# AppKernia Codex 交付报告

日期：2026-08-04  
范围：Backend + Admin frontend 完成性与 GitHub 开源开发体验；未自动 commit、push 或部署。

## 交付摘要

### 2026-08-04 HotGo 地区编码数据移植

- 从 HotGo 固定提交 `c6191f7126c0ece4f4357014684f479836643822` 的 `hg_sys_provinces` PostgreSQL Seed 转换出 3,663 条 AppKernia 版本化地区目录，完整性为 34 个根、357 个二级、3,272 个三级、0 重复编码、0 缺失父级、5 条缺失坐标。
- 新增确定性导入脚本、目录来源元数据和 HotGo MIT 完整声明；HotGo 参考仓库位于已忽略的 `tmp/`，交付构建不依赖它。
- Backend 新增 sqlc `UpsertCoreRegion` 与 Serializable `CoreRegions`，`ak-cli seed core` 现在输出 `regions=3663`；连续执行两次未产生重复编码，已有非空邮编/坐标不会被目录空值覆盖。
- 现有公开/管理地区 API 与 `/system/settings/regions` 后台树表无需新增视觉结构即可读取目录；专项 E2E 已改为以北京 `110000` → 市辖区 `110100` 验证懒加载。
- PostgreSQL 18 实测数值编码目录为 3,663 条、34 个根、0 个孤儿；公开 API 返回北京及子级且 `has_children=true`。本轮数据是 HotGo 源提交快照，未声明为 2026 年最新官方行政区划。
- 真实 Admin 登录与 `/admin-api/v1/regions` 根/子查询均返回 200；一次性验收用户和租户随后精确禁用，活动 Session 复核为 0，未写入或输出固定密码。

### 本轮实际命令与结果

| 命令 | 退出码 | 结果 |
|---|---:|---|
| `python3 scripts/import_hotgo_regions.py ...` | 0 | 确定性生成 3,663 条；层级 `34/357/3272` |
| `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` | 0 | `UpsertCoreRegion` 生成代码同步 |
| `go test ./internal/seed ./cmd/ak-cli` | 0 | 地区目录来源、数量、父级顺序、坐标范围和配置目录测试通过 |
| 本地 PostgreSQL 18 连续两次 `go run ./cmd/ak-cli seed core` | 0 / 0 | 两次均输出 `regions=3663`，幂等执行成功 |
| PostgreSQL 目录/孤儿/北京链路探针 | 0 | `3663 / 34 / 357 / 3272 / missing=5 / orphan=0`；北京三级链路字段正确 |
| 容器内 `GET /api/v1/regions` 根级及 `parent_code=110000` | 0 | 34 条初始化根数据可读，北京 `has_children=true`，子级返回市辖区 |
| 既有忽略文件中的本机测试账号登录 | 1（HTTP 401） | 凭据不属于当前数据库或已失效；未把失败写成通过，也未输出凭据 |
| 一次性 bootstrap → Admin 登录 → 两次 `/admin-api/v1/regions` → 精确禁用 | 0 | 登录、根查询、子查询均 HTTP 200；结束后 user/tenant=`disabled`、active sessions=`0` |
| 首轮 Admin validator 路径 | 1 | 误用不存在的 `tools/validate_blueprint.py`；更正为 `scripts/validate_blueprint_specs.py` 后退出 0 |
| `make -C server check` + `go test -json ./...` | 0 | gofmt/vet/默认测试通过；82 个 test pass events、0 failed |
| Backend/Admin/Mobile/i18n validators + Python py_compile + `git diff --check` | 0 | 四类契约、两个导入/E2E 脚本语法和补丁格式通过 |
| `docker compose build seed && docker compose run --rm --no-deps seed` | 0 | Go 1.26.5 Seed 镜像构建成功；容器输出 `regions=3663` |

### 2026-08-04 系统初始配置与云存储补充

- 参考 HotGo 在线系统配置页及忽略目录 `tmp/hotgo` 中的源码结构，独立实现 9 个分类/55 个安全默认配置项；`tmp/` 已加入 `.gitignore`，参考源码不会进入版本控制。
- `seed core` 现对所有活动租户幂等写入配置目录，保留现有值与 Secret；目录元数据由 `x-appkernia-catalog` 标记并在 Repository 层锁定，只开放当前值修改和 Secret 轮换。
- 云存储完成 local/S3-compatible/MinIO 配置 Adapter、AES-GCM Secret 解密、上传策略、provider/bucket/object key 持久化、OpenAPI/sqlc/生成 Client、provider 筛选、私有下载和删除闭环；生产拒绝 local 与非 TLS 远端配置。
- Admin 系统配置页完成 9 分类导航、URL 持久化、双语目录名称、目录字段只读保护、Secret 不回显；新增可复用 `AkFileUploader` 并接入系统配置、文件管理和 `AkFilePicker`。
- 本地 PostgreSQL 18/Docker API 实际完成一次 48 B 文件的分片上传、组装、私有下载内容比对、内部引用核对和 API 删除。公开响应不含 bucket/object key，配置 Secret 默认均为空。
- UI 使用项目内 `ui-ux-pro-max`，保存 request/output/decisions/checklist、Master/page override 与 4 张 1800×952 Chrome 截图；Chrome 中验证 `zh-CN`/`en-US` 即时切换和目录编辑抽屉仅开放 Current value。
- 本轮创建的本地 Chrome 验收 user/tenant 已精确设为 disabled，并撤销 11 个 Session/Refresh 记录；已知测试密码再次登录返回 HTTP 401，未影响用户原有账号或数据。
- 验证边界：没有 S3/MinIO 真实账号，真实云端请求未执行；local 容器链路、配置安全规则、MinIO SDK 编译与单元/集成验证已完成。暗色模式与 768 px Chrome 截图未执行，不冒充已验收。

### 本轮实际命令与结果

