# AGENTS.md — AppKernia Admin

本文件对所有在 AK Admin 仓库中工作的 AI Coding Agent 和开发者生效。

## 1. 先读后写

按顺序读取：`AK_ADMIN_FRONTEND_BLUEPRINT.md`、信息架构、路由授权、API/状态/安全、页面规格、UI skill 工作流、Route Registry、权限矩阵、任务图和后端 OpenAPI。不得猜测 API 字段、权限码、路由路径或菜单层级。

## 2. 固定技术边界

必须：React 19.2、TypeScript strict、Vite 8.1、TanStack Router file-based、TanStack Query、Zustand、Ant Design 6、Pro Components、React Hook Form + Zod、ECharts、Day.js、i18next、pnpm 11、Node.js 24 LTS。

禁止：Vue、Naive UI、Pinia、Next.js/SSR、Redux 保存服务端列表、业务组件手写未类型化响应、`any`、`@ts-ignore`、静默吞错。RHF + Zod 是业务表单唯一事实源；不得同时让 AntD Form/ProForm 与 RHF 双向维护同一表单。

## 3. ui-ux-pro-max 硬门禁

任何创建、重构或明显改变可视界面的任务，必须确认 skill 已安装，读取 `design-system/MASTER.md` 和页面 override，执行 React/表格/表单/可访问性查询，并把输入、输出、决策、检查表和截图保存到 `artifacts/ui-ux-pro-max/<task>/`。Skill 不存在时可以继续 API、类型和测试，但不得把最终 UI 标记完成；禁止伪造 Master。

## 4. 路由、菜单和权限

- 所有页面静态编译并注册在 Route Registry。
- 后端不能下发 import path、JavaScript URL 或组件代码。
- 未知 `component_key`：忽略菜单，记录 telemetry，应用继续运行。
- URL Search Params 是列表过滤、分页和排序事实源。
- 页面前检查 view permission，动作检查 action permission；Go API 再次强制授权。
- 菜单可见性不能推导 API 权限。

## 5. 鉴权和数据

- Access Token 只在内存；Refresh Token 必须 Secure + HttpOnly + SameSite Cookie。
- 401 使用 single-flight，只重试一次；403 不刷新 Token。
- 登出、租户切换和会话失效后清理用户 Query Cache 与会话状态。
- OpenAPI 生成代码放 `src/generated/api/`，禁止手改。
- Query Key 包含 tenant scope；Mutation 只精确失效相关 Key。
- 不把敏感 Query Cache 持久化到浏览器存储。

## 6. 必须复用的基础组件

`AkPage`、`AkPageHeader`、`AkSearchPanel`、`AkDataTable`、`AkTreeTable`、`AkFormDrawer`、`AkDetailDrawer`、`Can/useCan`、`AkStatusTag`、`AkEmptyState`、`AkErrorState`、`AkFilePicker`、`AkIconPicker`、`AkJsonViewer`、`AkAuditDiff`、`AkDangerConfirm`。

不得构建执行任意业务逻辑的万能 JSON CRUD 引擎。可抽象布局和字段适配，业务状态机、权限和副作用保留在 feature 中。

## 7. 完成定义

每个 feature 至少有 Zod/纯函数单测、MSW 组件测试、权限分支、Search Params、关键 Playwright E2E、视觉截图和 axe 检查。提交前实际执行：

```bash
pnpm lint
pnpm typecheck
pnpm test
pnpm test:e2e
pnpm build
python3 scripts/validate_blueprint_specs.py
```

报告必须列出真实命令、结果、变更文件、未完成项和风险。

## 8. 国际化硬约束

- 先读取 `blueprint/I18N_CONTRACT.md` 与 `docs/I18N_ARCHITECTURE.md`。
- P0 即初始化 i18next，必须交付完整 `zh-CN`、`en-US`；默认和最终回退为 `zh-CN`。
- 所有页面标题、菜单、字段标签、按钮、表格、校验、Toast、空状态、错误页和图表文本都必须使用 key，禁止 JSX 硬编码用户可见中英文。
- 菜单优先使用 `i18n_key`，后端 `title` 仅作回退；Route Registry 的 `title_key` 必须存在于两套语言包。
- 切换语言无需刷新，必须同步 Ant Design locale、Day.js locale、HTML `lang/dir`、document title 和已打开工作区标签。
- 登录用户切换语言时更新服务端偏好；匿名选择仅存非敏感本地存储。请求统一发送 `Accept-Language`。
- API 错误优先按 `message_key/error.code` 本地翻译，再回退后端 message。
- `pnpm check` 必须校验 key parity、placeholder parity、未翻译 key、硬编码可见字符串，并在 `zh-CN`/`en-US` 下运行关键 E2E 与视觉回归。
