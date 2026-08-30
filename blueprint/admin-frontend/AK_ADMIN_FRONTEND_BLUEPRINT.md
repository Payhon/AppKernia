# AppKernia（AK）后台管理 Web 前端开发蓝图

## 1. 定位与借鉴边界

AK Admin 是 Go Admin API 驱动的生产级管理 SPA，覆盖用户/组织、RBAC、系统配置、文件、通知、任务、审计、安全和运行状态。HotGo 用于参考功能覆盖、动态菜单、路由守卫、组织树、角色数据权限、文件选择器、日志、在线用户和定时任务等交互；不复制 Vue 3、Naive UI、Pinia、动态组件路径加载、在线代码生成或运行时插件安装。

通知中心包含 App/环境级“推送渠道”页面。厂商配置使用强类型字段，服务端秘密仅写入并只显示指纹；测试推送只能选择当前 App 已注册设备，投递状态必须区分厂商受理、失败、Token 失效与应用打开。

## 2. 技术栈

| 层 | 选型 | 规则 |
|---|---|---|
| Runtime | Node.js 24 LTS + pnpm 11 | 开发/CI；生产只部署静态资源 |
| UI Runtime | React 19.2 | SPA，不启用 RSC/SSR |
| Build | Vite 8.1 | 路由拆包，锁定安全 patch |
| Language | TypeScript strict | 开启 `noUncheckedIndexedAccess`、`exactOptionalPropertyTypes` |
| Router | TanStack Router v1 | file-based、类型安全 Search Params |
| Server State | TanStack Query v5 | Query factory、精确失效、请求取消 |
| Client State | Zustand | 仅主题、侧栏、工作区、会话快照 |
| UI | Ant Design 6 + Pro Components | AntD semantic tokens；ProForm 不与 RHF 双状态 |
| Forms | React Hook Form + Zod | 验证、Search Params、DTO 转换统一 |
| Charts/i18n | ECharts + i18next | `zh-CN`/`en-US` 完整语言包；默认/最终回退 `zh-CN`；运行时无刷新切换 |
| Tests | Vitest + Testing Library + MSW + Playwright + axe | 单元、组件、契约、E2E、视觉、a11y |
| Design | ui-ux-pro-max | 所有可视 UI 任务硬门禁 |

精确 patch 在 Phase 0 经兼容矩阵后写入冻结 lockfile，不在业务阶段自动跨 major 升级。

## 3. 架构原则

```text
Browser
  └─ React App Shell
      ├─ TanStack Router ── Static Route Registry
      ├─ Auth Session Manager
      ├─ TanStack Query ── Generated OpenAPI Client ── Go /admin-api/v1
      ├─ Zustand（仅 Client State）
      └─ Ant Design + AK UI + ui-ux-pro-max Design System
```

核心原则：**路由代码静态、菜单数据动态、授权由后端决定。** `sys.menus.component_key` 只能引用静态注册键；任何未知键都不执行。

## 4. 启动时序

1. 加载最小 App Shell、主题、i18n Catalog 和非敏感本地偏好；先解析规范 locale，再渲染任何用户可见内容。
2. 匿名路由读取 `auth/public-config`；受保护路由执行 Refresh/Context Bootstrap。
3. `GET /auth/context` 返回用户、活动租户、可用租户、角色、权限、菜单、feature flags 与 revision。
4. 将后端菜单与静态 Route Registry 求交集；未知键丢弃并告警。
5. Router `beforeLoad` 校验登录、feature flag 和 view permission。
6. Route Loader 预取首屏 Query；页面处理 loading/empty/error/403/stale。

## 5. 推荐目录

```text
apps/ak-admin/
├── src/
│   ├── app/{bootstrap,layouts,providers,theme,error-boundaries}/
│   ├── routes/{__root.tsx,_auth,_app,_plain}/
│   ├── features/{auth,dashboard,users,departments,positions,tenants,roles,permissions,menus,configs,dictionaries,files,notifications,schedules,api-clients,webhooks,audit,monitoring}/
│   ├── shared/{api,auth,components,hooks,i18n,lib,permissions,query,ui}/
│   ├── generated/api/
│   ├── routeTree.gen.ts
│   └── main.tsx
├── design-system/{MASTER.md,pages/}
├── artifacts/ui-ux-pro-max/
└── tests/
```