| 命令 | 退出码 | 结果 |
|---|---:|---|
| `git clone git@github.com:bufanyun/hotgo.git tmp/hotgo` | 0 | 只读参考源码，固定到 `c6191f...`，目录被 Git 忽略 |
| `make sqlc-generate`（server） | 0 | provider/bucket 查询与生成代码同步 |
| `GOTOOLCHAIN=go1.26.5 go mod tidy && make check`（server） | 0 | gofmt、vet、默认测试全通过；最终默认 suite 78 tests / 0 failed |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/storageadmin/repository ./internal/modules/systemsettings/repository ./internal/modules/storage/repository -count=1 -v` | 0 | 8 个 PostgreSQL 18 集成测试通过 |
| Docker/curl 上传生命周期探针 | 0 | policy、分片、完成、provider 列表、私有下载、内部引用及删除全部通过 |
| `npm run generate && npm run lint && npm run typecheck && npm run test`（Admin） | 0 | OpenAPI/i18n/routes 生成，10 files / 45 tests，strict typecheck 通过；首轮 lint 曾发现 1 个冗余条件，修复后重跑为 0 |
| `npm run build && npm run validate:bundle && npm run validate:blueprint && npm run check:ui-skill` | 0 | 4158 modules；最终 initial gzip 203,784 B；最大 chunk 165,429 B；Admin 蓝图与 Skill 检查通过 |
| `docker compose build admin` | 0 | Node 24.18.1 + pnpm 11.18.0 production build 通过 |
| Backend/Admin/Mobile/i18n 四项 validator | 0 | 0 errors/warnings；Admin 152 existing APIs + 0 delta |
| `govulncheck ./...` | 0 | 项目调用路径可达漏洞 0；另有 2 个 imported package 和 1 个 module 提示不在调用路径 |
| `npx --yes @redocly/cli@2.12.4 lint server/openapi/openapi.yaml` | 0 | OpenAPI 有效；保留 1 个既有 license、3 个既有 ambiguous file path warnings |
| `docker compose up --build -d api` / `AK_ADMIN_PORT=4174 docker compose up -d admin` | 0 | API/Admin/PostgreSQL healthy，`/healthz` 返回 `ok` |
| Backend/Admin checksum manifest 全量 `shasum -c` | 1 | 本轮修改的 7 个蓝图文件及新 `core-configs.json` 均为 OK；仓库既有 manifest 对其他历史已修改蓝图仍有 6/20 个陈旧项，本轮未批量覆盖无关历史指纹 |

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
- 地区生产数据的版本化导入 CLI 已于 2026-08-04 通过 HotGo 固定快照补齐；此前 250 子节点夹具仍仅用于大树上限测试，不作为初始化目录来源。
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

## 2026-08-04 Admin 导航信息架构修正追加

### 交付结果

- 根因是 Admin `resolveBackendMenus` 只返回 `page`，静默丢弃后端已经正确提供的 `directory` 和 `parent_id`，导致核心菜单在侧栏铺平；本轮改为递归树解析。
- 根导航现在固定为 `Dashboard` 与 `系统/System`。系统下按功能展示 8 个分类；用户要求的系统配置/字典、部门/用户/岗位、角色/菜单均处在正确的三级位置，既有地区、模块、权限目录等页面继续归入对应分类。
- 导航权限仍由后端授权和前端路由边界共同约束；解析器同时裁剪无权限、Feature Flag 关闭、未实现、循环和空目录节点，不接受额外任意根节点。
- App Shell 使用 TanStack Router 的响应式 pathname 驱动选中项与祖先展开；语言切换无页面重载，移动 Drawer 复用同一树且叶子导航后关闭。
- 真实视觉审查期间发现暗色菜单链接对比度问题；提供显式高对比文字色并等待 disclosure 动画稳定后，最终 axe serious/critical 为 0。

### 本次真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `ui-ux-pro-max`：design-system、UX、React、Web 四次搜索 | 0 | 保存 request/output/decisions/checklist；采用层级、焦点、active state、响应式建议，拒绝不适用的营销 Hero、CTA、外部字体和颜色建议 |
| `vitest run src/app/route-registry.test.ts`（首轮） | 0 | 7 项树解析、权限、未知组件、根节点约束和 redirect 定向测试通过 |
| 首轮 strict typecheck | 2 | 暴露目录/页面 path 的联合类型及 Dashboard 快速访问仍假定扁平结构；随后改为判别联合和 `flattenMenuPages` |
| 最终 `mise exec node@24.18.1 -- corepack pnpm --dir apps/ak-admin run check` | 0 | API/i18n/48 routes 生成、strict lint/typecheck、9 files/42 tests、2,554 modules production build、bundle budget、Admin validator全通过 |
| `docker compose -p appkernia_docker_test build admin` + `up -d --no-deps --force-recreate admin` | 0 | Node 24.18.1 镜像构建完成并替换 `localhost:4173` Admin；第二次最终 CSS 构建同样 exit 0 |
| 系统 Python 运行导航 E2E | 1 | 缺少 `playwright` Python 包，未冒充通过 |
| Playwright Skill wrapper `playwright_cli.sh --help` | 127 | wrapper 报 `playwright-cli: command not found`；使用隔离临时 venv 安装 Playwright 1.58.0 和 Chromium 145 继续验证 |
| 导航 E2E 中间运行 | 1 | 依次暴露测试选择器、Feature Flag 下租户叶子预期、展开动画对比度时序、服务端语言偏好和已展开目录重复点击问题；均修正后重跑 |
| 最终 `e2e_navigation_hierarchy.py`（4173 Docker） | 0 | `zh-CN`/`en-US` 根节点与 8 分类、三组核心叶子、无刷新切换、375 px Drawer、axe serious/critical=0 全部通过 |
| Backend/Admin/Mobile validators、i18n validator、UI Skill check | 0 | Backend 9 migrations/58 tables；Admin 35 menus/48 routes/147 APIs；Mobile blueprint；双语键/占位符和 Skill 可用性均通过 |
| 错误路径 `blueprint/backend/scripts/validate_blueprint_specs.py` | 2 | 文件不存在；改用真实 `blueprint/backend/tools/validate_blueprint.py` 后 exit 0，未隐藏失败 |
| `python3 -m py_compile apps/ak-admin/scripts/e2e_navigation_hierarchy.py` / `git diff --check` | 0 / 0 | E2E 脚本语法与补丁格式通过 |

### 视觉证据与边界

- 结果：[admin-navigation-hierarchy.evidence.json](../output/playwright/admin-navigation-hierarchy.evidence.json)
- 中文桌面：[admin-navigation-hierarchy.zh-CN.1440.png](../output/playwright/admin-navigation-hierarchy.zh-CN.1440.png)
- 英文桌面：[admin-navigation-hierarchy.en-US.1440.png](../output/playwright/admin-navigation-hierarchy.en-US.1440.png)
- 英文移动 Drawer：[admin-navigation-hierarchy.en-US.375.png](../output/playwright/admin-navigation-hierarchy.en-US.375.png)
- 仅 Chromium 145 在本机 Docker/API/PostgreSQL 环境实际执行；Firefox、Safari 与生产部署未执行。
- 本轮没有 Backend/OpenAPI/数据库/权限契约变化；蓝图菜单种子原本已符合目标，故未制造无意义的 migration。
- Android、iOS、Harmony 与本次 Web 导航无关且未执行，状态继续为 blocked / 未验证。
- 本轮未 commit、未 push。

## 2026-08-04 Admin 菜单图标与间距优化追加

### 交付内容

- 为 `admin-menu-seed.json` 与后端集成快照中的 35 条核心菜单逐项配置 Ant Design Icon，并通过既有幂等 Seed 更新当前 PostgreSQL。
- Admin 新增静态图标注册表，消费 Auth Context 的 `menu.icon`；所有后端字符串只作为 Map key，未知/缺失值统一回退，保持静态路由与静态组件安全边界。
- App Shell 不再只为 Dashboard/System/User Management 做硬编码判断；根、目录和叶子全部走统一渲染。
- 菜单图标固定为 16 px，图标与文案间距固定为 8 px；不压缩层级 indent，不降低菜单行触控区域。
- `ui-ux-pro-max` request/output/decisions/checklist 与三张截图保存在 `artifacts/ui-ux-pro-max/AKADM-navigation-icons/`、`output/playwright/`。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `ui-ux-pro-max` design-system + UX/React/Web 三项查询 | 0 | 采用一致 SVG、焦点、语义、bundle 与响应式建议；拒绝营销页、外部字体和新配色建议 |
| `pnpm install --lockfile-only --filter @appkernia/admin...`（Node 24.18.1） | 0 | 将既有锁定版本 `@ant-design/icons@6.3.2` 声明为 Admin 直接依赖，lockfile 可复现 |
| 定向 typecheck/lint/2 files 9 tests | 0 | 图标注册表、fallback、路由 icon 透传和菜单树测试通过 |
| 首轮组合门禁 | 0（组合 shell）/ Admin 子命令 1 | Admin lint 发现测试中多余 `??`；同轮 Backend check 和 validators 通过，但未把 Admin 子失败记为通过 |
| 最终 `set -e; pnpm --dir apps/ak-admin run check` | 0 | API/i18n/48 routes 生成、lint、strict typecheck、10 files/45 tests、4,155 modules build、bundle budget、Admin validator 全通过 |
| `make -C server check` | 0 | Go vet 与 Backend 全量 unit tests 通过 |
| Backend/Admin/Mobile validators + i18n + UI Skill check | 0 | 9 migrations/58 tables、35 menus/48 routes/147 APIs、Mobile 静态蓝图、双语契约与 Skill 均通过 |
| Docker build `seed admin` + `docker compose run --rm --no-deps seed` + Admin recreate | 0 | Seed 输出 `permissions=108 menus=35`；4173 Admin 最终镜像启动 |
| PostgreSQL core menu icon probe | 0 | `35|35`：全部 35 条 active core menu 的 icon 非空 |
| 图标 E2E 首轮 | 1 | 测试错误假设 34 个可见项目；真实环境因 Feature Flag/权限展示 32 项，调整为验证所有实际可见项后重跑 |
| 最终 `e2e_navigation_hierarchy.py` | 0 | 两种语言均 `32/32` 可见项有图标；计算宽度 16 px、gap 8 px；375 Drawer 与 axe 通过 |
| `git diff --check` | 0 | 补丁格式通过 |

### 证据与边界

- [中文桌面图标截图](../output/playwright/admin-navigation-icons.zh-CN.1440.png)
- [英文桌面图标截图](../output/playwright/admin-navigation-icons.en-US.1440.png)
- [英文移动 Drawer 截图](../output/playwright/admin-navigation-icons.en-US.375.png)
- [机器测量与 axe 证据](../output/playwright/admin-navigation-hierarchy.evidence.json)
- AppShell chunk gzip 从约 49 KB 增至约 62 KB，仍低于 180 KB chunk budget；初始 bundle 约 201 KB，低于 300 KB budget。
- 只验证 Chromium 145；Firefox、Safari 与生产部署未执行。
- Android、iOS、Harmony 与本次 Web 导航无关且未执行，继续标记 blocked / 未验证。
- 本轮仍未 commit、未 push。

## 2026-08-04 Admin DESIGN.md 视觉焕新与品牌资产追加

### 交付内容

- 更新 Ant Design semantic/component tokens：ink primary、near-white canvas、hairline、6/12px radius、层叠阴影与 data-dense table 规则。
- 更新 App Shell：真实品牌图、黑色侧栏、反相选中项、sticky blur header 和移动 Drawer 品牌标题。
- 更新认证框架：原创品牌 Logo、蓝青绿大尺度氛围光、近白网格画布和高对比表单卡；中英文文案与表单语义未改变。
- 更新 Dashboard：KPI 品牌光谱边、紧凑数据层级、ECharts 序列色/虚线网格/ink tooltip/低透明面积层，以及全局共享 surface 样式。
- 生成透明品牌 master 与 512/180/64/32 派生资产，接入 favicon、Apple touch icon、认证和导航；完整提示词与处理边界见 `apps/ak-admin/public/brand/README.md`。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + React/chart/UX 四条查询 | 0 | 输出、采用/拒绝项、决策和检查表已保存 |
| 内置 `imagegen` + chroma-key helper + `sips` 派生 | 0 | 1254 master 与 512/180/64/32 PNG 均有 alpha；soft matte 内部白色几何异常后改用经预览验证的 hard matte |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 warning/error；宿主 Node 26 engine warning 如实保留 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 48 routes 重新生成，TypeScript strict 通过 |
| `pnpm --filter @appkernia/admin test` | 0 | 10 files / 45 tests 通过 |
| `pnpm --filter @appkernia/admin build` | 0 | 4,155 modules，production build 成功 |
| `validate:bundle` / Admin blueprint validator | 0 / 0 | initial gzip 200,914 B；最大 lazy chunk gzip 165,429 B；35 menus/48 routes/108 permissions/147 APIs 通过 |
| Mobile blueprint + i18n validator + UI Skill check | 0 | Mobile 0 error/warning；`zh-CN`/`en-US` parity；Skill 可用 |
| 登录 1440/768/375 Chromium 截图与 axe | 0 | 三视口均无水平溢出，axe 0 violations |
| `visual_check.py` Dashboard mock-contract | 0 | 1440 axe 0 serious/critical；768 Drawer 转场后截图；不写数据库 |
| `docker compose ... build admin` | 0 | Node 24.18.1 Admin 镜像生产构建成功 |
| 首次无端口覆盖的 Compose 恢复 | 1 | PostgreSQL 55432 已被其他进程占用；改用该项目此前的 55433 显式恢复后成功，不掩盖失败 |
| `AK_POSTGRES_PORT=55433 ... docker compose ... up -d` | 0 | PostgreSQL/migrate/seed/API/Admin 依赖链启动；API healthy 后 Admin 启动 |
| Compose 终态复核与恢复 | state unhealthy → healthy | 先前未轮询的通用 `up -d` 会话随后用默认 55432 重建 PostgreSQL，导致 API health 503；显式 55433 再启动后 PostgreSQL/API/Admin 均 healthy，并停止、移除该会话额外创建的 Redis/MinIO 容器，恢复原测试栈范围 |
| `git diff --check` | 0 | 补丁格式通过 |

### 截图索引与边界

- [登录桌面 1440](../artifacts/ui-ux-pro-max/AKADM-310-design-refresh/screenshots/1440x900-light.png)
- [登录平板 768](../artifacts/ui-ux-pro-max/AKADM-310-design-refresh/screenshots/768x1024.png)
- [登录移动 375](../artifacts/ui-ux-pro-max/AKADM-310-design-refresh/screenshots/375x812.png)
- [Dashboard 桌面 1440](../artifacts/ui-ux-pro-max/AKADM-310-design-refresh/screenshots/dashboard-1440x900-light.png)
- [Dashboard 768 Drawer](../artifacts/ui-ux-pro-max/AKADM-310-design-refresh/screenshots/dashboard-768x1024-navigation.png)
- [透明品牌 master](../apps/ak-admin/public/brand/appkernia-mark.png)
- 本轮浏览器验证为 Chromium；Firefox/Safari 未执行。Dashboard 使用确定性的 HTTP mock-contract，仅验证 UI 渲染、响应式与 a11y，不代表新一轮 PostgreSQL/生产联调。
- 当前产品没有 dark theme algorithm/选择器，因此未生成或伪装 dark 截图。Mobile 未变更，Android/iOS/Harmony 未构建或真机验证。
- 本轮未 commit、未 push、未部署生产。

## 2026-08-04 Admin 侧栏祖先可读性与折叠交互追加

### 交付内容

- 根因是 Ant Design 的 `.ant-menu-submenu-selected` 让祖先目录继承了叶子选中态的深色文字，而祖先仍处在 `#111111/#0A0A0A` 背景；现在为选中/展开祖先显式提供高对比浅色。
- 原侧栏底部汉堡折叠按钮已删除；桌面侧栏新增位于内容边界垂直中点的白色 Chevron 控制，展开时向左、折叠时向右。
- 控制默认不抢占视觉，侧栏 hover、按钮 focus 或非 hover 指针设备时可见；保留全局 focus ring、双语 accessible name、`aria-expanded` 和 reduced-motion 兼容。
- 修复折叠/展开期间 Ant Menu 清理 `openKeys` 的副作用，重新展开后当前叶子的两级祖先会完整恢复。
- 同页字典空状态文字由 AntD 默认低对比灰调整为 design token 的 muted ink，使本次 axe 审计为零违规。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `ui-ux-pro-max` design-system + UX + React 三项查询 | 0 | 保存 request/output/decisions/checklist；采用对比度、语义 Button、键盘焦点、非 hover 指针与 reduced-motion 建议 |
| `npm --prefix apps/ak-admin run typecheck && ... lint` | 0 | 48 routes 生成、TypeScript strict 与 ESLint 通过 |
| `pnpm --dir apps/ak-admin test && ... build` | 0 | 10 files / 45 tests；4,155 modules production build 通过；宿主 Node 26 engine warning 如实保留 |
| Admin/Mobile 蓝图与 i18n validator | 0 | 35 menus/48 routes/108 permissions/147 APIs、Mobile 0 error/warning、双语契约通过 |
| 默认 Python / Codex bundled Python 运行视觉脚本 | 1 / 1 | 两个运行时均未安装 `playwright`，未冒充通过；改用 `uv run --with playwright==1.54.0` 的隔离依赖 |
| 视觉脚本中间运行 | 1 | 依次暴露重复“字典管理”选择器、hover 测试鼠标位置、展开过渡时序和折叠后丢失一层 open key；修正实现/脚本后全部重跑 |
| 最终 `visual_check.py`（4173 Docker） | 0 | 两级祖先色均为 `rgba(255,255,255,.92)`；隐藏、hover、折叠、展开、祖先恢复与水平溢出通过；axe 0 violations |
| 最终 `mise exec node@24.18.1 -- corepack pnpm --dir apps/ak-admin run check` | 0 | generate、lint、strict typecheck、10 files/45 tests、build、bundle budget、Admin validator 全部通过；initial gzip 201,720 B，最大 lazy chunk 166,087 B |
| `python3 blueprint/scripts/validate_i18n_contract.py` + Mobile validator + UI Skill check + `git diff --check` | 0 | `zh-CN/en-US` parity、Mobile 蓝图、Skill 可用与补丁格式通过 |
| Docker Admin build/recreate + `/healthz` | 0 | Node 24.18.1 镜像构建并替换 `localhost:4173` Admin，探针返回 `ok`；API/PostgreSQL 原测试栈未重建 |

