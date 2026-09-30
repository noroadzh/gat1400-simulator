## ADDED Requirements

### Requirement: DeviceID MUST 是 20 位数字字符串

有效的 `Node.ID`（DeviceID）MUST 恰好 20 个字符且全部为数字 0–9。任何未通过该检查的输入 MUST 导致 `Node.Sanity()` 返回 `ErrInvalidDeviceID`。

#### Scenario: 1 个字符的输入
WHEN 对 ID 为 `"x"` 的 Node 调用 `Sanity()`
THEN 错误 MUST 满足 `errors.Is(err, ErrInvalidDeviceID)`。

#### Scenario: 20 位全数字输入
WHEN 对 ID 为 `"41000000005030312222"` 的 Node 调用 `Sanity()`
THEN 错误 MUST 为 `nil`。

### Requirement: Resource Kind MUST 是七个已知值之一

`Resource.Kind` 的值 MUST 是以下之一：`Person`、`Face`、`Vehicle`、`Plate`、`NonMotorVehicle`、`Image`、`Object`。其他任何值 MUST 导致校验返回 `ErrInvalidKind`。

### Requirement: HTTP Status 码 MUST 映射到 5 值枚举

`response.Code` MUST 是 `0`（OK）、`1`（Invalid）、`2`（NotFound）、`3`（Unauthorized）、`4`（ServerError）之一。对应的 `ResponseStatus.StatusString` MUST 为 `"OK"`、`"INVALID"`、`"NOTFOUND"`、`"UNAUTHORIZED"` 或 `"SERVER_ERROR"`。

### Requirement: DeviceID 生成器 MUST 产出唯一 ID

每次调用 `Generator.DeviceID()` MUST 返回唯一的 20 位字符串。生成器 MUST 支持多个 goroutine 并发使用。

#### Scenario: 1000 个并发调用
WHEN 1000 个 goroutine 各调用一次 `DeviceID()`
THEN 结果集合 MUST 包含 1000 个互不相同的字符串。

### Requirement: DeviceID 生成器 MUST 遵循 `8+2+2+2+6` 布局

生成的 DeviceID MUST 可分解为：
- 字符 `[0..8]` → SiteCode
- 字符 `[8..10]` → IndustryCode（mod 100）
- 字符 `[10..12]` → TypeCode（例如 01）
- 字符 `[12..14]` → SubTypeCode（例如 01）
- 字符 `[14..20]` → 序列号

#### Scenario: SiteCode 41000000、IndustryCode 30、Seq 1
WHEN 用 `NewGenerator(41000000, 30)` 构造生成器并调用 `DeviceID()`
THEN 结果 MUST 以 `"41000000301"` 开头，且总长度为 20 字符。

### Requirement: Nonce 必须是密码学安全的随机值

`Generator.Nonce()` MUST 返回 32 个字符的小写十六进制字符串。输出 MUST 以 `crypto/rand` 作为熵源——不得使用 `time.Now()` 或其他可预测来源。

### Requirement: UUID 必须遵循 RFC 4122 v4

`Generator.UUID()` MUST 返回 36 字符的规范 UUID 集合形式（`8-4-4-4-12` 带连字符）。版本位 MUST 为 `4`，变体位 MUST 在 `8-b` 范围内。

### Requirement: SubscribeID 必须是 12 个大写字母数字字符

`Generator.SubscribeID()` MUST 恰好返回 12 个来自 `A–Z` 与 `0–9` 字母表的字符。

---

## ADDED Architecture Decisions

### Decision: 领域层零外部依赖

`internal/domain/` 下的包 MUST NOT 导入标准库之外的任何包。这保证了纯净性并防止与基础设施意外耦合。

### Decision: 校验返回哨兵错误

所有 `Sanity()` 方法 MUST 返回命名的哨兵错误之一（`ErrMissingID` 等），让调用方能用 `errors.Is` 做分派。

### Decision: ResourceKind 是穷举的

编译器 MUST 能通过对 `Kind` 的 switch 检测未处理的分支。新增 Kind MUST 要求同步更新 switch——这防止出现无声的协议缺口。