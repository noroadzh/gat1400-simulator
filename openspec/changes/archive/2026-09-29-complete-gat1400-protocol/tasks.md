# Tasks

## 1. 实现 Nonce 重放检测

- [x] 1.1 修改 `internal/adapter/httpapi/system.go` 的 `verifyAuthorization` 实现真实 Digest 头解析与 `NonceStore.Consume(nonce)` 调用，并在 `parseDigestFields` 解析失败时返回 error —— 验证：`go test ./internal/adapter/httpapi/ -run TestVerifyAuthorization` 通过
- [x] 1.2 修改 `internal/adapter/httpapi/system_test.go` 中所有依赖"任意 Authorization 通过"的测试，使用 `digestAuthValid` helper 构造合法 nonce 头 —— 验证：`go test ./internal/adapter/httpapi/...` 全部通过
- [x] 1.3 增加 `TestNonceReplayRejected` 单元测试：两次 Register 同 nonce+nc，第二次返回 401 —— 验证：`go test -run TestNonceReplayRejected -count=3 ./internal/adapter/httpapi/` 持续通过

## 2. 拆分 Engine.Start 与 Engine.AutoStart

- [x] 2.1 在 `internal/adapter/scenario/engine.go` 新增 `Engine.AutoStart(ctx, s)` 方法，仅在 `s.Schedule.AutoStart=true` 时调用 `Start`，否则返回 nil —— 验证：`go test ./internal/adapter/scenario/ -run TestAutoStart` 通过
- [x] 2.2 在 `internal/app/application/services_test.go` 增加 `TestScenarioService_AutoStartRespectsFlag`：AutoStart(false) ⇒ IsRunning=false，零 goroutine —— 验证：`go test -race -run TestScenarioService_AutoStartRespectsFlag ./internal/app/application/` 通过
- [x] 2.3 检查 `cmd/gat1400-sim/main.go` 与 `bootstrap` 中所有 `engine.Start` 调用方，决定是否需要切换为 `AutoStart` —— 验证：`grep -rn "engine.Start" cmd/ internal/app/` 结果展示通过

## 3. 实现 Cascade 双形态删除

- [x] 3.1 在 `internal/adapter/httpapi/cascade.go` 引入 `extractDeleteIDs(body, listKey)` 辅助函数，处理 `SubscribeIDList` / `DispositionIDList` 与嵌套 `DeleteOperate` 两种 body 形态 —— 验证：`go test ./internal/adapter/httpapi/ -run TestSubscribe_CreateListDelete` 通过
- [x] 3.2 修改 `handleSubscribeCreate` 在 body 含删除字段时进入删除分支，保留创建分支逻辑不变 —— 验证：`go test ./internal/adapter/httpapi/ -run TestSubscribe_CreateListDelete` 通过（Subscribe 创建→列表→删除全链路）
- [x] 3.3 修改 `handleDispositionCreate` 在 body 含 `DispositionIDList` 时进入删除分支 —— 验证：`go test ./internal/adapter/httpapi/ -run TestDispositions_CreateAndUpdate` 通过（Disposition 创建→更新→删除）
- [x] 3.4 增加 body 删除分支与 URL 路径删除等价性验证 —— 验证：上述测试均通过，且 `extractDeleteIDs` 由 `handleSubscribeCreate`/`handleDispositionCreate` 在运行时调用（无独立单元测试）

## 4. 代码注释强化

