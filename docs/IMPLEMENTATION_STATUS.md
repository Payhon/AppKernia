# AppKernia 实施状态

更新时间：2026-08-04（Asia/Shanghai）

## 总体状态

| Surface | 当前可交付状态 | 下一依赖 |
|---|---|---|
| Backend | Admin 蓝图所需 Backend 契约已完成至 `AKADM-310`：认证、自作用域、业务管理、API Client/Webhook、访问规则/服务状态、完整 MFA/OAuth 绑定均形成真实闭环 | 当前 Admin backlog 已收口；生产 Adapter 联调见风险 |
| Admin | `AKADM-000`—`AKADM-310` 依赖图内全部 Task 已实现并通过最终硬化门禁 | 当前 Admin backlog 已收口；跨浏览器/生产验收见风险 |
| Mobile | Home、Profile 及子页、文章列表/详情、会话/安全存储/刷新策略与内容契约已实现；Android/iOS 最终 compile-only、Harmony 最终未签名 `.hap` 及静态门禁均通过，主题 20/20 问题已修复 | 三端均未安装运行，设备、签名/发布与安全存储回读验收未完成 |
| Cross-platform i18n | 蓝图契约通过；Admin 与 Mobile 均有 `zh-CN`/`en-US` 语言包、运行时切换与服务端用户偏好接线 | Mobile 三端长英文/运行时视觉验收 |

## 2026-08-04 HotGo 地区编码初始化数据

- 新增 `blueprint/backend/spec/core-regions.json`，从 HotGo `c6191f7126c0ece4f4357014684f479836643822` 的 PostgreSQL `hg_sys_provinces` 数据确定性生成 3,663 条地区记录；34/357/3,272 条源 1/2/3 级数据按 AppKernia 契约规范化为 0/1/2 级，编码无重复、父级无缺失。
- `ak-cli seed core` 新增全局地区目录幂等 upsert，按父级优先写入 `sys.regions`；名称、全名、层级、状态和来源元数据跟随目录更新，已有邮编及源数据缺失时的已有坐标不会被空值覆盖。
- 保留 HotGo 拼音首字母、源层级和排序元数据；源中 5 条缺失坐标按 `NULL` 落库，不伪造邮编。根级/子级公开 API 及带 `sys.region.read` 的 Admin API 已在 PostgreSQL 18 上返回北京 → 市辖区真实数据，`has_children=true`。
- 新增可重复执行的 `scripts/import_hotgo_regions.py` 与完整 MIT 第三方声明；`tmp/hotgo` 继续被 `.gitignore` 排除，运行时和构建不依赖参考仓库。
- 现有 `/system/settings/regions` 后台页面直接消费同一 API，树表懒加载无需视觉结构调整；专项 E2E 夹具已改为验证初始化目录中的北京/市辖区，不再依赖手工 `E2E Child` 数据。
- 本地数据库连续执行两次 Seed 均输出 `regions=3663`；数值编码目录查询为 3,663 条、34 个根、0 个孤儿、5 条缺失坐标。该目录明确是 HotGo 固定提交快照，未核验为 2026 年最新官方行政区划。
- Admin API 验收使用一次性本机 bootstrap 身份完成登录、根查询与子级查询；结束后该用户和租户均设为 disabled，活动 Session 为 0。旧 `LOCAL_TEST_ACCESS` 对当前数据库已返回 401，未复用或暴露其中凭据。
- 最终 Backend `make check`、Backend/Admin/Mobile 蓝图和 i18n 校验全部通过；Go JSON 测试统计为 82 个通过事件、0 失败。Seed Docker 镜像使用 Go 1.26.5 构建成功并在容器内再次写入 `regions=3663`。

## 2026-08-04 系统初始配置与可配置云存储

