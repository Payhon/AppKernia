# AppKernia 实施状态

更新时间：2026-08-04（Asia/Shanghai）

## 总体状态

| Surface | 当前可交付状态 | 下一依赖 |
|---|---|---|
| Backend | Admin 蓝图所需 Backend 契约已完成至 `AKADM-310`：认证、自作用域、业务管理、API Client/Webhook、访问规则/服务状态、完整 MFA/OAuth 绑定均形成真实闭环 | 当前 Admin backlog 已收口；生产 Adapter 联调见风险 |
| Admin | `AKADM-000`—`AKADM-310` 依赖图内全部 Task 已实现并通过最终硬化门禁 | 当前 Admin backlog 已收口；跨浏览器/生产验收见风险 |
| Mobile | `AKMOB-000`、`AKMOB-030` 非 UI 部分完成 | 本轮未继续，平台构建仍 blocked |
| Cross-platform i18n | 蓝图契约通过；Admin 18 个 namespace、双语运行时与登录用户服务端偏好已闭环 | Mobile UI |

## 2026-08-04 GitHub 开源开发体验收口

- 新增标准 MIT `LICENSE`，根/Admin package metadata 同步声明 MIT；新增 `CONTRIBUTING.md` 与 `SECURITY.md`，明确契约同步、双语、UI Skill、安全报告和平台验证边界。
- 根 `package.json` 现提供 `dev/build/test/check/generate/preview` 等 npm/pnpm 入口；Admin package 新增真实 Vite `dev`、`preview` 与聚合 `generate`，内部聚合命令改用标准 `npm run`，同时保留 `pnpm-lock.yaml` 为唯一锁文件事实源。
- `server/Makefile` 现提供 help、API/Worker 联合或独立开发、三类二进制构建、测试/race/integration、数据库初始化、doctor、sqlc、迁移、Seed、交互式管理员初始化与安全清理入口；根 Makefile 提供 Backend/Admin 跨项目 setup/dev/build/test/check 入口。
- README 已补齐源码开发模式（Docker PostgreSQL + 宿主 Go/Vite）、全 Docker 模式、npm/pnpm、Makefile 命令、首次管理员、OpenAPI/i18n 契约和仓库结构；Backend/Admin 子 README 同步为真实入口。
- `scripts/doctor.sh` 改为跨平台检查必需工具，将 HBuilderX/Xcode/DevEco 标为可选，并对 Node 24、pnpm 11、npm 11+ 执行版本门禁。本机当前 Node 26.5.0 因此 doctor 按预期失败；Go 1.26.5、pnpm 11.18.0、npm 11.17.0 和 Docker 28.5.1 均被真实识别。
- 实际执行 Backend 全量 Go test 与三二进制 build、Admin 9 files/40 tests 与 Vite production build、npm 与 pnpm 两套脚本入口、冻结锁文件安装、三蓝图/i18n/Mobile 静态校验，均退出 0。
- 源码模式真实启动 API + Worker 和 Vite（临时端口 4175），readiness、Admin HTML、Vite `/admin-api` 代理及 `zh-CN`/`en-US` `Content-Language` 探针均通过；默认 4173 因本机既有 Docker 端口占用按 `strictPort` 退出 1，未伪报通过。
- 本轮没有可视 UI 修改，因此 `ui-ux-pro-max` 不适用且未伪造新 UI 产物；Android、iOS、Harmony 未执行真实构建/真机，继续标记 blocked / 未验证。
- 4173 本机 `appkernia_docker_test` 环境已通过交互式 bootstrap 创建本机管理员；原请求的 10 位密码因少于 12 位被真实策略拒绝，未降低策略，改用 12 位本机临时密码。凭据只写入 Git 忽略的 `docs/LOCAL_TEST_ACCESS.md`，Migration/Core Seed 仍禁止固定默认管理员和明文密码。

## 本轮继续完成（AKADM-250—310）

### Backend

- 完成 `AKADM-250` API Client 与 Webhook：凭据仅一次披露、数据库只保存 Hash/密封值，Webhook URL 执行 SSRF 防护，签名投递、重试、熔断/状态和脱敏审计均接入真实 PostgreSQL、OpenAPI、权限与集成测试。
- 完成 `AKADM-260` 访问规则与服务状态：规则影响确认、主体脱敏、精确权限、多租户 SQL 过滤、不可变审计与服务健康状态契约同步。
- 完成 `AKADM-300` 身份安全：RFC 6238 TOTP、AES-GCM 密封 Secret、仅 Hash 的恢复码、密码/TOTP step-up、OAuth state 单次消费、S256 PKCE 与绑定/解绑闭环；本地 OAuth Mock 仅在 development Feature Flag 下可用，production 配置为 local-mock 会拒绝启动。
- 新增 migration 9 的 OAuth binding challenge，当前 PostgreSQL 18 application migration version=9、dirty=false；蓝图校验统计为 9 组 migration、58 张总表、95 个索引、107 个外键引用。
- MFA/OAuth 路由、Application、Repository、OpenAPI、权限、嵌入式双语 Catalog、Admin 生成 Client 与 PostgreSQL 集成测试保持一致；一次性 Secret、恢复码、code/state 均未写入持久化测试产物。

