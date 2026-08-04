你位于 AppKernia 根目录，负责 AK Mobile。先读取根 `AGENTS.md`，再完整读取 `blueprint/mobile/AGENTS.md`、主蓝图、docs、spec、integration，以及 `blueprint/backend` 的 App API/OpenAPI 规范。审计现有代码后，严格按 `blueprint/mobile/spec/agent-task-backlog.json` 的依赖顺序连续开发 `apps/ak-mobile`，不要停在计划阶段，也不要逐任务等待确认。

固定使用 uni-app x、UTS/UVue、VDOM 与仓库固定的 uView Ultra；业务页面只能使用 AK UI 适配层。任何 UI 任务先运行 `ui-ux-pro-max` 并保存真实产物。后端缺口按 `app-api-delta.json` 同步实现 OpenAPI、权限、sqlc、审计和测试。执行真实 Blueprint 校验、UTS 检查、Android/iOS/Harmony 构建与可用平台测试；无法执行的平台标 blocked，禁止伪报通过。持续更新实现状态和交付报告。
