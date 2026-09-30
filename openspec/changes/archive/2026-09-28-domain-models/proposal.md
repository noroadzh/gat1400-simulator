# 提案：领域模型

## 状态
已归档 —— 全部领域类型已定义并通过测试。

## 背景动机

GA/T 1400 协议围绕一组少量持久化实体展开——节点（Nodes）、资源（Resources）、订阅（Subscriptions）、布控（Dispositions）——以及统一的响应信封。在构建任何 HTTP 层或场景引擎之前，我们需要稳定且经过充分测试的领域类型来编码协议不变量：
- DeviceID 是布局为 `8+2+2+2+6` 的 20 位数字字符串
- 资源 Kind（Person、Face、Vehicle、Plate、Image、Object、NonMotorVehicle）是穷举枚举
- HTTP Status 码（0/1/2/3/4）映射到协议层状态（OK/Invalid/NotFound/Unauthorized/ServerError）

## 目标

- `internal/domain/` 中的领域模型，零外部依赖
- DeviceID、UUID、Nonce 的 ID 生成器
- 每个实体上的校验函数（`Sanity()`），返回哨兵错误
- 覆盖全部有效/无效组合的完整单元测试

## 非目标

- 不做持久化（由后续 change 的 adapter-storage 负责）
- 不做 HTTP DTO（由 adapter-httpapi 负责）
- 不做 UI 展示（由 ui 服务端绑定负责）

## 待定问题

无。