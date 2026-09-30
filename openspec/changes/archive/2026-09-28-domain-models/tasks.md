## 任务

### 任务：定义 Node、Role、Capability、Status

- [x] 创建 `internal/domain/node/node.go`
- [x] 定义 `Node` 结构体及其所有字段
- [x] 定义 `Role`、`Capability`、`Status` 枚举
- [x] 实现 `Sanity()`，返回哨兵错误

### 任务：定义 Resource、Kind、Metadata

- [x] 创建 `internal/domain/resource/resource.go`
- [x] 定义 `Resource` 结构体
- [x] 定义 `Kind` 枚举（7 个值）
- [x] 实现 `Sanity()`

### 任务：定义 Subscription、Disposition

- [x] 加入 `internal/domain/resource/`（或新建包）
- [x] 为两者实现 `Sanity()`

### 任务：定义 Scenario、ScheduleSpec、NodeSpec 等

- [x] 创建 `internal/domain/scenario/scenario.go`
- [x] 同时定义 `yaml`（场景加载）与 `json`（HTTP）的 struct tag

### 任务：定义 ResponseStatus、Code

- [x] 创建 `internal/domain/response/response.go`
- [x] 定义 `Code` 枚举（5 个值）与 `ResponseStatus` 结构体
- [x] 添加辅助函数 `OK()`、`Invalid()` 等

### 任务：实现 ID 生成器

- [x] 创建 `internal/domain/ids/ids.go`
- [x] 实现带互斥锁的 `Generator`，`NewGenerator(siteCode, industryCode uint32)`
- [x] 实现 `DeviceID()`、`UUID()`、`Nonce()`、`SubscribeID()`
- [x] Nonce 与 UUID 使用 `crypto/rand`；DeviceID 序列号使用单调计数器

### 任务：单元测试

- [x] `internal/domain/node/node_test.go` —— Sanity() 覆盖度、HasCapability()
- [x] `internal/domain/resource/resource_test.go` —— Kind 穷举性
- [x] `internal/domain/scenario/scenario_test.go` —— YAML/JSON 往返
- [x] `internal/domain/response/response_test.go` —— Code 映射
- [x] `internal/domain/ids/ids_test.go` —— DeviceID 布局、并发、Nonce 唯一性

### 验证

- `go test ./internal/domain/...` → exit 0
- 领域包覆盖率 ≥ 90%