### Admin

- 完成 API Client/Webhook、访问规则/服务状态页面及 `AKADM-300` Profile MFA、OAuth Connections 和 Callback 页面；所有可见文案使用 `zh-CN`/`en-US` 翻译键，切换语言无需刷新。
- MFA enrollment 只显示一次 Secret/URI，恢复码要求确认后清除；轮换/禁用执行 step-up。OAuth callback 立即从浏览器 history 删除 code/state，回放被服务端稳定拒绝。
- `AKADM-250/260/300/310` 的 UI 均真实运行项目内 `ui-ux-pro-max` 并保存 request、原始 output、决策、检查表与截图。硬化阶段采纳可见焦点、对比度、reduced-motion、语义标签与长文本约束，明确拒绝不适合管理平台的营销视频、外部字体、scroll-snap 和高位移动效建议。
- 完成 `AKADM-310` 全量硬化：Vite manifest、自动 bundle budget 门禁、strict TypeScript、ESLint、Vitest、双语 Chromium E2E、axe 和响应式视觉回归全部接入 `pnpm check`/交付验证。
- 最终 Admin unit suite 为 9 个文件/40 个测试；全量 E2E 为 48 个 axe 场景全部 0 critical/serious violation，身份安全专项另覆盖 7 个双语/响应式场景且无意外控制台错误。
- 生产构建 bundle budget 通过：initial gzip 201,792 B（上限 307,200 B），最大 chunk `DashboardTrendChart` 165,895 B（上限 184,320 B）。

### 最终质量门禁

- Backend `make check`、全量 race、串行 PostgreSQL integration、`staticcheck`、`golangci-lint` 均退出 0；`govulncheck` 未发现可达漏洞，仍报告 1 个依赖包/模块层面的未调用漏洞信息。
- Backend/Admin/Mobile 蓝图 validator、i18n contract validator 与项目本地 UI Skill 检查全部退出 0；Redocly 最终退出 0，保留 4 个既有 warning。
- Admin `pnpm check`、production build、bundle budget 和全量 Chromium E2E 退出 0；Android、iOS、Harmony 本轮均未实际执行，保持 blocked/未验证。

## 2026-08-03 最终完成性复核

- 逐项对照 Admin 的 48 项 Route Registry 后，补齐此前仅有 registry 声明、没有可挂载页面的 API Client 详情、定时任务运行记录、显式 `/404` 与 `/offline` 四条路由；API Client 与 Schedule 列表改为 TanStack 父级 `Outlet` + index route，避免参数子路由只加载 chunk 却不挂载页面。
- 新增租户隔离的 `GET /admin-api/v1/api-clients/{id}`，同步 Route/Application/Repository、OpenAPI、Backend/Admin 蓝图、生成 Client、精确 `sys.api_client.read` 授权和 PostgreSQL integration；跨租户读取返回 not found，响应只含 Secret 元数据。
- Admin Backend API snapshot 现为 147 existing + 0 delta；权限 snapshot 与 Backend `core-permissions.json` 完全一致，实际 seed 为 108 permissions、35 baseline menus。当前持久化 Docker 数据库在 E2E 后有 42 个 active menus，其中 7 个是测试创建记录，不把它们误报为 seed 数量。
- 项目本地 `ui-ux-pro-max` 已为本次路由收口真实运行并保存 request/output/decisions/checklist/screenshot index；新页面视觉实查覆盖 `en-US` 1440 与 `zh-CN` 375，父页面专项继续覆盖 375/768/1024/1440。
- 最终专项 Chromium：Integrations 17 个 axe 场景全部零 violation；Schedules 10 个 axe 场景全部零 violation。验证详情 GET、机器 Token audience 隔离、Secret 撤销、Webhook local-mock、运行记录数据库状态与 4 条审计、双语即时切换、显式错误/offline 页面和无页面级横向溢出。
- 最终 Node 24.18.1 `admin check`、Backend 全量 race、全模块 PostgreSQL integration、staticcheck、golangci-lint、govulncheck、OpenAPI lint、三蓝图 validator、i18n validator 与 UI Skill 检查均退出 0。
- Docker 最终状态：PostgreSQL 18/API/Admin healthy，Worker running，migrate/seed jobs Exited (0)，application migration `9:false`；Admin `http://127.0.0.1:4174/healthz` 返回 200。

