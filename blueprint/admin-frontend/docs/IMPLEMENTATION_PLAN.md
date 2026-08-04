# AI Coding Agent 实施计划

- **Phase 0A**：仓库、Node/pnpm、React/Vite、TypeScript strict、CI、错误边界。
- **Phase 0B**：ui-ux-pro-max、Master Design System、AntD Token、Light/Dark、Showcase。
- **Phase 0C**：OpenAPI Client、Token/Refresh、Auth Context、Route Registry、App Shell、认证页、个人中心。
- **Phase 0D**：Dashboard。
- **Phase 1A**：部门、岗位、用户、租户。
- **Phase 1B**：角色、权限目录、菜单、数据范围。
- **Phase 1C**：审计、安全事件、在线会话。
- **Phase 2A**：配置、字典、地区、模块。
- **Phase 2B**：文件、公告、消息、模板、投递。
- **Phase 2C**：定时任务、API Client、Webhook、访问控制、服务状态。
- **Phase 3**：MFA/OAuth、Billing 扩展、全量硬化、i18n、视觉回归。

机器可执行依赖图见 `spec/agent-task-backlog.json`。每阶段报告必须列 Completed、实际命令与 PASS/FAIL、测试数、UI 证据、契约变更和遗留风险。

## i18n 实施调整

国际化不得只留到 Phase 3。新增 `AKADM-015`，在 App Shell 和认证页面前完成 i18next、`zh-CN/en-US`、Ant Design/Day.js 适配、语言切换和静态检查；Phase 3 只做全量完整性、文本扩展和视觉回归。
