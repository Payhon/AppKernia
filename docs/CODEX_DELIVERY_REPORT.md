# AppKernia Codex 交付报告

日期：2026-08-04  
范围：Backend + Admin frontend 完成性与 GitHub 开源开发体验；未自动 commit、push 或部署。

## 交付摘要

- 真实运行项目内 `ui-ux-pro-max`，保存 Master、登录/App Shell/Dashboard override、决策、检查表和截图索引。
- 将 Admin 从 core library 扩展为可运行 SPA：双语登录、App Shell、移动导航、Dashboard 空态、错误页、静态 registry/resolver、权限与安全 redirect。
- 将 Admin 运行时路由从手工 `createRoute` 树迁移为 TanStack Router file-based：最终 60 个路由源文件位于 `src/routes`，构建期生成类型安全 `routeTree.gen.ts`，Vite 自动路由级代码分割。
- 将 Admin i18n 改为从蓝图事实源生成的 18 个 namespace；运行时同步 i18next、AntD、Day.js、HTML lang/title 和 API locale。
- 新增 Backend `GET /admin-api/v1/auth/public-config`，同步 OpenAPI、蓝图 API snapshot/delta 与生成类型，未实现能力默认关闭。
- 完成注册与密码恢复闭环：服务端选定租户/默认角色、重复账号防枚举、仅 Hash 持久化的一次性短期 Token、冷却、跨 audience Session/Refresh 撤销、密码历史与脱敏审计；外部投递通过 Port/Adapter 隔离，生产禁止 local Adapter。
- 完成 Admin 注册、找回与重置页面；协议不预选、已知/未知账号相同结果、Token 立即从 URL/history 移除、feature-disabled 直达路由 fail-closed，并保存专项 `ui-ux-pro-max` 真实产物。
- 新增 `GET/PATCH /admin-api/v1/me`，语言偏好、sqlc、OpenAPI、事务审计与 Admin 状态同步形成真实闭环。
- 新增 `GET/DELETE /admin-api/v1/me/sessions[/{id}]`，以 user + tenant + Admin audience 自作用域列出/撤销会话；撤销 Refresh Token family 并写事务审计。
- 新增 Admin Profile Basic 与 Profile Security 页面；双语资料读写、会话列表、当前会话警告和非当前会话真实撤销形成浏览器闭环。
- 新增本人密码修改：当前密码校验、最近历史复用阻断、并发版本条件更新、保留当前会话/撤销其他会话和脱敏审计；Admin 提供密码管理器友好的双语表单。
- 新增本人 Web 设备闭环：登录携带随机安装 UUID，后端事务绑定 `iam.devices`/Session；列表按本人、租户和 audience 隔离，移除会撤销该设备跨 audience 的 Session/Refresh family、删除设备并写审计。Admin 提供双语响应式设备卡、当前设备警告和真实移除操作。
- 完成 `AKADM-070`：新增租户级 Dashboard summary/trends/activity API，服务端按权限裁剪查询与响应，Admin 提供 URL 时间范围、真实 KPI、lazy ECharts 趋势、语义表格替代和三类脱敏活动面板。
- 补齐登录事件事实源：成功登录与 Session 同事务写入，已知账号失败关联活动租户，未知账号不落身份提示；Dashboard E2E 已从 PostgreSQL 响应确认成功和失败趋势均非零。
- 修复统一错误 JSON：GoFrame `WriteStatus` 原会写入状态文本前缀，改为 `WriteHeader` 后错误码可被客户端稳定解析；readiness 503 同步修复。
- 新增 Backend/Admin production 多阶段 Docker 镜像、Nginx 同源代理、migration/seed/bootstrap jobs 和根级一键命令。
- 使用真实 PostgreSQL 18、Go API、Vite 和 Chromium 完成认证 E2E；没有把 Mock 结果冒充后端联调。
- 完成 `AKADM-060` 头像闭环：Storage Port、development-local Adapter、生产 fail-closed、OpenAPI/sqlc/i18n、私有内容读取、租户级 SHA-256 去重与冗余对象清理，以及 Admin 双语上传/预览/进度/错误态。
- 完成 `AKADM-100` 部门与岗位：9 个真实 Admin API、OpenAPI/sqlc/权限/租户数据库契约、递归防环、占用冲突和 Serializable 事务审计；Admin 提供双语响应式树/详情、岗位表格、URL 筛选、RHF + Zod 编辑与键盘移动表单。
- 完成 `AKADM-110` 用户管理：真实租户级 API、权限/最后管理员保护、角色和组织引用校验、会话及密码安全、CSV 导入导出、Admin 列表/详情/批量动作与双语响应式视觉闭环。
- 完成 `AKADM-120` 租户管理：真实租户 CRUD/状态/成员 API、跨租户隔离、租户切换新会话与 tenant-scoped Query cache 清理；Admin 列表/详情、URL 恢复、无刷新切换和双语响应式视觉闭环。
- 完成 `AKADM-130` 访问控制：角色 CRUD、权限/菜单/数据范围独立写入、只读权限目录、菜单树 CRUD/移动、精确动作权限、租户 SQL 过滤、系统记录不可变、3 层/防环/static component key 约束和脱敏事务审计。
- 角色、权限目录和菜单页严格使用 `zh-CN`/`en-US` 翻译键；核心菜单按 `i18n_key` 渲染，组织范围按 label 搜索，所有菜单页面组件只能从编译期 Route Registry 选择。
- 完成 `AKADM-140`：5 个审计/安全 API、OpenAPI/sqlc/权限与租户数据库契约、递归服务端脱敏、安全事件 Serializable 处理与不可变审计；Admin 提供操作/登录/安全事件表、脱敏 JSON 详情、隐藏深链路由、URL 恢复和双语响应式流程。
- 完成 `AKADM-150`：2 个在线会话 API 使用精确权限和服务端租户上下文，只返回脱敏账号/IP/设备提示；强制下线事务撤销 Session/Refresh family 并写审计。Admin 提供 URL 筛选、当前会话额外警告、动作权限、列表刷新和双语响应式页面。
- 项目内 `ui-ux-pro-max` 为 AKADM-140/150 分别真实生成并保存 request、skill output、decisions、review checklist；新增 Security Audit 与 Online Sessions override，营销 CTA、外部字体等不适用建议在决策文件中明确覆盖。
- 完成 `AKADM-200`：系统配置与字典的 11 个 Admin API、AES-256-GCM Secret 密封/轮换且永不回显、公共配置最小暴露、权限与事务审计；Admin 配置/字典页完成 URL 状态、RHF + Zod、系统锁定及双语响应式闭环。
- 完成 `AKADM-210` 的 API/Admin 验收：公开/管理地区查询支持大树懒加载，管理模块目录固定为编译期只读 catalog 并明确禁用运行时插件；Admin 地区/模块页完成双语、375/1440 与键盘/axe 验收。
- 项目内 `ui-ux-pro-max` 为 AKADM-200/210 真实生成并保存 request、skill output、decisions、review checklist；新增 System Configs、Dictionaries、Regions、Modules override。
- 完成 `AKADM-220`：新增租户级文件 Storage Application/Repository/HTTP 模块，5 MiB/100 MiB multipart、断点状态、取消、服务端组装/SHA-256/MIME 校验、列表/详情/usage、扫描门禁、鉴权下载、引用保护删除与不可变审计；OpenAPI、sqlc、Feature Flag、权限和 Docker 契约同步。
- 完成 Admin 文件管理与可复用 `AkFilePicker`：URL 筛选、权限动作、暂停/续传/取消、进度/错误 live region、详情/引用 Drawer、下载、引用警告和删除闭环；项目内 `ui-ux-pro-max` 真实产物与 System Files override 已保存。
- 修复真实 Docker 上传链路中的 Nginx 1 MiB 默认限制，将 `/admin-api` 单请求体上限设为 6 MiB，只允许略大于单个 5 MiB 分片；同时修复进度条名称、按钮/空表对比度、移动端空表滚动和 i18n 单花括号插值。
- 完成 `AKADM-230` 通知管理：Backend 新增公告/消息、模板和投递 Application/Repository/HTTP 模块，服务端精确收件计数、幂等物化收件人、HTML allowlist 清洗、JSON Schema 模板变量校验、脱敏投递详情与失败重试；OpenAPI、权限、审计和生成客户端同步。
- 完成 Admin 公告、站内消息、模板和投递页面：精确收件确认、安全预览、失败投递详情/确认重试、双语与响应式状态均接入真实 API；项目内 `ui-ux-pro-max` 5 条查询、通知设计系统、页面 override、决策、检查表和截图均已保存。
- 修复通知路由未进入静态实现白名单导致的安全重定向，并修复表单 label/id、AntD 对比度、移动端 Drawer 裁切和详情布局。Docker Chromium 真实验证精确受众计数、重试 HTTP 200、通知审计和 10 个零 axe violation 场景。
- AKADM-230 阶段测试计数：Admin Vitest 8 个文件/33 个测试；通知 Application 3 个具名测试 + PostgreSQL Repository 1 个集成测试；Backend 全量 Go race、Docker Chromium 通知 10 个双语/视口/状态 axe 场景均退出 0，无意外控制台错误。
- 完成 `AKADM-240` 定时任务 Backend/Admin：Go Route/Application/Repository、编译期 handler registry、River 0.42 Worker、Cron/IANA/DST、misfire/overlap、手动执行幂等、OpenAPI、migration/sqlc、权限、审计和生成客户端保持同步。
- Admin `/system/integrations/schedules` 提供 URL 筛选、RHF + Zod Drawer、服务端五次运行预览、暂停/恢复/立即执行确认及安全运行历史；所有可见文案由蓝图双语目录生成，语言切换无需刷新。
- 项目内 `ui-ux-pro-max` 5 条真实查询、Schedules Master/page override/request/output/decisions/checklist 均已保存。最终测试计数更新为 Admin 9 个文件/37 tests；定时任务 Repository 1 个 PostgreSQL 集成场景；Chromium 8 个双语/视口/状态 axe 场景全部零 violation。