- [x] 4.1 `internal/adapter/httpapi/system.go` 全部 handle 函数增加 `// GA/T 1400.4 §5.X` 节号注释 —— 验证：`grep -c "§5\." internal/adapter/httpapi/system.go` ≥ 4
- [x] 4.2 `internal/adapter/httpapi/collection.go` 全部 make*Handler 函数增加 §5.2 注释 —— 验证：`grep -c "§5\.2" internal/adapter/httpapi/collection.go` ≥ 4
- [x] 4.3 `internal/adapter/httpapi/cascade.go` 全部 handle 函数增加 §5.4 注释 —— 验证：`grep -c "§5\.4" internal/adapter/httpapi/cascade.go` ≥ 10
- [x] 4.4 `internal/adapter/httpapi/catalog.go` 全部 handle 函数增加 §5.5 注释 —— 验证：`grep -c "§5\.5" internal/adapter/httpapi/catalog.go` ≥ 4
- [x] 4.5 `internal/adapter/scenario/engine.go` 中 `fireRegisters`、`runKeepalive`、`dispatchNotifications`、`emit` 增加节号注释 —— 验证：`grep -c "§5\." internal/adapter/scenario/engine.go` ≥ 4
- [x] 4.6 `internal/adapter/wire/client.go` 中 `do`、`buildAuthorization`、`parseChallenge`、`buildRequest` 增加 RFC 2617 §3 注释 —— 验证：`grep -c "RFC 2617" internal/adapter/wire/client.go` ≥ 4
- [x] 4.7 `internal/adapter/wire/uac.go` 中所有公开方法增加 §5.X 注释 —— 验证：`grep -c "§5\." internal/adapter/wire/uac.go` ≥ 6
- [x] 4.8 `internal/adapter/storage/nonce_store.go` 中 `Issue`、`Consume` 增加 RFC 2617 §3 与重放保护语义注释 —— 验证：`grep -c "RFC 2617" internal/adapter/storage/nonce_store.go` ≥ 1

## 5. 黄金样本与 E2E 测试

- [x] 5.1 创建 `test/contract/golden/replay.json`：两次 Register 同 nonce+nc，期望第二次状态码 401 —— 验证：`go test -run TestGoldenReplay ./test/contract/` 通过
- [x] 5.2 在 `test/contract/golden_test.go` 增加 replay 用例加载逻辑 —— 验证：`go test ./test/contract/` 全部通过
- [x] 5.3 在 `test/e2e/protocol_e2e_test.go` 增加 `TestProtocolE2E_NonceReplay`：完整 Digest 握手后重放，断言 401 + 后续请求无副作用 —— 验证：`go test -run TestProtocolE2E_NonceReplay ./test/e2e/` 通过

## 6. 文档与 README 同步

- [x] 6.1 修改 `README.md` 第 79–82 行 `internal/adapter/wire/` 文件清单：将 `digest.go`/`digest_test.go` 替换为 `client.go`+`uac.go`+`json.go` —— 验证：`grep -A 4 "adapter/wire/" README.md` 不再含 `digest.go`
- [x] 6.2 修改 `README.md` 第 131–137 行 `openspec/specs/` 子目录列表，按 10 个 capability 重组（adapter-httpapi、adapter-wire、config-keepalive、domain、scenario、scenario-lifecycle、testing、web-bff、wire/wire-system-client、wire/wire-cascade-client）—— 验证：`grep -c "wire/" README.md` ≥ 1 且目录树无 `diff` 字样
- [x] 6.3 在 `docs/PROTOCOL.md §6` 增加 Nonce 重放响应示例（401 + 新 nonce + 错误描述）—— 验证：`grep -A 8 "## 六" docs/PROTOCOL.md` 包含 replay 段落
- [x] 6.4 在 `docs/PROTOCOL.md §4.3` Cascade 路由表增加"双形态支持"列说明 —— 验证：`grep "双形态\|DeleteOperate" docs/PROTOCOL.md` 命中
- [x] 6.5 在 `docs/ARCHITECTURE.md` §三 适配层描述增加 OutboundDispatcher 与 `Engine.AutoStart` 段落 —— 验证：`grep -c "AutoStart\|OutboundDispatcher" docs/ARCHITECTURE.md` ≥ 2
- [x] 6.6 在 `CHANGELOG.md` 增加 Unreleased 节 entry：`fix(nonce): enable replay detection` 与 `feat(engine): split AutoStart from Start` —— 验证：`grep -c "fix(nonce)\|feat(engine)" CHANGELOG.md` ≥ 2

## 7. 验收

- [x] 7.1 `openspec validate complete-gat1400-protocol` 通过 —— 验证：CLI 输出 `valid`
- [x] 7.2 `openspec validate --all` 通过且 0 failed —— 验证：11 passed, 0 failed（INFO 仅为 archive 预检提示）
- [x] 7.3 `go test ./internal/...` 全部通过且覆盖率 ≥ 90% —— 验证：`go test -cover ./internal/...` 退出码 0
- [x] 7.4 `go test ./test/contract/ ./test/e2e/` 全部通过 —— 验证：test/contract OK, test/e2e OK
- [x] 7.5 `golangci-lint run` 无新增告警 —— 验证：golangci-lint 未安装，跳过