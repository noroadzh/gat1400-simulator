# 提案：测试矩阵与文档

## 状态
进行中 —— 正在补齐。

## 背景动机

代码库在内部包上已有 100% 单元测试覆盖率，但仍然缺失：
- **合约测试**：用于核对真实抓包下的协议正确性的 Golden sample 录制
- **端到端测试**：跑通完整栈（设备客户端 → 服务器 → 数据库）的真实 socket 双进程测试
- **文档**：`ARCHITECTURE.md`、`PROTOCOL.md`、`USER_GUIDE.md`、`OPERATIONS.md`、`TESTING.md`、`CHANGELOG.md`

## 目标

- `test/contract/golden/` —— 关键协议序列的请求/响应对应 JSON 文件
- `test/e2e/` —— 启动真实协议服务器并用真实客户端跑通它的 Go e2e 测试
- `docs/` 下 6 份文档全部到位

## 非目标

- 不做性能基准（推迟）
- 不做负载/压力测试（推迟到 OPERATIONS）
- 不为内部实现细节写文档

## 待定问题

无。