### 截图索引与边界

- [展开侧栏与悬停折叠箭头](../artifacts/ui-ux-pro-max/AKADM-navigation-collapse/screenshots/dictionaries-expanded-hover-1440x900.png)
- [折叠侧栏与悬停展开箭头](../artifacts/ui-ux-pro-max/AKADM-navigation-collapse/screenshots/dictionaries-collapsed-hover-1440x900.png)
- 浏览器使用 Chromium + 确定性 HTTP mock-contract，只验证本次菜单渲染、交互与 a11y；不冒充真实 PostgreSQL、生产部署或第三方联调。
- Firefox、Safari 未执行。本轮未修改 Mobile，Android/iOS/Harmony 未构建或真机验证。
- 本轮未 commit、未 push；保留用户未提交的 `DESIGN.md` 与此前视觉焕新改动。

## 2026-08-04 Admin 顶部工具与账户菜单追加

### 交付内容

- 新增 `FullscreenToggle`：标准 Fullscreen API、Escape 状态同步、进入/退出图标与双语语义、失败反馈及不支持时禁用。
- `LocaleSwitcher` 增加 `icon` 变体：App Shell 使用翻译图标和可选中菜单，匿名认证页保留文字 Select；语言保存失败仍会回滚并保留 `role=alert`。
- 新增 `UserMenu`：36px Avatar、受保护 Blob 头像、首字母回退、用户/角色摘要、个人中心和危险色退出菜单。
- Header 删除直接显示的用户名和退出按钮，三个图标入口在 375px 仍保持可操作且没有横向溢出。
- Admin 双语事实源增加账户菜单与全屏语义键，应用 Catalog 由生成脚本同步，不手工漂移。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 项目内 `ui-ux-pro-max` design-system + UX + React + Web 查询 | 0 | 保存 request/output/decisions/checklist；采用 40px icon button、语义名称、键盘焦点、响应式与外部全屏状态同步建议 |
| 定向 Node 24 typecheck + lint | 0 | 48 routes 生成，TypeScript strict 与 ESLint 通过 |
| 首轮 `pnpm --dir apps/ak-admin test` | 0 | 10 files / 45 tests 通过 |
| 首轮 Header 视觉脚本 | 1 | Docker 构建从蓝图生成 Catalog，暴露新键只写应用文件会被覆盖；随后同步 `blueprint/i18n/admin` 并重跑 |
| 视觉脚本中间运行 | 1 | 依次暴露角色文本测试选择器包含可访问标签、375px 浮层重定位等待、axe 调用参数与 popup region；修正后全部重跑 |
| 最终 `visual_check.py`（4173 Docker） | 0 | Avatar 初始字母、用户/角色、个人中心/退出、语言选中/切换、全屏进入/退出、375px 无溢出通过；axe violations 0 |
| 最终 `mise exec node@24.18.1 -- corepack pnpm --dir apps/ak-admin run check` | 0 | generate、lint、strict typecheck、10 files/45 tests、4,157 modules build、bundle budget、Admin validator 全部通过；initial gzip 201,867 B，最大 lazy chunk 166,087 B |
| i18n + Mobile validator + visual script py_compile + UI Skill check + `git diff --check` | 0 | 双语事实源/占位符、Mobile 蓝图、脚本语法、Skill 可用与补丁格式全部通过 |
| Docker Admin build/recreate + `/healthz` | 0 | Node 24.18.1 镜像构建并替换 `localhost:4173` Admin，健康探针返回 `ok` |

### 截图索引与边界

- [账户菜单桌面](../artifacts/ui-ux-pro-max/AKADM-header-utilities/screenshots/header-account-1440x900.png)
- [语言菜单桌面](../artifacts/ui-ux-pro-max/AKADM-header-utilities/screenshots/header-language-1440x900.png)
- [全屏模式桌面](../artifacts/ui-ux-pro-max/AKADM-header-utilities/screenshots/header-fullscreen-1440x900.png)
- [账户菜单移动端](../artifacts/ui-ux-pro-max/AKADM-header-utilities/screenshots/header-account-375x812.png)
- 浏览器验证使用 Chromium 与确定性 HTTP mock-contract；头像安全加载路径由现有 AuthSession/query 实现复用，但本轮截图使用无头像首字母状态。
- Firefox、Safari 与生产部署未执行。Mobile 应用未修改，Android/iOS/Harmony 未构建或真机验证。
- 本轮未 commit、未 push，并完整保留此前未提交的视觉焕新、侧栏与用户 `DESIGN.md` 改动。

## 2026-08-05 Admin 登录图形验证码追加

### 交付内容

- 服务端新增 30 分钟失败窗口、第三次失败触发、5 分钟 PNG 数字挑战、范围 HMAC、盐化答案 Hash、一次性消费与 5 次尝试上限；成功登录重置失败状态。
- 新增 migration `000010` 的 2 张 IAM 表、sqlc 查询/Repository、匿名验证码接口、登录请求字段与两个稳定错误码；同步 OpenAPI、生成 Client、双语 Catalog、Backend/Admin 蓝图契约与数据库文档。
- Admin 登录页只信任服务端 `IAM.AUTH.CAPTCHA_REQUIRED`，不使用可绕过的前端失败计数；新增双语可访问验证码输入、PNG 图片、刷新/加载/错误状态和响应式布局。
- 实际执行项目内 `ui-ux-pro-max`，保存 request、skill output、decisions、review checklist、登录页 design-system override 和三档关键视口截图。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make sqlc-generate` | 0 | migration/query 对应的 sqlc 模型与方法生成成功 |
| 首次定向 IAM integration | 1 | 误用旧数据库口令，PostgreSQL 拒绝认证；读取当前 Compose 配置后重跑 |
| 第二次定向 IAM integration | 1 | 新增过期挑战 fixture 首次违反 `expires_at > created_at` 检查约束；同时回拨创建/过期时间后重跑 |
| 最终定向验证码 integration | 0 | 第三次门槛、刷新不可绕过、错误/过期/已消费挑战、成功与阈值重置通过 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/iam/application -count=1` | 0 | IAM PostgreSQL 全量集成测试通过，耗时 7.974s |
| `make check` | 0 | Go vet 与 Backend 全量单元测试通过 |
| `npm run test -- --run src/pages/LoginPage.test.tsx` | 0 | 前两次隐藏、第三次显示、刷新行为与 axe-core 0 violations 通过 |
| `npm run check` | 0 | API/i18n/48 routes 生成、lint、strict typecheck、11 files/47 tests、4,158 modules build、bundle budget、Admin validator 全通过 |
| `docker compose build admin && docker compose up -d --no-deps admin` | 0 | Node 24.18.1 生产镜像重建到 4174；最终 API/Admin/PostgreSQL 均 healthy |
| Chromium 真实 API 验收 | 0 | 第三次后显示、刷新后仍强制、PNG 图片、自动聚焦、双语错误即时切换；375/768/1440 的 `scrollWidth == clientWidth` |
| PostgreSQL 精确验收清理 | 0 | 只删除管理员浏览器测试 scope 的 6 条 challenge 与 1 条 failure state，未清理其他用户/集成 fixture |

### 截图索引与验证边界

- [中文移动 375](../artifacts/ui-ux-pro-max/AKADM-login-captcha/screenshots/login-captcha-zh-CN-375.jpg)
- [英文平板 768](../artifacts/ui-ux-pro-max/AKADM-login-captcha/screenshots/login-captcha-en-US-768.jpg)
- [英文桌面 1440](../artifacts/ui-ux-pro-max/AKADM-login-captcha/screenshots/login-captcha-en-US-1440.jpg)
- 真实浏览器使用本机 Chrome/Chromium 与 Docker Go API/PostgreSQL，不是 mock；验证码答案未由代理读取或提交，成功/消费路径由受控 PostgreSQL integration fixture 验证。
- axe-core 在 jsdom 渲染路径为 0 violations；真实浏览器同时核对 accessibility tree、可见 label、焦点与三视口无溢出。Firefox/Safari 与生产环境未执行。
- Mobile 无改动；Android/iOS/Harmony 未构建或真机验证。本轮未 commit、未 push。