- 新增 `blueprint/backend/spec/core-configs.json` 作为初始配置目录事实源，共 9 个分类、55 个配置项：基本、邮件、短信、登录注册、提现、云存储、地理位置、支付和微信。`seed core` 会对全部活动租户幂等补齐目录，同时保留租户已修改的当前值和已密封 Secret；本机 PostgreSQL 18 验证为 `55/55` 目录标记、`0` 个默认 Secret 密文。
- 初始目录项允许修改当前值或轮换 Secret，但模块、分组、键、名称、类型、默认值、校验规则、可见性、排序和状态由服务端锁定，避免后台误改系统契约；普通租户自定义配置仍保持原有 CRUD 能力。
- 系统配置页改为分类优先的信息架构，分类选择写入 URL；9 个分类和 55 个配置名均有 `zh-CN`/`en-US` 语义翻译。密钥只显示“未配置/已配置”，编辑抽屉只允许替换，服务端和浏览器均不回显明文。
- 对象存储改为租户配置驱动的 `configured` Adapter：开发环境支持 local，生产支持 S3-compatible 与 MinIO；Endpoint、TLS、path-style、region、bucket、路径前缀、AccessKey/Secret/Session Token、文件/图片大小和 MIME 白名单均从租户配置解析。Secret 继续使用既有 AES-256-GCM 密封值，非开发环境拒绝 local 和明文 HTTP Endpoint。
- 新增 `GET /admin-api/v1/files/upload-policy`，文件会话和持久化记录保存 provider/bucket/object key，但 Admin API 只返回 provider，不返回 bucket/object key。文件列表新增 provider 筛选；头像和后台文件上传均路由到配置 Adapter。
- 新增可复用 `AkFileUploader`，包含策略预取、文件类型/大小前置校验、分片、进度、暂停、续传、取消、失败重试和完成回调；已接入文件管理、文件选择器及云存储配置页测试上传。
- 真实本地 API 生命周期已完成：policy → create session → upload part → complete → provider filter/list → private download → delete；数据库确认内部引用为 `local|appkernia-local|object_key`，公开响应确认不含 bucket/object key。
- 项目内 `ui-ux-pro-max` 产物、Master/page override、双语 1800×952 Chrome 截图和目录配置编辑保护截图已保存。当前 Admin Shell 无暗色模式控制，且已连接 Chrome 窗口未安全调整到 768 px，因此暗色/768 截图明确记为未验证。
- Backend `make check` 通过，默认 Go suite 为 78 个测试；本轮存储/配置 PostgreSQL 集成专项 8 个测试通过。Admin 为 10 个文件/45 个 Vitest，lint、strict typecheck、Vite production build、bundle budget、Node 24 Docker build、三蓝图/i18n 校验均通过。
- 验证边界：local Adapter 已完成真实容器与对象卷闭环；S3/MinIO 没有提供真实凭据，本轮仅完成实现、配置解析、端点/安全规则单元测试与正确 Node/Go 生产构建，不声明真实云服务上传已验收。更换已有文件所依赖的 provider/bucket 前需先迁移对象。

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
- `AKADM-210` 的 API/Admin 验收已完成：地区懒加载大树和编译期只读模块目录均通过 PostgreSQL/Chromium 验证；版本化地区初始化目录及幂等导入 CLI 已于 2026-08-04 补齐。
- `AKADM-220` 已完成：文件列表/详情、multipart cancel/resume、scan gate、usage viewer、`AkFilePicker`、delete-in-use 警告、真实下载、双语响应式 UI、PostgreSQL 与 Docker Chromium 验收均闭环。
- `AKADM-230` 已完成：公告、消息、模板、精确收件确认、服务端内容清洗、投递详情与失败重试的 Backend/Admin 闭环已通过 PostgreSQL、Docker Chromium 与双语视觉验收。
- `AKADM-240` 已完成：编译期 handler、Cron/IANA/DST、misfire/overlap、River Worker、手动执行幂等、事务审计、Admin 双语响应式 UI 和真实运行历史均通过 PostgreSQL/Docker/Chromium 验收。
- Backend/Admin 已完成 Admin 蓝图至最终 `AKADM-310`。生产通知供应商 Adapter/Worker、真实 OAuth Provider 与外部投递因缺少第三方设施/凭据尚未联调；development Port/Adapter、Feature Flag 和本地 Mock 已闭环，不伪报生产送达或第三方账号绑定通过。
- Mobile 本轮没有新增 UI；Android/iOS/Harmony 未实际构建或真机验收，继续标记 blocked。
- Playwright Skill 自带 wrapper 当前调用不存在的 `playwright-cli`；本轮使用本机 Playwright 1.54.0 + Chromium 完成真实浏览器测试，并记录 wrapper 非零结果。
- 已验证本地 Docker Backend/Admin/PostgreSQL 18、Chromium 与 Vite production build；未部署、未验证 Firefox/Safari、未执行生产设备闭环。

