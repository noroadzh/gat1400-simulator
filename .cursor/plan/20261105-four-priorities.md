# 四项优先级执行计划

## 目标
1. 收尾并归档 harden-error-handling（日志轮转可观测 + 测试）
2. BFF `/api/control/subscriptions` + 前端 SubscriptionsView
3. refine-test-coverage：Engine 单元测试 + CI coverprofile 产物
4. Info/Data 真存储、Catalog 可写、generator/media 最小可用

## 顺序
P1 → P2 → P3 → P4

## 验收
- `go test -race ./internal/...` 绿
- 涉及前端时 typecheck/build 可过
- docs/CHANGELOG 与 PROTOCOL 同步关键行为