## 2026-08-05 个人头像裁剪与 configured ObjectStore 上传追加

### 交付内容

- 新增公共 `AkImageCropper`、`AkAvatarUploader` 与纯函数裁剪模块，支持拖动、Slider 缩放、90 度旋转、键盘微调、512×512 PNG Canvas 导出、对象 URL 生命周期、进度、成功/错误 live feedback 和失败后原裁剪重试。
- `/profile/basic` 通过 typed upload callback 复用现有本人作用域 `/me/avatar` API，上传成功更新 Profile Query、私有 Blob 与 Auth Context；客户端不接收 Bucket、对象键或云存储凭据。
- local development、S3、MinIO 均沿用系统配置驱动的 ObjectStore adapter；OpenAPI/蓝图明确服务端随机对象键、MIME magic/尺寸/大小/所有权校验与私有读取边界。
- 本地 Compose 与 `.env.example` 默认打开头像 Feature Flag，使 4174 开箱可见；生产仍由显式环境配置控制。
- 双语事实源、设计系统 Profile override、`ui-ux-pro-max` 四类产物与 7 张真实截图已保存。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + UX + React + Web 查询 | 0 | 采用现有 Admin Master、浏览器裁剪、具名键盘控制、进度/live region 与对象 URL 清理；拒绝不一致营销布局、外部字体和替代色板 |
| 定向 lint + strict typecheck + Vitest | 0 | 裁剪数学、文件预检、组件裁剪/进度/axe、头像 API 两阶段上传通过；Slider accessible name 首轮 axe serious 后改用官方 `ariaLabelForHandle` 并清零 |
| 最终 `npm run check` | 0 | 13 files / 54 tests、4,161 modules production build、initial gzip 204,737 B、最大 lazy chunk 165,429 B、Admin blueprint 全部通过 |
| `make check` | 0 | Go vet 与 Backend 全量单元测试通过，storage/configured ObjectStore 测试包含在内 |
| Backend/Admin/Mobile 蓝图 + i18n + `git diff --check` | 0 | 10 migrations/60 tables、35 menus/48 routes/153 APIs、Mobile 静态蓝图与中英文 key/占位符一致性通过 |
| Docker Admin/API build + Compose recreate | 0 | 4174 Admin、API、PostgreSQL 均 healthy；头像 Feature Flag 已启用 |
| 4174 Docker Chromium + PostgreSQL 实际上传 | 0 | 512×512 PNG、进度 100%、页面/顶部头像同步；DB 为 local provider、ready/skipped 文件、completed session，上传对象 248226 bytes |
| 精确验收清理 | 0 | 恢复管理员 `avatar_file_id=NULL`；只删除本次 usage/session/file 与对应 local 对象，复核计数均为 0 |

### 截图索引与验证边界

- [中文桌面裁剪 1440](../artifacts/ui-ux-pro-max/AKADM-profile-avatar-crop/screenshots/zh-CN-crop-1440.jpg)
- [中文桌面上传成功 1440](../artifacts/ui-ux-pro-max/AKADM-profile-avatar-crop/screenshots/zh-CN-upload-success-1440.jpg)
- [英文平板预览 768](../artifacts/ui-ux-pro-max/AKADM-profile-avatar-crop/screenshots/en-US-ready-768.jpg)
- [英文移动个人中心 375](../artifacts/ui-ux-pro-max/AKADM-profile-avatar-crop/screenshots/en-US-profile-375.jpg)
- [英文移动裁剪 375](../artifacts/ui-ux-pro-max/AKADM-profile-avatar-crop/screenshots/en-US-crop-375.jpg)
- 真实浏览器验证使用本机 Docker Go API/PostgreSQL 和 configured local adapter；S3/MinIO 仅完成代码、配置解析和自动化测试，未使用真实第三方 Bucket/凭据，因此不标记生产云联调通过。
- Firefox/Safari 与生产部署未执行。Mobile 未修改，Android/iOS/Harmony 未构建或真机验证。本轮未 commit、未 push。

## 2026-08-05 系统配置表单模式与统一保存追加

### 交付内容

- 新增公共 `AkConfigValueField`，根据配置定义渲染文本、URI/Email、长文本、整数/小数、布尔、枚举、JSON、日期、Secret 及 File ID 控件；File ID 复用 `AkFilePicker` 和既有云存储上传组件。
- 系统配置页默认表单模式，顶部 Segmented 可切换至完整保留的表格定义管理模式；默认模式不写冗余 URL，表格模式持久化 `mode=table`。
- 统一保存只处理变更项，保留乐观版本与 Secret rotation；顺序执行的部分失败会逐项反馈，失败草稿不丢失。分类/模式切换与浏览器离开均保护脏数据。
- 新增组件验证测试与真实 E2E；修复真实 axe 暴露的未选中 Switch 和表格公开标签对比度问题。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm lint` / `pnpm typecheck` / `pnpm build` | 0 / 0 / 0 | ESLint、TypeScript strict、48 routes 和 4,163 modules production build 通过；宿主 Node 26 engine warning 如实保留 |
| `pnpm test` | 0 | 14 files / 56 tests 通过，新增动态配置字段的 label/axe 与 JSON/数字校验覆盖 |
| `pnpm validate:bundle` | 0 | initial gzip 205,602 B；最大 lazy chunk 165,429 B；均在预算内 |
| Admin blueprint / Mobile blueprint / i18n / UI Skill | 0 | 35 menus、48 routes、108 permissions、153 APIs；Mobile 0 error/warning；双语 key/占位符一致；Skill 可用 |
| 首次系统 Python E2E | 1 | 缺少 `playwright` 模块；改用已有 `/Users/payhon/.venv/3.12` 环境后继续，不冒充通过 |
| Chromium E2E 中间运行 | 1 | 真实发现 Ant 默认 Switch 与 cyan Tag 对比度不足，以及测试定位/默认取消文案差异；修正样式与稳健定位后从头重跑 |
| 最终 `e2e_config_form_mode.py`（4174） | 0 | 默认表单、脏数据拦截、保存/恢复 HTTP 200、表格模式、双语、文件选择/上传入口、375 无溢出、控制台错误 0；四组 axe violations 均为 0 |
| PostgreSQL 精确复核 | 0 | 浏览器测试完成后 `site.name` 已恢复为 `AppKernia` |
| Docker Admin build/recreate + `docker compose ps admin` | 0 | 4174 Admin 使用最终生产构建且 healthy |
| `python3 -m py_compile ...` + `git diff --check` | 0 | E2E 脚本语法与补丁格式通过 |

### 截图索引与验证边界

- [中文表单模式 1440](../artifacts/ui-ux-pro-max/AKADM-system-config-form-mode/screenshots/zh-CN-form-1440.png)
- [中文表格模式 1440](../artifacts/ui-ux-pro-max/AKADM-system-config-form-mode/screenshots/zh-CN-table-1440.png)
- [英文表单模式 375](../artifacts/ui-ux-pro-max/AKADM-system-config-form-mode/screenshots/en-US-form-375.png)
- 真实浏览器为本机 Docker Go API/PostgreSQL + Chromium；统一保存是一项 UI 操作，但复用现有逐项版本化接口，不是数据库原子批处理。
- 未部署生产，Firefox/Safari 未执行。Mobile 未修改，Android/iOS/Harmony 未构建或真机验证。本轮未 commit、未 push，并保留全部既有未提交改动。

## 2026-08-05 登录页图标语言菜单追加

### 交付内容

- `AuthFrame` 改用公共 `LocaleSwitcher variant="icon"`，因此登录、注册、找回密码和重置密码页面保持统一语言入口。
- 图标按钮复用 Ant Design `TranslationOutlined` 与 selectable Dropdown；当前 locale 使用 `selectedKeys`，按钮具备双语 accessible name、可见 focus，并关联异步保存错误。
- Dropdown 挂载至触发器父区域，避免 portal 内容落在页面 landmark 之外；同时提高当前语言文字对比度至 WCAG AA。
- 新增图标菜单单测与真实 Chromium E2E，覆盖键盘打开/选择、无刷新切换、双语选中态、四视口无溢出、axe 与控制台。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + UX + React + Web 查询 | 0 | 保存 request/output/decisions/checklist；采用语义 icon button、选中菜单、visible focus、键盘与 WCAG AA 建议 |
| 首次图标 Dropdown 单测 | 1 | jsdom 缺少 `ResizeObserver`；补充与仓库其他 Ant Popup 测试一致的浏览器 API 桩后重跑 |
| `pnpm exec vitest run src/components/LocaleSwitcher.test.tsx` | 0 | 2 项测试通过：匿名图标菜单切换与已认证偏好保存失败回滚 |
| `pnpm lint` / `pnpm typecheck` / `pnpm build` | 0 / 0 / 0 | ESLint、TypeScript strict、48 routes 和 4,163 modules production build 通过；宿主 Node 26 engine warning如实保留 |
| `pnpm test` | 0 | 14 files / 57 tests 全部通过 |
| `pnpm validate:bundle` | 0 | initial gzip 205,596 B，最大 lazy chunk 165,429 B，均在预算内 |
| Admin / Mobile / i18n / UI Skill validators | 0 | 35 menus、48 routes、108 permissions、153 APIs；Mobile 0 error/warning；双语契约与 Skill 检查通过 |
| Chromium E2E 中间运行 | 1 | 先修正测试使用的错误标题；随后真实发现 Dropdown portal landmark 和当前菜单项 4.38:1 对比度问题，修正后从头重跑 |
| 最终 `e2e_login_locale_menu.py`（4174） | 0 | 原生 Select 不存在、图标存在、键盘切换、双语选中态、不刷新、375/768/1024/1440 无溢出；两组 axe 0 violation、控制台错误 0 |
| Docker Admin build/recreate + `docker compose ps admin` | 0 | 4174 Admin 使用最终生产镜像且 healthy |
| E2E `py_compile` + `git diff --check` | 0 | 自动化脚本语法和补丁格式通过 |

### 截图索引与验证边界

- [中文桌面语言菜单 1440](../artifacts/ui-ux-pro-max/AKADM-login-locale-icon-menu/screenshots/zh-CN-menu-1440.png)
- [英文移动语言菜单 375](../artifacts/ui-ux-pro-max/AKADM-login-locale-icon-menu/screenshots/en-US-menu-375.png)
- 真实浏览器为本机 Docker Admin + Chromium；本功能不涉及新的后端或数据库写入。
- 未部署生产，Firefox/Safari 未执行。Mobile 未修改，Android/iOS/Harmony 未构建或真机验证。本轮未 commit、未 push，并保留全部既有未提交改动。

## 2026-08-05 Mobile Framework、内容与版本治理追加（实现完成；平台真实设备验收未完成）

### 交付内容

- Mobile：实现 Home、Profile 和可见子页（基本资料、安全中心、设备、通知偏好、语言/外观、帮助/关于、退出登录），以及文章列表/详情、分类筛选、游标分页、书签和分享。所有业务页面通过 `ak-*` 适配层；Home 将本人、未读数与精选文章查询拆分，次要卡片失败不清空整个页面。
- 安全与网络：会话只持久化 Refresh 凭据；恢复时通过 Refresh 换取短期 Access Token。请求带 `Accept-Language`、超时、取消与结构化错误；401 只在显式 opt-in 的 GET/HEAD 上最多重放一次，写请求绝不隐式重放。文章资源经带鉴权 Header 的下载 Port 获取，URL 不携带 Token。
- 内容安全：文章正文解析为允许的 heading/paragraph/callout block DTO；详情页未使用 `rich-text` 或 `v-html`。收藏状态以服务端确认结果为准，分享只经平台 Port，不下载或执行远端脚本。
- Backend：内容/分类/书签、移动个人资料与版本治理的 API、OpenAPI、PostgreSQL migration、sqlc、权限和 Admin/Mobile 客户端契约已随实现同步；实施方提供了临时 PostgreSQL 18 实测证据。没有在本次文档整理阶段重跑会变更或依赖该临时库的测试。
- Admin：文章/分类管理与移动版本发布页面已实现，包含权限分支、冲突展示、`lock_version`、双语 UI 与响应式表格/Drawer。保存的 E2E 使用确定性 HTTP mock-contract，不冒充真实 Backend/PostgreSQL 集成。
- i18n：Mobile 使用 `AkI18n` 与完整 `zh-CN`/`en-US` 语言包；默认与最终回退为 `zh-CN`，登录用户偏好走后端，匿名选择仅存非敏感本地偏好。静态契约已验证，不把它替代为三端运行时 UI 证据。
- 安全存储：`ak-secure-storage` 提供 Android Keystore + AES-GCM、iOS Keychain `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`、Harmony Asset Store 的接口实现，且无普通 Storage fallback 或凭据日志。此结论的验证边界见下方平台项。
- DCloud：已记录官方 rules/MCP 来源、导入 commit、许可证、`@dcloudio/uni-app-x-mcp@0.0.5` 及本 worktree 的显式 `projectPath`。本 task 已经以 direct-stdio 完成 MCP initialize、tools/list 与 call，结果保存为 `apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-mobile-framework/mcp-easycom-scan.json`。本机原生 `codex mcp list` 因 vendor binary 损坏不能执行；项目级 MCP server 仍需从仓库根新开/重启 Codex 后才会载入原生工具列表，故不把后者写为已验证。

### 本文档整理阶段实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 37 routes、3 tabs、26 baseline APIs、34 API delta、11 permission delta、33 components、11 privacy capabilities、26 tasks、3 platforms；0 errors / 0 warnings |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN`/`en-US`、default/fallback `zh-CN` 及 backend/admin/mobile reference packs 契约通过 |
| `python3 apps/ak-mobile/scripts/verify-mobile-framework.py` | 0 | 路由、OpenAPI 片段、文章安全渲染、请求取消、Bootstrap、会话与刷新策略的源级契约通过 |
| `python3 apps/ak-mobile/scripts/verify-secure-storage-contract.py` | 0 | Android Keystore/AES-GCM、iOS Keychain、Harmony Asset Store、无普通 Storage fallback 和无 native credential logging 的源级契约通过 |
| `python3 apps/ak-mobile/scripts/test_refresh_policy.py` | 0 | 可执行 fake HttpPort 测试覆盖 1 次 GET 重放、禁用重放、4 种写方法不重放、Refresh 失败清除会话及文章资源 Header 边界 |
| HBuilderX 5.06 CLI，前一轮唯一临时项目 `ak-mobile-framework-ios-rootqa` 的 iOS compile-only | 0 | 20 页 + `ak-secure-storage`；日志含 `UTS编译完毕`、`ready in 39282ms`；标准基座警告 secure-storage 插件不生效，需 custom base 且 iOS 13+ |
| HBuilderX 5.06 CLI，第 3 次最终隔离项目 `ak-mobile-framework-ios-final-evidence` 的 iOS 合并代码验证 | 0 | 20 页 + `ak-secure-storage`、UTS 完成、`ready in 30446ms`、正常“已停止运行”；清理前严格搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality` 均为空；仅标准基座/custom base/iOS 13+ 预期提示 |
| Android 唯一项目 `ak-mobile-framework-android-final-clean-1785906028` 的最终 compile-only | launch exit 未可信捕获；close 成功 | 20 页至 Android class、UTS、`ready in 19314ms`、正常停止；5 处 UTS 值比较修复后静态 7 项、严格错误/警告/Identity 搜索均为 0；仅 compile-only，已清理且无进程 |
| HBuilderX 5.06 CLI，唯一项目 `ak-mobile-framework-harmony-final.ECScJj` 的 Harmony 最终源码回归 | 0 | 20 页、UTS、`ready in 15236ms`、工程/依赖/运行包制作成功，恰 1 个 `entry-default-unsigned.hap`；严格搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality`、`currentColor` 零匹配（`rg` exit 1） |
| Admin `pnpm check`（实施方最新运行） | 0 | 16 files / 64 tests、production build、bundle 与 Admin blueprint check 通过 |
| `go test ./... -count=1 && go vet ./...`（`server`） | 0 | Backend 全量 Go 测试与 vet 通过 |
| Backend/Admin/Mobile blueprint + i18n + Mobile framework/secure-storage/refresh + `bash -n apps/ak-mobile/scripts/build-platform.sh` + `git diff --check` 组合回归 | 0 | 全部子命令 exit 0 |