## 2026-08-04 Admin 三级菜单布局修正

### 状态：完成

- 修复 Admin 菜单解析器丢弃 `directory` 节点并把所有页面铺成一级菜单的问题；当前严格只接受蓝图定义的 `dashboard`、`system` 两个根节点。
- 真实保留后端 `id/parent_id/sort` 树和排序：`Dashboard` 为唯一根页面，其他业务页面全部位于 `系统` 下的系统设置、用户管理、权限设置、文件存储、通知中心、任务集成、审计安全和运行监控分类。
- 页面仍按静态 Route Registry、精确权限和 Feature Flag 过滤；不可访问或未实现叶子被移除，空目录自动裁剪，任意后端根节点不会进入编译期导航。
- 当前路由叶子自动选中并展开其全部祖先；移动端保留同一树形 Drawer，点击叶子后自动关闭。
- Dashboard 快速访问改为显式提取树中可访问页面，不再依赖已废弃的扁平菜单返回值。
- 中文和英文菜单全部使用既有语义翻译键；真实浏览器切换语言时 `performance.timeOrigin` 不变，无刷新或重启。
- `ui-ux-pro-max` 请求、真实输出、决策、页面 override、检查表、截图索引与三张真实截图已保存到 `artifacts/ui-ux-pro-max/AKADM-navigation-hierarchy/` 和 `output/playwright/`。
- Playwright/Chromium 在 1440 px 双语桌面和 375 px English Drawer 上通过；axe serious/critical 为 0。Firefox、Safari 未执行，不标记通过。
- 本机 `http://localhost:4173` 已使用最终 Admin 镜像重新构建并替换，用户可直接刷新后验证。

## 2026-08-04 Admin 菜单图标与间距优化

### 状态：完成

- 35 条核心菜单 Seed 全部配置语义化 Ant Design Icon，PostgreSQL 实际查询为 `35/35` 非空图标；`appkernia_docker_test` 已重跑 core seed。
- 新增编译期 `ConfiguredMenuIcon` 白名单，Admin 仅按后端 `icon` 名称查表渲染；未知或空值安全回退 `AppstoreOutlined`，不允许动态 import 或执行服务端字符串。
- 根目录、功能分类和页面叶子统一使用 16 px 图标；图标与文字的计算间距统一为 8 px，保留原三级缩进与 44 px 菜单行操作区域。
- 当前 4173 环境因 Feature Flag/权限实际展示 32 项；真实 Chromium 在 `zh-CN`、`en-US` 下均测得 `32/32` 有图标、图标宽度仅 `16`、间距仅 `8`。
- 375 px Drawer 使用同一图标体系；语言切换、权限过滤、当前页面选中和祖先展开行为保持不变，axe serious/critical 为 0。
- Admin Node 24 门禁通过：10 个测试文件、45 项测试、strict typecheck、lint、production build、bundle budget 和蓝图校验全部通过。
- 本机 `http://localhost:4173` Admin 已重建并处于 healthy，用户刷新即可看到新图标与紧凑间距。

## 2026-08-04 Admin DESIGN.md 视觉焕新与品牌资产

### 状态：完成（Web light theme）

- 使用用户提供的 `DESIGN.md` 将 Admin 全局视觉更新为 ink/near-white/hairline/stacked-shadow 体系；Ant Design 主题、表格、卡片、输入、Drawer、Modal、App Shell、认证页和 Dashboard 均统一映射。
- 使用内置 imagegen 参考附件生成原创 AppKernia 标志，按 chroma-key 工作流输出透明 master，并派生 512/180/64/32 四种 Web 尺寸；已接入登录页、桌面/移动导航、favicon 和 Apple touch icon。
- 品牌渐变限制在 Logo、认证品牌面和 KPI 2px 顶部光谱线；管理操作仍使用可访问的 ink/semantic 颜色，不把后台改造成营销落地页。
- `design-system/appkernia-admin/MASTER.md`、App Shell、Login/Auth 与 Dashboard override 已同步；本次 request、真实 Skill 输出、决策、检查表、可复核视觉脚本和 5 张截图保存在 `artifacts/ui-ux-pro-max/AKADM-310-design-refresh/`。
- 新增视觉检查：登录 1440/768/375 均无水平溢出且 axe 0 violations；Dashboard mock-contract 1440 axe 0 serious/critical，768 Drawer 真实截图通过。
- 静态与构建门禁：ESLint、strict typecheck、10 files/45 tests、Vite production build、bundle budget、Admin/Mobile/i18n validators、UI Skill check、`git diff --check` 全部退出 0。
- 本轮仅交付并验证当前 light theme；应用尚无 dark algorithm/选择器，因此没有伪造 dark 截图。Dashboard 新截图使用本地 API 边界 mock-contract，只证明渲染与响应式，不冒充真实 PostgreSQL/生产联调。

