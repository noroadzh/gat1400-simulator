# Tasks

## 1. wire 子类型高层 API（wire-system-client / wire-cascade-client）

- [x] 1.1 在 `wire/` 新建 `uac.go`，定义 `System` struct（持有 `*Client`）和 `Cascade` struct，实现：
  - `System.Register(ctx, baseURL, obj RegisterObject) error`
  - `System.UnRegister(ctx, baseURL, deviceID) error`
  - `System.Keepalive(ctx, baseURL, deviceID) error`
  - `System.ServerTime(ctx, baseURL) (string, error)`
  - `Cascade.SubscribeCreate(ctx, baseURL, body map[string]any) (string, error)`
  - `Cascade.SubscribeDelete(ctx, baseURL, subscribeID) error`
  - `Cascade.SubscribeList(ctx, baseURL) ([]any, error)`
  - `Cascade.DispositionCreate(ctx, baseURL, body map[string]any) (string, error)`
  - `Cascade.SubscribeNotificationPush(ctx, baseURL, body map[string]any) error`
  - `Client.System() *System` 和 `Client.Cascade() *Cascade`
  - 验证：`go test ./internal/adapter/wire/... -run TestUAC -v` 通过

- [x] 1.2 在 `wire/uac_test.go` 中编写 httptest 测试，覆盖：
  - Register 首次 401 → 自动重试成功
  - Keepalive 无鉴权端点成功
  - ServerTime 返回含 "T" 的时间字符串
  - Cascade SubscribeCreate → SubscribeList → SubscribeDelete 全链路
  - 验证：`go test ./internal/adapter/wire/... -v` 全部通过

## 2. OutboundDispatcher 扩展（dispatcher-extend）

- [x] 2.1 在 `scenario/dispatcher.go` 的 `OutboundDispatcher` 上新增 6 个方法：
  - `DispatchRegister(ctx, target, body) error` → POST `/VIID/System/Register`
  - `DispatchKeepalive(ctx, target) error` → POST `/VIID/System/Keepalive`（body 为 nil）
  - `DispatchUnregister(ctx, target) error` → POST `/VIID/System/UnRegister`
  - `DispatchSubscribe(ctx, target, body) error` → POST `/VIID/Subscribes`
  - `DispatchSubscribeNotification(ctx, target, body) error` → POST `/VIID/SubscribeNotifications`
  - `DispatchDisposition(ctx, target, body) error` → POST `/VIID/Dispositions`
  - 验证：`go build ./internal/adapter/scenario/...` 无编译错误

- [x] 2.2 将所有 dispatch 方法的抓包记录逻辑提取为 `record()` helper（void 返回），替换各方法中重复的 recorder 块
  - 验证：`go vet ./internal/adapter/scenario/...` 无错误

- [x] 2.3 在 `scenario/dispatcher_test.go` 中补充测试：验证 DispatchKeepalive 发到正确 URI、DispatchUnregister 失败不 panic、recorder 为 nil 时正常工作
  - 验证：`go test ./internal/adapter/scenario/... -run Dispatch -v` 通过

## 3. Engine 生命周期改造（engine-lifecycle）

- [x] 3.1 将 `Engine.dispatcher` 字段类型从 `Dispatcher` 接口改为 `*OutboundDispatcher`，更新 `NewEngine` 构造函数签名
  - 验证：`go build ./...` 无编译错误

- [x] 3.2 `materialise` 末尾：对每个 device 节点，在 `ExceptionSpec.RegisterDropRate` 判定不跳过时，异步调用 `d.DispatchRegister`；注册失败仅 warn log，不阻断 materialise
  - 验证：e2e 测试中节点上线后服务端收到 Register 请求

- [x] 3.3 `Engine.run` 中新增 keepalive goroutine：每个 device 节点启动一个 goroutine，按 `keepaliveInterval + jitter` 周期调用 `d.DispatchKeepalive`，goroutine 在 ctx cancel 时退出
  - 验证：`go test ./internal/adapter/scenario/... -run TestKeepalive -v` 通过（需先写测试）

- [x] 3.4 `Engine.Stop` 中，在 `h.cancel()` 之前对所有 device 节点调用 `d.DispatchUnregister`；失败仅 log 不阻断 Stop
  - 验证：`go test ./internal/adapter/scenario/... -run TestStop -v` 通过

- [x] 3.5 修复 `postNotification`：当 `SubscriptionSpec.NotificationURL` 非空时，通过 `d.DispatchSubscribeNotification` POST 通知到端点，而非仅 log
  - 验证：`go build ./...` 无错误

## 4. 配置透传（config-wireup）

- [x] 4.1 `config.Config` 新增 `KeepaliveInterval time.Duration` 字段，`Default()` 返回默认值 `30 * time.Second`，YAML 键为 `keepaliveInterval`
  - 验证：写 `testdata/keepalive.yaml` 含 `keepaliveInterval: "15s"`，`config.Load("testdata/keepalive.yaml")` 返回 `KeepaliveInterval == 15s`

- [x] 4.2 `configs/default.yaml` 新增一行 `keepaliveInterval: "30s"`

- [x] 4.3 `cmd/gat1400-sim/main.go` 将 `cfg.KeepaliveInterval` 透传给 `NewEngine`
  - 验证：`go build ./cmd/...` 成功

## 5. 端到端验证（e2e-test）

- [x] 5.1 新建 `test/e2e/uac_lifecycle_e2e_test.go`，用真实 TCP socket（httptest.NewServer）验证完整生命周期：Register → Keepalive → Persons push → UnRegister；以及 Cascade Subscribe → List → Disposition → Delete 全链路
  - 验证：`go test ./test/e2e/... -run 'TestUAC' -v` 全部通过

- [x] 5.2 全工程回归测试：`go test ./... -count=1 -timeout=120s`，所有包必须通过
  - 验证：命令返回 exit code 0