表中前 5 条是本代理实际运行的静态/模拟门禁；合计为 5 个命令入口、0 个失败。其余命令结果由对应实施代理提供并按来源标注。所有这些门禁都不等价于安装包、模拟器或真机测试。

### 已保存的 Admin 浏览器/视觉证据

- 内容管理：确定性 mock E2E 覆盖中文文章列表、编辑 Drawer、分类和英文 375；axe JSON 记录 `zh-CN.1440`、`zh-CN.categories.1440`、`en-US.categories.375` 均为 0 serious/critical。
- 版本管理：确定性 mock E2E 覆盖中文 1440 列表、冲突 Drawer、英文 375；axe JSON 的三个场景均为 0 serious/critical、0 violations，唯一预期控制台记录为一次 409 Conflict。
- 截图：[内容管理中文 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-content-management/screenshots/1440x900-light.png)、[内容编辑 Drawer](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-content-management/screenshots/drawer-zh-CN-1440.png)、[内容管理英文 375](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-content-management/screenshots/375x812-en-US.png)、[版本管理中文 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshots/1440x900-light.png)、[版本 Drawer](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshots/drawer-zh-CN-1440.png)、[版本管理英文 375](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshots/375x812-en-US.png)。
- Mobile 的 UI Skill request/output/decisions/checklist 已保存于 `apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-mobile-framework/`；该 checklist 中的 Android 360×800、iOS 390×844、Harmony 430×932 截图、动态字体、reduced motion 和英文扩展仍未勾选。因此本节不列移动端截图为通过证据。

### 平台、设备与风险边界（实现完成；真实设备验收未执行）