## 2026-08-04 Admin 侧栏祖先状态与边界折叠控制

### 状态：完成

- 修复第三层叶子选中后，一级与二级祖先目录错误继承白底叶子的 ink 文字色，导致暗色侧栏文字和图标近乎不可见的问题；选中/展开祖先现在保持 `rgba(255,255,255,.92)`。
- 移除侧栏右下角汉堡折叠按钮，改为侧栏与主内容分界线垂直中点的左右箭头：侧栏悬停或键盘聚焦时出现，折叠后箭头反向并可再次展开。
- 新控制使用语义 Button、既有双语 `aria-label`、`aria-controls` 与 `aria-expanded`；无 hover 指针设备保持可见，`prefers-reduced-motion` 下关闭自定义过渡。
- 折叠时保留受控 `openKeys`，重新展开后恢复当前路由的两级祖先，不丢失“系统 / 系统设置”定位上下文。
- `ui-ux-pro-max` request/output/decisions/checklist 与两张 Chromium 截图保存在 `artifacts/ui-ux-pro-max/AKADM-navigation-collapse/`。
- 浏览器 mock-contract 验证 `/system/settings/dictionaries`：两级祖先计算色均为 `rgba(255,255,255,.92)`，隐藏/悬停/折叠/展开状态均通过，axe violations 为 0。
- Node 24.18.1 全量 Admin check 通过：10 个测试文件、45 项测试、lint、strict typecheck、4,155 modules production build、bundle budget 与 Admin 蓝图均通过；i18n 与 Mobile 蓝图校验通过。
- 本机 `http://localhost:4173` Admin 已重新构建并替换，健康探针返回 `ok`。未部署生产，Firefox/Safari 未执行；Mobile 未改动且未构建或真机验收。

## 2026-08-04 Admin 顶部全屏、语言与账户菜单

### 状态：完成

- 顶部右侧从“语言 Select + 用户名 + 退出登录”收敛为全屏、语言、圆形 Avatar 三个图标入口；顺序、间距、40px 操作区和 hover/focus 状态已同步 App Shell override。
- Avatar 优先读取现有受保护头像 Blob，并正确释放 Object URL；未设置头像时使用显示名称/邮箱的首个 Unicode 字符大写回退。
- Avatar 下拉层分为用户信息与菜单两部分：显示当前用户名、Auth Context 真实角色，并提供个人中心和退出登录。角色为空时使用双语回退文案。
- 语言入口在 App Shell 使用翻译图标；下拉菜单显示简体中文/English，当前语言采用高对比浅蓝选中态。登录/注册等匿名页继续使用原文字 Select。
- 全屏入口使用标准 Fullscreen API，图标、双语 accessible name 与 `aria-pressed` 随状态反转；浏览器 Escape 退出通过 `fullscreenchange` 同步。
- 新增双语键写入 `blueprint/i18n/admin` 事实源并重新生成应用语言包，避免 Docker/CI 构建覆盖。
- Chromium mock-contract 真实点击覆盖账户菜单、角色、个人中心、语言选中与英文切换、全屏进入/退出、375px 移动端和无水平溢出；最终 axe violations 为 0。
- `ui-ux-pro-max` request/output/decisions/checklist 与 4 张截图保存在 `artifacts/ui-ux-pro-max/AKADM-header-utilities/`。
- 本机 `http://localhost:4173` Admin 已使用 Node 24.18.1 Docker 镜像重建，健康探针返回 `ok`。未部署生产，Firefox/Safari 未执行。

## 2026-08-05 Admin 登录失败三次后图形验证码

### 状态：完成

