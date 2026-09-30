# Design: complete-gat1400-protocol

## Context

当前实现已经在功能层覆盖了 51/53 个 capability 要求（详见 explore 报告）。剩余差距集中在 3 处行为细节 + 1 处文档同步：

| 差距 | 位置 | 当前状态 | 期望状态 |
|---|---|---|---|
| Nonce 重放检测 | `httpapi/system.go:55` `verifyAuthorization` | 返回 `nil`（空桩） | 调用 `NonceStore.Consume` |
| AutoStart API 形状 | `scenario/engine.go:77` | 仅有 `Start()` | 同时暴露 `Start()` 与 `AutoStart()` |
| Cascade 双形态删除 | `httpapi/cascade.go:195-205` | 仅 `DELETE /:id` | 同时支持 `POST` body 含 `SubscribeIDList`/`DeleteOperate` |
| 文档同步 | `README.md` `internal/adapter/wire/` 文件清单 + `docs/PROTOCOL.md` §6 | `digest.go` 不存在；无重放响应字段示例 | 修正为 `client.go`+`uac.go`+`json.go`；新增重放响应字段 |
| 代码注释稀疏 | `system.go`/`cascade.go`/`collection.go`/`catalog.go`/`client.go`/`uac.go`/`engine.go`/`nonce_store.go` | 缺 GA/T 1400.4 节号与协议字段语义 | 补齐节号与字段语义 |

约束：
- 不引入新依赖；使用既有的 `crypto/subtle`、`modernc.org/sqlite`
- 所有改动向后兼容（仅补齐实现 + 增加新方法，不删除任何已部署端点）
- 遵循 `internal/` 分层纪律：`adapter/` 依赖 `domain/` 与 `app/`，不得反向依赖

## Approach

### 1. Nonce 重放检测（httpapi）

在 `httpapi/system.go` 中将 `verifyAuthorization(authHeader, method, uri)` 改造为真实实现：

```go
func (s *Server) verifyAuthorization(authHeader, method, uri string) error {
    if !strings.HasPrefix(authHeader, "Digest ") {
        return errors.New("auth: not a Digest header")
    }
    // 1. 解析 nonce 字段
    fields := parseDigestFields(authHeader)
    nonce, ok := fields["nonce"]
    if !ok {
        return errors.New("auth: missing nonce")
    }
    // 2. 调用 NonceStore.Consume；失败 ⇒ nonce 已消费或已过期
    if err := s.nonce.Consume(nonce); err != nil {
        return fmt.Errorf("nonce consume: %w", err)
    }
    return nil
}
```

关键设计点：
- 解析失败 ⇒ 401 挑战（中间件层处理）
- `NonceStore.Consume` 已实现 `RowsAffected=0` ⇒ `ErrNonceUnknown` 语义
- 不在 `verifyAuthorization` 内做密码比对（密码验证在客户端侧通过 mock 测试），服务端仅做 nonce 去重
- 由于既有实现是空桩，本变更的 verify 实际是「注册"消费过"的语义」而非「RFC 2617 全套」——这与既有 spec 中的"Nonce 重放 MUST 被拒绝"语义一致

### 2. AutoStart 方法（scenario）

在 `scenario/engine.go` 中增加 `AutoStart` 方法，复用 `Start` 内部逻辑：

```go
// AutoStart 根据 Scenario.Schedule.AutoStart 字段决定是否启动。
// 与 Start 的差别：Start 无条件启动；AutoStart 仅在 AutoStart=true 时启动。
func (e *Engine) AutoStart(ctx context.Context, s scenario.Scenario) error {
    if !s.Schedule.AutoStart {
        return nil
    }
    return e.Start(ctx, s)
}
```

关键设计点：
- 不修改 `Start` 的语义，保持向后兼容
- `Scenario.Schedule.AutoStart` 字段已存在于 `domain/scenario`（在 yaml 加载层）
- `AutoStart(false)` MUST 是幂等的 no-op，不产生 side effect

### 3. Cascade 双形态（httpapi）

在 `httpapi/cascade.go` 中：

**a)** `handleSubscribeCreate` 改造为同时支持创建与 body-删除：

