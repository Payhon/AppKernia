# 可直接交给 AI Coding Agent 的启动提示词

你负责开发 AppKernia Admin（AK Admin），后端为 AppKernia Go Admin API。

开始前完整读取并遵守：`AGENTS.md`、主蓝图、信息架构、路由授权、API/状态/安全、UI skill 工作流、测试门禁、实施计划、菜单种子、Route Registry、权限矩阵、页面 API/Schema 映射、任务图、后端 OpenAPI 与 Backend Blueprint。

固定栈：React 19.2、TypeScript strict、Vite 8.1、TanStack Router、TanStack Query、Zustand、Ant Design 6、Pro Components、React Hook Form + Zod、ECharts、Day.js、i18next、pnpm 11。国际化从 P0 开始，必须完整实现 `zh-CN`、`en-US`，默认/最终回退 `zh-CN`，所有可见文案使用翻译键。严禁改用 Vue/Naive UI/Pinia/Next.js。HotGo 只作能力和交互参考。

任何 UI 代码前先检测并使用 ui-ux-pro-max，读取 `design-system/MASTER.md` 和页面 override，保存 `artifacts/ui-ux-pro-max/<task>/` 证据。Skill 未安装时不得把 UI 标记完成。

所有页面静态编译；后端菜单只引用静态 component_key。Access Token 内存保存，Refresh Token Secure HttpOnly Cookie，401 single-flight，禁止 localStorage Token。

现在领取 `AKADM-000`，严格按 `spec/agent-task-backlog.json` 推进。每个 Task 完成后运行真实 lint/typecheck/test/e2e/build，输出改动文件、命令结果、测试数、UI 证据、契约变更、未完成项和风险。后端契约缺口可使用 typed MSW fixture 做测试，但不得伪装成已联调。
