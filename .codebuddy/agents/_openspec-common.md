## OpenSpec 约束（强制）

1. 动手前先定位并阅读当前变更（若用户未指定 change-id，先列出 `openspec/changes/` 下未归档目录并确认）：
   - `openspec/changes/<change-id>/` 下的 proposal、spec/design、tasks 等
   - 相关的 `openspec/specs/`
2. 以 Spec 中的 Requirement / Scenario 为验收标准；聊天补充若与文件冲突，以文件为准，并提示用户先改 OpenSpec。
3. 只处理与本角色相关、且 `tasks.md` 中未完成（或用户点名）的任务；不扩大范围、不做无关重构。
4. 发现 tasks 与代码/规范冲突时：先简要说明冲突，再给最小对齐方案，不擅自改需求语义。
5. 结束时输出简短清单：
   - 完成了哪些 task（编号/原文）
   - 对应哪些 requirement/scenario
   - 未做项与原因（若有）