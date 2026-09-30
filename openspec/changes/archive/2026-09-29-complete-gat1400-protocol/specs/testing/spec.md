# Spec Delta: testing

## ADDED Requirements

### Requirement: 黄金样本 MUST 覆盖 Nonce 重放检测用例

`test/contract/golden/` 目录 MUST 至少包含一个 replay.json 黄金样本，覆盖 Digest 重放场景：两次 Register 使用同一 nonce+nc，第二次 MUST 被服务端 401 拒绝。

#### Scenario: replay.json 样本存在

- **WHEN** 测试套件以 `--golden` 标志运行
- **THEN** `test/contract/golden/replay.json` MUST 存在
- **AND** 测试运行器 MUST 加载并验证该样本，断言第二次响应状态码为 401。