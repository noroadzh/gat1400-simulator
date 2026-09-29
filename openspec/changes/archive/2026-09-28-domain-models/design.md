# Design: Domain Models

## Entities

### Node
Represents a single instance — a device (UAC) or a platform (UAS).

```go
type Role string
const (
    RoleDevice        Role = "device"        // UAC
    RolePlatformSmall Role = "platform-small" // UAS
    RolePlatformLarge Role = "platform-large" // UAS large
)

type Capability string
const (
    CapSystem     Capability = "system"     // Register / Keepalive / Time
    CapCollection Capability = "collection" // Persons, Faces, Vehicles, etc.
    CapCascade    Capability = "cascade"    // Subscribes, Notifications, Dispositions
)

type Status string
const (
    StatusOnline  Status = "online"
    StatusStopped Status = "stopped"
    StatusError   Status = "error"
)

type Node struct {
    ID           string       `json:"DeviceID"`
    Name         string       `json:"Name"`
    Role         Role         `json:"Role"`
    Status       Status       `json:"Status"`
    Capabilities []Capability `json:"Capabilities"`
    HTTPListen   string       `json:"HTTPListen"`
    Upstream     string       `json:"Upstream"`
    Tags         []string     `json:"Tags"`
    Metadata     Metadata     `json:"Metadata"`
    CreatedAt    time.Time    `json:"CreatedAt"`
    UpdatedAt    time.Time    `json:"UpdatedAt"`
    LastSeenAt   time.Time    `json:"LastSeenAt"`
}
```

Validation: `Sanity()` checks ID/Name non-empty, Role is a known value, Capabilities contains only known values.

### Resource

Represents any entity pushed from device to platform: Person, Face, Vehicle, Plate, NonMotorVehicle, Image, Object.

```go
type Kind string
const (
    KindPerson         Kind = "Person"
    KindFace           Kind = "Face"
    KindVehicle        Kind = "Vehicle"
    KindPlate          Kind = "Plate"
    KindNonMotorVehicle Kind = "NonMotorVehicle"
    KindImage          Kind = "Image"
    KindObject         Kind = "Object"
)

type Resource struct {
    ID          string    `json:"ID"`         // resource instance ID
    Kind        Kind      `json:"Kind"`
    SourceNode  string    `json:"SourceNode"` // node that produced the resource
    Timestamp   time.Time `json:"Timestamp"`
    Attributes  Metadata  `json:"Attributes"`
    DataSource  string    `json:"DataSource"` // URI or inline
    StoragePolicy string  `json:"StoragePolicy"` // hot, warm, cold
}
```

### Subscription / Disposition

```go
type Subscription struct {
    ID         string    `json:"SubscribeID"`
    Title      string    `json:"Title"`
    Resource   Kind      `json:"Resource"`
    Criteria   Metadata  `json:"Criteria"`
    StartTime  time.Time `json:"StartTime"`
    EndTime    time.Time `json:"EndTime"`
    Notifier   string    `json:"Notifier"`
    CreatedAt  time.Time `json:"CreatedAt"`
    UpdatedAt time.Time `json:"UpdatedAt"`
}

type Disposition struct {
    ID         string    `json:"DispositionID"`
    Title      string    `json:"Title"`
    Reason     string    `json:"Reason"`
    Targets    []Target  `json:"Targets"`
    CreatedAt  time.Time `json:"CreatedAt"`
}
```

### Scenario

```go
type Scenario struct {
    ID          string         `yaml:"id"`
    Name        string         `yaml:"name"`
    Schedule    ScheduleSpec   `yaml:"schedule"`
    Nodes       []NodeSpec     `yaml:"nodes"`
    Resources   []ResourceSpec `yaml:"resources"`
    Subscribes  []SubscribeSpec `yaml:"subscribes"`
    Faults      []FaultSpec    `yaml:"faults"`
}

type ScheduleSpec struct {
    AutoStart bool          `yaml:"autoStart"`
    Interval  time.Duration `yaml:"interval"`
}
```

### ResponseStatus

```go
type Code int
const (
    CodeOK           Code = 0
    CodeInvalid      Code = 1
    CodeNotFound     Code = 2
    CodeUnauthorized Code = 3
    CodeServerError  Code = 4
)

type ResponseStatus struct {
    StatusCode   Code   `json:"StatusCode"`
    StatusString string `json:"StatusString"`
    Description  string `json:"Description"`
}
```

### ID Generator

```go
type Generator struct {
    mu           sync.Mutex
    siteCode     uint32
    industryCode uint32
    seq          uint64
}
```

DeviceID layout: `8 + 2 + 2 + 2 + 6 = 20` digits
- `[0:8]`  SiteCode (8 digits, left-padded with zeros)
- `[8:10]` IndustryCode clamped to 0–99 (2 digits)
- `[10:12]` TypeCode (e.g., 01 = video) (2 digits)
- `[12:14]` SubTypeCode (e.g., 01 = IPC) (2 digits)
- `[14:20]` Sequence number, monotonic, max 999999

Other helpers: `UUID()` returns RFC 4122 v4 string, `Nonce()` returns 32 hex chars, `SubscribeID()` returns 12-char upper alphanumeric.

## Validation

Each entity exposes `Sanity() error` that returns one of:
- `ErrMissingID` / `ErrMissingName` / `ErrInvalidRole` / `ErrInvalidCapability`
- `ErrMissingKind` / `ErrMissingSourceNode`
- `ErrMissingSubscribeID` / `ErrMissingDispositionID`

The errors are exported and usable with `errors.Is`.