- Android：此前隔离 HBuilderX CLI 的 exit 0 证据不替代最终合并代码。最终 `ak-mobile-framework-android-final-clean-1785906028` 完成 20 页至 Android class、UTS、`ready in 19314ms` 和正常停止。Terra 修复原 identity-equality 4 条并在复编译再修复 profile edit 1 条，共 5 处 `Any?/Boolean` 与 `Int` 长度的 UTS 值比较；静态 7 项、严格错误/警告/Identity 搜索均为 0。launch 数值 exit 因外层 `PIPESTATUS` 误用未可信捕获，project close 成功，因此只记录 compile-only。无 APK/AAB、安装、模拟器、真机或安全存储回读证据，临时项目已清理且无进程。
- iOS：前一轮 `ak-mobile-framework-ios-rootqa` 的 exit 0 证据保留；第 3 次最终合并代码在 `ak-mobile-framework-ios-final-evidence` 的 HBuilderX 5.06 隔离验证中 CLI exit 0，完成 20 页/`ak-secure-storage` UTS，`ready in 30446ms` 后正常“已停止运行”。清理前严格搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality` 均为空；标准基座/custom base/iOS 13+ 为预期提示。模拟器安装/启动、签名、Keychain 回读/清除均未执行；项目已关闭/清理且无进程。
- Harmony：最终源码在唯一项目 `ak-mobile-framework-harmony-final.ECScJj` 上以 HBuilderX 5.06 CLI 可信 exit 0 回归；20 页、UTS、`ready in 15236ms`，工程生成、依赖安装和运行包制作成功，恰 1 个 `entry-default-unsigned.hap`。清理前严格搜索 `ERROR`、`Error:`、`错误`、`编译失败`、`Identity equality`、`currentColor` 均零匹配（`rg` exit 1）。secure-storage 修复为 interface 显式 class、三端 availability new 实例与 Harmony TextDecoder options 显式类型，Asset Store 仍保留。包名/签名/未签名为预期 warning；没有安装、Asset Store 回读或真机 smoke。项目已 close，临时项目/日志已清理，无进程。主题 20/20 覆盖已移除 `currentColor`，并为三种 button variant 的 loading 使用显式 token。
- Backend 临时 PostgreSQL 18 实测、Admin mock E2E 和本表静态检查都不是生产部署/生产数据验收。真实 Go API/PostgreSQL 浏览器联调、第三方 OAuth/Push、应用商店升级、弱网、三端真机与生产发布仍 blocked/未验证。
- 本代理没有改动 `apps/ak-mobile` 活跃源码；当前工作树原有的其他未提交实现均予以保留。提交/推送状态由主代理在最终范围和回归结论确定后单独记录。

## 2026-08-05 Mobile 四页面 ImageGen 视觉概念（历史设计阶段，后续已实现）

### 交付内容

- 新增 Mobile Master 设计系统与 Home、Profile、Articles override，采用 Minimal Swiss、浅灰页面、白色卡片、克制蓝色、系统字体、44px 触控目标与安全区规则。
- 使用内置 ImageGen 分别生成 Home、Profile、Article List、Article Detail；后续页面以上一张成品作为严格视觉参考，保持色彩、图标、圆角、排版和几何插图一致。
- 文章页面按堆栈视觉概念设计且不修改既有三 Tab；该设计阶段尚无 Article route/API/permission，后续 Mobile Framework 交付已补齐相关实现与契约。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 --version` | 0 | Python 3.9.6 |
| `ui-ux-pro-max` design-system 查询 | 0 | Minimalism & Swiss Style；蓝色 CTA、浅灰背景、清晰层级与无搜索/导航反模式建议 |
| `ui-ux-pro-max` Mobile UX 查询 | 0 | 12 项结果；44px 触控目标、8px 间距、可预测返回与手势冲突约束 |
| `ui-ux-pro-max` Vue stack 查询 | 0 | 3 项列表/渲染建议；本轮未把 raster 概念冒充 UVue 实现 |
| 内置 ImageGen 4 次生成 | succeeded | 4 张独立 PNG 均生成并复制到 workspace；每张 853×1844 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 35 routes、3 tabs、26 baseline APIs、28 API delta、9 permission delta、33 components、26 tasks、3 platforms；0 error/warning |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US`、默认与最终回退 `zh-CN` 通过 |
| `git diff --check` | 0 | 文本补丁格式通过 |

### 截图索引与验证边界

- [Home 视觉稿](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-imagegen-four-surfaces/screenshots/home-853x1844-light.png)
- [Profile 视觉稿](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-imagegen-four-surfaces/screenshots/profile-853x1844-light.png)
- [Article List 视觉稿](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-imagegen-four-surfaces/screenshots/articles-list-853x1844-light.png)
- [Article Detail 视觉稿](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-imagegen-four-surfaces/screenshots/article-detail-853x1844-light.png)
- 该设计阶段只完成 raster 视觉审查和静态蓝图/i18n 校验；UVue、API、Route Registry、语言包与业务实现由后续 Mobile Framework 交付完成。
- Android/iOS/Harmony 构建、模拟器/真机、双语运行时、动态字号、交互状态与可访问性自动化未执行。本轮未 commit、未 push。

## 2026-08-05 字典驱动化、云存储与通知发送完善

### 交付内容

- 字典由独立 CRUD 升级为统一选项源：增加 visibility、`fixed/open/registered/s3_compatible` 扩展策略、内置锁定、租户覆盖、locale 回退、public/internal 隔离和消费接口；配置 Schema 使用 `x-appkernia-dictionary` 做前后端双重校验。
- `core-dictionaries.json` 接入 `ak-cli seed core`：4 个核心类型、40 个全局双语项、22 个全局 SMS/Email 模板可幂等初始化；稳定协议枚举继续保留在代码和数据库约束中。
- 云存储保留 `local/s3/minio`，新增 `cos/oss/qiniu` S3-compatible profile；OSS/COS/Qiniu 强制虚拟主机寻址，自定义驱动仅允许 `adapter=s3_compatible`。文件列表、上传策略、配置表单与 OpenAPI Client 均改为消费 `storage.driver`，不再维护固定 provider 数组。
- 短信配置按腾讯云/阿里云隔离密钥和供应商字段；Adapter 使用腾讯云 SMS v20210111、阿里云 Dysmsapi 20170525 官方 Go SDK。Secret 继续 AES-GCM 加密且不回显。
- 通知模板增加 `body_format`、HTML 净化、声明变量渲染与转义、SMS 外部模板绑定；真实测试投递使用目标/载荷密文、事务内 River 入队、`dedupe_key` 和保守重试分类。短信未知结果不自动重放，人工重试必须确认重复扣费风险。
- SMTP 改用 `wneessen/go-mail`，支持 context、STARTTLS/implicit TLS 和超时；生产拒绝明文与跳过证书校验。找回密码已接入注册租户的 `email/password_reset` 异步投递，同时保持匿名账号枚举防护。
- Admin 字典、配置、文件、通知模板与投递页完成字典 Select、供应商条件字段、结构化 S3 元数据、SMS 绑定、真实测试抽屉和风险提示；根 `AGENTS.md` 已加入枚举与字典决策强制规则。
- Migration/OpenAPI、Admin Client、权限/页面/API/Schema 快照、Backend/Admin 蓝图和双语事实源均已同步。Mobile 当前仓库没有 OpenAPI→UTS Client 生成实现，本轮只同步 Mobile 可消费的公开 OpenAPI 并通过 Mobile 蓝图校验，未伪造生成产物。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| migration `up → seed×2 → down → up → seed` | 0 | PostgreSQL 18 最终 `version=11, dirty=false`；核心 4 类型、40 全局双语项、22 全局 Email/SMS 模板 |
| `make check` | 0 | gofmt、Go vet 与 Backend 77 个顶层单元/契约测试通过 |
| `AK_TEST_DATABASE_URL=... AK_TEST_S3_ENDPOINT=127.0.0.1:9000 make test-integration` | 0 | IAM、Job、System Settings、Notification、Storage 定向 6 个包通过 |
| `AK_TEST_DATABASE_URL=... AK_TEST_S3_ENDPOINT=127.0.0.1:9000 go test -tags=integration ./...` | 0 | 全仓 28 个顶层 integration tests 通过；含字典并发/覆盖、加密 River 入队、找回密码和真实 MinIO 自定义 S3 Put/Open/Delete |
| `npm run check` | 0 | 14 files / 58 tests、ESLint、TypeScript strict、4,163 modules build、bundle budget、158 APIs/54 tables/33 pages Admin validator 全通过 |
| Backend / Mobile / i18n / UI Skill validators | 0 | 11 migrations/61 tables；Mobile 0 error/warning；双语 key/占位符一致；Skill 可用 |
| 最终 Docker build + force recreate | 0 | API、Worker、Admin、Migrate、Seed、Bootstrap 工具镜像成功；API/Admin/PostgreSQL healthy，Worker running |
| `e2e_dictionary_notification_drivers.py` | 0 | Docker Chromium + 真实 API/PostgreSQL；4 个截图状态均 0 serious/critical、无溢出、控制台错误 0；Email test 返回 202，目标明文不存在于 ciphertext，审计日志 2 条 |
| `python3 -m py_compile ...` + `git diff --check` | 0 | E2E 脚本语法与文本补丁格式通过 |

### 截图索引与验证边界

- [中文字典桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-notification-drivers/screenshots/zh-CN-dictionaries-1440.png)
- [英文短信绑定桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-notification-drivers/screenshots/en-US-sms-binding-1440.png)
- [英文短信测试风险桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-notification-drivers/screenshots/en-US-sms-test-risk-1440.png)
- [中文通知模板移动 375](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-notification-drivers/screenshots/zh-CN-templates-375.png)
- [结构化 Playwright 证据](../output/playwright/dictionary-notification-drivers-e2e-results.json)
- 本机 MinIO 是真实 S3-compatible 网络集成，不代表腾讯 COS、阿里 OSS、七牛 Kodo 的生产凭据联调。未提供隔离云/SMS/SMTP 凭据，因此这些外部 smoke test 均为未验证，不以 Mock 或本机 MinIO 替代。
- 未部署生产；Firefox/Safari、Android/iOS/Harmony、真机和真实 SMTP 未执行。本轮未 commit、未 push；已删除本任务失败调试截图并精确清理临时 E2E 用户/租户（最终计数 0）。

## 2026-08-05 字典管理分类导航与行编辑优化

### 交付内容

- 对照 HotGo 的 `pid` 字典类型树后，AppKernia 采用不污染消费模型的前端分类：从稳定字典代码首段自动形成 namespace 分类，左侧以可折叠“分类 → 字典类型”展示，并保留选中态、锁定/扩展策略和 URL 状态。
- 类型导航默认一次加载 100 条并纠正过期页码；右侧将显示标签和字典键值拆列，桌面固定标签/键值和操作列，移动端使用容器内横向滚动。
- 可扩展内置行现在显示“编辑”，保存时创建当前租户覆盖；已有覆盖再次编辑仍识别为覆盖模式，键值、语言和编译期能力元数据不可改。内置行不显示删除，租户覆盖继续支持编辑/删除。
- Repository 对 `registered` 与已注册 `s3_compatible` 值要求覆盖元数据与全局项语义一致；新增 S3 值仍强制 `adapter=s3_compatible`。集成测试夹具增加精确字典清理，并清除了本机 23 条符合测试 UUID/hex 规则的历史残留字典类型。
- 重新执行 `ui-ux-pro-max` 并保存 request/output/decisions/checklist、页面 override、双语截图和结构化 Playwright 结果。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `npm run lint` + `npm run typecheck` + 定向 Vitest | 0 | ESLint、TypeScript strict 通过；新增 `dictionaryPresentation.test.ts` 2/2 通过 |
| `npm run check`（Admin） | 0 | 生成 Client/i18n/routes、ESLint、TypeScript strict、15 files / 60 tests、4,164 modules build、bundle budget 与 Admin 蓝图校验全部通过 |
| `go test ./internal/modules/systemsettings/repository ./internal/modules/systemsettings/application` | 0 | System Settings 单元测试通过 |
| `make check`（Backend） | 0 | go vet 与全仓 Go 单元/契约测试通过 |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/systemsettings/repository -count=1 -v` | 0 | 4 个具名测试通过；覆盖内置元数据覆盖、未知 registered 拒绝、自定义 S3 约束，并确认测试字典残留计数 0 |
| `docker compose build admin` + `docker compose up -d --no-deps admin` | 0 | Admin production image 构建并启动 |
| `e2e_dictionary_category_editing.py` | 0 | 真实 API/PostgreSQL：租户覆盖 POST=201、再次编辑 PATCH=200；zh-CN/en-US 1440 与 en-US 375 均 0 serious/critical axe、0 页面溢出、0 console/HTTP error |

### 截图索引与验证边界

- [中文分类与编辑桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-category-editing/screenshots/zh-CN-dictionaries-category-edit-1440.png)
- [英文分类与编辑桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-category-editing/screenshots/en-US-dictionaries-category-edit-1440.png)
- [英文分类与编辑移动 375](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-category-editing/screenshots/en-US-dictionaries-category-edit-375.png)
- [结构化 Playwright 证据](../output/playwright/dictionary-category-editing-e2e-results.json)
- 本轮验证 Docker Chromium 与 PostgreSQL 18；Firefox/Safari、生产部署及移动真机未执行。本轮未 commit、未 push。

## 2026-08-05 字典项颜色与样式类可创建选择器

### 交付内容