```go
func (s *Server) handleSubscribeCreate(c echo.Context) error {
    var body map[string]any
    if err := c.Bind(&body); err != nil {
        return c.JSON(http.StatusBadRequest, response.Error(...))
    }
    // body 删除分支：POST 含 SubscribeIDList 或 DeleteOperate
    if ids, ok := s.extractDeleteIDs(body, "SubscribeIDList"); ok {
        for _, id := range ids { s.subRepo.delete(id) }
        return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
    }
    // 创建分支（现有逻辑）
    ...
}

// extractDeleteIDs 抽取 body 中可能的 DeleteOperate 或 顶层 List 字段。
func (s *Server) extractDeleteIDs(body map[string]any, listKey string) ([]string, bool) {
    if arr, ok := body[listKey].([]any); ok {
        out := make([]string, 0, len(arr))
        for _, v := range arr { if s, ok := v.(string); ok { out = append(out, s) } }
        return out, true
    }
    if op, ok := body["DeleteOperate"].(map[string]any); ok {
        if arr, ok := op[listKey].([]any); ok {
            out := make([]string, 0, len(arr))
            for _, v := range arr { if s, ok := v.(string); ok { out = append(out, s) } }
            return out, true
        }
    }
    return nil, false
}
```

**c)** `handleDispositionCreate` 同理处理 `DispositionIDList` body 删除。

**d)** 保留 `DELETE /VIID/Subscribes/:id` 与 `DELETE /VIID/Dispositions/:id` 端点不动（向后兼容）。

关键设计点：
- 通过 `extractDeleteIDs` 统一处理顶层与嵌套 `DeleteOperate` 两种 body 形式
- 创建分支与删除分支通过 ID 字段是否为空区分；任一删除字段存在即走删除路径
- 不修改 subscribeRepo 的现有 CRUD 方法，仅在 entry 层做路由分发

### 4. 代码注释强化（system/）

在每个 handle 函数 docstring 顶部添加 GA/T 1400.4 节号。覆盖：

| 文件 | 函数 | 节号 |
|---|---|---|
| system.go | handleRegister | §5.1 Register |
| system.go | handleUnRegister | §5.1 UnRegister |
| system.go | handleKeepalive | §5.3 Keepalive |
| system.go | handleTime | §5.3 Time |
| collection.go | makeCollectionHandler/makeDataListHandler/... | §5.2 Collection |
| cascade.go | handleSubscribeCreate/Update/Delete/List | §5.4 Subscribe |
| cascade.go | handleNotification* | §5.4 SubscribeNotification |
| cascade.go | handleDisposition* | §5.4 Disposition |
| catalog.go | handleAPEs/APSs/Tollgates/Lanes | §5.5 Catalog |
| scenario/engine.go | fireRegisters | §5.1 Register |
| scenario/engine.go | runKeepalive | §5.3 Keepalive |
| scenario/engine.go | dispatchNotifications | §5.4 SubscribeNotification |
| wire/client.go | do | RFC 2617 §3 |
| wire/client.go | buildAuthorization | RFC 2617 §3.2.2.1 |
| wire/uac.go | Register | §5.1 |
| wire/uac.go | SubscribeCreate / NotificationPush / DispositionCreate | §5.4 |
| adapter/storage/nonce_store.go | Issue / Consume | RFC 2617 §3 + 重放保护语义 |

### 5. 文档同步

- `README.md`：在 `internal/adapter/wire/` 文件清单中将 `digest.go`/`digest_test.go` 替换为 `client.go`+`uac.go`+`json.go`；`openspec/specs/` 子目录描述按 10 个 capability 重组
- `docs/PROTOCOL.md §6`：增加重放响应字段示例（401 + 新 nonce + 提示"previous nonce already consumed"）
- `docs/PROTOCOL.md §4.3`：Cascade 路由表增加双形态说明
- `docs/ARCHITECTURE.md §三`：适配层描述补齐 OutboundDispatcher 与 engine.AutoStart 段落
- `CHANGELOG.md`：在 "Unreleased" 节增加 entry

## Risks & Mitigations

| 风险 | 缓解措施 |
|---|---|
| `verifyAuthorization` 真实化后既有的"任意 Authorization 通过"测试失败 | 一并修改 `system_test.go` 使用真实的 Digest 头（带合法 nonce） |
| `AutoStart(false)` 误调 ⇒ 影响现有自动加载流程 | 在 `cmd/gat1400-sim/main.go` 检查所有 `Start` 调用方，确保显式 `AutoStart` 调用 |
| Cascade 双形态可能误把合法创建 body 当成删除 | `extractDeleteIDs` 严格只在 `SubscribeIDList` / `DispositionIDList` / `DeleteOperate` 字段存在时进入删除路径；空 Body 走原有创建分支 |
| 文档同步遗漏导致 spec 状态不符 | 在 tasks.md 中显式列出每个 doc 文件的同步条目，并跑 `openspec validate --specs` |
| 注释过长影响代码可读性 | 注释仅 1-2 行，使用 `// §5.X ...` 简短格式 |