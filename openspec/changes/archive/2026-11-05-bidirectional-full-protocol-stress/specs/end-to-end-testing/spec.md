# Spec Delta

## ADDED Requirements

### Requirement: 双向协议压测 MUST 有真实 TCP e2e

`test/e2e/` MUST 包含至少一条用例：本进程 UAS 监听真实端口，UAC（wire 客户端）经真实 TCP 完成 Register → 集合推送 → Catalog 上报 → DispositionNotification 主路径；禁止 mock `http.Handler` 替代 socket。

#### Scenario: 回环全协议主路径 e2e

- **WHEN** 运行该 e2e（空闲端口）
- **THEN** 上述各步骤 MUST 返回协议成功
- **AND** UAS 侧 MUST 可查询到对应持久化或内存对象

### Requirement: 契约黄金样本 MUST 覆盖新增 UAC 请求形

对新增高层客户端构造的代表性请求/响应（至少 Collection 分页列表与 DispositionNotification POST）MUST 有 `test/contract/golden/` 样本或等价契约断言。

#### Scenario: DispositionNotification 契约

- **WHEN** 契约测试回放 DispositionNotification 推送样本
- **THEN** 路径、Content-Type 与关键包装字段 MUST 与 PROTOCOL 一致
