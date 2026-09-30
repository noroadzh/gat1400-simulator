# 领域模型 — 主规格

> 合并来源：`archive/domain-models/specs.md`

## Purpose

定义核心领域模型与生成器的约束：DeviceID 格式校验、资源 Kind 枚举、HTTP 响应码映射、DeviceID 生成器（8+2+2+2+6 布局）、Nonce 与 UUID 生成器规范。领域层零外部依赖。

## Requirements

### Requirement: DeviceID MUST be 20 位十进制字符串

合法的 `Node.ID`（即 DeviceID）MUST 为恰好 20 个字符，且全部为 0–9 的数字。任何不符合此约束的输入，MUST 使 `Node.Sanity()` 返回 `ErrInvalidDeviceID`。

#### Scenario: 输入 1 个字符

- **WHEN** 调用 `Sanity()`，节点 ID 为 `"x"`
- **THEN** 返回的错误 MUST 可通过 `errors.Is(err, ErrInvalidDeviceID)` 判定。

#### Scenario: 输入 20 位全数字

- **WHEN** 调用 `Sanity()`，节点 ID 为 `"41000000005030312222"`
- **THEN** 返回的错误 MUST 为 `nil`。

### Requirement: 资源 Kind MUST 为已知的 7 个枚举值之一

`Resource.Kind` MUST 等于以下之一：`Person`、`Face`、`Vehicle`、`Plate`、`NonMotorVehicle`、`Image`、`Object`。任何其它值 MUST 使校验返回 `ErrInvalidKind`。

#### Scenario: 合法 Kind 校验通过

- **WHEN** `Resource.Kind` 设为 `"Face"`
- **THEN** 校验 MUST 返回 `nil`。

### Requirement: HTTP 状态码 MUST 映射为 5 值枚举

`response.Code` MUST 取值为 `0`（OK）/ `1`（Invalid）/ `2`（NotFound）/ `3`（Unauthorized）/ `4`（ServerError）。对应的 `ResponseStatus.StatusString` MUST 为 `"OK"` / `"INVALID"` / `"NOTFOUND"` / `"UNAUTHORIZED"` / `"SERVER_ERROR"`。

#### Scenario: 5 值枚举映射

- **WHEN** 传入 `response.Code = 0`
- **THEN** `StatusString` MUST 等于 `"OK"`。

### Requirement: DeviceID 生成器 MUST 输出唯一 ID

每次调用 `Generator.DeviceID()` MUST 返回一个唯一的 20 位字符串。生成器 MUST 适用于多 goroutine 并发调用。

#### Scenario: 1000 个并发调用

- **WHEN** 1000 个 goroutine 各自调用一次 `DeviceID()`
- **THEN** 结果集合 MUST 包含 1000 个互不相同的字符串。

### Requirement: DeviceID 生成器 MUST 遵循 `8+2+2+2+6` 布局

生成的 DeviceID MUST 按以下规则切分：
- `[0..8]` 位 → SiteCode（区划码）
- `[8..10]` 位 → IndustryCode（行业代码，模 100）
- `[10..12]` 位 → TypeCode（设备类型）
- `[12..14]` 位 → SubTypeCode（子类型）
- `[14..20]` 位 → Sequence（序号）

#### Scenario: SiteCode 41000000、IndustryCode 30、Seq 1

- **WHEN** 构造生成器 `NewGenerator(41000000, 30)` 并调用 `DeviceID()`
- **THEN** 结果 MUST 以 `"41000000301"` 开头，总长度为 20。

### Requirement: Nonce MUST 由密码学安全随机源生成

`Generator.Nonce()` MUST 返回 32 位小写十六进制字符串。实现 MUST 使用 `crypto/rand` 作为熵源——严禁使用 `time.Now()` 等可预测源。

#### Scenario: Nonce 格式验证

- **WHEN** 调用 `Generator.Nonce()`
- **THEN** 结果 MUST 为 32 位小写十六进制字符（字符集 `0-9a-f`）。

### Requirement: UUID MUST 符合 RFC 4122 v4

`Generator.UUID()` MUST 返回 36 位的标准 UUID 形式（`8-4-4-4-12`，含连字符）。版本位 MUST 为 `4`，变体位 MUST 落在 `8-b` 范围。

#### Scenario: UUID 版本位为 4

- **WHEN** 调用 `Generator.UUID()`
- **THEN** 第 14 位 MUST 为 `"4"`（版本 4）。

### Requirement: SubscribeID MUST 为 12 位大写字母数字

`Generator.SubscribeID()` MUST 返回恰好 12 个字符，字符集为 `A–Z` 与 `0–9`。

#### Scenario: SubscribeID 格式验证

- **WHEN** 调用 `Generator.SubscribeID()`
- **THEN** 结果 MUST 恰好为 12 个字符，且字符集在 `A-Z0-9` 范围内。

---

## ADDED Architecture Decisions

### Decision: 领域层零外部依赖

`internal/domain/` 包 MUST NOT 引入任何标准库之外的包。该规则强制领域层纯粹性，避免与基础设施耦合。

### Decision: 校验返回哨兵错误

所有 `Sanity()` 方法 MUST 返回具名哨兵错误（如 `ErrMissingID`），调用方通过 `errors.Is` 分发处理。

### Decision: ResourceKind 必须穷举

编译器 MUST 能通过对 `Kind` 的 switch 检测遗漏分支。新增 Kind MUST 同步更新 switch——避免协议契约被悄悄遗漏。