每个 feature 使用 `api/{keys,queries,mutations}.ts`、`components/`、`pages/`、`schemas/`、`model/`、`tests/`。禁止跨 feature 深层导入内部文件。

## 6. App Shell 与体验

桌面侧栏 240/64px，顶部 56px，内容区 24px；小于 992px 使用 16px。768px 仍须可操作：侧栏变 Drawer，表格按优先级折叠，行操作进入更多菜单。可选工作区标签只保存路由位置，不默认保活整个 DOM。

## 7. 状态、表单和安全

- TanStack Query 管理全部服务端数据；Query Key 必须包含 `['tenant', tenantId, ...]`。
- Zustand 只保存 Access Token 内存引用、主题、侧栏、语言、工作区标签和命令面板。
- Access Token 不进入 localStorage/sessionStorage/IndexedDB；Refresh Token 使用 HttpOnly Cookie；CSRF 使用同源 Cookie + Header/Origin 校验。
- 401 刷新 single-flight；失败后原请求最多重试一次。
- Zod 同时校验路由 Search Params、表单和 DTO；服务端字段错误映射回 RHF。
- Secret 字段使用“保持不变/替换/轮换”三态；并发更新使用 version/updated_at 和 409 对比。

## 8. 通用组件与页面模式

列表页统一为：标题/说明 → Search Params 搜索区 → 主动作 → 表格/树表 → Drawer/详情路由。每个异步页面必须有 Loading、Empty、Error、Forbidden、Offline/Stale 状态。破坏性动作说明对象、数量和不可逆影响。

## 9. HotGo 能力在 AK 中的处理

- 用户、部门、岗位、角色、菜单、字典、配置、附件、公告、日志、在线用户、定时任务、监控：保留能力并按“系统”分组重构。
- 按钮/接口权限与菜单分离；权限目录只读并由代码/Seed 维护。
- 文件选择器重写为 `AkFilePicker`，走 S3/MinIO 直传、扫描状态和引用检查。
- 在线代码生成只保留 `ak gen` CLI/Agent 工作流，不设生产菜单。
- 插件只展示编译期模块信息，不允许在线安装二进制。
- 支付/资金为可选 Billing 扩展，默认关闭。

## 10. Agent 执行

从 `AKADM-000` 开始，严格按 `spec/agent-task-backlog.json` 的依赖图推进。每个 UI Task 在写 JSX/CSS 前必须使用 ui-ux-pro-max 并保留证据。每个阶段完成后运行真实 lint/typecheck/test/e2e/build，契约缺口先更新 OpenAPI/权限/Seed，不把长期 mock 当作联调完成。

## 11. 多语言架构

统一契约见 `blueprint/I18N_CONTRACT.md`，详细实现见 `docs/I18N_ARCHITECTURE.md`。i18n 是 P0 基础设施，不允许等到 AKADM-310 才开始替换硬编码文案。

必须实现：

- i18next namespace 分包与按路由懒加载。
- `zh-CN`、`en-US` 两套完整资源；构建时 key/占位符一致性检查。
- Header/Profile 语言切换入口；切换不刷新页面。
- Ant Design `zh_CN/en_US`、Day.js `zh-cn/en`、HTML `lang/dir`、document title 同步。
- 所有请求附带 `Accept-Language`；Auth Context 返回的用户 locale 与本地状态同步。
- 菜单使用 `i18n_key`，Route Registry 使用 `title_key`；fallback title 不能成为主要显示方式。
- 错误优先按稳定 code/message_key 本地翻译；字段错误映射到本地字段标签。
- 数字、日期、时间、相对时间、百分比和货币通过统一 Formatter；不得在页面自行拼接。
- 两种语言的 Playwright、axe 和视觉基线；英文扩展文本不得截断。