## 实际命令与退出码

| 命令 | 退出码 | 结果 |
|---|---:|---|
| `.codex/skills/ui-ux-pro-max/scripts/search.py ... --design-system --persist`（Master + 3 page overrides +专项查询） | 0 | 真实 Skill 产物已保存 |
| `pnpm --filter @appkernia/admin generate:routes`（误命中 Node 26/Corepack） | 1 | 工具链漂移；改用项目 Node 24.18.1 |
| 同命令，显式 Node 24.18.1 | 0 | 生成 48 个 Admin route 记录 |
| `playwright_cli.sh --help` | 127 | wrapper 引用的 `playwright-cli` 不在当前 `@playwright/mcp` 包中 |
| `pnpm --filter @appkernia/admin check:ui-skill`（修复 CWD 前） | 1 | 从 filtered package CWD 误报 MISSING |
| 同命令（按脚本位置解析 repo root 后） | 0 | `FOUND:.codex/skills/ui-ux-pro-max/scripts/search.py` |
| E2E 开发轮次 | 1 | 真实发现 label 关联、3.36:1 对比度、AntD listbox 名称及测试等待问题；均修复 |
| 最终 `AK_E2E_PASSWORD=<ephemeral> python apps/ak-admin/scripts/e2e_visual.py` | 0 | 真实 Backend 登录、双语、4 视口、axe、键盘、reduced-motion 通过 |
| `curl ... /admin-api/v1/auth/public-config` | 0 | HTTP 200；en-US、Vary、no-store、feature flags fail-closed |
| `PATH=<node24> make check` | 0 | Backend/Admin/Mobile/四类蓝图/i18n 总门禁通过 |
| `go test -race ./...` | 0 | Backend 15 个默认测试通过 |
| `go test -tags=integration ...` | 0 | 两个包共 6 tests；含 2 个 PostgreSQL 专项 |
| `staticcheck ./...` | 0 | 通过 |
| `golangci-lint run` | 0 | 0 issues |
| `govulncheck ./...` | 0 | 可达漏洞 0；依赖图提示不在调用路径 |
| `sqlc v1.31.1 generate` | 0 | 生成成功 |
| `pnpm --filter @appkernia/admin check` | 0 | API/i18n/route 生成、lint、typecheck、20 tests、build、blueprint 全通过（最终总门禁见下方） |
| `pnpm peers check` | 0 | 无 peer dependency 问题 |
| `pnpm audit --audit-level=high --registry=https://registry.npmjs.org` | 0 | 无已知漏洞 |
| `npx @redocly/cli lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；1 个既有 proprietary license warning |
| Admin/Mobile/backend/i18n 四个 blueprint validator | 0 | 35 menus、48 routes、108 permissions；i18n parity 通过 |
| 源码 Secret pattern scan | 0 | 未发现私钥、常见 token 或本轮临时 E2E 密码 |
| PostgreSQL 18 `migrate up` | 0 | version=6、dirty=false |
| `bootstrap-admin`（密码经 stdin） | 0 | 本机 E2E 管理员获 108 permissions、35 menus |
| fresh `docker compose up --build -d postgres migrate seed api admin`（独立 project/volume） | 0 | Backend/Admin 镜像构建，migration/seed 0，API/Admin healthy |
| 首次重复 `bootstrap-admin` | 1 | 真实发现固定 `migrate up 6` 在最新版本返回 file does not exist；随后修复幂等性 |
| 修复后重复 migrate/seed/bootstrap | 0 | version=6 dirty=false；管理员初始化成功 |
| `curl` Docker `/healthz` 与双语 `/admin-api/v1/auth/public-config` | 0 | 200；`zh-CN`/`en-US` Content-Language 正确 |
| 首次从根目录调用 `python scripts/e2e_visual.py` | 2 | 路径记录错误；真实脚本位于 `apps/ak-admin/scripts/`，随后修正 |
| 首次会话集成测试使用错误本地 DB 密码 | 1 | PostgreSQL SASL 拒绝；读取 Compose 开发配置后使用正确开发连接重跑 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/iam/application ./internal/modules/iam/repository` | 0 | 5 个 PostgreSQL 集成场景通过，含 self-scope、Access/Refresh 失效、密码历史/会话策略和事务审计验证 |
| 会话安全页 E2E 开发轮次 1 | 1 | axe 发现 Descriptions 标签对比度 3.36:1；提升至设计系统高对比辅助色 |
| 会话安全页 E2E 开发轮次 2 | 1 | 取消 Popconfirm 后隐藏节点导致确认按钮 strict locator 冲突；限定可见确认框 |
| 最终 `AK_E2E_BASE_URL=http://localhost:4174 ... apps/ak-admin/scripts/e2e_visual.py` | 0 | Docker 登录、资料读写、真实双会话撤销、双语、4 视口、9 组 axe 无 violations、服务端语言持久化和新会话恢复 |
| `make check` | 0 | Backend/Admin/Mobile 静态门禁及四类蓝图/i18n 校验通过；最终重跑后 Admin 20 tests |
| 首次查询会话撤销审计按 `created_at` 排序 | 1 | 真实 Schema 字段为 `occurred_at`；修正查询后重跑 |
| `psql ... iam.me.session.revoke ORDER BY occurred_at` | 0 | 最新记录为 `200 / succeeded / active → revoked` |
| `sqlc generate`（密码撤销 SQL 首轮） | 1 | 子查询 `id` 歧义；改为 `iam.sessions AS s` 显式限定后通过 |
| 密码 API 422 只读探针首轮 `jq` | 5 | 发现正文为 `Unprocessable Entity{...JSON}`；定位 GoFrame `WriteStatus` 写正文行为 |
| 修复后密码 API 422 探针 | 0 | 纯 JSON，稳定码 `IAM.PASSWORD.CURRENT_INVALID`、message_key 和 request_id 可解析 |
| 密码表单 E2E 首轮 | 1 | `New password` 模糊命中确认字段；改用 accessible label exact 查询 |
| 最终密码表单 Docker E2E | 0 | autocomplete、双语本地化错误态、9 组 axe 无 violations；成功修改由临时账号集成测试覆盖 |
| `psql ... iam.me.password.change ORDER BY occurred_at` | 0 | 最新记录 `200/succeeded`、version `1→2`、撤销 1 个其他会话，审计 JSON 不含 `password_hash` |
| `docker compose exec postgres psql ... iam.users/audit.operation_logs` | 0 | locale=`en-US`；`iam.me.update` before=`zh-CN`、after=`en-US`、session 非空 |
| `./scripts/doctor.sh` | 0 | Go 1.26.5、pnpm 11.18、Docker 28.5.1 可用；宿主 Node 26.5 与项目 Node 24 基线不一致 |
| `docker compose config --quiet && git diff --check` | 0 | Compose 与补丁格式通过 |
| 设备专项 `ui-ux-pro-max` 三条本地查询 | 0 | request、真实输出摘要、decisions、review checklist 和 Profile Security override 已保存 |
| `make sqlc-generate` | 0 | 设备 upsert/list/lock/revoke/delete/audit 查询生成成功 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/iam/... -count=1 -v` | 0 | 6 个 PostgreSQL 集成场景通过；设备跨 audience 撤销、外部用户隔离、删除和审计均通过 |
| `pnpm --filter @appkernia/admin lint/typecheck/test/build` | 0 | 24 个 Vitest；设备 header、GET、非重放 DELETE、当前设备清理和本地 UUID 持久化通过 |
| `AK_POSTGRES_PORT=55433 AK_ADMIN_PORT=4174 ... docker compose -p appkernia_docker_test up -d --build api admin` | 0 | Node 24 Admin 镜像与 Go API 镜像重建，migrate/seed 退出 0，API/Admin healthy |
| 首次 `pnpm ... test:e2e` | 1 | 系统 `python3` 缺少 Playwright 模块；切换已安装 Playwright 1.54.0 的本机 Python 3.12 环境 |
| `AK_E2E_BASE_URL=http://localhost:4174 ... /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py` | 0 | 真实设备列表、当前设备取消警告、非当前设备删除、双语、4 视口和 9 组 axe 通过 |
| `make check`（设备闭环最终重跑） | 0 | Backend/Admin/Mobile 静态门禁、四类蓝图/i18n；Admin 24 tests，API 快照 106 existing + 34 delta |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run` | 0 | Race、静态检查与 lint 全通过，golangci-lint 0 issues |
| `govulncheck ./...` | 0 | 可达漏洞 0；依赖图提示 2 个不在调用路径的漏洞 |
| `npx @redocly/cli lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅 1 个既有 proprietary license warning |
| AKADM-100 `ui-ux-pro-max` 部门/岗位/UX/React 四条真实查询 | 0 | request、完整 skill output、decisions、review checklist 与两个 page override 已保存 |
| `make sqlc-generate`（组织 CRUD、占用、审计） | 0 | 组织查询与生成代码成功；9 个 OpenAPI 操作同步生成 Admin 类型 |
| 组织集成测试首次使用旧本地数据库密码 | 1 | PostgreSQL 返回 `28P01`；随后从当前 Docker 容器以不打印方式读取测试密码并重跑 |
| `AK_TEST_DATABASE_URL=... go test -p=1 -tags=integration ./internal/modules/iam/... ./internal/modules/dashboard/... ./internal/modules/storage/... ./internal/modules/org/... -count=1 -v` | 0 | IAM/Dashboard/Storage/Org 全部单元与 PostgreSQL 场景通过；组织覆盖租户隔离、循环、占用、部门筛选和审计 |
| 组织 E2E 开发轮次 | 1 | 真实发现系统 Python 无 Playwright、注册测试租户配置漂移、整页导航清空内存 Token、树展开/等待条件、GoFrame 通用参数读取导致 Move Body 422，以及 AntD 对比度 1.83/3.36；均逐项修复后重跑 |
| `Get("id")` 失败复现与 `GetRouter("id")` 修复后探针 | 422 → 200 | 路径参数读取不再与 JSON Body 合并；真实 Move 成功并写审计 |
| 双语循环冲突 HTTP 探针 | 0 | `zh-CN`/`en-US` 均返回 HTTP 409、稳定码 `ORG.UNIT.CYCLE`、对应 `Content-Language` 与本地化消息 |
| 最终 Node 24 Docker API/Admin rebuild | 0 | migration/seed 退出 0，API/Admin/PostgreSQL 18 均 healthy |
| 最终 Chromium 组织全流程 E2E | 0 | 部门创建 201/201、键盘移动 200、岗位创建 201、匿名直达守卫、双语、4 视口截图；22 组 axe 无 serious/critical violations且无意外控制台错误 |
| `make check`（AKADM-100 收口） | 0 | Backend/Admin/Mobile 蓝图、i18n、Go vet/tests、Admin 28 tests、lint/typecheck/build 全通过 |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run`（AKADM-100） | 0 | Race、静态检查与 lint 全通过；golangci-lint 0 issues |
| `govulncheck ./...`（AKADM-100） | 0 | 可达漏洞 0；另有 1 个包与 1 个模块漏洞不在当前调用路径 |
| `pnpm audit --audit-level=high` / `pnpm peers check` / Redocly | 0 | 无已知高危依赖、无 peer 问题；OpenAPI 有效，保留 1 个既有 license warning |
| AKADM-060 头像专项 `ui-ux-pro-max` 三条本地查询 | 0 | 保存 request、完整 skill output、decisions、review checklist 与 Profile Basic override |
| 首次顺序 i18n 校验前的并行 generate/validate | 1 | 生成器与校验器竞争读取；改为先生成再校验后退出 0 |
| 头像 Storage 默认/集成测试首轮 | 1 | 先修复 GoFrame Response Writer 类型；数据库集成再发现审计 session 外键夹具无真实会话，补齐真实 Session 后通过 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/storage/... -count=1 -v` | 0 | 图片真实性、self/tenant scope、事务、usage、审计、私有读取、重复内容去重和冗余对象删除通过 |
| Admin 头像 session/blob 单测与 `pnpm check` | 0 | 28 个 Vitest；写请求不自动重放、私有 GET、双语 i18n、lint/typecheck/build/蓝图通过 |
| 头像 E2E 开发轮次 | 1 | 真实发现系统 Python 缺 Playwright、Chromium response body cache、重复文件 23505、Query cache 时序、status locator 冲突及原生英文文件文案；逐项修复 |
| 重复头像 PostgreSQL 回归测试首轮 | 1 | 命中 `uq_files_tenant_dedup`；改为复用 ready file 并清理未引用对象后通过 |
| `govulncheck ./...` 首轮 | 3 | `x/image/webp v0.32.0` 两条可达 2026 漏洞；升级至 `v0.43.0` |
| `govulncheck ./...`（升级后） | 0 | 可达漏洞 0 |
| 全包并行 PostgreSQL integration 首轮 | 1 | 测试包并行共享 DB 触发一次 SQLSTATE 40001；`go test -p=1 -tags=integration ...` 全部通过，未把首轮失败隐藏为通过 |
| `make check` / `go test -race ./...` / `staticcheck ./...` / `golangci-lint run` | 0 | 四蓝图/i18n、Go vet/tests/race、Admin 28 tests/build 与静态检查通过；lint 0 issues |
| `pnpm audit --audit-level=high` / `pnpm peers check` / Redocly | 0 | 无高危依赖、无 peer 问题；OpenAPI 有效，保留 1 个既有 license warning |
| 最终 Node 24 Docker rebuild + Chromium 双语 E2E | 0 | 头像 session 201、upload 200、private content 200；16 组 axe 无 violations，4 视口、双语及既有全流程通过 |
| PostgreSQL/object-volume/匿名内容探针 | 0 | 当前头像 private/ready、usage=1、审计存在、对象文件=1；匿名内容请求 401 |
| AKADM-050 认证恢复专项 `ui-ux-pro-max` 三条本地查询 | 0 | 保存 request、真实 skill output、decisions、review checklist 与 Auth Recovery override |
| `make sqlc-generate`（注册/恢复与默认租户角色） | 0 | 注册事务、challenge、一次性重置、跨 audience 撤销及默认角色 SQL 生成成功 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/iam/... ./internal/modules/dashboard/... -count=1 -v` | 0 | IAM 注册/恢复、资料/会话/设备/密码及 Dashboard PostgreSQL 集成全部通过 |
| 注册恢复 E2E 开发轮次 1 | 1 | axe 发现注册页辅助文字对比度 3.36:1；提升至高对比辅助色后修复 |
| 注册恢复 E2E 开发轮次 2 | 1 | 注册返回 503；定位测试租户缺少默认 `member` 角色，补充幂等默认角色 seed 并重建 API |
| 注册恢复 E2E 开发轮次 3 | 1 | 找回成功页切换语言时被 locale 相关 feature query 重新挂载；改为 locale 无关缓存键 |
| Admin 重建后的首次浏览器探针 | 1 | 容器尚未 ready 导致 `ERR_CONNECTION_REFUSED`；健康检查确认 healthy 后重跑，不记为功能通过 |
| `AK_E2E_BASE_URL=http://localhost:4173 ... /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py` | 0 | 注册/找回/无效重置、Dashboard、Profile、双语与 4 视口；14 组 axe 均为 0 violations |
| `pnpm --filter @appkernia/admin check`（AKADM-050 收口） | 0 | API/i18n/48 routes 生成、lint、typecheck、26 tests、Vite production build、Admin 蓝图通过；宿主 Node 26 产生 engine warning |
| Backend/Admin/Mobile/i18n 四个 validator（AKADM-050 收口） | 0 | 112 existing APIs + 28 deltas；0 errors/warnings；双语 key/placeholder parity 通过 |
| 首次 AKADM-050 validator 批处理（错误 backend 路径） | 2 | 使用了不存在的 `blueprint/backend/scripts/...`；改用真实 `blueprint/backend/tools/validate_blueprint.py` 后退出 0 |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run`（AKADM-050 收口） | 0 | Race、静态检查与 lint 通过；golangci-lint 0 issues |
| `govulncheck ./...`（AKADM-050 收口） | 0 | 可达漏洞 0；另有 2 个依赖漏洞不在当前调用路径 |
| `npx @redocly/cli lint server/openapi/openapi.yaml`（AKADM-050 收口） | 0 | OpenAPI 有效；仅 1 个既有 proprietary license warning |
| `pnpm add @tanstack/router-plugin@1.170.18`（镜像源） | 1 | 镜像与官方均不存在该插件版本；没有强行安装不存在版本 |
| 官方 registry 查询/安装 `@tanstack/router-plugin@1.170.18` | 1 | 官方确认 package not found；随后采用其声明兼容 React Router `^1.170.18` 的 plugin `1.168.23` |
| 安装 `@tanstack/router-plugin@1.168.23` 与 `@tanstack/router-cli@1.167.21` | 0 | Plugin peer contract 精确接受当前 React Router；CLI 与 Plugin 内置 generator 同版本，lockfile 更新 |
| `pnpm --filter @appkernia/admin generate:file-routes` | 0 | 从 `src/routes` 生成 11 个类型安全文件路由及 `routeTree.gen.ts` |
| 文件路由迁移后 `lint && typecheck && test && build` | 0 | 26 tests；2377 modules；路由级 chunks 与 494.24 kB DashboardTrendChart 独立 chunk |
| `docker compose -p appkernia_docker_test build admin`（文件路由版） | 0 | Node 24.18.1 production 镜像构建成功，Admin healthy |
| 文件路由版完整 Docker 双语 E2E | 0 | 登录/注册/找回/重置、直接 URL 守卫、Dashboard/Profile、14 组 axe 与 4 视口全部通过 |
| `pnpm audit --audit-level=high --registry=https://registry.npmjs.org` | 0 | 新增 Router 构建依赖后无已知漏洞 |
| `pnpm peers check` | 0 | 无 peer dependency 问题 |
| `psql ... iam.me.device.remove / iam.devices` | 0 | 最新设备审计为 200/succeeded；验收账号保留 2 个已登记 Web 设备 |
| Dashboard 专项 `ui-ux-pro-max` 三条本地查询 | 0 | 保存 request、skill output、decisions、review checklist 与 Dashboard page override |
| `make sqlc-generate`（登录审计租户归属） | 0 | 已知账号失败事件通过活动 tenant membership 归属租户，未知账号仍为空 |
| 首次 IAM + Dashboard 集成重跑（旧 DB 密码） | 1 | PostgreSQL SASL 拒绝；读取当前 Compose 开发连接后重跑，不计作代码通过 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/iam/... ./internal/modules/dashboard/... -count=1 -v` | 0 | IAM 4 个 Application + 2 个 Repository、Dashboard 1 个 Repository PostgreSQL 场景及相关单元测试通过 |
| `docker compose -p appkernia_docker_test up -d --build api` | 0 | 最新登录审计和 Dashboard API 镜像构建；migrate/seed 退出 0，API healthy |
| Docker 健康等待脚本首轮 | 1 | zsh 保留只读变量 `status`；改用任务专用变量后退出 0，未影响容器 |
| 最终 Dashboard 双语视觉 E2E | 0 | PostgreSQL 7 天趋势断言 success=13、failure=1；4 视口、9 组 axe 均无 violations，lazy chunk 已请求 |
| Backend/Admin/Mobile/backend-i18n 四个 blueprint validator（Dashboard 收口） | 0 | 109 existing APIs + 31 deltas；0 errors/warnings；i18n parity 通过 |
| `make check`（server） | 0 | gofmt、go vet、默认 Go tests 通过 |
| `pnpm --filter @appkernia/admin check`（Dashboard 收口） | 0 | 生成 API/i18n/48 routes、lint、typecheck、25 tests、Vite build、Admin blueprint 全通过 |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run` | 0 | Race、静态检查与 lint 通过；0 issues |
| `govulncheck ./...` | 0 | 可达漏洞 0；另有 2 个依赖漏洞不在当前调用路径 |
| `npx @redocly/cli lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅 1 个既有 proprietary license warning |
| AKADM-110 `ui-ux-pro-max` 用户列表/详情专项查询 | 0 | 保存 request、skill output、decisions、review checklist 与两个 page override |
| `make sqlc-generate`（用户管理查询） | 0 | 用户、角色、组织分配、会话、导入导出所需 sqlc 代码生成成功 |
| `AK_TEST_DATABASE_URL=... go test -p=1 -tags=integration ./internal/modules/iam/... ./internal/modules/dashboard/... ./internal/modules/storage/... ./internal/modules/org/... ./internal/modules/useradmin/... -count=1 -v` | 0 | PostgreSQL 18 全部集成场景通过；用户专项覆盖租户隔离、引用校验、启停、密码、会话与脱敏审计 |
| `make check`（AKADM-110 最终） | 0 | 四类蓝图/i18n、Go vet/tests、Admin API/i18n/routes 生成、lint、typecheck、28 tests、build、Mobile 静态检查全部通过 |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run ./...`（AKADM-110） | 0 | Race 和静态检查通过；golangci-lint 为 0 issues |
| `govulncheck ./...`（AKADM-110） | 0 | 可达漏洞 0；1 个包和 1 个模块提示不在调用路径 |
| 默认镜像源 `pnpm audit --audit-level high` | 1 | `npmmirror` 未实现 audit endpoint；未伪报依赖安全通过 |
| `pnpm --config.registry=https://registry.npmjs.org audit --audit-level high` | 0 | 官方 registry 返回无已知漏洞 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；保留 1 个既有 proprietary license warning |
| `docker compose -p appkernia_docker_test build api admin && ... up -d` | 0 | Go 1.26.5 API 与 Node 24.18.1 Admin 真实镜像构建，API/Admin healthy |
| 本地 Docker 测试库精确 E2E fixture 清理事务 | 0 | 仅删除固定 `managed-/bulk-/browser-register-` 邮箱与 `Platform/Engineering + 8位后缀` 节点；不可恢复但由下一轮 E2E 重建，未触碰 Docker Admin |
| AKADM-110 Chromium E2E 开发轮次 | 1 | 真实发现无 Outlet、选择器遮挡、插值格式混用、损坏 PNG fixture、1024px 表格溢出、375px 动作列遮挡及测试路由衔接问题；均修复并重跑 |
| 最终 `AK_E2E_BASE_URL=http://localhost:4173 ... /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py` | 0 | 创建 201、角色/组织替换 200、导入 2 行、批量禁用 200/200、导出下载、URL 恢复、匿名守卫、双语与 4 视口；26 组 axe 无 serious/critical |
| AKADM-120/130 `ui-ux-pro-max` 本地真实查询（租户、角色、权限目录、菜单） | 0 | request、skill output、decisions、review checklist、3 个 access page override 和截图索引均已保存 |
| `make check`（AKADM-130 最终前回归） | 0 | Backend/Admin/Mobile 蓝图、i18n、Go vet/tests、Admin 31 tests、lint/typecheck/production build 全通过 |
| `make check`（Ant 6 label-search 修复后的首轮） | 2 | lint 拒绝 deprecated 顶层 `optionFilterProp`；改用 `showSearch.optionFilterProp` 后重跑 |
| `make check`（Ant 6 `showSearch.optionFilterProp` 修复后） | 0 | Backend/Admin/Mobile 蓝图、i18n、Go vet/tests、Admin 31 tests、lint/typecheck/Vite production build 与 Mobile 静态检查全部通过 |
| `go test -p=1 -tags=integration ... ./internal/modules/accessadmin/repository`（首次默认密码） | 1 | 当前 Docker PostgreSQL 返回 SQLSTATE 28P01；未计为通过 |
| 从当前测试容器读取密码到不输出的 shell 变量后重跑同一集成套件 | 0 | 8 个 PostgreSQL 包通过；Access Control 覆盖租户隔离、分离写入、custom scope、深度、循环、component key 与审计 |
| `go test -race ./...` / `staticcheck ./...` / `golangci-lint run ./...` | 0 | Race 与静态检查通过；golangci-lint 0 issues |
| `govulncheck ./...` | 0 | 项目代码可达漏洞 0；1 个包与 1 个模块漏洞不在调用路径 |
| `pnpm --dir apps/ak-admin exec redocly ...` | 1 | Admin workspace 未安装该二进制；改用根目录 `npx @redocly/cli` |
| `npx --yes @redocly/cli lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅保留既有 proprietary license warning |
| `docker compose -p appkernia_docker_test up -d --build postgres migrate seed api admin` | 0 | PostgreSQL 18、Go 1.26.5 API、Node 24.18.1 Admin 真实构建；migration/seed 退出 0，API/Admin healthy |
| AKADM-130 Chromium E2E 开发轮次 | 1 | 真实发现注册目标租户装配、Ant 消息/placeholder 对比度、虚拟 Select 定位、组织 label 搜索和 Drawer 退场时序；均修复并重跑，不伪报中间轮次通过 |
| 最终 `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py` | 0 | Role 201、permission/menu/data-scope PUT 200、menu 201、租户切换 200/200；双语、4 视口、38 组 axe 全部 0 serious/critical，无意外控制台错误 |
| `docker compose -p appkernia_docker_test build admin`（Ant 6 最终源码） | 0 | Node 24.18.1 重新生成 API/i18n/routes 并完成 Vite production build，最新 Admin 镜像构建成功 |
| 首次最终源码 E2E 启动 | 1 | 一次性管理员显示名未传给测试进程，登录后的 fixture 断言失败；无业务通过结论，随后按脚本环境变量契约修正启动命令 |
| `AK_E2E_EMAIL=<ephemeral> AK_E2E_DISPLAY_NAME=<ephemeral> AK_E2E_PASSWORD=<ephemeral> AK_E2E_TENANT_CODE=<ephemeral> /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py`（最终源码） | 0 | 最新 Docker 镜像上 38 组 axe 全部 0 serious/critical、意外控制台错误 0；Access Control 与 Tenant 真实写链路、双语即时切换、375/768/1024/1440 视口通过 |
| 最终组合复核中的 `python3 blueprint/backend/scripts/validate_blueprint_specs.py` 子命令 | 2 | Backend 蓝图不存在该统一路径；实际入口为 `tools/validate_blueprint.py`，随后修正，不以组合命令最终退出码掩盖该错误 |
| `python3 blueprint/backend/tools/validate_blueprint.py` | 0 | 7 对迁移、57 张表、91 个索引、34 个触发器、105 个外键引用静态一致性通过，0 errors / 0 warnings |
| `python3 blueprint/admin-frontend/scripts/validate_blueprint_specs.py` / `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` / `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | Admin、Mobile 与跨端 zh-CN/en-US 契约最终复核通过；Mobile 仅为蓝图静态校验，不代表平台构建 |
| AKADM-140 `ui-ux-pro-max` 设计系统/UX/React/Web 查询 | 0 | 5 条项目本地命令真实执行；request、输出、决策、检查表与 Security Audit override 已保存 |
| `make sqlc-generate`（审计与在线会话） | 0 | 审计分页/详情/处理及在线会话列表/锁定/撤销/Refresh/Audit 查询生成成功 |
| 审计 PostgreSQL 集成首轮 | 1 | 夹具调用未安装的 `digest()`；移除非必要 Hash 夹具后重跑，不把首轮计为通过 |
| `go test -tags=integration ./internal/modules/auditadmin/...` | 0 | 租户隔离、嵌套脱敏、安全登录提示、跨租户 404、仅处理一次与不可变审计通过 |
| AKADM-140 Admin lint 首轮 | 1 | 真实发现 Ant 6 `Alert.message` deprecated 和未使用 import；改为 `title` 并清理后通过 |
| AKADM-140 Chromium 开发轮次 | 1 | 依次暴露 JSON strict locator、Drawer 动画对比度、隐藏 Select、表格可聚焦性、夹具 UUID 状态行、详情父路由缺少 Outlet 与 Modal 重复标题；逐项修复，所有非零轮次均未冒充通过 |
| AKADM-150 `ui-ux-pro-max` 设计系统/UX/Web/React 查询 | 0 | 4 条项目本地命令真实执行；Online Sessions request/output/decisions/checklist 与 page override 已保存 |
| `go test ./internal/modules/sessionadmin/...` | 0 | 精确权限、认证 tenant/current-session 作用域和非法筛选单元测试通过 |
| `AK_TEST_DATABASE_URL=<local> go test -p=1 -tags=integration ./internal/modules/sessionadmin/... -count=1 -v` | 0 | 脱敏账号/IP、当前标记、租户隔离、跨租户拒绝、Session/Refresh 撤销、审计和重复撤销通过 |
| `pnpm ... redocly lint server/openapi/openapi.yaml` 首轮 | 1 | 新增 inline description 未加引号导致 YAML 逗号被解析为属性；加引号后复核通过 |
| `pnpm ... redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅保留既有 proprietary license warning |
| `make check`（AKADM-140/150 收口前回归） | 0 | 四类蓝图/i18n、Go vet/tests、Admin 31 tests、lint/typecheck/Vite build 与 Mobile 静态检查通过 |
| `go test -race ./... && staticcheck ./... && golangci-lint run ./... && govulncheck ./...` | 0 | Race、静态检查、lint 0 issues、项目可达漏洞 0；另有 1 个包和 1 个模块漏洞不在调用路径 |
| `docker compose -p appkernia_docker_test build api admin` | 0 | Go 1.26.5 与 Node 24.18.1 production 镜像真实构建成功 |
| 审计 E2E SQL 夹具首次 `psql -c` | 1 | Shell 引号导致 UUID 被当作数值；改为安全双引号 SQL 后两条 INSERT 退出 0，不把首次失败计为夹具成功 |
| AKADM-140/150 最终 E2E 环境与数据累积轮次 | 1 | 依次暴露 Compose feature flags 未带入、租户名与显示名重复、多个失败登录 strict locator、多个活动会话下 `.first` 漂移；修正环境和测试绑定固定 row key 后重跑 |
| `AK_E2E_*=<ephemeral> /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_visual.py`（最终源码） | 0 | 48 个 axe 分组全部零 violation；audit resolve=200、session revoke=200、服务端脱敏、列表刷新、匿名守卫、双语即时提示和意外控制台错误 0 |
| `AK_TEST_DATABASE_URL=<local> go test -p=1 -tags=integration ./internal/modules/iam/... ./internal/modules/dashboard/... ./internal/modules/storage/... ./internal/modules/org/... ./internal/modules/useradmin/... ./internal/modules/tenantadmin/... ./internal/modules/accessadmin/... ./internal/modules/auditadmin/... ./internal/modules/sessionadmin/... -count=1 -v` | 0 | 9 个模块树共 43 个具名测试串行通过，覆盖租户隔离、精确权限、脱敏、事务撤销与不可变审计 |
| `make check`（AKADM-140/150 最终） | 0 | Backend/Admin/Mobile 蓝图与 i18n、Go vet/tests、Admin 31 tests、lint/typecheck/Vite production build、Mobile 静态检查全部通过 |
| `pnpm --filter @appkernia/admin exec prettier --version` | 1 | 项目未安装 Prettier；未伪造格式化结果，改用 `apply_patch` 并由 ESLint/typecheck 验证 |
| `pnpm dlx @redocly/cli lint ...` | 1 | 新版包暴露多个 binary，无法自动选择；按提示显式选择 `redocly` 后重跑 |
| `pnpm --package=@redocly/cli dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅保留既有 proprietary license warning |
| `python3 -m py_compile ...`、两份 locale JSON、`git diff --check`、Docker health | 0 | E2E 脚本语法、双语 JSON、补丁格式通过；PostgreSQL/API/Admin 均 healthy |
| AKADM-200 `ui-ux-pro-max` 真实查询 | 0 | System Configs/Dictionaries 的 request、skill output、decisions、review checklist 与两个 page override 已保存 |
| `AK_TEST_DATABASE_URL=<local> go test -tags=integration ./internal/modules/systemsettings/... -count=1 -v` | 0 | 公共配置隔离、Secret 不泄漏、配置/字典事务行为，以及 250 子节点地区懒加载和编译期模块过滤通过 |
| 公共配置 Docker SQL/HTTP smoke 首轮 | 1 | 首轮 psql 变量引号错误发生在插入前；修正后插入 public/private/secret 夹具，HTTP 仅暴露 public，随后按固定键精确清理 |
| AKADM-200 Chromium 开发轮次 | 1 | 真实发现生成 namespace/raw key、Secret 交互与 Ant Tag 对比度问题；逐项修复后重跑，不把中间轮次计为通过 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_system_settings.py` | 0 | 两种语言、375/1440、配置/字典共 6 个场景全部零 axe violation；Secret 创建/轮换不泄漏，意外控制台错误 0 |
| AKADM-210 `ui-ux-pro-max` 5 条真实查询 | 0 | Regions/Modules 的 request、skill output、decisions、review checklist 与两个 page override 已保存 |
| AKADM-210 Chromium 开发轮次 | 1 | 真实发现 namespace、Tag 对比度、未加载树节点展开按钮、移动端可滚动区域焦点和测试 selector 问题；修复后重跑 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_regions_modules.py` | 0 | 地区/模块双语及 375/1440 共 6 个场景零 axe violation；子节点请求 200，意外控制台错误 0 |
| `make check`（AKADM-200/210 最终） | 0 | Backend/Admin/Mobile 蓝图、跨端 i18n、Go gofmt/vet/tests、Admin 7 文件/31 tests、lint/typecheck/Vite build、Mobile 静态检查全部通过 |
| `go test -race ./internal/modules/systemsettings/... ./internal/modules/platform/...` | 0 | 配置/字典/地区/模块及公共配置相关包 race 专项通过 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；仅保留既有 proprietary license warning |
| `python3 -m py_compile apps/ak-admin/scripts/e2e_system_settings.py apps/ak-admin/scripts/e2e_regions_modules.py && git diff --check` | 0 | 两份专项 E2E 脚本语法与补丁格式最终通过 |
| `docker compose -p appkernia_docker_test ps` | 0 | PostgreSQL 18、Go API、Node 24 Admin 均为 healthy |
| AKADM-220 `ui-ux-pro-max` 5 条真实查询 | 0 | 文件管理/AkFilePicker request、skill output、decisions、review checklist 与 System Files override 已保存；375/1440 与生产构建项已按真实结果更新 |
| `make sqlc-generate`（AKADM-220） | 0 | 生成文件上传会话、分片、文件、引用和审计所需 pgx/sqlc 查询代码 |
| `gofmt ...` 首次使用重复 `server/` 路径 | 2 | 工作目录已在 `server`，路径不存在；改为模块内相对路径后通过，未隐藏该失败 |
| `go test ./internal/modules/storageadmin/... ./internal/modules/storage/... -count=1` | 0 | multipart resume/complete、非法分片/权限、扫描门禁、未知浏览器 MIME 和头像回归通过 |
| PostgreSQL 专项首轮 | 1 | 隔离库缺失/历史半迁移；后续仅重建明确命名的 `appkernia_test`，未触碰开发库 `appkernia` |
| `dropdb --force appkernia_test && createdb ...`、7 个 migration、`seed core` | 0 | 仅重建隔离测试库；version=7、dirty=false，108 permissions、35 menus |
| `AK_TEST_DATABASE_URL=<local-test> go test -tags=integration ./internal/modules/storageadmin/... ./internal/modules/storage/... -count=1 -v` | 0 | 10 个具名测试通过；覆盖租户隔离、usage 删除保护、事务审计、私有对象和头像回归 |
| `python3 apps/ak-admin/scripts/e2e_files.py` 首轮 | 1 | 系统 Python 缺 Playwright；切换项目既有 `/Users/payhon/.venv/3.12` 运行时 |
| AKADM-220 Chromium/Docker 开发轮次 | 1 | 依次真实发现旧 API 镜像 404、Nginx 请求体限制、语言偏好 selector、插值、进度条名称、按钮/空表对比度和空表滚动；逐项修复，非零轮次均未伪报通过 |
| `docker compose build api/admin` 与 `up -d --force-recreate` | 0 | Go 1.26.5 API、Node 24.18.1 Admin 最新镜像真实构建；PostgreSQL/API/Admin 均 healthy |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_files.py`（最终源码） | 0 | 双语 1440/375、隐藏详情深链及 5 个 axe 场景零 violation；断点续传 201、下载、delete-in-use、解除引用删除、取消和审计通过 |
| `make check`（AKADM-220 首轮） | 2 | 新 PostgreSQL 集成测试尚未 gofmt；格式化后重跑，不把首轮计为通过 |
| `make check`（AKADM-220 最终） | 0 | Backend/Admin/Mobile 蓝图与 i18n、Go fmt/vet/tests、Admin 7 文件/31 tests、lint/typecheck/Vite build、Mobile 静态检查全部通过 |
| `GOTOOLCHAIN=go1.26.5 go test -race ./...` | 0 | Backend 全量 race 通过 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；1 个既有 license warning，另有 3 个因蓝图固定 `/files/upload-sessions/{id}` 与 `/files/{id}/*` 结构产生的 ambiguous-path warnings；Docker HTTP E2E 已验证真实路由 |
| `python3 blueprint/scripts/validate_i18n_contract.py` / Backend/Admin/Mobile validators / `git diff --check` | 0 | 双语 key/占位符、三个蓝图、迁移/schema/权限/API 分类和补丁格式通过 |
| AKADM-230 项目内 `ui-ux-pro-max` 5 条真实查询 | 0 | 通知 request、skill output、decisions、review checklist、通知 Master 与页面 override 已保存 |
| `GOTOOLCHAIN=go1.26.5 go test ./internal/modules/notificationadmin/... -count=1 -v` | 0 | 3 个 Application 测试通过，覆盖 HTML allowlist、模板变量校验和收件范围 |
| 在仓库根目录误执行 `go test -tags=integration ./internal/modules/notificationadmin/...` | 1 | 当前目录不是 Go module；切换 `server` 后使用相同目标重跑，未隐藏首轮失败 |
| `AK_TEST_DATABASE_URL=<local-test> GOTOOLCHAIN=go1.26.5 go test -tags=integration ./internal/modules/notificationadmin/... -count=1 -v`（`server`） | 0 | 3 个 Application + 1 个 PostgreSQL Repository 测试通过，覆盖租户隔离、收件物化、投递约束与审计 |
| `python3 apps/ak-admin/scripts/e2e_notifications.py` 首轮 | 1 | 系统 Python 缺 Playwright；切换项目既有 `/Users/payhon/.venv/3.12` 运行时 |
| AKADM-230 Chromium/Docker 开发轮次 | 1 | 真实发现通知路由白名单、label/id、选择器时序、对比度和移动 Drawer 裁切问题；逐项修复后重跑 |
| 默认 `docker compose up` 使用 Admin 端口 4173 | 1 | 宿主端口已占用；改用 `AK_ADMIN_PORT=4174` 后启动成功 |
| `AK_ADMIN_PORT=4174 docker compose up --build -d seed api admin` | 0 | Go 1.26.5 API、Node 24.18.1 Admin 最新源码镜像构建；PostgreSQL/API/Admin healthy |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_notifications.py`（最终源码） | 0 | 双语、375/768/1024/1440 共 10 个 axe 场景零 violation；两个精确收件计数均为 1，重试 HTTP 200，意外控制台错误 0 |
| `make check`（AKADM-230 最终） | 0 | Backend/Admin/Mobile 蓝图与 i18n、Go fmt/vet/tests、Admin 8 文件/33 tests、lint/typecheck/Vite build、Mobile 静态检查全部通过 |
| `GOTOOLCHAIN=go1.26.5 go test -race ./...`（`server`） | 0 | Backend 全量 race 通过 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；保留 1 个既有 license 与 3 个既有 ambiguous-path warning |
| `python3 -m py_compile apps/ak-admin/scripts/e2e_notifications.py`、`git diff --check`、`docker compose ps` | 0 | E2E 脚本语法、补丁格式和最终 PostgreSQL/API/Admin health 均通过 |
| 最终文件核对首轮使用 zsh 保留变量 `path` 且正则含未转义反引号 | 127 | 反引号被执行，`path` 覆盖命令搜索路径，导致后续 docker/git 未找到；改用单引号与 `artifact_path` 后重跑 |
| AKADM-240 项目内 `ui-ux-pro-max` 5 条真实查询 | 0 | Schedules request、skill output、decisions、review checklist、Master 与页面 override 已保存 |
| `python3 blueprint/backend/tools/validate_blueprint.py` 首轮 | 1 | 新 migration 起初缺少蓝图要求的显式 `BEGIN/COMMIT`；补齐后重跑通过，未隐藏首轮失败 |
| `python3 blueprint/backend/tools/validate_blueprint.py`（最终） | 0 | 8 组 up/down migrations、57 tables、93 indexes、34 triggers、105 foreign-key references，0 warning |
| `GOTOOLCHAIN=go1.26.5 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` | 0 | 从 migration 8 重新生成 pgx/sqlc 数据库类型 |
| `GOTOOLCHAIN=go1.26.5 go test ./internal/modules/jobadmin/... ./internal/bootstrap ./cmd/ak-worker ./cmd/ak-cli -count=1` | 0 | Cron/DST、handler/payload 限制、默认值、API bootstrap、CLI 与 Worker 编译通过 |
| 未给 test DSN 加引号 / 在仓库根执行 Go 集成测试 | 1 / 1 | zsh 将 `?` 当 glob，随后根目录不是 Go module；加引号并切换 `server` 后重跑通过 |
| `docker compose run --rm -e AK_DATABASE_URL=<local-test> migrate` | 0 | 隔离测试库实际执行应用 migration version=8、River 官方 migrations=7、dirty=false |
| `AK_TEST_DATABASE_URL=<local-test> GOTOOLCHAIN=go1.26.5 go test -tags=integration ./internal/modules/jobadmin/repository -count=1 -v` | 0 | 1 个 PostgreSQL 场景通过：租户隔离、状态迁移、幂等单次 River 入队、重叠冲突和 4 条审计 |
| `AK_ADMIN_PORT=4174 docker compose up --build -d postgres migrate seed api worker admin` | 0 | Go 1.26.5 API/Worker、Node 24.18.1 Admin 真构建；应用 migration=8、River migrations=7；PostgreSQL/API/Admin healthy、Worker running |
| `apps/ak-admin/scripts/e2e_schedules.py` 开发轮次 | 1 | 真实发现直达 reload 等待、结果 JSON selector、Descriptions/Select 对比度和无焦点内部滚动；逐项修复后重跑 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_schedules.py`（最终） | 0 | 真实创建、DST 预览 5 次、执行成功、暂停/恢复、URL 状态、无刷新双语切换；8 组 axe 零 violation、控制台错误 0、审计 4 |
| 将专项任务 `next_run_at` 设为到期并轮询 25 秒上限 | 0 | 15 秒动态调度器真实产生 `trigger_type=schedule` 运行，最终落库 `succeeded|schedule|JOBS.HEALTH.OK` |
| `make check`（`server`） | 0 | 全量 gofmt/vet/unit tests 通过；jobadmin Application 测试纳入全量门禁 |
| `GOTOOLCHAIN=go1.26.5 go test -race ./internal/modules/jobadmin/... -count=1` | 0 | 定时任务 Application/Repository/HTTP/Worker 专项 race 通过 |
| `pnpm --filter @appkernia/admin check` | 0 | API/i18n/route 生成、ESLint、strict TypeScript、9 文件/37 tests、Vite production build、Admin 蓝图校验通过；宿主 Node 26 仅记录 engine warning |
| Backend/Admin/Mobile 三个 validator + `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | 三蓝图 0 error；Admin 131 existing APIs + 14 deltas；`zh-CN`/`en-US` key 与占位符完全一致 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；保留 1 个既有 license 和 3 个既有 files ambiguous-path warning |
| 最终文档搜索首轮正则含未转义反引号 | 127 | zsh 执行了反引号内文本；改用不含命令替换的安全搜索后复核，不影响文件 |
| `python3 -m py_compile apps/ak-admin/scripts/e2e_schedules.py`、`git diff --check`、私钥签名扫描 | 0 | E2E 脚本语法、已跟踪补丁格式和源码私钥扫描通过；工作树整体仍为未跟踪，未 commit/push |
| `docker compose ps -a`（最终） | 0 | PostgreSQL/API/Admin healthy，Worker running，migrate/seed jobs 均 Exited (0) |

## AKADM-250—310 追加交付

- `AKADM-250` API Client/Webhook 已完成：一次性凭据披露、Hash/密封存储、SSRF 校验、签名投递、状态/重试、精确权限、租户隔离、审计与 Admin 双语页面形成真实闭环。
- `AKADM-260` 访问规则/服务状态已完成：规则影响确认、脱敏主体、服务健康读取、权限/数据库/OpenAPI 契约及 Admin 双语交互一致。
- `AKADM-300` 完整 MFA/OAuth 绑定已完成：RFC 6238 TOTP、AES-GCM Secret、仅 Hash 恢复码、密码/TOTP step-up、单次 state、S256 PKCE、绑定/解绑和回放拒绝均有 Backend/PostgreSQL/Admin/Chromium 证据。
- OAuth local-mock 只在 development Feature Flag 下启用；production 配置为 local-mock 会拒绝启动。没有第三方 Provider 凭据，因此未把本地 Mock 声称为真实外部账号绑定。
- `AKADM-310` 最终硬化已完成：项目内 `ui-ux-pro-max` 产物、Vite manifest、bundle budget、strict lint/typecheck/unit/build、双语全量 E2E、axe、reduced-motion 和响应式回归均进入最终门禁。
- Admin API/OpenAPI、权限 seed、migration/sqlc、Backend embedded catalog、18 个 Admin namespace 和生成 Client 已同步；PostgreSQL 18 application migration 最终为 version=9、dirty=false。

### 本阶段真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `.codex/skills/ui-ux-pro-max`：1 次 `--persist` design-system + UX/React/Web 3 次搜索 | 0 | 真实生成 `design-system/appkernia-admin-hardening/MASTER.md` 与 AKADM-310 request/output/decisions/checklist；采纳焦点、对比度、reduced-motion、语义标签，拒绝营销视频/外部字体/scroll-snap |
| `python3 blueprint/backend/tools/validate_blueprint.py`（AKADM-300 初轮） | 1 | OAuth challenge 契约初轮校验失败；修正 migration/schema/spec 后最终重跑，不隐藏失败 |
| `python3 blueprint/backend/tools/validate_blueprint.py`（最终） | 0 | 9 组 migration、58 tables、95 indexes、107 foreign-key references，0 error/warning |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint server/openapi/openapi.yaml`（AKADM-300 初轮 / 最终） | 1 / 0 | 初轮 Conflict `$ref` 与 YAML 逗号问题已修复；最终 OpenAPI 有效，保留 4 个既有 warning |
| `go test ./internal/modules/identitysecurity/... -count=1` | 0 | TOTP、恢复码、step-up、feature flag、OAuth state/PKCE 生命周期与 replay 拒绝通过 |
| identitysecurity PostgreSQL integration（初轮 / 最终） | 1 / 0 | 初轮审计列名使用错误，改为 `client_ip` 后通过真实 migration 9 与租户/单次消费验证 |
| `apps/ak-admin/scripts/e2e_identity_security.py` 开发轮次 | 1 | 依次暴露 Tag 对比度、Result 非语义标题、Popconfirm selector、预期 403 console 分类与 fixed skip-link 截图拼接问题；逐项修复 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_identity_security.py`（最终） | 0 | 7 个双语/响应式 axe 场景零 violation；MFA verify=200、错误 proof=403、rotate=200、OAuth replay=422、unbind=200、disable=200，一次性值未持久化、意外 console error=0 |
| `pnpm --filter @appkernia/admin test:e2e`（初轮 / 最终） | 1 / 0 | 初轮临时租户缺默认 `member` role 导致 register=503；补齐隔离 E2E fixture 后 48 个 axe 场景全为 0 critical/serious，全部业务流通过 |
| `pnpm --filter @appkernia/admin check`（初轮 / 最终，Node 24.18.1） | 2 / 0 | 初轮 `exactOptionalPropertyTypes` 拒绝显式 `className={undefined}`；改为条件 spread 后 API/i18n/routes 生成、lint、strict typecheck、9 文件/39 tests、production build、bundle budget、Admin validator 全部通过 |
| `node apps/ak-admin/scripts/check-bundle-budget.mjs` | 0 | initial gzip 200,604 B ≤ 307,200 B；最大 chunk 165,895 B ≤ 184,320 B；结果写入 `output/admin-bundle-budget.json` |
| `make check`（`server`） | 0 | gofmt、vet、全量 unit tests 通过 |
| `go test -race ./... -count=1`（`server`） | 0 | Backend 全量 race 通过 |
| `AK_TEST_DATABASE_URL=<local-test> go test -p=1 -tags=integration ./internal/modules/... -count=1`（`server`） | 0 | 全部可用模块 PostgreSQL 集成测试串行通过，包含 identitysecurity |
| `staticcheck ./...` / `golangci-lint run ./...` | 0 / 0 | 静态检查通过；golangci-lint 报告 0 issues |
| `govulncheck ./...` | 0 | 无被当前调用链触达的漏洞；工具仍报告 1 个 package/module 层面但未调用的漏洞信息 |
| Backend/Admin/Mobile validators + `python3 blueprint/scripts/validate_i18n_contract.py` + UI Skill 检查 | 0 | 三蓝图、双语 key/占位符、API/权限/schema 分类和项目本地 Skill 均通过 |
| `docker compose up -d --build migrate seed api worker admin` | 0 | PostgreSQL 18、Go API/Worker、Node 24.18.1 Admin 真实构建运行；migration/seed jobs exit 0，服务健康 |
| E2E/截图产物敏感值扫描 | 0 | identity/full E2E JSON 未保存 TOTP Secret、恢复码、OAuth code/state 或凭据 fixture |
| `AK_ADMIN_PORT=4174 docker compose build admin`（最终源码） | 0 | Node 24.18.1 重新安装锁定依赖、生成 API/i18n/routes、构建 2,539 modules，Vite manifest 与最终 Admin 镜像成功产出 |
| `AK_ADMIN_ORIGIN=http://127.0.0.1:4174 AK_ADMIN_PORT=4174 docker compose up -d --no-deps --force-recreate api admin` | 0 | 撤销 E2E 临时 feature flags 并恢复 Compose 默认开发配置；API/Admin 均以最终镜像重建 |
| 最终 health 探针首轮 / 修正后 | 1 / 0 | 首轮误用 zsh 只读变量 `status`，未影响容器；改用 `service_health` 后 Admin/API/PostgreSQL healthy，migrate/seed exit 0，public config 显示 registration/avatar/password recovery=false、oauth=true |
| `python3 -m py_compile ...`、三份结果 JSON parse、`git diff --check` | 0 | 身份 E2E 语法、结果与 bundle JSON 均有效，补丁格式通过 |
| `python3 blueprint/scripts/check_ui_skill.py` / `bash blueprint/admin-frontend/scripts/check_ui_skill.sh` | 2 / 0 | 首个路径不存在；改用蓝图真实脚本后确认 `FOUND:.codex/skills/ui-ux-pro-max/scripts/search.py` |

## 浏览器与视觉证据

浏览器：Playwright 1.54.0 / Chromium，最新 Docker Admin 镜像 + 真实 Go API + PostgreSQL 18。

当前验收 project `appkernia` 仍在本机运行：Admin `localhost:4174`、PostgreSQL `localhost:55432`；PostgreSQL、API 与 Admin 均为 healthy。

- axe：全局套件 48 组全部零 critical/serious violation；另有 System Configs/Dictionaries 6 组、Regions/Modules 6 组、Files 5 组、Notifications 10 组、Schedules 8 组、API/Webhook Integrations 11 组、Access Rules/Service Status 8 组和 Identity Security 7 组专项结果全部零 violation。
- 视口：375、768、1024、1440。
- 行为：在既有认证/Profile/Dashboard/组织/用户/租户/访问控制链路上，新增审计操作 JSON 脱敏、安全事件详情与处理、登录结果 URL 筛选、在线会话脱敏提示、固定目标会话撤销与列表刷新。强制下线成功提示在不刷新页面的语言切换后即时更新；菜单深度/循环、跨租户破坏及 Refresh family 撤销由 PostgreSQL 集成测试覆盖，不把未执行的浏览器破坏操作冒充通过。
- 结果 JSON：[admin-e2e-axe-results.json](../output/playwright/admin-e2e-axe-results.json)
- 系统设置结果：[admin-system-settings-e2e-results.json](../output/playwright/admin-system-settings-e2e-results.json)
- 地区/模块结果：[admin-regions-modules-e2e-results.json](../output/playwright/admin-regions-modules-e2e-results.json)
- 文件存储结果：[admin-files-e2e-results.json](../output/playwright/admin-files-e2e-results.json)
- 通知管理结果：[admin-notifications-e2e-results.json](../output/playwright/admin-notifications-e2e-results.json)
- 定时任务结果：[admin-schedules-e2e-results.json](../output/playwright/admin-schedules-e2e-results.json)
- 身份安全结果：[admin-identity-security-e2e-results.json](../output/playwright/admin-identity-security-e2e-results.json)
- Bundle budget：[admin-bundle-budget.json](../output/admin-bundle-budget.json)
- 截图索引：[SCREENSHOT_INDEX.md](../artifacts/ui-ux-pro-max/SCREENSHOT_INDEX.md)

## 设计决策

`ui-ux-pro-max` 输出直接影响了实现：采用企业蓝/深海军蓝、数据密集但可访问的布局、可见焦点、无位移 hover、系统字体、reduced-motion、破坏性操作二次确认、明确当前会话与异步状态。原始 Skill 的 Google Fonts、营销 CTA 和 hover transform 建议被保留为原始产物，但在 decisions 中基于性能、隐私、语义与布局稳定性覆盖。

## 未完成项与风险

- Admin task backlog `AKADM-000`—`AKADM-310` 已按依赖图收口；不再把已完成的 P2/P3 页面或完整 MFA 记为后续项。
- 密码恢复的真实邮件供应商未联调：缺少第三方凭据时按契约使用开发期本地内存 Adapter，生产环境启用 local Adapter 会拒绝启动；浏览器 E2E 未冒充“收到邮件并点击链接”。
- `AKADM-060/300` 均已完成；TOTP 与本地 OAuth Mock 已通过真实 API/PostgreSQL/Chromium 验证。真实外部 OAuth Provider 因无第三方凭据未联调，production 明确禁止 local-mock。
- `AKADM-070` 已完成；后续 P1 Dashboard 指标会随用户、组织、任务、通知等业务模块完善而自然增加真实数据量。
- `AKADM-100` 已完成；部门与岗位的后端授权、租户隔离、递归防环、占用冲突、审计和 Admin UI 已闭环。
- `AKADM-110` 已完成；用户列表/详情、启停/解锁/重置、角色和组织分配、会话、导入导出及批量动作已闭环。
- `AKADM-120/130/140/150/200/210/220/230/240/250/260/300/310` 已完成；当前 Admin 蓝图没有剩余依赖任务。
- 地区生产数据的版本化导入 CLI 尚未实现；当前验证的是真实 PostgreSQL 查询、250 子节点懒加载和 Admin UI，不把测试夹具等同于生产地区数据管道。
- `AKADM-220` 的 multipart cancel/resume、scan gate、usage viewer、`AkFilePicker` 与 delete-in-use Backend/Admin 闭环已完成。生产对象存储和实际恶意软件扫描 Adapter/Worker 因未配置第三方设施尚未联调；本轮 development-local 完成后标记 `skipped`，pending/infected/failed 均 fail-closed，不能据此声称生产扫描通过。
- OpenAPI 最终 lint 退出 0 但保留 4 个 warning：1 个既有 proprietary license warning，3 个由蓝图固定 `/files/upload-sessions/{id}` 与 `/files/{id}/usages|presign-download|content` 产生的结构歧义提示；真实 GoFrame 路由和 Docker HTTP E2E 已通过，但文档工具 warning 未隐藏。
- Backend 为当前 Admin 蓝图提供的注册/密码恢复、自作用域、业务管理、API Client/Webhook、访问规则、服务状态和 MFA/OAuth 契约均已闭环。生产对象存储、恶意软件扫描、通知投递和外部 OAuth Adapter 因未提供第三方凭据/设施尚未联调；各 local Adapter 在 production 均 fail-closed。
- AKADM-140/150 页面已实际复核 375/1440；其 `ui-ux-pro-max` checklist 对“每页 375/768/1024/1440 全截图”和在线会话 reload 恢复项保持未勾选，未把全局四视口覆盖冒充为每页四视口覆盖。
- AKADM-210 页面实际复核 375/1440；768/1024 专项截图仍未执行，checklist 保持未勾选。仅 Chromium 已验证，未给出 Firefox/Safari 结论。
- AKADM-220 页面实际复核 375/1440；768/1024 专项截图仍未执行，checklist 保持未勾选。文件上传失败/续传、引用警告和双语状态均在 Chromium 验证，未给出 Firefox/Safari 结论。
- AKADM-230 通知专项实际覆盖 375/768/1024/1440 和 `zh-CN`/`en-US`，但不是每个通知页面都在每个视口分别截图；仅 Chromium 已验证，未给出 Firefox/Safari 结论。
- AKADM-240 定时任务专项实际覆盖 375/768/1024/1440、`zh-CN`/`en-US`、空态/编辑预览/成功运行与 URL 状态；仅 Chromium 已验证，未给出 Firefox/Safari 结论。River Worker 已在本地 PostgreSQL 18 实际消费，但未部署生产。
- 生产通知供应商 Adapter/Worker 与真实外部送达因缺少第三方设施/凭据尚未联调；本轮失败重试验证的是服务端状态约束和安全入队，不能据此声称邮件、短信或推送实际送达。
- 本轮 Vite production build 与 bundle budget 通过，但未部署到生产；仅真实验证 Chromium，没有 Firefox/Safari 兼容性结论。
- Android、iOS、Harmony 本轮未实际执行构建或真机测试，均为 blocked/未验证，不写 passed。
- Playwright Skill wrapper 本身不可用；本轮使用本机 Playwright 真实浏览器替代，wrapper 缺陷仍需上游修复。
- Git 工作树全部仍为未提交文件；没有 commit，也没有 push。
- 最终 Admin 门禁与 Docker 构建使用项目要求的 Node 24.18.1 并通过；较早宿主 Node 26 的 engine warning 仅为中间环境记录，不作为最终通过依据。

## 2026-08-03 Backend + Admin 最终完成性审计追加

### 新增与修正

- 补齐 4 个 registry/蓝图声明但此前没有真实可挂载页面的路由：API Client 详情、Schedule 运行记录、显式 404 与 offline。参数子路由的父文件改为 `Outlet`，列表移动到 index route；这修复了“详情 chunk 已加载但页面未挂载、GET 未发出”的真实问题。
- 新增 `GET /admin-api/v1/api-clients/{id}`，实现 UUID 校验、精确读取权限、SQL tenant filter、跨租户 not found、OpenAPI/生成 Client/Backend 蓝图同步，并增加 same-tenant/cross-tenant PostgreSQL integration。
- Admin API/权限 delta 已归零：147 existing APIs + 0 delta，108 permissions；Backend fresh seed 日志为 108 permissions、35 menus。持久化验收库当前 42 active menus，额外 7 条来自真实 E2E 创建，不属于 seed。
- API Client 详情只展示 client/permission/CIDR/Secret metadata；一次性明文不由 GET 返回。Schedule 深链展示有界服务端结果，375 px 横向区域可键盘聚焦。
- 404 增加主 landmark；offline 改为独立语义卡片与安全重试说明，未完成写请求不会自动重放。双语切换均无需刷新。

### 本次真实命令、退出码与证据

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `.codex/skills/ui-ux-pro-max`：1 次持久化 design-system + UX/React/Web 3 次搜索 | 0 | 生成 route-completion Master 与 request/output/decisions/checklist；拒绝与既有 Admin Master 冲突的营销页、视频、外部字体和 burgundy/gold token 建议 |
| `go test -tags=integration ./internal/modules/apiclientadmin/... -count=1 -v` | 0 | 同租户 GET 成功、跨租户 not found、Secret 明文不回传 |
| `make check`（`server`） | 0 | gofmt、vet、Backend unit tests 通过 |
| `AK_ADMIN_ORIGIN=http://127.0.0.1:4174 AK_ADMIN_PORT=4174 docker compose up --build -d postgres migrate seed api worker admin` | 0 | PostgreSQL 18、Go API/Worker、Node 24.18.1 Admin 真实构建；migration/seed jobs exit 0，API/Admin/PostgreSQL healthy |
| Integrations E2E 首轮 | 1 | 等待详情 GET 超时；定位为父 route 缺 `Outlet`，不是 API 路由缺失，随后修复 |
| 使用已存在 `local` tenant 的临时 bootstrap | 1 | PostgreSQL `uq_tenants_code` 冲突；改用每轮唯一 tenant code 后成功，未掩盖失败 |
| Schedules E2E 首轮 | 1 | 375 px axe 报 `scrollable-region-focusable` serious；给内部 Ant Table 横向滚动区域增加真实键盘焦点后修复 |
| Schedules E2E 中间两轮 | 1 / 1 | 预览 blur 自动请求与按文案找 loading 按钮发生竞态；最终按 `.ak-schedule-preview` 结构定位，并只接受请求体包含目标时区的响应 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_integrations.py`（最终 Docker 镜像） | 0 | 17 个 axe 场景零 violation；详情 GET、machine audience 拒绝 Admin API、Secret 撤销、Webhook mock、404/offline、两语言和 4 视口父流程通过 |
| `/Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_schedules.py`（最终 Docker 镜像） | 0 | 10 个 axe 场景零 violation；深链双语、DST 时区预览、运行 `succeeded|JOBS.HEALTH.OK|active` 与 4 条审计通过 |
| `mise exec node@24.18.1 -- corepack pnpm --filter @appkernia/admin check` | 0 | 9 files/40 tests、strict lint/typecheck、2,554 modules production build、bundle budget、Admin validator 全通过 |
| `go test -race ./... -count=1` / `staticcheck ./...` / `golangci-lint run ./...` | 0 / 0 / 0 | Backend race 与静态检查通过，golangci-lint 为 0 issues |
| `govulncheck ./...` | 0 | 当前调用链 0 vulnerability；仍提示依赖中 1 个 package/module 层面但未调用的问题 |
| 全模块 PostgreSQL integration 首轮 | 1 | `appkernia_test` 停在 migration 8，缺 `iam.oauth_binding_challenges`；确认是测试库漂移 |
| `AK_ENV=test ... ak-cli migrate up` 子进程 | 1 | 非 development 缺 JWT 私钥，按安全配置拒绝启动；外层诊断 shell 因随后查询而为 0，此处记录真实子进程失败 |
| `AK_ENV=development ... go run ./cmd/ak-cli migrate up` | 0 | `appkernia_test` 从 `8:false` 真实迁移到 `9:false` |
| `AK_TEST_DATABASE_URL=... go test -p=1 -tags=integration ./internal/modules/... -count=1`（最终） | 0 | Backend 全部可用模块 PostgreSQL integration 通过 |
| Backend/Admin/Mobile validators + i18n validator + UI Skill check | 0 | Backend 9 migrations/58 tables/95 indexes/107 FKs；Admin 48 routes/147 APIs/108 permissions/0 delta；Mobile 静态蓝图与双语契约通过 |
| `redocly lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；保留并明确记录 4 个既有 warning |
| `docker compose ps -a` + SQL/health 探针 | 0 | API/Admin/PostgreSQL healthy、Worker running、migrate/seed Exited (0)、application migration `9:false`、Admin healthz=200 |

### 最终验证边界

- 当前 Backend + Admin 蓝图 backlog `AKADM-000`—`AKADM-310` 已完成；本轮没有自动 commit 或 push。
- 仅 Chromium 在本机 Docker/PostgreSQL 18 上做了真实浏览器与视觉验证；Firefox、Safari 和生产部署未执行。
- 密码邮件、生产对象存储/恶意软件扫描、通知供应商和真实外部 OAuth Provider 缺少第三方凭据/设施，继续由 Port/Adapter、Feature Flag 与 development local mock 承接；production 对 local adapter fail-closed。
- Android、iOS、Harmony 本轮均未执行构建或真机测试，状态保持 **blocked / 未验证**，没有伪报通过。

## 2026-08-04 GitHub 开源与本地开发入口追加

### 交付内容

- MIT：新增根 `LICENSE`，根/Admin package metadata 为 `MIT`。
- 开源协作：新增 `CONTRIBUTING.md`、`SECURITY.md`，保留现有 GitHub Actions CI。
- Frontend：根与 Admin `package.json` 补齐 npm/pnpm 的 dev、test、build、preview、generate、lint、typecheck、E2E 和 check 入口；依赖安装仍以 `pnpm-lock.yaml` 为唯一可复现锁文件。
- Backend：`server/Makefile` 补齐联合/独立源码开发、API/Worker/CLI 构建、数据库初始化、doctor、测试与帮助入口；根 Makefile 提供跨端便利命令。
- Onboarding：重写根、Backend、Admin README，覆盖源码开发、全 Docker、首次管理员、代理、命令表、OpenAPI/i18n 契约和安全注意事项。
- Toolchain：doctor 可跨平台运行，必需工具缺失/版本不符会失败，移动 IDE 未安装只标 optional。

### 本轮真实命令与退出码

| 命令 / 阶段 | Exit | 结果 |
|---|---:|---|
| `make -C server help` / `make help` | 0 / 0 | 新公开入口可发现 |
| `make -C server test` | 0 | Backend 全量 Go packages 通过 |
| `make -C server build` | 0 | 生成 `server/bin/ak-api`、`ak-worker`、`ak-cli` |
| `pnpm test && pnpm build` | 0 | Admin 9 test files / 40 tests，Vite 2,554 modules production build |
| `npm test && npm run build` | 0 | 标准 npm script 调用链同样通过 40 tests 与 production build |
| `pnpm install --frozen-lockfile && pnpm check` | 0 | 冻结锁文件未漂移；API/i18n/48 routes 生成、lint、strict typecheck、40 tests、build、bundle budget、Admin validator 全通过 |
| `make dev-deps && make db-setup && make -C server doctor` | 0 | PostgreSQL 18 healthy，application migration `9:false`，Seed 108 permissions/35 menus，数据库 doctor ready |
| `make dev-backend` + readiness probe | started / probe 0 | API 和 Worker 从源码启动；`/internal/v1/health/ready` 返回 ready，验收后 Ctrl-C 优雅停止（命令因人为停止为 130） |
| `pnpm dev` 默认端口 | 1 | 4173 被本机既有 Docker 映射占用，Vite `strictPort` 明确失败；未将其写成通过 |
| `npm --prefix apps/ak-admin run dev -- --port 4175` + HTTP probes | started / probes 0 | Vite ready；HTML title 可读，`/admin-api` 代理到源码 API，`zh-CN` 与 `en-US` 均返回对应 `Content-Language`；验收后人为 Ctrl-C |
| Backend/Admin/Mobile validators + `validate_i18n_contract.py` + Mobile static check | 0 | Backend 9 migrations/58 tables，Admin 48 routes/147 APIs，Mobile blueprint、双语 key/placeholder 和静态工程检查通过 |
| `make -C server check` | 0 | gofmt、go vet、Backend tests 通过 |
| `make toolchain`（最终） | 2 | 正确识别本机 Node 26.5.0 不符合项目 Node 24.x 门禁；Go 1.26.5、npm 11.17.0、pnpm 11.18.0、Docker 28.5.1 已识别 |
| `npm install --package-lock=false --ignore-scripts` 探索 | terminated | npm 在本机 52 秒内未结束，主动停止且未生成 `package-lock.json`；README 因此只承诺 npm script 调用，不把 npm 作为第二锁文件安装路径 |

### 验证边界与风险

- 当前宿主 Node 为 26.5.0，超出仓库锁定的 Node 24.x；Admin 测试/构建虽实际通过，但会显示 engine warning。CI 和正式门禁仍以 Node 24.18.1 为准，doctor 会阻止误用。
- 本轮修改的是开发脚本和文档，没有可视 UI 变更，因此没有运行或伪造新的 `ui-ux-pro-max` 产物与视觉截图。
- 双语 API 协商和既有 40 个 Admin unit tests 已验证；本轮没有重新运行需要专用账号/审计 fixture 的完整浏览器 E2E，不能把历史浏览器结果当作本轮新结果。
- Android、iOS、Harmony 本轮未执行构建或真机测试，均为 **blocked / 未验证**。
- Git 工作树在开始与结束时均为未提交文件集合；没有 commit、push 或远程发布。
- 本机 4173 测试环境的临时管理员已经 bootstrap，并通过真实 `/admin-api/v1/auth/login` 返回 `code=OK`。原请求的 10 位密码被策略拒绝且未绕过；实际本机临时凭据位于被 Git 忽略的 `docs/LOCAL_TEST_ACCESS.md`，不会进入开源代码或初始化 SQL。