## 此前完成

### Backend

- 新增匿名 `GET /admin-api/v1/auth/public-config`，返回规范 locale、支持语言和 fail-closed 的 Admin 认证 feature flags。
- 完成匿名 `POST /admin-api/v1/auth/register`、`POST /admin-api/v1/auth/password/forgot`、`POST /admin-api/v1/auth/password/reset`：注册租户和默认 `member` 角色均由服务端配置决定，客户端不能注入 tenant/role；重复邮箱与找回已知/未知账号保持不可枚举响应。
- 密码恢复 Token 使用加密安全随机值，数据库仅保存 SHA-256 Hash，15 分钟过期且 60 秒冷却；本地开发通过有界内存 Adapter 接收，生产配置禁止启用 local Adapter，第三方投递凭据缺失时默认 fail-closed。
- 重置密码在 Serializable 事务内完成一次性 challenge 消费、密码历史校验与更新、跨 `ak-admin`/`ak-api` audience 的 Session/Refresh family 撤销和脱敏审计；PostgreSQL 集成测试验证 Token 不可复用、旧密码不可登录、新密码可登录。
- `seed core` 会为每个活动租户幂等补齐 `tenant-admin` 与默认 `member` 角色，避免开放注册指向已存在租户时缺少最小角色。
- 同步 Go Route、OpenAPI、Admin API snapshot/delta、页面 backend status 和生成客户端。
- 运行时验证 `Content-Language`、`Vary: Accept-Language`、`Cache-Control: no-store`。
- 保持 Argon2id、Ed25519 JWT、Audience 隔离、Refresh 轮换/重放撤销、CSRF、RBAC/Menu 既有闭环。
- PostgreSQL 18 当前应用迁移 version=9、dirty=false，River 官方 migrations=7；本机验收账号只用于 E2E，不写入仓库。
- 新增自作用域 `GET/PATCH /admin-api/v1/me`，同步 sqlc、OpenAPI、Admin API snapshot/delta；更新 display name/locale/time zone 与不可变 `audit.operation_logs` 在同一 Serializable 事务提交。
- 新增自作用域 `GET /admin-api/v1/me/sessions` 与 `DELETE /admin-api/v1/me/sessions/{id}`；SQL 同时约束当前 user、tenant、`ak-admin` audience，撤销会话、Refresh Token family 和 `iam.me.session.revoke` 审计在同一 Serializable 事务提交。
- PostgreSQL 集成测试验证：列表只标记一个当前会话、不能撤销其他用户会话、被撤销 Access Token/Refresh family 均失效且写入一条不可变审计记录。
- 新增 `POST /admin-api/v1/me/password/change`：Argon2id 校验当前密码、拒绝当前/最近 5 个历史密码复用，条件更新 password version；保留当前会话并事务撤销其他 Access/Refresh family，审计仅记录版本和撤销数量，不记录密码/Hash。
- 新增 `GET/DELETE /admin-api/v1/me/devices[/{id}]`：Admin 登录用可选随机 UUID `X-AK-Device-Key` 绑定 Web 设备；列表按 user、tenant、`ak-admin` audience 隔离并显式返回当前设备。移除设备在 Serializable 事务中撤销该用户此设备跨 audience 的全部 Session/Refresh family、删除设备并写入 `iam.me.device.remove` 审计。
- PostgreSQL 设备集成测试验证两个设备、唯一当前标记、其他用户设备不可见/不可删、同设备 `ak-admin`/`ak-api` Token 同时失效、Refresh family 失效、设备删除和审计撤销数量。
- 新增租户级 `GET /admin-api/v1/dashboard/summary|trends|activity`：时间范围限定为 `7d/30d/90d`，KPI、趋势和活动类目在 Application 层按权限裁剪，Repository 查询全部带 tenant/time-window 条件，活动响应不返回审计 details、任务 payload/output 或安全事件原始内容。
- 新增自作用域头像闭环：`POST /admin-api/v1/me/avatar/upload-session`、development-local `PUT .../content` 与鉴权 `GET /me/avatar/content`；Storage Port/Adapter、Feature Flag、5 MB 限制、JPEG/PNG/WebP 实际解码、尺寸/像素限制、私有文件、SHA-256 去重、usage、用户引用和脱敏审计均在服务端约束。
- 本地对象 Adapter 仅允许 development；production 启用 local 配置会拒绝启动。重复上传相同图片复用租户级去重文件并删除未引用的新对象，PostgreSQL 集成测试覆盖首次上传、相同内容替换、本人/租户隔离和私有读取。
- 登录成功审计与 Session 创建在同一 Serializable 事务提交；已知账号登录失败通过其首个活动租户写入租户级 `audit.login_events`，未知账号保持 tenant/user 为空。两类失败均只记录稳定原因、IP/UA/request ID，不记录邮箱、密码或设备安装键。
- Dashboard PostgreSQL 集成测试覆盖 6 类 KPI、5 条趋势、3 类活动及租户隔离；IAM 集成测试同时验证成功/失败登录事件脱敏和已知账号失败的租户归属。
- 修正统一错误 transport：GoFrame `WriteStatus` 会把状态文本写进 JSON 正文前缀，现改为 `WriteHeader` 后输出纯 JSON；readiness 503 同步修复。
- `migrate up` 改为应用全部待执行迁移并将 `ErrNoChange` 视为成功，修复重复 `docker compose up`/`bootstrap-admin` 被最新版本阻断的问题。
- 新增 Backend/Admin 多阶段镜像、Nginx 同源 `/admin-api` 代理、migration/seed job、健康依赖和交互式管理员初始化；fresh Docker project 已真实构建运行。
- 完成 `AKADM-100` 组织后端：9 个 `/admin-api/v1/org/units|positions` 路由与 OpenAPI/sqlc 生成契约同步；Application 对每个读写动作分别校验 `org.unit.*`/`org.position.*` 权限，Repository 所有查询强制使用认证上下文的 `tenant_id`。
- 部门移动使用 PostgreSQL Recursive CTE 检测自身/后代循环；删除部门检查直属子节点与成员，删除岗位检查成员分配，冲突返回稳定错误码和占用计数。所有成功写操作在 Serializable 事务内写入脱敏 `audit.operation_logs`。
- PostgreSQL 18 集成测试覆盖租户隔离、循环阻断、子节点/成员占用、按部门岗位分配筛选与审计；双语 HTTP 探针确认同一 `ORG.UNIT.CYCLE` 在 `zh-CN`/`en-US` 下返回一致错误码与对应 `Content-Language`。
- 完成 `AKADM-110` 租户用户管理：列表/详情/创建/更新、启停、解锁、管理员重置密码、角色替换、组织分配、会话读取/撤销、CSV 导入导出与角色选项均接入 `/admin-api/v1`，同步 OpenAPI、sqlc、权限 seed、审计和生成客户端。
- 所有用户/角色/部门/岗位/会话查询由服务端使用认证上下文 `tenant_id` 过滤；角色与组织引用在 Serializable 事务内重新校验，最后一个活动 `super-admin` 禁用保护、本人禁用保护、跨租户引用拒绝及密码重置跨 audience 撤销均有稳定错误码。
- 导入限制为 2 MiB/500 行并返回逐行稳定错误码；导出只允许安全字段且最多 500 行。临时密码只用于 Argon2id Hash，进入 Repository 前即清空，审计不记录密码或 Hash。
- 完成 `AKADM-120` 租户管理：租户列表/详情/创建/编辑、状态变更、成员读取/移除和当前租户切换均接入真实 `/admin-api/v1`；后端按系统权限与 tenant context 隔离，切换签发新作用域会话并清理 Admin 租户缓存。
- 完成 `AKADM-130` 访问控制后端：角色 CRUD、权限分配、菜单分配、数据范围、只读权限目录、菜单树 CRUD/移动共 12 个 `/admin-api/v1` 操作；每个动作使用精确权限码，所有查询在 SQL 层绑定当前租户。
- 系统角色与核心菜单不可变；自定义角色/菜单写入使用 Serializable 事务和脱敏审计。自定义组织范围重新验证同租户 unit，菜单 component key 只接受编译期静态 Route Registry，菜单移动拒绝循环和超过 3 层深度。
- PostgreSQL 18 集成测试覆盖跨租户拒绝、权限/菜单分离写入、自定义范围、3 层菜单、第四层拒绝、循环拒绝、任意 component key 拒绝和审计记录。
- 完成 `AKADM-140` 审计安全后端：租户级操作日志、登录日志、安全事件列表/详情与处理共 5 个 `/admin-api/v1/audit/*` 操作；30 天默认/180 天上限、精确权限、SQL 租户过滤和分页上限均在服务端执行。
- 操作与安全事件 JSON 在服务端递归脱敏密码、Token、Secret、Authorization、Cookie、OTP、MFA、API Key、私钥和 Bearer 值；API 不返回登录标识 Hash、原始设备/地理信息、User-Agent 或错误消息。安全事件处理使用 Serializable 行锁、拒绝重复处理，并在同一事务写不可变操作审计。
- 完成 `AKADM-150` 在线会话后端：`GET/DELETE /admin-api/v1/online-sessions[/{id}]` 使用 `iam.session.read/revoke` 精确权限和认证租户上下文；响应只提供服务端脱敏账号/IP/设备提示。强制下线在 Serializable 事务内锁定活动会话、提升 token version、撤销 refresh family 并写审计，跨租户与重复撤销均拒绝。
- 完成 `AKADM-200` 系统配置与字典后端：11 个 `/admin-api/v1/configs|dictionaries` 操作同步 Route、Application、Repository、OpenAPI 与权限；Secret 使用 AES-256-GCM 密封且读接口、审计和浏览器均不回显明文，系统记录锁定，写入使用租户上下文和事务审计。
- 公共配置仅返回全局、active、`is_public=true` 且 `is_secret=false` 的设置；`/api/v1/public/config` 和 Admin 匿名配置使用同一 `settings` 契约，Docker/PostgreSQL 探针证明租户配置和 Secret 不泄漏。
- 完成 `AKADM-210` 地区与模块目录后端：新增公开/管理地区查询和管理模块查询，地区支持根节点、按父级懒加载、搜索、层级/状态过滤及服务端 limit；模块目录为编译期只读 catalog，明确禁止运行时上传、安装和二进制插件。
- 地区 PostgreSQL 集成测试使用 1 个根节点和 250 个子节点验证 `has_children`、懒加载和 100 条上限；精确校验 `sys.region.read`、`sys.module.read`，所有管理读取继续绑定认证上下文。
- 完成 `AKADM-220` 文件存储后端：上传会话、5 MiB 分片写入/覆盖、断点状态读取、取消、完成、列表、详情、引用、受保护下载与删除共 10 个 `/admin-api/v1/files*` 操作；单文件上限 100 MiB，随机对象键、服务端 SHA-256 和 MIME 检测、租户 SQL 过滤、精确权限、Feature Flag 与审计契约均已接入。
- 完成时在服务端重新组装对象并校验总大小/分片完整性；只有 `ready` 且扫描状态为 `clean` 或开发期明确 `skipped` 才能选择/下载。业务引用存在时 Repository 在事务内拒绝删除，成功取消/完成/删除写入不可变审计。
- 保留头像 Storage 既有接口并完成回归；PostgreSQL 18 集成测试覆盖租户隔离、引用保护、完成/删除审计和私有对象，开发期 local Adapter 可用，production 仍禁止 local Adapter。
- 完成 `AKADM-230` 通知管理后端：公告/消息列表、详情、创建、更新、预览、发布、取消与收件统计，通知模板列表/创建/更新，以及投递列表、详情和失败重试均接入 `/admin-api/v1`，同步 OpenAPI、权限 seed、审计与生成客户端。
- 收件范围只接受服务端解析的 `all` 或显式用户集合；发布确认返回精确计数和有界脱敏预览，发布事务幂等物化 `notify.recipients`。HTML 正文由服务端 allowlist 清洗，模板语言限制为 `zh-CN`/`en-US`，变量按 JSON Schema 校验。
- 投递 API 只返回脱敏目标提示与安全失败摘要；重试仅接受失败投递且有界入队，不自动重放成功或处理中记录。PostgreSQL 18 集成测试覆盖租户隔离、收件物化、审计与状态约束。
- 完成 `AKADM-240` 定时任务后端：新增编译期 handler registry、Cron/IANA 时区与 DST 预览、任务 CRUD/暂停/恢复/立即执行/运行历史共 9 个 `/admin-api/v1` 操作；同步 OpenAPI、权限、迁移、sqlc、审计与 Admin 生成客户端。
- 新增 River 0.42 Worker 与 15 秒动态调度器；计划运行、misfire `ignore/fire_once/catch_up`、overlap `allow/skip/replace`、手动执行幂等键和 River 入队均在 PostgreSQL 事务边界内处理。当前唯一真实 handler `system.health.snapshot` 只执行编译期 PostgreSQL 健康检查，不允许 Shell、SQL、上传代码或任意 handler/payload。
- PostgreSQL 18 已实际执行应用 migration version=8、River 官方 7 个迁移；Repository 集成测试验证租户隔离、暂停/恢复、幂等只入队一次、重叠冲突与 4 条不可变审计，Docker Worker 的手动与 15 秒动态计划调度均实际消费，计划运行落库 `succeeded|schedule|JOBS.HEALTH.OK`。