- 登录保护门槛由服务端持久化计算：同一规范化账号、Admin audience 与来源 IP 在 30 分钟内累计失败 3 次后，后续登录必须提交验证码；页面刷新、更换前端会话或直接调用登录 API 均不能绕过。
- 新增 `iam.login_failure_states` 与 `iam.login_captcha_challenges`、migration `000010` 和 sqlc Repository。挑战有效期 5 分钟、最多尝试 5 次、正确答案一次性消费；答案只保存随机盐 SHA-256 Hash，登录范围使用独立配置密钥 HMAC，不在接口、DOM、日志或数据库中保存答案明文。
- 新增匿名 `POST /admin-api/v1/auth/login/captcha`，登录请求支持 `captcha_id`/`captcha_answer`；稳定错误码为 `IAM.AUTH.CAPTCHA_REQUIRED` 与 `IAM.AUTH.CAPTCHA_INVALID`，OpenAPI、Admin 生成 Client、Backend/Admin 双语 Catalog 与蓝图快照已同步。
- Admin 前两次失败保持原表单；第三次服务端返回稳定错误码后才展示 6 位数字 PNG、可见 label、`one-time-code`、加载 live region、刷新按钮和错误 alert，并自动聚焦验证码输入。失败提示保存翻译键，因此已有错误在 `zh-CN`/`en-US` 切换时即时更新。
- `ui-ux-pro-max` 产物、登录页 override、决策与 review checklist 已保存。真实 Chromium 验证 375/768/1440 三视口均无水平溢出，并确认中英文运行时切换、PNG data URL、焦点和响应式布局；axe-core 单元渲染检查为 0 violations。
- PostgreSQL 集成测试覆盖前两次通用失败、第三次强制验证码、刷新不可绕过、错误/过期/已消费挑战、正确登录及成功后阈值重置；IAM 全量 integration suite、Backend `make check` 与 Admin `npm run check` 均通过。
- 本机 Compose 已统一并重建到 `http://localhost:4174`，API/Admin/PostgreSQL 健康。验收后已精确删除管理员测试范围内 6 条挑战和 1 条失败状态，浏览器恢复为空白且不显示验证码的登录页。
- 未部署生产；Firefox/Safari、Android、iOS、Harmony 与真实移动设备未执行，本项仅完成本机 Chromium + Docker/PostgreSQL 闭环。

## 2026-08-05 个人中心头像裁剪与云存储上传

### 状态：完成（本机 configured ObjectStore）

- 新增可复用 `AkAvatarUploader` 与 `AkImageCropper`：JPEG/PNG/WebP 预检、浏览器端正方形裁剪、拖动、缩放、90 度旋转、键盘方向微调、512×512 PNG 输出、预览、进度、失败重试和 Object URL 释放均由公共能力承载。
- 个人中心复用既有本人作用域头像 API；服务端负责租户/用户/随机对象键，按系统云存储配置路由 local、S3 或 MinIO，并继续执行 MIME magic、尺寸、大小、所有权与存储策略校验。上传成功同步 Profile Query、私有头像 Blob 和 Auth Context，无需刷新。
- 本地 Compose 默认启用 `AK_AVATAR_UPLOAD_ENABLED`，避免功能代码存在但个人中心不可达；生产环境仍可显式关闭 Feature Flag。
- 中英文事实源、OpenAPI 说明、Admin/Backend 蓝图、Profile 页面 override 与 `ui-ux-pro-max` request/output/decisions/checklist 已同步。
- Docker Chromium 在 4174 实际完成裁剪与上传，PostgreSQL 记录 local provider、ready 文件和 completed session；验收后精确清理测试头像及对象并恢复管理员原状态。375/768/1440 无水平溢出，移动端头像说明改为纵向布局。
- Admin 全量 13 个测试文件、54 项测试、strict typecheck、lint、production build 与 bundle budget 通过。Backend `make check`、四套蓝图/i18n 校验、Docker Admin/API 构建均通过。
- 外部 S3/MinIO 因未提供真实 Bucket/凭据未联调，不冒充生产云上传；Firefox/Safari、Android/iOS/Harmony 与真实移动设备未执行。本轮未 commit、未 push。

## 2026-08-05 系统配置表单模式与统一保存

### 状态：完成（本机 Docker Chromium + PostgreSQL）