- 新增公共 `AkCreatableSelect`，用受控单值 tags Select 同时承载“默认（不指定）”、搜索预设、选择预设与键盘 Enter 创建自定义值；与 React Hook Form 的 `color/css_class` 字符串保持兼容，内部默认标识不会提交 API。
- 颜色下拉提供 7 个语义预设，样式类下拉提供 6 个编译期预设；选项包含色块/样式预览、双语名称和原始值，字段下方明确说明自由创建方法。
- 仅允许编译期样式预设在管理端 DOM 进行效果预览；自定义类名不会注入管理页面 class。自定义颜色仅对安全十六进制值显示色块，原始字符串仍完整保存和显示。
- 外观表格列可同时显示颜色与样式类；选择器弹层挂回字段父容器，消除了 Ant Design 默认 portal 引起的 axe `region` moderate 问题。
- `ui-ux-pro-max` 全流程、Master/page override、双语蓝图事实源/生成 Catalog、组件/helper 单测与真实 Playwright 写链路均已保存。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + UX + React 查询 | 0 | 采用受控组合框、键盘创建、显式标签、文字+效果双重表达和可见焦点建议；产物保存到 `dictionary-appearance-selects/` |
| 定向 Vitest + typecheck + lint | 0 | 新增组件与 helper 共 5 项测试通过；自由输入 Enter、预设选择、默认态、axe 与受控预览 allowlist 均覆盖 |
| `npm run check`（最终源码） | 0 | Client/i18n/routes 生成、ESLint、TypeScript strict、17 files / 65 tests、4,166 modules build、208,539 B initial gzip、bundle budget 与 Admin 蓝图全部通过 |
| Docker Admin build/recreate | 0 | 4174 最终 production image 启动并 healthy |
| `e2e_dictionary_category_editing.py`（扩展外观用例） | 0 | 预设 POST=201、自定义 PATCH=200；7 个截图状态 axe 0 violations、页面无溢出、console/HTTP error 为 0 |
| Mobile blueprint / i18n / UI Skill / E2E py_compile / `git diff --check` | 0 | Mobile 0 error/warning、双语键/占位符一致、Skill 可用、脚本语法和文本补丁格式通过 |
| PostgreSQL 精确清理复核 | 0 | 临时 E2E 用户 0、测试标签覆盖项 0 |

### 截图索引与验证边界

- [中文预设下拉桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-appearance-selects/screenshots/zh-CN-dictionary-appearance-presets-1440.png)
- [中文自定义值桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-appearance-selects/screenshots/zh-CN-dictionary-appearance-custom-1440.png)
- [英文自定义值桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-appearance-selects/screenshots/en-US-dictionary-appearance-custom-1440.png)
- [英文自定义值移动 375](../apps/ak-admin/artifacts/ui-ux-pro-max/dictionary-appearance-selects/screenshots/en-US-dictionary-appearance-custom-375.png)
- [结构化 Playwright 证据](../output/playwright/dictionary-appearance-selects-e2e-results.json)
- 本轮真实浏览器边界为本机 Docker Admin/Go API/PostgreSQL + Chromium；未部署生产，Firefox/Safari、Android/iOS/Harmony 与真实移动设备未执行。本轮未 commit、未 push，并原样保留用户已有 Mobile 未跟踪目录。

## 2026-08-05 Admin 折叠侧栏 BUG 修复与样式优化

### 交付内容

- 桌面折叠态不再传入受控 `openKeys`，Ant Design Menu 恢复原生 hover/focus 展开与关闭；展开桌面保留当前路径祖先，移动 Drawer 保持 click 驱动的受控树，不受桌面折叠状态影响。
- 所有目录浮层使用 `ak-navigation-submenu-popup`：86% 深色透明表面、14px blur、轻 hairline 与两层阴影；贴近父级的一侧取消圆角，外侧保留 10px 圆角。
- 折叠控件由侧栏绝对定位改为视口固定定位：22×40px、图标 16px、无横向 padding；在 900px 高视口中 `top=430 / bottom=470 / centerY=450`，展开/折叠时分别贴合 `x=248 / x=80` 侧栏边界。
- 更新 App Shell design-system override，保存 `ui-ux-pro-max` 请求、真实检索输出、决策、review checklist、双语截图和可重复 E2E 脚本。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + UX + style + React 查询 | 0 | 采用渐进披露、原生键盘语义、半透明 blur、轻边框与层叠阴影；拒绝营销结构、外部字体和高噪声液态效果 |
| `npm run typecheck` / `npm run lint` / `npm test` | 0 | TypeScript strict 与 ESLint 通过；17 个测试文件、65 项测试通过；JSDOM 仅输出已知 pseudo-element/Canvas 不支持提示 |
| `npm run build` / `npm run validate:bundle` | 0 | Vite production build 4,166 modules；initial gzip 209,222 B、最大 chunk 165,429 B，均在预算内 |
| Docker Admin build + force recreate | 0 | 当前源码镜像构建成功，4174 Admin healthy |
| 系统 Python 直接运行 E2E | 1 | 当前 Python 缺少 `playwright`；未冒充通过，切换为隔离 `uv run --with playwright` |
| E2E harness 中间运行 | 1 | 依次修正测试脚本的 regex、`wait_for_function` 参数、账户语言持久化和 localhost CSRF origin；产品代码无需回退 |
| 最终 `e2e_sidebar_flyouts.py` | 0 | zh-CN/en-US 均得到 popup `0 → 1 → 2 → 0`；两层计算样式符合透明度/圆角/blur，控制位置在 80/248px 边界且垂直中心精确为 450px |
| axe / overflow / console | 0 | serious/critical=0、页面无水平溢出、控制台错误=0 |
| Admin/Mobile/i18n/UI Skill validators | 0 | 35 menus/48 routes/161 APIs/54 tables/33 pages；Mobile 0 error/warning；双语契约与 Skill 可用性通过 |
| E2E 测试账号与临时服务清理 | 0 | 原始密码哈希、失败次数和锁定状态恢复并比对成功；登出会话，删除临时凭据文件并停止临时 4173 Admin |
| `git diff --check` | 0 | 最终文本补丁格式通过 |

### 截图索引与验证边界

- [中文折叠三级浮层 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-collapsed-flyout/screenshots/zh-CN-collapsed-third-level-1440.png)
- [英文折叠三级浮层 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-collapsed-flyout/screenshots/en-US-collapsed-third-level-1440.png)
- [结构化浏览器证据](../output/playwright/sidebar-collapsed-flyout.evidence.json)
- 本轮真实浏览器范围为本机 Docker Admin/API/PostgreSQL + Chromium 139，UI 页面使用 Dashboard 的同一 `AppShell` 验证；本次没有后端、OpenAPI、数据库、权限或 i18n 文案变更。
- Firefox/Safari、生产部署、Android/iOS/Harmony 与真实移动设备未执行。本轮未 commit、未 push，并保留工作区全部既有字典、地区、通知、Mobile 和 Backend 未提交改动。

## 2026-08-05 地区管理可编辑化追加

### 交付内容

- Backend 新增 `sys.region.create/update/delete`、`POST /admin-api/v1/regions`、`PATCH/DELETE /admin-api/v1/regions/{code}`，并在 Serializable 事务内完成层级推导、乐观锁、叶子校验、软删除和 `sys.region` 操作审计。
- `000014_region_management` 增加 `version/is_manually_managed/deleted_at` 与活动父级索引；所有地区读取和 `has_children` 排除软删除。核心目录 upsert 对手工记录保持现值，因此不会覆盖编辑或恢复已删除节点。
- Admin 使用动作权限而非角色名控制操作；省/市可新增直接下级，编辑抽屉锁定编码/父级/层级，删除无级联入口。写入后更新当前分支 `has_children` 并失效相关搜索查询。
- OpenAPI、sqlc/生成 Client、权限种子、页面权限矩阵、页面 API 映射、Schema 快照、Backend/Admin 蓝图、设计系统 override 和 `zh-CN/en-US` 均已同步。
- 浏览器验收期间发现删除说明误用了 `{{name}}`；已按项目单花括号插值契约修为 `{name}` 并真实复核显示地区名。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make sqlc-generate`（server） | 0 | `UpsertCoreRegion` 生成代码与查询同步 |
| 隔离 PostgreSQL 18 migration `up → down → up` | 0 | `000012` 可往返，最终 migration 12 clean |
| `AK_TEST_DATABASE_URL=... go test -tags=integration ./internal/modules/systemsettings/repository` | 0 | 大分支 `has_children`、创建/更新/版本冲突、深层拒绝、父节点拒删、叶子软删除、种子保留手工状态及 4 条审计全部通过 |
| `make check`（server） | 0 | gofmt、go vet、全仓 Go 单元/契约测试通过 |
| 地区范围 ESLint + 定向 Vitest | 0 | 地区页面、settings hooks、auth session 相关文件 lint 通过；2 files / 27 tests 通过 |
| Admin `npm run typecheck` / `npm run build` / bundle budget | 0 | TypeScript strict、4,163 modules production build通过；initial gzip 210,109 B、最大 chunk 165,429 B 均在预算内 |
| Admin 全量 `npm run lint` / `npm test -- --run` | 1 / 1 | 被工作树其他在途改动阻塞：`AdminServiceHealthPage.tsx` 3 个 lint；已删除模块菜单后旧 `menu-icons.test.tsx` 仍期望 35、实际 34。地区定向测试不受影响；未越权修改这些文件 |
| Backend/Admin/Mobile/i18n validators | 0 | 当前合并工作树为 13 migrations、111 permissions、160 APIs、54 tables、32 pages；地区 3 项写权限和 3 个写接口均包含在内，双语 key/占位符一致 |
| `e2e_regions_modules.py`（一次性 venv + 本机 Chrome） | 0 | 父节点删除拦截；省下新增市、市下新增区县、编辑、两次叶子删除；只读动作隐藏；中英文 1440、中文 375；地区 `main` 五状态 axe=0、无页面溢出、console error=0 |
| 公开 API 查询软删除 E2E 编码 | 0 | `items=[]`，删除节点不再公开返回 |

### 截图索引与验证边界

- [英文可管理桌面 1440](../output/playwright/admin-regions-manage.en-US.1440.png)
- [英文编辑抽屉桌面 1440](../output/playwright/admin-regions-edit-drawer.en-US.1440.png)
- [中文可管理桌面 1440](../output/playwright/admin-regions-manage.zh-CN.1440.png)
- [中文窄屏 375](../output/playwright/admin-regions-manage.zh-CN.375.png)
- [英文只读桌面 1440](../output/playwright/admin-regions-read-only.en-US.1440.png)
- [结构化 E2E 证据](../output/playwright/admin-regions-modules-e2e-results.json)
- 地区页 `<main>` 在五个截图状态均为 axe 0 violation。全页审计单独暴露既有深色侧栏 `System Settings / System Configuration / Dictionaries` 英文文字对比度约 2.34–2.57:1，本轮未扩展修改全局导航样式，风险保留。
- 本轮只验证本机 PostgreSQL 18、Go API、Vite Admin 和 Chrome；未部署生产，Firefox/Safari、Android/iOS/Harmony 与真实移动设备未执行。临时数据库、API/Vite 进程和一次性 Python venv在交付前清理；未 commit、未 push。

## 2026-08-05 Admin 侧栏三态控制与持久化

### 交付内容

- 新增 Zustand 侧栏偏好 store，以 `expanded / collapsed / hidden` 作为唯一状态源；叶子路由只关闭移动 Drawer，不再修改桌面侧栏模式。
- 中间控件缩为 18×40px，并增加稳定层叠层级、紧邻侧栏边界的 hover 命中区、反色背景/图标、可见边框和阴影；折叠态上方新增 18×28px CloseOutlined 完全隐藏控件。
- 完全隐藏态同时提供左侧 48% 透明边缘恢复条和 Header 操作组首位恢复按钮；两者都直接恢复 248px 展开态，恢复后同时消失。
- 双语事实源、生成 Catalog、Master/App Shell override、`ui-ux-pro-max` request/output/decisions/checklist、可复现 E2E 与截图全部同步。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` design-system + UX + web + React 查询 | 0 | 采用单一持久状态、渐进披露、图标 accessible name、可见 hover/focus 与无布局位移动画 |
| 侧栏定向 ESLint + Vitest | 0 | `AppShell`/store 相关 ESLint 通过；`sidebar-store.test.ts` 5/5 通过 |
| `npm run typecheck` / `npm run build` / `npm run validate:bundle` | 0 | TypeScript strict、4,163 modules production build通过；initial gzip 210,115 B，预算通过 |
| Admin 全量 `npm run lint` | 1 | 工作区另一项 `AdminServiceHealthPage.tsx` 的布尔比较与 2 个废弃属性错误；未越权修改 |
| Admin 全量 `npm run test` | 1 | 20 files 中 19 通过、75 项中 74 通过；唯一失败为菜单已改 34 项但旧测试仍硬编码 35 |
| Docker Admin build + force recreate | 0 | 4174 production image 使用最终侧栏源码重建并 healthy |
| `e2e_sidebar_flyouts.py` | 0 | 路由后保持 80px；18px 控件居中/反色；隐藏 48%→100%；Header/边缘双恢复；中英文 1440 与移动 Drawer 375 均通过 |
| axe / overflow / console | 0 | serious/critical=0、页面无水平溢出、控制台错误=0 |
| Admin/Mobile/i18n/UI Skill validators + py_compile + `git diff --check` | 0 | 34 menus/47 routes/111 permissions/160 APIs/32 pages；Mobile 0 error/warning；双语契约、脚本语法与补丁格式通过 |
| E2E 测试账号恢复 | 0 | 密码哈希、失败次数与锁定状态退出时恢复并比对一致，测试会话完成登出 |