### Admin

- 项目本地 `ui-ux-pro-max` 已真实执行并生成：
  - `design-system/appkernia-admin/MASTER.md`
  - 登录、App Shell、Dashboard、Profile Basic、Profile Security page override
  - `artifacts/ui-ux-pro-max/` 请求、输出、决策、检查表和截图索引
- 建立真实 React/Vite SPA、AntD token、路由级代码分割；ECharts 仅在 Dashboard 独立 lazy chunk 中加载，当前最大 chunk 494.24 kB（gzip 167.73 kB），不触发 500 kB 警告。
- 建立由蓝图生成的 48 项静态 Route Registry、未知 component key 丢弃、feature/view permission 过滤和 redirect 白名单。
- 路由运行时已迁移为 TanStack Router file-based 模型：`src/routes` 是真实文件路由源，`routeTree.gen.ts` 由 `tsr generate` 构建期生成；Vite Router Plugin 自动按路由切块，不再手写 `createRoute` 树。
- 建立响应式 App Shell、移动 Drawer、403/404/500、Dashboard 空态和安全登录页。
- 登录使用 RHF + Zod、显式 label/id、username/current-password autocomplete、内存 Access Token、HttpOnly Refresh Cookie、CSRF 和 single-flight。
- Admin Public Config 控制注册/找回入口；未实现的写 API保持不可操作，不伪造请求。
- 完成 `/register`、`/forgot-password`、`/reset-password` 三个匿名路由与真实写 API；关闭 feature flag 时入口和直达路由均 fail-closed，已登录访问会安全返回 Dashboard。
- 注册协议默认不勾选；邮箱/新密码字段具备密码管理器 autocomplete。找回页对已知/未知账号展示完全相同结果，重置页读取一次性 Token 后立即从浏览器地址和 history 中移除，所有状态均使用双语翻译键与稳定错误码。
- `zh-CN`、`en-US` 拆分为 18 个 namespace 文件；i18next、AntD、Day.js、HTML lang、document title 和 API `Accept-Language` 同步。
- 匿名语言遵循“显式本地选择 > 浏览器语言 > zh-CN”；登录成功切换到服务端用户 locale。
- 已登录语言切换调用 `PATCH /admin-api/v1/me`；非幂等 PATCH 不自动 refresh/replay，失败时回滚原语言并用 `role=alert` 双语提示。
- fresh 浏览器会话重新登录已验证读取服务端 `en-US` 偏好，无需刷新或重启。
- 新增真实 `/profile/basic` 和 `/profile/security`：基本资料/语言/时区读写、密码修改表单、活动会话列表、当前会话标识、双语撤销确认、非当前会话真实撤销及列表刷新；当前会话撤销客户端会清空认证上下文并回登录页。
- 密码表单具备显式 label/id、`current-password`/`new-password` autocomplete、可见性切换、确认一致性和稳定错误码本地化；E2E 验证错误态，成功修改使用临时账号 PostgreSQL 集成测试，固定验收账号密码未改变。
- 会话 `DELETE` 不走自动 401 refresh/replay；后端只接受当前认证主体范围内的 session UUID，不信任客户端 user/tenant。
- Admin 使用本地随机、非敏感 UUID 作为 Web 设备安装标识；设备列表、当前设备文字标识、双语移除确认、加载/空/成功/错误态均已实现。设备 `DELETE` 同样不自动 refresh/replay；当前设备移除会清空认证上下文并回登录页。
- `AKADM-070` Dashboard 已完成：URL 保存 `7d/30d/90d` 范围，服务端按权限省略无权 KPI/趋势/活动，三块各自提供加载/空/可重试错误态；趋势图具备语义数据表替代且 reduced-motion 下关闭动画。
- Docker E2E 从 Dashboard trends 响应真实断言 7 天登录成功/失败均非零，避免把静态图或 Mock 数据当作联调完成。
- Profile Basic 新增双语头像选择、预览、分阶段进度、成功/错误 live region 与立即更新；文件二进制、上传能力和 Blob URL 不持久化。中文文件选择使用翻译按钮触发隐藏 input，不暴露浏览器原生英文文案。
- 完成 `/system/users/departments` 与 `/system/users/positions`：真实树/详情、URL 筛选、RHF + Zod 新增编辑、键盘可触发移动、占用删除影响、按部门岗位筛选、路由权限阻断和动作权限控制均已接入真实 API；写请求不自动 401 重放。
- 完成 `/system/users/accounts` 与隐藏详情路由：URL 搜索/状态/部门/排序/分页恢复、创建编辑 Drawer、角色和组织分配、登录会话、导入导出、行选择与批量启停均接入真实 API；写请求不自动 refresh/replay。
- 用户页在 1200px 以下切换单栏，在 768px 以下收敛为“用户+状态+更多动作”并保留内部表格滚动；真实视觉 review 修复了 1024px 页面溢出与 375px 固定动作列遮挡。
- 完成 `AKADM-120` 租户列表/详情与无刷新租户切换：URL 筛选和返回状态可恢复，切换后按 tenant query key 清理缓存，匿名直达安全回登录页。
- 完成 `AKADM-130` 角色列表/五标签详情、只读权限目录和菜单树：权限、菜单、数据范围分别提交；菜单创建只从静态 component selector 选择，核心记录只读，移动是独立动作，破坏性删除展示影响确认。
- 核心菜单通过 `i18n_key` 渲染，权限目录不展示单语种子名称；自定义组织多选按 label 搜索。真实视觉回归修复 Ant 消息/placeholder 对比度以及 Drawer/Select 动画审计时序。
- 完成 `AKADM-140` 操作日志、登录日志、安全事件与隐藏详情路由：筛选/分页写入 URL，操作详情展示服务端脱敏 JSON，登录表只消费安全字段，安全事件按动作权限确认处理并在返回列表后恢复查询状态。修复了父级事件路由缺少 `Outlet` 导致详情不渲染的问题。
- 完成 `AKADM-150` 在线会话页：账号/应用/平台/IP/活跃日期/状态 URL 筛选、服务端分页、当前会话文字警告、权限控制的强制下线、当前会话额外影响提示、成功/错误 live feedback、双语即时切换和响应式表格均接入真实 API。
- `AKADM-140/150` 均先真实运行项目内 `ui-ux-pro-max`，保存 request、skill output、decisions、review checklist，并新增 Security Audit 与 Online Sessions page override。
- `AKADM-200/210` 均先真实运行项目内 `ui-ux-pro-max`，保存 request、skill output、decisions、review checklist；新增 System Configs、Dictionaries、Regions、Modules 页面 override。
- 完成 `/system/settings/configs` 与 `/system/settings/dictionaries`：URL 筛选、RHF + Zod 编辑、Secret 创建/轮换但绝不回显、系统记录只读、字典类型/条目管理和双语响应式状态均接入真实 API。
- 完成 `/system/settings/regions` 与 `/system/settings/modules`：地区树按需请求子节点并提供可见键盘焦点，模块只展示编译期 catalog/capability 与运行时插件禁用提示；375/1440、`zh-CN`/`en-US` 真实 Chromium 视觉与 axe 结果均为零 violation。
- `AKADM-220` 先真实运行项目内 `ui-ux-pro-max` 并保存 request/output/decisions/checklist 与 System Files override；随后完成 `/system/storage/files`、真实 multipart 续传/暂停/取消、URL 筛选、详情与 usage Drawer、权限动作、下载、删除确认和可复用 `AkFilePicker`。
- 文件页修复 Nginx 默认 1 MiB 请求体导致 5 MiB 分片被 413 拦截、进度条无可访问名称、link 按钮/空表对比度、空表移动端可滚动区不可聚焦及单/双花括号插值混用；最终 Docker Chromium 在中英文、375/1440、引用警告与错误恢复状态均为零 axe violation。
- `AKADM-230` 先真实运行项目内 `ui-ux-pro-max` 5 条查询并保存 request/output/decisions/checklist、通知设计系统和页面 override；随后完成公告、站内消息、模板、投递列表与详情，接入精确收件确认、安全 HTML 预览、模板变量校验和失败投递重试。
- 修复通知路由未进入静态实现白名单导致安全重定向、表单 label/id、AntD 对比度、移动端 Drawer 裁切和详情布局问题；Docker Chromium 使用真实 Go API/PostgreSQL 覆盖双语与 375/768/1024/1440 通知场景，10 组 axe 均为零 violation。
- `AKADM-240` 先真实运行项目内 `ui-ux-pro-max` 5 条查询并保存 request/output/decisions/checklist、Schedules Master 与页面 override；随后完成 `/system/integrations/schedules` 的 URL 筛选、RHF + Zod 编辑、服务端五次运行预览、确认暂停/恢复/立即执行和自动轮询运行历史。
- 定时任务页面只从服务端编译期 registry 选择 handler；所有文案由蓝图 `zh-CN`/`en-US` 生成，语言切换不刷新并保留 URL/列表状态。Docker Chromium 使用真实 API/Worker/PostgreSQL 覆盖 8 组双语/视口/状态 axe 场景，全部零 violation、控制台错误 0。
- Chromium 真实 E2E 覆盖 375/768/1024/1440、两种语言、无刷新切换、真实后端登录、账号枚举一致性、匿名直达守卫、键盘顺序与 reduced-motion。
- axe 48 个关键页面/语言/视口分组全部零 violation；除既有页面外已覆盖操作/登录日志、安全事件详情与列表、在线会话的双语及 375/1440 关键视口。全局套件仍覆盖 375/768/1024/1440，截图见 `output/playwright/`。
- Admin 9 个 Vitest 文件、37 个测试通过。

