# Spec Delta

## Purpose

定义双向全协议压测的可观测行为：场景可编排对真实或回环对端的 UAC 出站与 UAS 入站流量，并给出最小验收指标与安全默认（不自动对外部加压）。

## ADDED Requirements

### Requirement: 压测场景 MUST 可声明目标与速率

系统 MUST 支持 YAML 场景声明上游/下游目标 URL、Digest 凭据引用（非明文入库）、资源 Kind、目标 QPS（或 Rate）与持续时间；`Schedule.AutoStart` 默认 MUST 为 false。

#### Scenario: 加载压测样例不自动启动

- **WHEN** 加载压测样例 YAML 且 AutoStart 未显式为 true
- **THEN** 引擎 MUST NOT 自动对外发起请求

### Requirement: 压测场景 MUST 覆盖双向主路径动作

同一压测能力 MUST 能编排至少下列出站动作：Register/Keepalive、集合 POST、Catalog 上报、Subscribe/Disposition 创建、DispositionNotification 推送；入站侧由本进程 UAS 承接真实设备请求。

#### Scenario: 回环双向主路径

- **WHEN** 以本机协议端口为对端运行回环压测场景
- **THEN** 出站请求 MUST 被本机 UAS 接受并记入抓包或对象存储
- **AND** 入站 Register + 资源 POST MUST 返回协议成功响应

### Requirement: 压测结果 MUST 可观测

压测运行期间 MUST 暴露成功/失败计数与最近错误原因（日志或 BFF/控制面事件至少一种），以便验收与排障。

#### Scenario: 对端 5xx 可观测

- **WHEN** 对端对资源 POST 持续返回 5xx
- **THEN** 失败计数 MUST 增加且日志或事件 MUST 含非空错误原因

### Requirement: 真实对端地址 MUST 由配置注入

指向真实平台/设备的 baseURL MUST 仅来自配置文件或环境变量；仓库样例 MUST 使用占位符或 localhost，MUST NOT 提交生产密钥。

#### Scenario: 样例无生产密钥

- **WHEN** 审查仓库内压测样例与默认配置
- **THEN** MUST NOT 出现非开发默认以外的明文生产口令
