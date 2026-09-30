## ADDED Requirements

### Requirement: Golden 样本必须覆盖全部四类路由族

Golden 样本集必须至少各含一条以下类别的交互：
- System 路由（Register、Keepalive、Time）
- Collection 路由（至少 Person 的 POST + GET）
- Cascade 路由（至少 Subscribe 的 POST + GET）
- Catalog 路由（至少 APE 的 GET）

#### Scenario: 缺失 APE catalog golden 样本
WHEN 测试套件以 `--golden` 标志运行
THEN 每个 `test/contract/golden/*.json` 文件中的 catalog 步骤都必须通过
AND 缺失 catalog 样本绝不能导致测试被静默跳过。

### Requirement: E2E 测试必须使用真实 TCP socket

E2E 测试绝不能 mock HTTP 层。服务器必须通过 `net.Listen` 或 `httptest.Server` 绑定到真实端口。客户端必须通过真实 TCP 连接。

#### Scenario: 真实 socket 往返
WHEN `TestProtocolE2E_RegisterAndPush` 运行
THEN 客户端必须向服务器发送 TCP SYN
AND 响应必须通过同一条 TCP 连接返回
AND 绝不能使用任何 mock 或假的 HTTP transport。

### Requirement: E2E 测试必须验证端到端正确性

每个 E2E 测试必须在 HTTP 状态码之外断言至少一个可观察效果：
- 服务器的内存状态（ResourceStore、NodeService）反映了客户端请求
- CaptureStore 中为每个请求记录了一条条目
- Register 之后 NodeService 将该节点报告为在线

### Requirement: Golden 样本运行器必须把不一致报告为失败

当一个 golden 样本被加载并回放时，响应状态码或关键 body 字段中的任何不一致都必须导致 `t.Fatal`。

#### Scenario: 状态码不一致
WHEN golden 样本期望状态 200 但服务器返回 401
THEN 测试必须失败，并给出描述该不一致的信息。

### Requirement: 所有内部包必须达到 ≥90% 测试覆盖率

`internal/` 下的每个包都必须达到 `go test -cover` 报告的至少 90% 行覆盖率。覆盖率报告必须在 CI 中发布。

### Requirement: 文档必须与代码保持同步

当一个 change 引入了新路由、新配置键或新 YAML 字段时，对应的文档必须在同一个 change 内更新。

---

## ADDED Architecture Decisions

### Decision: Golden 样本使用受版本控制的 JSON

Golden 样本是提交到仓库的纯 JSON 文件。这使它们易于评审、diff，并在协议语义变化时易于更新。

### Decision: E2E 测试放在 test/e2e/

E2E 测试位于顶层 `test/` 目录，而不是 `internal/` 内部，因为它们跨越多个层级，不应被 `internal/` 的包边界规则约束。

### Decision: 文档放在 docs/ 而不是源码树内

文档由 OpenSpec 规范生成，存放在 `docs/` 中，并作为项目 README 的一部分对外提供。实现细节保留在源码注释里。