## 当前部分完成 / 未完成

- `AKADM-040`：App Shell、48 项静态 registry、菜单解析、未知 key 丢弃、直接 URL 守卫、403/404/500、file-based route tree 与响应式视觉验收均已完成；其余业务 component 的页面实现归属后续 P1–P3 Task，不再错误计入本 Task 未完成项。
- `AKADM-050`：登录、注册、找回与重置密码、公开开关、账号枚举防护、键盘/密码管理器和双语响应式 E2E 均已完成；真实邮件供应商因未提供第三方凭据未联调，本轮仅验证 Port、开发期本地 Adapter 与 PostgreSQL 事务闭环。
- `AKADM-060`：基本设置、头像上传、密码修改、本人会话/Web 设备管理、MFA delta contracts、自作用域测试、会话撤销 E2E 与响应式截图均已完成；依赖任务 `AKADM-300` 的完整 MFA/OAuth 绑定也已完成。
- `AKADM-070` 已满足 permission-pruned cards、分区空/错态、URL 状态和 lazy ECharts 退出条件并完成双语视觉验收。
- `AKADM-100` 已完成：部门/岗位 API、租户隔离、逐动作权限、递归防环、占用冲突、事务审计、Admin 双语响应式 UI、键盘移动、Docker/PostgreSQL/Chromium 验收均通过。
- `AKADM-110` 已完成：用户 API、租户过滤、权限矩阵、最后管理员保护、角色/组织引用校验、会话与密码安全、导入导出、Admin 双语响应式 UI、Docker/PostgreSQL/Chromium 验收均通过。
- `AKADM-120` 已完成：租户 API、状态/成员约束、跨租户隔离、切换会话与缓存清理、双语响应式 UI 和 Chromium 验收均通过。
- `AKADM-130` 已完成：角色/权限/菜单/数据范围 API、精确授权、租户 SQL 过滤、系统记录不可变、菜单深度/循环/static component 约束、事务审计及双语响应式 UI 均闭环。
- `AKADM-140` 已完成：审计列表、服务端递归脱敏、安全事件详情/处理、URL 恢复、事务审计与双语响应式 UI 闭环。
- `AKADM-150` 已完成：租户在线会话安全提示、精确读写权限、脱敏标识、Refresh family 撤销、不可变审计、URL 状态和列表刷新闭环。
- `AKADM-200` 已完成：配置/字典权限、Secret 密封与不回显、公共配置最小暴露、事务审计及 Admin 双语响应式 UI 闭环。
- `AKADM-210` 的 API/Admin 验收已完成：地区懒加载大树和编译期只读模块目录均通过 PostgreSQL/Chromium 验证；生产地区数据的版本化导入 CLI 尚未实现，保持为后续 Backend 运维工具项，不伪报完成。
- `AKADM-220` 已完成：文件列表/详情、multipart cancel/resume、scan gate、usage viewer、`AkFilePicker`、delete-in-use 警告、真实下载、双语响应式 UI、PostgreSQL 与 Docker Chromium 验收均闭环。
- `AKADM-230` 已完成：公告、消息、模板、精确收件确认、服务端内容清洗、投递详情与失败重试的 Backend/Admin 闭环已通过 PostgreSQL、Docker Chromium 与双语视觉验收。
- `AKADM-240` 已完成：编译期 handler、Cron/IANA/DST、misfire/overlap、River Worker、手动执行幂等、事务审计、Admin 双语响应式 UI 和真实运行历史均通过 PostgreSQL/Docker/Chromium 验收。
- Backend/Admin 已完成 Admin 蓝图至最终 `AKADM-310`。生产通知供应商 Adapter/Worker、真实 OAuth Provider 与外部投递因缺少第三方设施/凭据尚未联调；development Port/Adapter、Feature Flag 和本地 Mock 已闭环，不伪报生产送达或第三方账号绑定通过。
- Mobile 本轮没有新增 UI；Android/iOS/Harmony 未实际构建或真机验收，继续标记 blocked。
- Playwright Skill 自带 wrapper 当前调用不存在的 `playwright-cli`；本轮使用本机 Playwright 1.54.0 + Chromium 完成真实浏览器测试，并记录 wrapper 非零结果。
- 已验证本地 Docker Backend/Admin/PostgreSQL 18、Chromium 与 Vite production build；未部署、未验证 Firefox/Safari、未执行生产设备闭环。