- `/system/settings/configs` 默认进入表单模式，顶部提供有明确标签的“表单模式 / 表格模式”切换；原表格筛选、配置定义新建/编辑及 Secret 单独轮换能力完整保留。
- 表单按服务端配置定义动态生成文本、长文本、数字、布尔、枚举、JSON、日期、Secret 和文件选择控件；Logo 文件配置复用公共 `AkFilePicker` 及云存储上传入口。
- “保存全部更改”只提交脏字段，继续使用后端版本号和 Secret 专用接口。一次 UI 操作会顺序调用现有版本化接口，并明确呈现部分失败，失败草稿保留重试；不伪装成数据库原子批处理。
- 切换分类、模式或离开页面前保护未保存内容；锁定、禁用或无权限配置不可编辑；Secret 既有值不回显，空白表示不变。
- `zh-CN`/`en-US` 事实源与生成 Catalog 已同步；项目内 `ui-ux-pro-max` request/output/decisions/checklist、页面 design-system override 和三张截图已保存。
- Admin lint、strict typecheck、production build、14 files / 56 tests、bundle budget、Admin/Mobile/i18n 蓝图校验、UI Skill 检查和 `git diff --check` 全部退出 0。
- 4174 Docker Admin 已重建且 healthy。真实 Chromium 覆盖默认模式、脏数据拦截、统一保存并精确恢复、表格模式、双语、文件上传入口、375 无横向溢出；四组 axe 均为 0 violation，控制台错误为 0。
- PostgreSQL 复核 `site.name = AppKernia`。未部署生产；Firefox/Safari、Android/iOS/Harmony 和真实移动设备未执行。本轮未 commit、未 push。

## 2026-08-05 登录页图标语言菜单

### 状态：完成（本机 Docker Chromium）

- 登录及共享认证页面的原生语言 Select 已替换为 40px 翻译图标按钮；点击或键盘激活后弹出 `简体中文 / English` 菜单，当前语言具有语义选中态和高对比可见背景。
- 复用公共 `LocaleSwitcher` 的 icon variant，不新增登录页专用状态；匿名本地偏好、无刷新切换以及登录后服务端偏好保存/失败回滚行为保持不变。
- 语言弹层挂载回触发器所在 landmark，图标按钮使用现有双语 accessible name，并通过 `aria-describedby` 关联可能的保存错误。
- 真实 Chromium 使用键盘打开菜单并切换至英文，`performance.timeOrigin` 不变；375/768/1024/1440 均无横向溢出，桌面中文与移动英文开放菜单的 axe 均为 0 violation，控制台错误为 0。
- Admin 14 files / 57 tests、lint、strict typecheck、production build、bundle budget、Admin/Mobile/i18n 蓝图校验、UI Skill 检查、E2E 脚本 py_compile 和 `git diff --check` 全部退出 0。
- 4174 Admin 已使用最终镜像重建且 healthy。未部署生产；Firefox/Safari、Android/iOS/Harmony 和真实移动设备未执行。本轮未 commit、未 push。

## 2026-08-05 Mobile Framework、内容与版本治理（实现完成；平台真实设备验收未完成）

### 状态：实现完成 — 静态/契约、Android/iOS 最终 compile-only 与 Harmony 最终未签名 `.hap` 均已实测；三端设备验收尚未完成