### 截图索引与验证边界

- [中文折叠三级浮层 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-three-state-controls/screenshots/zh-CN-collapsed-third-level-1440.png)
- [中文完全隐藏 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-three-state-controls/screenshots/zh-CN-hidden-1440.png)
- [英文完全隐藏 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-three-state-controls/screenshots/en-US-hidden-1440.png)
- [英文隐藏态移动 Drawer 375](../apps/ak-admin/artifacts/ui-ux-pro-max/sidebar-three-state-controls/screenshots/en-US-hidden-mobile-drawer-375.png)
- [结构化浏览器证据](../output/playwright/sidebar-three-state-controls.evidence.json)
- 本轮真实范围为本机 Docker Admin/API/PostgreSQL + Chromium；未部署生产，Firefox/Safari、Android/iOS/Harmony 与真实移动设备未执行。本轮未 commit、未 push，并保留工作区全部其他在途改动。

## 2026-08-05 服务状态收敛与真实模块初始化

### 交付内容

- 独立模块管理功能已从菜单、路由、页面、Admin Client、Backend API、OpenAPI、权限种子、契约快照和双语资源中彻底移除；保留 `sys.modules` 仅供运行摘要读取。
- 核心目录固定为 8 个真实领域模块，包含稳定编码、双语名称/说明键、编译期能力和状态；种子在 Serializable 事务内幂等 upsert 并清除目录外测试记录。
- 新增共享 `buildinfo`，API/CLI/Worker/种子/运行摘要使用同一构建版本，Makefile 和 Docker 支持 ldflags 注入，本地回退 `dev`。
- 服务状态页补齐语义模块信息、双语能力标签、未知编码回退、等高状态卡、24/16px 响应式垂直节奏、16px 运行摘要内部间距、移动堆叠和键盘可聚焦表格滚动。
- 更新 Service Status 设计系统 override，并保存真实 `ui-ux-pro-max` 请求、输出、决策、检查清单、截图索引及可复现 Playwright 脚本。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make migration-up && make seed-core && make seed-core` | 0 | Migration 13 clean；两次初始化均为 modules=8、build_version=dev |
| `make migration-down && make migration-up && make seed-core` | 0 | `000013` 可往返；最终 migration 13 clean、8 模块恢复正确 |
| PostgreSQL 精确复核 | 0 | module_count=8、unknown_count=0、version_count=1/version=dev、`sys.module.read`=0；旧菜单 status=disabled 且 permission_id 已清空 |
| `AK_TEST_DATABASE_URL=... make test-integration` | 0 | Seed、Ops、IAM、Jobs、System Settings、Notification、Storage 隔离库集成测试通过；覆盖幂等、版本更新、精确 8 项和未知 fixture 清理 |
| `make check` / `go test -race ./...`（server） | 0 / 0 | gofmt、go vet、全仓 Go 测试和 Race 测试通过 |
| `govulncheck ./...` | 0 | 当前调用链受已知漏洞影响数为 0 |
| `staticcheck ./...` / `golangci-lint run` | 1 / 1 | 仅报告工作区既有 notification/storage ST1005 错误文案大小写；未扩大范围修改无关在途代码 |
| `npm run check`（Admin） | 0 | Client/i18n/routes 生成、ESLint、TypeScript strict、20 files / 75 tests、4,163 modules production build、bundle budget 和 Admin 蓝图通过 |
| Admin/Mobile/i18n 蓝图校验 | 0 | 34 menus、47 routes、111 permissions、160 APIs、32 pages；Mobile 0 error/warning；双语 key/占位符一致 |
| Backend 独立 `validate_blueprint_specs.py` | 未执行 | 仓库不存在该脚本；Backend 由 `make check`、integration、OpenAPI Client 生成和现有契约校验覆盖 |
| Docker `admin` 最终重建 | 0 | Admin/API/migrate/seed 使用最终源码构建并启动，API healthy |
| `e2e_service_status.py` | 0 | `zh-CN/en-US × 375/768/1024/1440` 共 8 组；axe=0、页面无水平溢出、每组 2 个可聚焦滚动区、console error=0；旧 modules API=404 |
| `git diff --check` | 0 | 最终文本补丁格式通过 |

### 截图索引与验证边界

- [中文桌面 1440](../output/playwright/service-status/service-status.zh-CN.1440.png)
- [中文移动 375](../output/playwright/service-status/service-status.zh-CN.375.png)
- [英文桌面 1440](../output/playwright/service-status/service-status.en-US.1440.png)
- [英文移动 375](../output/playwright/service-status/service-status.en-US.375.png)
- [结构化 E2E 证据](../output/playwright/service-status/e2e-results.json)
- 浏览器验证使用最终 Docker Admin 和契约等价 mock auth/data；真实 PostgreSQL 模块目录、运行摘要 Repository 与旧 API 404 由独立 DB/integration/API 请求验证。由于本机 API 拒绝新 bootstrap 凭据，未标记为生产登录后 UI 闭环。
- 未部署生产；Firefox/Safari、Android/iOS/Harmony 与真实移动设备未执行。本轮未 commit、未 push，并保留工作区全部其他在途改动。

## 2026-08-06 GitHub Actions CI 失败修复

### 交付内容

- 真实检查 [CI run 31014624161](https://github.com/Payhon/AppKernia/actions/runs/31014624161)：`admin` 在 setup-node 的 pnpm cache 阶段找不到 pnpm；`mobile-blueprint` 在项目检查阶段找不到 `rg`。
- Admin job 先执行 `pnpm/action-setup@v4`（11.18.0），再执行 `actions/setup-node@v5`；Mobile job 在项目检查前通过 apt 安装 `ripgrep`。
- 补齐 `rg` 后，Mobile 扫描进一步发现 9 处禁用 `any`。i18n 插值参数收紧为字符串 Map，数值由调用方显式转字符串；Catalog 与响应 Header 通过 `UTSJSONObject.getString` 读取，不使用无边界动态类型。

### 真实命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `gh run view 31014624161 --log-failed` | 0 | 确认 admin 与 mobile-blueprint 原始失败日志；同 run 的 backend、blueprint 已成功 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile Blueprint、i18n、VDOM、禁用模式扫描及三平台构建脚本目标检查通过 |
| `make check-blueprints` | 0 | Backend、Admin、Mobile 与统一 i18n 静态契约全部通过 |
| `pnpm install --frozen-lockfile && pnpm --filter @appkernia/admin check` | 0 | lint、strict typecheck、16 files / 64 tests、production build、bundle budget、Admin Blueprint 均通过 |
| Ruby YAML parse / `actionlint v1.7.7` / `git diff --check` | 0 / 0 / 0 | Workflow 语法、job 定义、Shell 和补丁格式通过 |

### 验证边界

- 本机 Node 为 26.5.0，Admin 检查产生 Node 24 engine warning；GitHub workflow 固定 Node 24.18.1，pnpm 固定 11.18.0。
- 这次只修复 CI 环境和被检查器暴露的 Mobile 类型边界；没有 API、数据库、权限或可见 UI 变更。
- Android/iOS/Harmony 的 compile、安装、模拟器和真机均未在本轮重新执行，不能用静态 CI 结果替代平台验收。

## 2026-08-06 Mobile Framework 主分支同步复核

### 交付结论

- 本地 `main` 与 `origin/main` 均为 `ddaaa4e`，且 `6256b90` 是其祖先；逐路径集合比对确认原 worktree 中的 `apps/ak-mobile` 框架文件在主分支缺失数为 0。
- 主分支包含 20 个 UVue 页面并全部使用 `ak-theme-root`，同时保留 `6256b90` 之后新增的四页面视觉证据与 5 个 Mobile 类型边界修正，没有把当前主分支降级为旧 worktree 快照。
- 修复静态 verifier 对旧 `Map<string, any>` 阅读时长插值的硬编码，改为校验当前 `Map<string, string>` 的 `minutes.toString()` 边界。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| Mobile Framework 文件集合与 Git 祖先关系比对 | 0 | `6256b90` 已在 `main`；框架文件缺失 0，页面 20/20，主题根 20/20 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 37 routes、3 tabs、34 API delta、11 permission delta，0 error/warning |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN`/`en-US` key、占位符、default/fallback 契约通过 |
| Mobile framework / secure-storage / refresh-policy 校验 | 0 | 三项源级与 fake HttpPort 门禁通过 |
| `git diff --check` | 0 | 补丁格式通过 |

### 验证边界

- 本轮是主分支文件完整性和静态契约复核，没有重新执行 Android/iOS/Harmony compile、安装、模拟器或真机；既有平台证据仍以此前交付记录为准。
