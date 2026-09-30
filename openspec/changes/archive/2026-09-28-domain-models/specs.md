## ADDED Requirements

### Requirement: DeviceID MUST be a 20-digit string

A valid `Node.ID` (DeviceID) MUST be exactly 20 characters, all digits 0–9. Any input that fails this check MUST cause `Node.Sanity()` to return `ErrInvalidDeviceID`.

#### Scenario: 1-character input
WHEN `Sanity()` is called on a Node whose ID is `"x"`
THEN the error MUST satisfy `errors.Is(err, ErrInvalidDeviceID)`.

#### Scenario: 20-character all-numeric input
WHEN `Sanity()` is called on a Node whose ID is `"41000000005030312222"`
THEN the error MUST be `nil`.

### Requirement: Resource Kind MUST be one of seven known values

A `Resource.Kind` value MUST equal one of: `Person`, `Face`, `Vehicle`, `Plate`, `NonMotorVehicle`, `Image`, `Object`. Any other value MUST cause validation to return `ErrInvalidKind`.

### Requirement: HTTP Status codes MUST map to a 5-value enum

`response.Code` MUST be one of `0` (OK), `1` (Invalid), `2` (NotFound), `3` (Unauthorized), `4` (ServerError). The corresponding `ResponseStatus.StatusString` MUST be `"OK"`, `"INVALID"`, `"NOTFOUND"`, `"UNAUTHORIZED"`, or `"SERVER_ERROR"`.

### Requirement: DeviceID generator MUST produce unique IDs

Each call to `Generator.DeviceID()` MUST return a unique 20-digit string. The generator MUST be safe for concurrent use from multiple goroutines.

#### Scenario: 1000 concurrent calls
WHEN 1000 goroutines each call `DeviceID()` once
THEN the resulting set MUST contain 1000 distinct strings.

### Requirement: DeviceID generator MUST follow layout `8+2+2+2+6`

A generated DeviceID MUST decompose as:
- Characters `[0..8]` → SiteCode
- Characters `[8..10]` → IndustryCode (mod 100)
- Characters `[10..12]` → TypeCode (e.g., 01)
- Characters `[12..14]` → SubTypeCode (e.g., 01)
- Characters `[14..20]` → Sequence number

#### Scenario: SiteCode 41000000, IndustryCode 30, Seq 1
WHEN the generator is constructed with `NewGenerator(41000000, 30)` and `DeviceID()` is called
THEN the result MUST start with `"41000000301"` and be 20 chars in total.

### Requirement: Nonce must be cryptographically random

`Generator.Nonce()` MUST return a 32-character lowercase hexadecimal string. The output MUST use `crypto/rand` as the entropy source — not `time.Now()` or other predictable sources.

### Requirement: UUID must follow RFC 4122 v4

`Generator.UUID()` MUST return a 36-character string in the canonical UUID form (`8-4-4-4-12` with dashes). The version digit MUST be `4` and the variant digit MUST be in `8-b`.

### Requirement: SubscribeID must be 12 uppercase alphanumeric chars

`Generator.SubscribeID()` MUST return exactly 12 characters from the alphabet `A–Z` and `0–9`.

---

## ADDED Architecture Decisions

### Decision: Domain layer has zero external dependencies

`internal/domain/` packages MUST NOT import any package outside the standard library. This enforces purity and prevents accidental coupling to infrastructure.

### Decision: Validation returns sentinel errors

All `Sanity()` methods MUST return one of the named sentinel errors (`ErrMissingID`, etc.) so callers can use `errors.Is` for dispatch.

### Decision: ResourceKind is exhaustive

The compiler MUST be able to detect unhandled cases via a switch over `Kind`. New Kinds MUST require an update to the switch — this prevents silent protocol gaps.