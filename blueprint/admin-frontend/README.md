# AppKernia Admin 前端开发蓝图

本资料包用于直接驱动 AI Coding Agent 开发 **AppKernia Admin（AK Admin）**。它参考 HotGo V2 的后台能力覆盖、动态菜单、路由守卫和通用 CRUD 交互，但不复制其 Vue/Naive UI 实现；AK Admin 固定使用已确定的 React 技术栈。

## Agent 入口

1. `AGENTS.md`：代码仓库根级强制规则。
2. `AK_ADMIN_FRONTEND_BLUEPRINT.md`：总体架构与技术决策。
3. `docs/INFORMATION_ARCHITECTURE.md`：最终菜单和隐藏路由。
4. `docs/PAGE_SPECIFICATIONS.md`：33 个页面契约。
5. `docs/UI_UX_PRO_MAX_WORKFLOW.md`：所有 UI 任务的硬门禁。
6. `spec/admin-menu-seed.json`：35 行菜单种子。
7. `spec/admin-route-registry.json`：48 条构建期静态路由。
8. `spec/page-permission-matrix.json`：页面动作权限矩阵。
9. `spec/page-api-schema-map.json`：页面—API—Schema 追踪矩阵。
10. `spec/schema-ui-coverage.json`：51 张核心表的 UI/聚合/后端专用分类。
11. `spec/agent-task-backlog.json`：23 个带依赖和验收命令的任务。
12. `prompts/AGENT_BOOTSTRAP_PROMPT.md`：可直接交给 Coding Agent 的启动提示词。
13. `scripts/validate_blueprint_specs.py`：独立一致性校验器。
14. `VALIDATION_REPORT.md`：已执行检查、限制和目标仓库复验命令。

## 不可变决策

- 一级菜单仅有 `Dashboard` 与 `系统`；平台型功能全部归入“系统”。
- 菜单最大三级：系统 → 功能组 → 页面。
- 登录、注册、找回/重置密码、个人中心和详情页是静态隐藏路由，不写入 `sys.menus`。
- 后端菜单只提供导航元数据；`component_key` 必须映射构建期静态 Route Registry。
- Server State 只进入 TanStack Query；Zustand 只保存小型 Client State。
- Access Token 仅在内存；Refresh Token 使用 Secure + HttpOnly Cookie。
- 所有创建或明显修改可视 UI 的任务必须使用 `ui-ux-pro-max`，并保存设计系统和截图证据。
