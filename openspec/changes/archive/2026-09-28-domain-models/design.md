# 设计：领域模型

## 实体

### Node

代表单个实例——一台设备（UAC）或一个平台（UAS）。

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
    CapCollection Capability = "collection" // Persons, Faces, Vehicles 等
    CapCascade    Capability = "cascade"    // Subscribes、Notifications、Dispositions
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

校验：`Sanity()` 检查 ID/Name 非空、Role 为已知值、Capabilities 仅包含已知值。

### Resource

代表从设备推送到平台的任意实体：Person、Face、Vehicle、Plate、NonMotorVehicle、Image、Object。

```go
type Kind string
const (
    KindPerson          Kind = "Person"
    KindFace            Kind = "Face"
    KindVehicle         Kind = "Vehicle"
    KindPlate           Kind = "Plate"
    KindNonMotorVehicle Kind = "NonMotorVehicle"
    KindImage           Kind = "Image"
    KindObject          Kind = "Object"
)

type Resource struct {
    ID            string    `json:"ID"`         // 资源实例 ID
    Kind          Kind      `json:"Kind"`
    SourceNode    string    `json:"SourceNode"` // 产生该资源的节点
    Timestamp     time.Time `json:"Timestamp"`
    Attributes    Metadata  `json:"Attributes"`
    DataSource    string    `json:"DataSource"`    // URI 或 inline
    StoragePolicy string    `json:"StoragePolicy"` // hot、warm、cold
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
    UpdatedAt  time.Time `json:"UpdatedAt"`
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
    ID          string          `yaml:"id"`
    Name        string          `yaml:"name"`
    Schedule    ScheduleSpec    `yaml:"schedule"`
    Nodes       []NodeSpec      `yaml:"nodes"`
    Resources   []ResourceSpec  `yaml:"resources"`
    Subscribes  []SubscribeSpec `yaml:"subscribes"`
    Faults      []FaultSpec     `yaml:"faults"`
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

### ID 生成器

```go
type Generator struct {
    mu           sync.Mutex
    siteCode     uint32
    industryCode uint32
    seq          uint64
}
```

DeviceID 布局：`8 + 2 + 2 + 2 + 6 = 20` 位
- `[0:8]` SiteCode（8 位数字，左侧补零）
- `[8:10]` IndustryCode（钳制到 0–99，2 位）
- `[10:12]` TypeCode（例如 01 = video，2 位）
- `[12:14]` SubTypeCode（例如 01 = IPC，2 位）
- `[14:20]` 序列号，单调递增，最大 999999

其他辅助方法：`UUID()` 返回 RFC 4122 v4 字符串，`Nonce()` 返回 32 个十六进制字符，`SubscribeID()` 返回 12 个大写字母数字。

## 校验

每个实体都暴露 `Sanity() error`，其返回下列哨兵错误之一：
- `ErrMissingID` / `ErrMissingName` / `ErrInvalidRole` / `ErrInvalidCapability`
- `ErrMissingKind` / `ErrMissingSourceNode`
- `ErrMissingSubscribeID` / `ErrMissingDispositionID`

这些错误被导出并可用于 `errors.Is`。