- Mobile 已实现 Home、Profile 及基本资料/安全中心/设备/通知偏好/语言外观/帮助关于/退出登录等可见子页，以及文章列表、详情、分类、书签和分享入口；页面静态登记、移动路由注册表、OpenAPI App API 契约与 Bootstrap 运行时接线均已同步。文章正文只接受受限 block DTO，不使用 `rich-text` 或 `v-html` 渲染不可信内容；私有文章资源下载携带 Bearer Header 且支持取消。
- Backend 已补齐内容、文章分类、书签、移动个人资料与移动版本治理的 Route/Application/Repository/sqlc/OpenAPI/migration/权限种子契约，并在临时 PostgreSQL 18 库进行实现方实测。该结果不是生产部署或第三方服务联调声明。
- Admin 已实现文章/分类内容管理和 Android/iOS/Harmony 版本发布管理。两项页面均保留项目内 `ui-ux-pro-max` request、output、decision、review checklist 与截图；Chromium 确定性 mock-contract E2E 的已保存 axe 证据在中英文关键视口均为 0 serious/critical。最新 Admin `pnpm check` exit 0（16 files / 64 tests、build、bundle 与 blueprint check 通过）；该 Admin 浏览器/Mock 证据不替代真实 Go API/PostgreSQL 端到端验收。
- Mobile P1 主题覆盖 20/20 已修复：移除 `currentColor`，并为三种 button variant 的 loading 状态提供显式 token；Harmony 最终源码实测已通过，修复不替代尚未执行的真实设备验收。
- `AkI18n`、`zh-CN`/`en-US` 语言包、`Accept-Language` 请求头、登录用户偏好同步和匿名非敏感 locale 偏好均已接入。2026-08-05 本地 i18n 契约校验通过；这仅证明键/占位符契约，不替代三端长英文布局与运行时切换验证。
- Refresh 凭据经 `ak-secure-storage` Port 保存，不回退至普通 `uni.setStorage`：Android 设计为 Android Keystore + AES-GCM，iOS 为 ThisDeviceOnly Keychain，Harmony 为 Asset Store。静态契约检查与 Swift 语法解析已通过；尚无三端 Keychain/Keystore/Asset Store 真机写入、读取、轮换和登出清除证据。
- 已引入 DCloud 官方 Codex rules（commit `9ec6ebb2ba57c3634a7be454f2d7c21a02635759`）及 `@dcloudio/uni-app-x-mcp@0.0.5` 的项目级 MCP 配置；仓库根规则仍为更高优先级。本 task 已通过 direct-stdio 的 initialize、tools/list 与 call 验证 MCP 成功，结果保存在 `apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-mobile-framework/mcp-easycom-scan.json`。本机原生 `codex mcp list` 因 vendor binary 损坏不能执行，且项目级 server 仍需从仓库根重启/重新打开 Codex 才会载入原生工具列表；该限制不影响已保存的 direct-stdio 验证。
- Mobile `ui-ux-pro-max` 产物已保存，但 Home/Profile/Article List/Detail 的 Android 360×800、iOS 390×844、Harmony 430×932 真机截图及动态字体、reduced motion、英文扩展审查仍待补齐，不能标记移动端视觉验收完成。
- 平台边界：前一轮唯一临时项目 `ak-mobile-framework-ios-rootqa` 曾以 HBuilderX 5.06 完成 iOS compile-only（20 页与 `ak-secure-storage`，`UTS编译完毕`、`ready in 39282ms`、CLI exit 0）。第 3 次最终隔离证据项目 `ak-mobile-framework-ios-final-evidence` 使用 HBuilderX 5.06 CLI exit 0：20 页与 `ak-secure-storage`、UTS 完成、`ready in 30446ms`、正常“已停止运行”；清理前逐行严格搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality` 均为空。仅标准基座/需 custom base/iOS 13+ 为预期提示；未执行模拟器安装/启动、签名或 Keychain 回读，项目已关闭/清理且无进程。Android 最终唯一项目 `ak-mobile-framework-android-final-clean-1785906028` 已完成 20 页至 Android class、UTS、`ready in 19314ms` 与正常停止；Terra 修复原 4 条 identity-equality warning，复编译额外发现并修复 profile edit 1 条，共 5 处 `Any?/Boolean` 与 `Int` 长度的 UTS 值比较。静态 7 项、严格错误/警告/Identity 搜索均为 0；launch 数值 exit 因外层误用 `PIPESTATUS` 未采集，project close 成功，且仅为 compile-only（非 APK/安装/真机），临时项目已清理且无进程。Harmony 最终最新源码在唯一项目 `ak-mobile-framework-harmony-final.ECScJj` 以 HBuilderX 5.06 CLI 可信 exit 0 实测：20 页、UTS、`ready in 15236ms`、工程生成/依赖/运行包制作成功，恰 1 个 `entry-default-unsigned.hap`。清理前严格逐行搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality`、`currentColor` 均零匹配（`rg` exit 1）。仅包名/签名/未签名为预期 warning；未安装/未真机/未执行 Asset Store 回读，项目已 close，临时项目和日志已清理且无进程。生产发布、三端真机、真实 Push/OAuth/商店升级和三端网络/弱网验收均仍未验证。
