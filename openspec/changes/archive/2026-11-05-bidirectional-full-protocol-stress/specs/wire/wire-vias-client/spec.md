# Spec Delta

## Purpose

为 UAC 提供 VIAS 分析系统子集客户端（注册/心跳、任务、控制、规则与能力查询），使模拟器可对真实分析平台或本机 VIAS 端点做联调与压测。

## ADDED Requirements

### Requirement: VIAS 子类型 MUST 支持 System Register/Keepalive/Time

`Client.VIAS()` MUST 提供对 `/VIAS/System/Register`、`/VIAS/System/UnRegister`、`/VIAS/System/Keepalive`、`/VIAS/System/Time` 的高层封装，语义对齐 VIID System 对应方法。

#### Scenario: VIAS 注册成功

- **WHEN** 调用方对合法 baseURL 执行 VIAS Register
- **THEN** 客户端 MUST POST `/VIAS/System/Register` 且 200 时无 error

### Requirement: VIAS 子类型 MUST 支持 Tasks CRUD 子集

VIAS 子类型 MUST 支持 GET/POST `/VIAS/Tasks`、GET/PUT/DELETE `/VIAS/Tasks/:id`。

#### Scenario: 创建并查询任务

- **WHEN** 调用方 POST 创建任务后 GET by id
- **THEN** 返回对象 MUST 含与请求一致的任务标识字段

### Requirement: VIAS 子类型 MUST 支持 TaskControls 与 AnalysisRules

VIAS 子类型 MUST 支持 POST `/VIAS/TaskControls`（含 Action Start/Pause/Stop）以及 GET/POST `/VIAS/AnalysisRules` 与按 ID DELETE。

#### Scenario: 启动任务控制

- **WHEN** 调用方 POST TaskControl 且 Action=Start
- **THEN** 对端 MUST 收到该 Action 且成功时调用方无 error

### Requirement: VIAS 子类型 MUST 支持能力查询

VIAS 子类型 MUST 提供对 `/VIAS/SystemCapability/AnalysisCapability`、`EnhancementAndRestorationCapability`、`RetrievalCapability` 的 GET。

#### Scenario: 查询分析能力

- **WHEN** 调用方 GET AnalysisCapability
- **THEN** 客户端 MUST 返回解析后的 JSON 对象 map
