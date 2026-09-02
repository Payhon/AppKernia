# AI Coding Agent 任务积压表

| ID | 阶段 | 任务 | 依赖 | UI Skill | 退出条件（节选） |
|---|---|---|---|---|---|
| `AKADM-000` | P0A | 初始化 React/Vite 工程与质量门禁 | — | 非 UI | pnpm lint；pnpm typecheck；pnpm test |
| `AKADM-010` | P0B | 初始化 ui-ux-pro-max 与 Master Design System | AKADM-000 | 必须 | scripts/check_ui_skill.sh；Playwright showcase screenshots；axe 0 critical/serious |
| `AKADM-015` | P0 | i18n 基础设施与中英文语言包 | AKADM-010 | 必须 | key/placeholder parity；运行时无刷新切换；中英文 E2E |
| `AKADM-020` | P0C | 生成 OpenAPI Client 与 API 基础层 | AKADM-000 | 非 UI | generated client clean diff；contract test |
| `AKADM-030` | P0C | 认证、Refresh single-flight 与 Auth Context | AKADM-020 | 非 UI | parallel 401 refresh test；cold reload recovery E2E；logout cache purge test；CSRF test |
| `AKADM-040` | P0C | App Shell、静态路由注册表与菜单解析 | AKADM-010, AKADM-015, AKADM-030 | 必须 | unknown key test；direct URL permission test；responsive screenshots |
| `AKADM-050` | P0C | 登录、注册、找回与重置密码 | AKADM-010, AKADM-015, AKADM-030 | 必须 | auth E2E；account enumeration test；keyboard/password manager check |
| `AKADM-060` | P0C | 个人中心：基本与安全设置 | AKADM-030, AKADM-040 | 必须 | self-only test；session revoke E2E；responsive screenshots |
| `AKADM-070` | P0D | Dashboard | AKADM-040 | 必须 | permission-pruned cards；empty/error states；lazy ECharts chunk |
| `AKADM-100` | P1A | 部门与岗位 | AKADM-040 | 必须 | cycle prevention；occupancy warning；keyboard move fallback |
| `AKADM-110` | P1A | 用户管理与用户详情 | AKADM-100 | 必须 | URL search restore；action permission matrix；bulk E2E |
| `AKADM-120` | P1A | 租户管理与租户切换 | AKADM-030, AKADM-110 | 必须 | feature flag test；cache purge test；cross-tenant isolation |
| `AKADM-130` | P1B | 角色、权限、菜单与数据范围 | AKADM-100, AKADM-110 | 必须 | permission/menu separation；depth/cycle test；component key selector |
| `AKADM-140` | P1C | 操作/登录日志与安全事件 | AKADM-040 | 必须 | redaction fixture；URL filters；resolve audit E2E |
| `AKADM-150` | P1C | 在线会话 | AKADM-110 | 必须 | current-session warning；revoke permission；refresh after revoke |
| `AKADM-200` | P2A | 系统配置与字典管理 | AKADM-040 | 必须 | secret never echoed；URL state；locked dictionary UX |
| `AKADM-210` | P2A | 地区与模块信息 | AKADM-040 | 必须 | large tree test；no runtime plugin install |
| `AKADM-220` | P2B | 文件存储与 AkFilePicker | AKADM-020, AKADM-040 | 必须 | cancel/resume；scan gate；delete-in-use warning |
| `AKADM-230` | P2B | 公告、消息、模板与投递 | AKADM-040 | 必须 | recipient confirmation；sanitization contract；retry UX |
| `AKADM-240` | P2C | 定时任务 | AKADM-040 | 必须 | timezone/DST；handler restriction；execute confirmation |
| `AKADM-250` | P2C | API Client 与 Webhook | AKADM-040 | 必须 | one-time disclosure；SSRF validation；delivery E2E |
| `AKADM-260` | P2C | 访问控制与服务状态 | AKADM-140 | 必须 | impact confirmation；redacted subjects；no secret exposure |
| `AKADM-270` | P2C | 第三方登录配置与 App 绑定 | AKADM-020, AKADM-040 | 必须 | write-only secret；preflight lifecycle；atomic four-provider binding |
| `AKADM-300` | P3 | OAuth 绑定与完整 MFA | AKADM-060 | 必须 | PKCE/state test；one-time secret；step-up E2E |
| `AKADM-310` | P3 | 全量硬化、性能、i18n 完整性与视觉回归 | AKADM-070, AKADM-130, AKADM-150, AKADM-200, AKADM-220, AKADM-230, AKADM-240, AKADM-250, AKADM-260, AKADM-270 | 必须 | pnpm lint；pnpm typecheck；pnpm test |

Agent 一次只领取一个 Task ID。UI Task 先读取/生成 Master 和页面 override；API 缺口先更新 OpenAPI 与 delta；完成后记录实际命令、测试数、截图和风险。


## AKADM-015 国际化专项验收

- 初始化 i18next namespace 架构，默认/最终回退 `zh-CN`。
- 完整提供 `zh-CN`、`en-US`，菜单、路由、认证、个人中心、通用状态和错误不得缺 key。
- 同步 Ant Design、Day.js、HTML lang、document title 和工作区标签。
- 语言切换无需刷新；登录用户偏好同步后端。
- 建立 key parity、placeholder parity、硬编码可见文本检查。
- 两种语言分别执行登录、菜单、用户列表、表单错误和 403/404 E2E，并保存视觉基线。
