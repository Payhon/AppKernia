# AppKernia Codex 交付报告

日期：2026-08-30
范围：AppKernia 全仓交付记录；本轮消息推送文档已提交、推送并发布到 GitHub Pages。

## 2026-08-30 消息推送功能文档站交付

### 已交付

- 新增中英文“消息推送架构”，用 Mermaid 展示 Admin/M2M → NotificationService → PostgreSQL/River → 发布/扇出/单设备 Worker → 九渠道 → Mobile/opened → 运营统计的完整链路，并明确厂商受理、设备展示、opened 和已读的不同语义。
- 新增中英文“推送渠道配置”，覆盖 APNs、FCM、华为 Android、荣耀、小米、OPPO、vivo、魅族、Harmony 所需字段、write-only 凭据、预检/激活/故障状态、Admin 问号 Tab 申请指引、Google/China 互斥变体、隐私初始化和生产真机门禁。
- 新增中英文“消息运营工作台”，说明概览、发布运行、队列任务、失败中心四个 URL 可恢复 Tab，任务/运行状态、15 秒条件轮询、90 天/13 个月保留、权限、排障顺序及 `unknown_after_write` 风险确认规则。
- 新增中英文“移动端权限中心”，记录六种稳定权限状态、用户主动授权顺序、iOS/Android/Harmony 设置行为、`PermissionPort`、未使用能力不展示不申请，以及旧 API 缺少 `share`/`push` 时的 fail-closed 启动兼容。
- 新增中英文“通知与推送 API”，区分 `ak-api` 与 `ak-mobile` 身份面，记录 API Client App allowlist、权限、M2M 幂等提交/状态/取消、Mobile 偏好/设备/opened 路径和内部 Go `NotificationService` 接入边界。同步 Guide/Concepts/API 导航、入口页、Mobile 开发和资源页交叉链接。

### 实际验证

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm check:api-docs`（`apps/ak-docs`） | 0 | 147 条手写 API 路径全部存在于 `server/openapi/openapi.yaml`。 |
| `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm check` | 0 | rslint 0 error/0 warning，TypeScript、Prettier、OpenAPI 同步、死链/锚点、Mermaid、80 页 Rspress 静态构建、双语 parity 与 Sitemap 全通过。 |
| 新增静态产物与 LLM 文档探针 | 0 | 5 组、共 10 个中英文 HTML 页面均存在；`llms-full.txt` 与英文版本均包含新导航、正文和交叉链接。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 45 routes、4 tabs、58 API delta、11 permission delta、40 components、11 privacy capabilities、26 tasks、3 platforms，0 error/0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US` 默认与回退契约通过。 |
| `git diff --check` | 0 | 文档补丁无空白错误。 |
| `git fetch origin main` / `git fetch gitee main` | 0 / 0 | 发布前确认 GitHub、Gitee 与本地基线均为 `17cf4472a8d3da22cf039c2bfed73ff08bd11f45`。 |
| `git push origin main` / `git push gitee main` | 0 / 0 | 文档内容提交 `4a69cddf8ed4f933c51c785f5d1b447632183686` 到达两个远端。 |
| GitHub Pages run `33289498354` | 0 | build 52 秒、deploy 10 秒，最终 conclusion=`success`。 |
| 线上 10 个中英文页面 HTTP/正文探针 | 0 | 五组页面均为 HTTP 200；中英文架构标题正文标记可读。 |
| 线上 OpenAPI SHA-256 探针 | 0 | `/openapi.yaml` 与 `server/openapi/openapi.yaml` 均为 `74dbd5c535cbaa6350ebe462892bf3eabd1010083b7d0284c9c2d8a295df27f6`。 |

### 验收边界

- 新页面已发布到 `https://payhon.github.io/AppKernia/`。GitHub Pages 配置仍为 workflow 且 `cname=null`；不能把该地址描述为已经绑定 `appkernia.com`。
- 没有新增页面主题、组件或交互，因此未产生新的 UI Skill 或浏览器截图；静态构建、链接和双语检查不等于 Safari/Firefox/移动浏览器视觉验收。
- 文档准确记录现有实现，但没有取得九渠道生产账号、签名包或物理设备证据；不把 Mock、编译、模拟器或厂商受理表述为真实到达。
- 工作树中既有两个未跟踪 Skill 目录和 `output/playwright/ak-news-admin-debug.png` 未修改、未纳入本轮范围。

## 2026-08-29 消息推送运行时、可观测工作台与统一权限中心

### 已交付

- Backend：新增 `jobqueue.Enqueuer`/River Adapter、编译期任务 Registry、任务运行/尝试安全投影、消息流水线、日统计、Worker Middleware、对账/聚合/保留任务和低基数 Prometheus 指标。发布、扇出、投递分别固定到 `notifications` 队列、5 次最大尝试及 30/90/90 秒 Worker 超时；未注册 kind、错误队列或超限尝试在写入 River 前拒绝。
- 通知门面与 M2M：新增 `platform/notification.Service` 的 `Submit/SubmitTx/Status/Cancel`，支持受控双语内联或模板内容、用户/广播收件人、站内基础渠道、可选 Push、计划/过期/TTL/collapse/thread/受控路由和严格幂等。`ak-api` Machine Principal 使用真实 Tenant、API Client subject、短期无 Refresh JWT，并强制 Client 状态、到期、CIDR、权限和 App allowlist。
- Admin：新增 App 级消息运营 8 个 API 与四 Tab 工作台，覆盖摘要/趋势、发布运行、队列任务、失败中心、URL 筛选恢复、可见性/积压驱动轮询、手动刷新、可访问数据表和安全重试。新增 `notify.observability.read`、`notify.task.retry`；旧投递路由保留兼容跳转。
- Mobile：新增 `ak-permissions` Port、稳定状态与编译期能力 Registry，完成 iOS/Android/Harmony 通知权限查询、用户主动申请、通知专属设置优先和通用设置回退；应用权限页联动 Push 总开关、Provider 通道、设备注册和通知偏好。页面加载不申请权限，OS 状态不上传服务端。
- 数据与运维：新增 22—24 号双向迁移、ADR-0022、消息推送架构图、M2M 使用手册、消息运营 Runbook、Mobile 权限中心手册及三端蓝图/权限/i18n/设计系统同步。

### 实际验证

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| PostgreSQL 18 独立临时库：24 组 migration + Core Seed + `go test -tags=integration` | 0 | 5 条实际数据库通知流程及相关单元/重试子用例通过；覆盖 `SubmitTx`、幂等复用/冲突、tenant+app 隔离、事务回滚、取消、River task projection、OTP/密码重置加密入队，以及提交→发布→扇出→Mock 受理→opened→运营汇总；临时库已删除。 |
| `make check && make build`（server） | 0 | gofmt、vet、全包测试及 API/Worker/CLI 三个二进制构建通过。 |
| `go test -count=1 -json ./...` 统计 | 0 | 214 项 test pass events、43 个通过包、0 failed。 |
| `go test -race` 高风险范围 | 0 | JobQueue、NotificationService、Machine Auth、通知 Repository、通知 Worker 共 5 个包通过。 |
| `npm run check`（Admin） | 0 | OpenAPI/i18n/routes 生成、360 个接口标题本地化、lint、strict typecheck、44 个 Vitest 文件/168 项测试、production build、bundle、OpenAPI docs 和 Admin Blueprint 全通过。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | 45 routes、4 tabs、58 API delta、11 permission delta、40 components、11 privacy capabilities、26 tasks、3 platforms；i18n/client/startup snapshot/upgrade/Node tests 通过。 |
| Backend/Admin/Mobile/i18n validators + `git diff --check` | 0 | 24 up/down migrations、109 张总表、165 indexes、66 triggers、244 FK references；48 menus、59 routes、163 permissions、215 existing APIs + 13 deltas及双语契约均通过。 |
| HBuilderX 5.24 Android/iOS/Harmony 构建 | 0 / 0 / 0 | 本轮前序最终态 36 页面三端源码编译通过；Harmony 完成依赖与未签名 HAP，未配置数字证书。 |
| `docker compose -p appkernia-news-demo up -d --build seed api worker admin` | 0 | Node 24.18.1 Admin 与 Go 1.26.5 API/Worker 镜像完成；PostgreSQL/API/Admin healthy、Worker running，数据库 `24|false`。 |
| 本地 HTTP/数据库安全探针 | 0 | health、运营页面、OpenAPI 为 200；未认证运营 API/M2M 提交为 401；6 项权限和 5 张运行时表存在，近 5 分钟 API/Worker 日志无 panic/fatal/error。 |

本地最终镜像摘要：Admin `sha256:5b43d2387450...`、API `sha256:346c34c2bc9a...`、Worker `sha256:f20b85733a86...`。升级前备份为 `tmp/local-backups/appkernia-news-demo-before-notification-runtime-v24-20260829.dump`（637,381 B，SHA-256 `11636715525c0f12307166fd60dfc2b39b14a2c06156645cb76776c309fe1e3e`）；24→23→24 往返和唯一统计触发器此前已通过。

验证过程中的失败已如实处理：临时库首轮错误使用 `AK_ENV=test`，配置按设计因缺少外部 JWT 私钥退出 1，改为隔离的 development 临时库；随后真实集成测试暴露旧 audience 列写入、空 `push_route_params` 及旧 pending 断言，修复后从干净临时库完整重跑通过。Admin 首轮曾暴露新增 OpenAPI 分组/中文标题缺失及 42 个 lint 问题，修复后完整 `npm run check` 通过；Mobile 首轮 Android/iOS 分别暴露 UTS 类型冲突和 Swift 参数标签问题，修复后各平台重跑通过。失败结果未写成成功。

### 未完成的外部门禁与风险

- 当前浏览器停在 `http://127.0.0.1:4174/login?redirect=%2Fsystem%2Fnotifications%2Foperations`。没有读取、输入或重置用户凭据，因此消息运营和应用权限页的登录态双语、1440/768/375、暗色、axe、键盘/读屏截图尚未完成。
- 未取得九渠道生产账号/凭据、生产签名包、TestFlight 或厂商物理设备；没有执行真实 APNs/FCM/国内厂商/Harmony 到达、权限恢复和点击矩阵。Mock、源码编译和未签名 HAP 不构成生产验收。
- 本轮没有 commit、push 或生产部署；当前本地 `4174` 栈已更新，所有既有无关工作树修改保持原状。

## 2026-08-29 多厂商离线消息推送

### 已交付

- Backend：在既有 `notify.messages/recipients/deliveries/push_devices` 上完成 21 号迁移、九厂商稳定枚举、加密 Provider 配置和 Token、Mobile/Admin API、权限与审计、发布/定时/扇出/投递 River Worker、全局 Kill Switch、用户分类偏好及受理/失效/打开统计。APNs 使用 HTTP/2 + ES256 短期 JWT，FCM 使用 HTTP v1 + OAuth2，其余渠道使用各厂商 REST；鉴权故障会 fault 配置，写出后结果未知不自动重放。内部 Prometheus 暴露按 App/Provider/Category/Result 的投递与打开计数、队列等待 sum/count、积压及故障配置，不使用 Token/User 标签。
- Mobile：新增统一 `ak-push` UTS Port、iOS APNs 生命周期、Android FCM/华为/荣耀/小米/OPPO/vivo/魅族唯一通道选择、Harmony Push Kit、Token 更新注册、系统设置恢复、前台事件、冷/热启动点击、受控路由和 opened 回传。设置页新增 Push 总开关、服务安全及资讯运营偏好，先过隐私同意与 OS 权限，失败会回滚且不影响站内消息。
- 构建：新增 `AK_ANDROID_PUSH_VARIANT=google|china` 互斥生成器、精确版本门禁、Firebase 自动初始化禁用、China/GMS 边界和 APK/AAB marker 扫描；公开构建字段与服务端 Secret 分离。当前仓库默认仍为 `disabled` development 变体，避免无真实配置时误打生产包。
- Admin：新增应用级“推送渠道”页面、强类型 Provider 表单、64 KiB write-only 凭据轮换、指纹、预检、启停、已注册设备测试和准确结果统计；同步 6 个权限、OpenAPI/生成 Client、双语 Catalog、设计系统及 UI Skill 产物。没有提供原始 Token 输入或任意 JSON 编辑器。
- 文档：新增 ADR-0021、厂商 SDK/许可证/隐私清单、构建和生产发布手册、三端蓝图及机器可读契约。DCloud 原生插件需要自定义基座或云打包，厂商版本/隐私信息必须在获得正式 SDK 后补齐并固定。

### 实际验证

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| PostgreSQL 18 临时库：`migrate up` → `down 1` → `up` | 0 | 空库双向迁移通过；`notify.push_provider_configs` 存在、6 个新权限存在、最终 `21:false`。 |
| 最终 PostgreSQL 18 原生 SQL：`000001..000021 up` → `000021 down` → `000021 up` | 0 | 回滚后 `notify.push_provider_configs` 不存在，再升级后存在；六个 Push 权限精确计数为 6。临时容器已删除。 |
| PostgreSQL 20 fixture：旧 `hms` + 同 App/设备双活 → 21 → 20 → 21 | 0 | `hms` 转为 `huawei_android/android/android_china`；较旧绑定停用并标记 `migration_replaced_duplicate`；down 恢复 `hms`，再 up 后 `version=21 dirty=false`。临时数据库已删除。 |
| `make -C server sqlc-generate` | 0 | 21 号 Schema 的 Provider 配置、Push Device、Delivery、Message/Template 字段同步到 sqlc models。 |
| `make -C server check` | 0 | gofmt、go vet、后端全包默认测试通过。 |
| `go test -count=1 -json ./... \| jq -s ...` | 0 | 191 个 test pass events、39 个通过包、0 failed；包含 Provider 响应/重试分类、unknown-after-write、APNs 连接隔离、Prometheus 标签脱敏、发布取消及扇出状态测试。 |
| `pnpm --dir apps/ak-admin check` | 0 | OpenAPI/i18n/routes 生成、reference、lint、TypeScript strict、40 个 Vitest 文件/160 项测试、Vite production build、bundle、OpenAPI docs 和 Admin Blueprint 全通过。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile Blueprint、双语 key/placeholder、生成 API Client、Push/Upgrade 静态门禁通过；Node 专项 4/4。 |
| `apps/ak-mobile/scripts/build-platform.sh android` | 0 | HBuilderX 5.24、35 页面、development `disabled` 变体源码编译通过；不是 Google/China 厂商 SDK 产物。 |
| `AK_REQUIRE_PUSH_CONFIG=1 AK_ANDROID_PUSH_VARIANT=disabled ...` 反向门禁及默认配置恢复 | 0（门禁按预期拒绝） | 生产模式拒绝 disabled；恢复后的配置只有 `minSdkVersion=26`，摘要为 disabled 且声明不含服务端 Secret。 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.24、35 页面及 `ak-push` iOS 13+ 源码编译通过；无 APNs 签名/TestFlight/真机。 |
| `apps/ak-mobile/scripts/build-platform.sh harmony` | 0 | HBuilderX 5.24、35 页面编译及未签名 `.hap` 制作成功；工具明确提示未配置数字证书。 |
| Mobile/Admin Blueprint、统一 i18n、`git diff --check` | 0 | 44 Mobile routes、57 API delta、48 Admin menus、58 Admin routes、158 permissions、206 APIs + 13 deltas及双语契约通过；补丁格式通过。 |
| `python3 blueprint/backend/tools/validate_blueprint.py` | 0 | 21 对 up/down migration、104 张表、156 个索引、63 个触发器、229 个外键引用，0 error / 0 warning。 |

验证过程中的失败已如实处理：首次 Go JSON 统计因漏写 `jq -s` 退出 5，修正后重新执行并得到上表统计；迁移 fixture 首两次插入因遗漏既有设备/成员外键退出 3，补齐完整 Tenant/App/User/Member/Device 事实链后从干净的 20 号状态执行并通过；最终审计曾误用不存在的 Backend validator 旧路径而退出 2，改用仓库真实 `blueprint/backend/tools/validate_blueprint.py` 后发现 down migration 缺少静态门禁要求的 `IF EXISTS`，补齐后静态与 PostgreSQL 18 往返均通过。首个最终权限探针用 `LIKE 'notify.push%'`，不会命中 `notify.operations.publish`，因此组合布尔值为 false；改成六项精确集合后计数为 6。上述诊断步骤均未被写成成功。

### 未完成的外部门禁与风险

- 未获得 APNs/FCM/华为 Android/荣耀/小米/OPPO/vivo/魅族/Harmony 的生产账号、证书、权益或频控授权；Provider 预检当前证明配置结构、密钥可解密及可解析，不能替代厂商控制台与真实外网鉴权验收。
- 未生成真实 `android_google` / `android_china` APK/AAB，因而尚未对最终安装包执行互斥 SDK marker 扫描；环境注入的国内 SDK 坐标、校验和、许可证与隐私清单必须在正式接入账号后锁定。
- 未执行任何渠道的物理设备前台/后台/被终止/离线恢复、权限拒绝与设置恢复、重装/升级/Token 刷新、双语点击跳转；iOS 编译和 Harmony 未签名 HAP 不能替代这些验收。
- 推送渠道 Admin 页和 Mobile 设置页尚无本轮登录态双语截图、375px 长英文、axe、键盘或读屏证据；对应 Skill screenshot index 明确登记为未取证，不伪造完成状态。
- Admin 全量检查运行在 Node 26.5.0，仓库声明范围为 `>=24 <25`；所有门禁退出 0，但正式 CI/发布仍应使用 Node 24。没有 commit、push、生产部署或真实厂商请求，原有无关工作树修改全部保留。

### 本地后台环境更新

- 已在既有 `appkernia-news-demo` Compose 项目内更新 `migrate`、`seed`、`api`、`worker`、`admin`，保留 PostgreSQL 与对象存储数据卷；未改动独立运行并占用宿主机 `8080` 的 `appkernia-news-demo-api-host` 容器。
- 升级前使用 `pg_dump -Fc` 生成 `tmp/local-backups/appkernia-news-demo-before-push-v21-20260829.dump`，大小 584 KiB，SHA-256 为 `7226bce4d2b3f09bc7ecc2e280eccd2702a31501c91b2606f695b13044feb937`。
- `AK_ADMIN_PORT=4174 docker compose -p appkernia-news-demo up -d --build postgres migrate seed api worker admin` 退出 0；Node 24.18.1 Admin production build与 Go 1.26.5 API/CLI/Worker 镜像构建成功。
- 运行态为 PostgreSQL/API/Admin healthy、Worker running、migrate/seed Exited (0)；应用迁移为 `21:false`，新表存在，六个 Push 权限均已 Seed，`super-admin` 映射数量为 6。
- `http://127.0.0.1:4174/healthz`、Admin public config、`/system/notifications/push-channels` 与 `/openapi/` 均为 HTTP 200；未登录读取 Push Provider 配置返回预期 HTTP 401。Admin/API/Worker 镜像摘要分别为 `ee913e0c...`、`4cbdc271...`、`29caeb21...`。

## 2026-08-29 分享配置管理与 App 绑定

本轮完成租户级可复用分享配置、App/Provider 唯一绑定、三端身份预检、最小化公开运行配置、Admin 管理页与 App 绑定 Drawer、构建导出 CLI，以及 Mobile 微信 Provider 检测和系统分享降级。微信 Provider 为代码注册能力，后台不能注入任意 Provider 或动态代码；微信 AppSecret 不进入本系统。

| 命令/验收 | 退出码 | 结果 |
|---|---:|---|
| PostgreSQL 18 `000020 up → down → up` | 0 | 两张表、7 个权限和菜单可双向迁移；最终 `schema_migrations=20, dirty=false` |
| PostgreSQL 回滚事务约束探针 | 0 | 同 App/Provider 重复、跨租户配置、非法场景及非 HTTPS 落地域名均被拒绝，事务最终回滚 |
| `go test ./...`（`server`） | 0 | Backend、CLI 与全仓 Go 测试通过；分享模块覆盖严格配置、危险 URL、预检最小化和导出幂等 |
| `go test ./cmd/ak-cli ./internal/modules/shareconfig/...` | 0 | 4 个 CLI/3 个应用层专项测试通过；保留无关 Manifest Provider 与 Entitlements，身份不匹配失败 |
| 本地数据库 fixture `app-share export` + `--check` | 0 / 0 | 临时目录生成 Android/iOS/Harmony `uni-share.weixin` 与 `applinks:share.example.com`；漂移检查通过，fixture 随后从数据库清理 |
| `pnpm check`（`apps/ak-admin`）及分享菜单回归 | 0 | 36 个 Vitest 文件/149 项测试、OpenAPI 生成、双语 Catalog、57 路由、lint、typecheck、production build、bundle、文档和 Admin 蓝图通过；新增测试锁定分享配置路由权限与可见性及 App 操作菜单图标/权限规则 |
| `./scripts/check-project.sh`（Mobile） | 0 | 运行配置 DTO、生成客户端、三端 Provider 检测与系统降级接线通过静态门禁 |
| Backend/Admin/Mobile/i18n validators | 0 | 20 组迁移、47 菜单、57 路由、152 权限、77 Schema、333 OpenAPI operation 和双语 parity 通过 |
| HBuilderX 5.24 Android/iOS/Harmony 编译 | 0 / 0 / 0 | 35 页面三端源码编译成功；Harmony 依赖安装和未签名 HAP 制作成功，未配置数字证书 |
| Chromium UI fixture + axe | 0 | 1440/768/375、zh-CN/en-US 共 5 张截图；4 个 axe 范围 serious/critical=0，console error=0 |
| 本地 Admin 菜单可见性修复与部署 | 0 | 修复 `system.settings.share-configs` 客户端已实现路由白名单遗漏；数据库菜单、父级链和 `super-admin` 授权均有效，新生产包已同步至 `4174`，容器 healthy，目标路由 HTTP 200 |
| App 管理操作下拉菜单 Chromium 验收 | 0 | 操作列实测 112px；当前权限下 5 个菜单项全部带图标，axe serious/critical 0，console error 0；编辑、升级、内容、分享配置与停用入口均可见 |
| 分享配置页容器间距 Chromium 验收 | 0 | 页面根节点切换为共享 `.ak-page-container`；1440px 下左右 padding 与右侧实际留白均为 48px，375px 下左右 padding 均为 16px，axe serious/critical 0，console error 0 |

编译补充：Android 首轮被既有资讯 Markdown 图片正则捕获的 `String?` 类型错误阻断；将捕获组显式收窄后，Android、iOS、Harmony 最终均退出 0。外部边界：当前仓库没有已审核微信开放平台 AppID、Android 发布签名、iOS Universal Link/Associated Domains、Harmony Bundle Name 对应的正式配置，因此只用本地数据库 fixture 验证了导出/漂移，没有把测试 AppID 写入项目 Manifest，也没有执行带微信 Provider 的三端自定义基座编译或物理设备三场景分享。本轮不把 fixture、普通源码编译或系统分享降级表述为微信直分享验收；正式配置后台保存后仍必须重新导出和打包。

## 2026-08-28 “应核 AppKernia”资讯 App 一期功能闭环

本轮完成 Backend、Admin、Mobile、OpenAPI、迁移、i18n、设计系统和发布文档的同一契约闭环：三种资讯、两级分类、多分类、专题、标签、搜索、收藏、首页聚合、评论先审、举报/拉黑、统一登录 Sheet、四 Tab、三类详情和跨平台分享。附件只作为信息层级参考，未复制 Apple 品牌或素材。

| 命令/验收 | 退出码 | 结果 |
|---|---:|---|
| `make check`（`server`） | 0 | gofmt、Go vet、全量 Go 测试通过；内容模块与 bootstrap 编译通过 |
| PostgreSQL 18.6 新库顺序执行 `000001`—`000019` | 0 | 19 个 up migration；专题、举报、视频白名单、敏感词与 trigram 索引存在 |
| PostgreSQL 18→19 旧文章升级 + `000019.down.sql` | 0 | 旧文章保留并默认为 `article`、分类关系回填；down 后旧文章仍在且一期新表移除 |
| `pnpm run check`（`apps/ak-admin`） | 0 | 30 个 Vitest 文件、131 tests；OpenAPI 321 操作映射、lint、typecheck、production build、bundle、文档与 Admin 蓝图通过 |
| `bash scripts/check-project.sh`（Mobile） | 0 | 43 routes、4 tabs、57 API delta、34 components；i18n/catalog/client/启动快照与静态测试通过 |
| Backend/Admin/Mobile/i18n validators | 0 | Backend 19 up/down、Admin 56 routes/145 permissions、Mobile 与双语契约通过 |
| `bash scripts/build-platform.sh android` | 0 | HBuilderX 5.24，34 页面 Android class 最终源码编译成功，最终态 `ready in 28409ms` |
| `bash scripts/build-platform.sh ios` | 0 | HBuilderX 5.24，34 页面 iOS 最终源码编译成功，最终态 `ready in 29723ms` |
| `bash scripts/build-platform.sh harmony` | 0 | HBuilderX `ready in 28413ms`；34 页面编译、DevEco 6.0.2.640 依赖和未签名 `.hap` 制作成功；未配置数字证书 |
| `docker compose -p appkernia-news-demo up -d --build ...` + migration/seed/bootstrap | 0 | PostgreSQL 18、API、Worker、Admin 均 healthy；数据库 version=19、dirty=false，Admin 对外地址为 `http://127.0.0.1:4174` |
| 真实 Admin/Mobile API 演示数据脚本 | 0 | 发布 article/gallery/video 各 1 篇，创建两级分类、1 个专题、3 个标签；收藏、评论创建、待审、后台通过、公开读取闭环通过 |
| 公开 API 与分享页 curl 探针 | 0 | 首页去重、列表、搜索、分类、专题、三类详情、公开媒体、评论和 `/s/{slug}` 均返回预期状态；公开接口不要求登录，失效 Bearer 返回 401 |
| Python Playwright + Chromium | 0 | Admin 资讯/分类/评论在 1440、资讯列表在 768/375、公开分享页在 390 完成截图；7 个内容 API 为 200，console error=0 |
| HBuilderX 5.24 iOS 自定义基座云制作、安装与同步 | 0 | `com.appkernia.mobile` 安装到 iPhone 16 Pro / iOS 18.6；最终基座依赖同时包含 `uni-video`、`uni-loading`，34 页面同步并启动成功 |
| Maestro 2.3.0 iOS 模拟器流程 | 0 | 四 Tab、游客认证 Sheet、登录/收藏、搜索/文章/分享、评论读取/提交、图文/视频类型筛选与详情、视频进度及暂停流程完成；10 份有效 JUnit 均 failures=0 |
| `go test -race ./... -count=1` | 0 | Backend 全包 race 回归通过 |
| `pnpm --dir apps/ak-admin run check` | 0 | 30 个 Vitest 文件、131 tests、lint/typecheck/build、OpenAPI/蓝图和 bundle 门禁通过；宿主 Node 26 仅产生 `>=24 <25` engine warning |
| Mobile project check + Backend/Mobile/i18n validators + `git diff --check` | 0 | 43 routes、4 tabs、57 API delta、34 components、19 migrations及双语契约通过；工作树补丁格式通过 |

补充收口：草稿允许 `zh-CN`、`en-US` 独立未完成保存；发布会重新读取当前锁版本，并在同一数据库事务中复核双语完整性、活动分类/专题/标签、媒体扫描状态、500 MiB MP4 限制和外链白名单。Admin 发布前会定位缺失语言字段；正文插图只保存受控文件 ID，Mobile 通过公开媒体投影映射为原生图片节点。

环境偏差：Admin 仓库要求 Node `>=24 <25`，本轮宿主为 Node 26.5.0 / pnpm 11.18.0，完整命令虽退出 0，但正式 CI 应继续使用 Node 24。Playwright Skill wrapper 不可用，本轮使用本机 Python Playwright 驱动真实 Chromium；Computer Use 无法稳定取得模拟器窗口，移动端改用本机 Maestro 2.3.0 运行真实 iOS 模拟器流程。

运行补充：初始 Google 示例 MP4 对 Range 请求已返回 403，导致播放器持续 loading；演示数据通过真实 Admin PATCH 改为 App 白名单内、Range 206 的 W3C HTTPS MP4。iOS 模拟器随后取得开始播放、等待 10 秒后不同视频帧、原地暂停三段证据，HBuilderX 日志无 `uni-video`/`uni-loading` 缺模块错误。未执行 Android/Harmony 资讯运行时、三端物理设备、动态字号和读屏验收；微信 AppID、Android 签名、iOS Universal Link 未配置，微信三场景直分享、签名包、商店上传与审核均未执行。本轮未 commit、未 push；本地演示栈保持运行。

## 2026-08-10 本地管理员与安全 Seed 引导

- 在当前 `appkernia-acceptance` 数据库既有 `local` 租户中创建 `admin@appkernia.local`，赋予 active `super-admin` 关系；密码只通过终端 stdin 进入 Argon2id 哈希流程，没有写入 Git、命令参数、环境变量、日志或本文档。
- 修复 `bootstrap-admin` 遇到既有 tenant code 时直接违反 `uq_tenants_code` 的问题。身份、凭据、成员、角色、权限、菜单和租户配置现由同一 Serializable 事务编排；重复 bootstrap 复用 user/tenant 且保持原密码。
- `seed core` 支持可选的 development-only 管理员初始化：`AK_SEED_ADMIN_EMAIL` 默认 `admin@appkernia.local`，密码必须来自 `AK_SEED_ADMIN_PASSWORD_FILE`；生产环境 fail closed，不内置固定密码。`.secrets/` 已加入 Git/Docker ignore。

| 命令/验收 | 退出码 | 结果 |
|---|---:|---|
| `make -C server sqlc-generate` | 0 | 新增 active tenant member 查询并同步 sqlc 生成代码 |
| `go test ./cmd/ak-cli ./internal/seed -count=1` | 0 | CLI 密码文件边界与 Seed 单元测试通过 |
| `go test -tags=integration ./internal/seed -run ... -count=1 -v` | 0 | 3/3 PostgreSQL 18 集成测试通过：既有租户复用、重复执行保留凭据、跨租户同邮箱 fail closed、模块目录幂等 |
| 交互式 `bootstrap-admin` | 0 | 本地管理员创建成功；未输出密码 |
| 完整 `seed core`，密码文件使用 `/dev/stdin` | 0 | `development_admin=true`；144 permissions、46 menus、8 modules、3,663 regions、66 dictionaries 同步成功 |
| 两次真实 `POST /admin-api/v1/auth/login` | 0 | 创建后及重复 Seed 后均 HTTP 200、`code=OK`、Access Token 存在；Token 值未输出 |
| `make -C server check` + `go test -json ./...` 统计 | 0 | gofmt、go vet、后端全量默认测试通过；133 tests passed / 0 failed |
| `go test -race ./cmd/ak-cli ./internal/seed -count=1` | 0 | CLI 与 Seed 专项 race 通过 |
| Backend/Admin/Mobile blueprint + i18n validators | 0 | Backend 17 组迁移静态校验及三端契约均通过 |
| `git diff --check` | 0 | 当前混合工作树无补丁格式错误 |

验证边界：这是本机 development 数据库与源码 API 的真实登录闭环，不代表生产账号、生产部署或外部身份系统验收；本轮没有修改 Migration、OpenAPI、权限码或 Admin UI。

## 2026-08-08 Mobile Apple UI refresh

- 使用仓库 `ui-ux-pro-max`，并从 skills.sh 安装/读取框架无关的 `ios-hig-design`，形成 Apple HIG 启发但不复制 Apple 资产的视觉方向；request、skill output、decisions、checklist 与截图已保存。
- 完成全局安全区、语义色/间距/圆角、AK UI 按钮/卡片/表单/状态/模态/开关、原创 TabBar 图标，以及 28 个 Mobile 页面家族的统一刷新。
- iPhone 16 Pro 模拟器验证登录、注册、找回、隐私政策返回、Home、Notifications、Articles、Profile、Language；`zh-CN` / `en-US` 首页和原生 TabBar 均有最终截图。
- 运行时登录曾稳定返回 422；只读诊断确认 iOS UTS bridge 把设备 UUID 变成对象字符串。最终实现改为运行时内存 UUID + 安全会话恢复，并对请求头做 JSON primitive materialization。真实首次登录与重启 refresh 均成功，测试账号保持登录。

| 命令/验收 | 退出码 | 结果 |
|---|---:|---|
| `bash apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile blueprint、i18n、目录与新增 UI/UUID 防回归门禁通过 |
| `bash apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.06，28 页面，UTS 编译完成；无 UTS/UCSS error |
| `bash apps/ak-mobile/scripts/build-platform.sh android` | 0 | 28 页面 Android class 编译完成；过程中修复 2 处 Android UTS 强类型差异后重跑通过 |
| `bash apps/ak-mobile/scripts/build-platform.sh harmony` | 0 | 28 页面 UTS 编译、鸿蒙工程依赖与未签名 `.hap` 制作成功；未配置包名/证书 |
| `git diff --check` | 0 | 当前混合工作树补丁格式通过 |
| iPhone 16 Pro / iOS 18.6 模拟器 | 0 | 双语 Home/TabBar、登录与重启 refresh、返回链路、页面家族状态通过 |

未执行：Android/Harmony 安装运行、三端物理设备、签名/发布、暗色模式、动态字体和 VoiceOver；不将三端编译或 iOS 模拟器结果表述为这些验收已通过。

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

## 2026-08-07 iOS 模拟器国际化白屏修复

### 交付内容

- 删除 `ak-i18n.uts` 对导入 JSON 执行 `as UTSJSONObject` 后调用 `toMap()/getString()` 的启动级路径，改用生成的 `Array<string>` 条目构造 `Map<string, string>`，避开 HBuilderX 5.06 iOS app-service 的普通对象/UTSJSONObject 运行时差异。
- 新增 `scripts/check-i18n-catalogs.py` 与生成 Catalog；`check-project.sh` 和 Mobile framework verifier 会拒绝语言包漂移，以及再次在 i18n 启动路径引入 `.toMap()`。
- 该变更不修改可见文案、布局、交互、API、数据库或权限；因此不创建新的 `ui-ux-pro-max` UI 设计产物。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 apps/ak-mobile/scripts/check-i18n-catalogs.py` | 0 | 119 组 `zh-CN` / `en-US` 词条与生成 UTS Catalog 完全一致 |
| `python3 apps/ak-mobile/scripts/verify-mobile-framework.py` | 0 | Mobile framework 源级契约通过，i18n 启动路径不再调用 `.toMap()` |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile Blueprint、统一 i18n、生成 Catalog、VDOM、禁用模式扫描和三平台脚本目标检查通过 |
| `python3 apps/ak-mobile/scripts/verify-secure-storage-contract.py` | 0 | 三端安全存储源级契约及无普通 Storage fallback 检查通过 |
| `python3 apps/ak-mobile/scripts/test_refresh_policy.py` | 0 | Refresh fake HttpPort 策略回归通过 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.06 编译 20 页面与 `ak-secure-storage`，`UTS编译完毕`，`ready in 44138ms`，compile-only 正常停止 |
| HBuilderX 5.06 `launch app-ios`，iOS 18.6 / iPhone 16 Pro 模拟器 | 启动成功；取证后手动结束 CLI 日志会话 | UTS `ready in 48518ms`；标准基座重签、安装、同步和启动成功；应用进入中文登录页，无原 `source.toMap` TypeError |
| 模拟器 `log show` 与 `unpackage` 精确搜索 | 0 | `catalogFromJson`、`source.toMap`、TypeError、fatal/exception 均无匹配 |
| `git diff --check` | 0 | 补丁格式通过 |

### 截图索引与验证边界

- [iOS 18.6 / iPhone 16 Pro 中文登录页](../apps/ak-mobile/artifacts/runtime/AKMOB-ios-white-screen-fix/screenshots/ios-iphone16pro-login-zh-CN.png)，1206×2622，SHA-256 `3446299ae2bfa19c9e807fb5e2e4c076f8288f1ca1a76aed2ca754fb37e8607a`。
- 标准基座的 `ak-secure-storage` 原生配置/第三方 SDK 不生效提示仍然存在，需自定义基座才能验证 Keychain；本轮没有把该提示当错误，也没有声明安全存储已通过运行时验收。
- 本轮未执行 Android/Harmony 安装运行、iOS 真机、Release/签名、Keychain 回读、英文切换、动态字体或完整自动化流程；模拟器通过不替代三端真机验收。该次取证阶段尚未 commit、push。

## 2026-08-07 App 管理与移动认证/法律内容交付报告

### 交付范围

- Backend：新增 App 领域迁移和 App membership，完成 App 范围的内容、页面 revision/法律同意、通知/推送偏好、移动版本、session、登录/安全事件；OpenAPI、sqlc、权限/菜单种子与 Admin/Mobile 契约同步。旧移动版本复制到每个现有 App，回滚采用确定性折叠策略。
- Admin：新增“App 管理”一级菜单及应用、用户、内容二级页面；内容包括文章和单页内容。应用/用户/单页列表实现服务端筛选、分页与 total，操作使用乐观锁并返回最新 DTO。
- Mobile：补齐注册、邮箱验证、找回/重置密码、用户协议、隐私政策入口及 App 配置/认证网络边界；请求携带 `X-AppID`，用户协议、隐私政策、关于我们由 App 单页内容读取。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make sqlc-generate`（`server`） | 0 | App、mobile profile、内容和 IAM 查询生成成功。 |
| `GOTOOLCHAIN=go1.26.5 go test ./...`（`server`） | 0 | 全量 Go 测试通过，覆盖 App 管理、内容、IAM、移动 profile 回归。 |
| Node 24 Admin typecheck / lint / test / build / check | 0 | 严格类型、ESLint、24 个测试文件/91 项测试、生产构建及 Admin check 通过。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile 静态项目检查、i18n/VDOM/禁用模式和脚本目标检查通过。 |
| Backend/Admin/Mobile 蓝图与 i18n validator | 0 | 三端蓝图静态契约及 `zh-CN`/`en-US` 键、占位符一致性通过。 |
| OpenAPI YAML/参数覆盖断言、`git diff --check` | 0 | 207 paths 可解析；Mobile `/api/v1` operation 声明 AppID；补丁格式通过。 |

### 设计证据

- Admin：[request](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/request.md)、[skill output](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/skill-output.md)、[decisions](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/decisions.md)、[review checklist](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/review-checklist.md)、[screenshot index](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/screenshot-index.md)。
- Mobile：[request](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/request.md)、[skill output](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/review-checklist.md)、[screenshot index](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/screenshots/INDEX.md)。

### 未完成项、阻塞与风险

- Docker daemon 无响应，PostgreSQL 18 migration `up → down → up` 未实际运行；静态检查和 Go 测试不能替代数据库验收。
- 本次扩展后的 HBuilder X iOS 构建只推进到 28 页，随后卡在 `ak-secure-storage`；没有 UTS 完成、安装或运行结果，因此新增注册、找回和法律页未做模拟器验收。此前 20 页面白屏修复的 iOS 模拟器证据保留在上一节。
- 未捕获 Admin 真实登录浏览器截图；Admin 构建/测试通过不代表真实登录态视觉验收。
- OTP 邮件外部通道没有真实凭据，没有发送邮件或进行验证码接收端到端验收。未部署生产；Android/iOS/Harmony 真机、Firefox/Safari 和真实第三方邮件服务均未验证。

## 2026-08-08 App 管理本地验收修复报告

### 修复内容

- Backend：修正通知管理 PostgreSQL integration fixture，确保第二位用户同时属于测试 tenant 和当前 App，不放宽生产 App/tenant 过滤。
- Admin：容忍系统预留单页在首个 revision 前没有 translations；补齐双语编辑初值和安全标题回退；应用状态翻译键与稳定 API `active` 枚举对齐。
- Mobile：移除 iOS UCSS 不支持的百分比 `min-height`；修复 `uni.request` 响应 header bridge 对象误调用 `getString()` 导致的运行时异常，并增加静态回归门禁。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `GOTOOLCHAIN=go1.26.5 go test -tags=integration ./... -count=1`（`server`，隔离 PostgreSQL 18） | 0 | 全量 integration 包通过；通知管理 application/repository/worker 目标复测均通过。 |
| `make check && make build`（`server`） | 0 | gofmt、go vet、全量 Go 测试及 API/Worker/CLI 三个二进制构建通过。 |
| `pnpm check`（`apps/ak-admin`） | 0 | ESLint、TypeScript strict、24 个测试文件/93 项测试、Vite production build、bundle budget、Admin blueprint 全部通过。 |
| Admin 双语 Chromium 验收 | 0 | 应用/单页 `zh-CN`、`en-US` 共 4 个状态；axe serious/critical=0，console error=0，两个真实 API GET 均为 200。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile blueprint、i18n、Catalog、VDOM、禁用模式、UCSS/header bridge 新门禁及平台脚本目标通过。 |
| HBuilderX 5.06 iOS build | 0 | 28 页面 UTS/UCSS 编译通过，无 compile error。 |
| iPhone 16 Pro / iOS 18.6 模拟器交互 | 通过 | 登录、找回密码、注册表单、隐私政策、用户协议均渲染并可导航；运行日志无应用 UTS/JS exception。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` / `python3 blueprint/scripts/validate_i18n_contract.py` | 0 / 0 | Mobile 规格及 Admin/Mobile 双语键、占位符契约通过。 |
| `git diff --check` | 0 | 最终文本补丁格式通过。 |

### 设计与截图证据

- Admin：[request](../apps/ak-admin/artifacts/ui-ux-pro-max/acceptance-repair/request.md)、[skill output](../apps/ak-admin/artifacts/ui-ux-pro-max/acceptance-repair/skill-output.md)、[decisions](../apps/ak-admin/artifacts/ui-ux-pro-max/acceptance-repair/decisions.md)、[review checklist](../apps/ak-admin/artifacts/ui-ux-pro-max/acceptance-repair/review-checklist.md)、[screenshots](../apps/ak-admin/artifacts/ui-ux-pro-max/acceptance-repair/screenshots/INDEX.md)。
- Mobile：[request](../apps/ak-mobile/artifacts/ui-ux-pro-max/acceptance-repair/request.md)、[skill output](../apps/ak-mobile/artifacts/ui-ux-pro-max/acceptance-repair/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/acceptance-repair/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/acceptance-repair/review-checklist.md)、[screenshots](../apps/ak-mobile/artifacts/ui-ux-pro-max/acceptance-repair/screenshots/INDEX.md)。

### 验证边界和风险

- HBuilderX UI 在完成编译后，本机资源同步阶段仍停滞；模拟器运行使用 HBuilderX 最新生成的 `unpackage/dist/dev/app-ios` 覆盖已安装官方标准基座的数据容器。已证明最终编译资源的渲染和交互，不将其描述为 HBuilderX 一键同步闭环。
- 官方标准基座不会加载 `ak-secure-storage` 自定义原生配置；Keychain/安全存储仍需自定义基座或真机验证。
- 未执行 iOS 真机、Android/HarmonyOS、Release 签名、Firefox/Safari、真实 OTP 邮件收发或生产部署。
- 本轮未 commit、未 push；工作区保留本轮修复与证据，未覆盖其他 Agent 的无关改动。

## 2026-08-08 Mobile 登录与访客返回链路修复报告

### 修复内容

- 登录页移除全宽“忘记密码”次按钮，在主按钮下提供“忘记密码 / 注册账号”双文字链接；注册入口继续由应用发布策略控制。
- 公共返回控件改用原生点击处理；认证、法律页面导航栏避开状态栏，并在没有页面历史时回到登录页。
- 设备键改为持久化 UUID，并在 `uni.request` 的头部字面量中直接携带，避开 iOS UTS bridge 将动态对象值序列化为 `{}`；OpenAPI 同步 UUID 契约。
- 安全存储插件调用使用稳定的 key/value/callback 接口；本地重建 iOS 模拟器基座后完成登录、重启和会话恢复。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile 蓝图、双语 Catalog、登录结构、访客返回兜底、VDOM/UCSS 和安全存储源级门禁通过。 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.06 编译 28 页面及原生模块，UTS/UCSS 无错误。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | Mobile 机器可读规格通过。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US` 键和占位符一致。 |
| iPhone 16 Pro / iOS 18.6 模拟器坐标交互 | 通过 | 登录、忘记密码、注册、隐私政策、用户协议和顶部返回均通过；登录响应 200。 |
| 模拟器重启与会话恢复 | 通过 | 重启后进入已认证 Home，并显示本地测试用户。 |
| `git diff --check` | 0 | 补丁格式通过。 |

### 设计与截图证据

- [request](../apps/ak-mobile/artifacts/ui-ux-pro-max/login-navigation-repair/request.md)、[skill output](../apps/ak-mobile/artifacts/ui-ux-pro-max/login-navigation-repair/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/login-navigation-repair/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/login-navigation-repair/review-checklist.md)、[simulator screenshots](../apps/ak-mobile/artifacts/ui-ux-pro-max/login-navigation-repair/screenshots/INDEX.md)。

### 验证边界和风险

- 测试用户、已发布法律内容和数据库激活仅存在本机隔离验收环境，不是远端或生产账号/数据。
- 官方 HBuilderX 标准基座不会加载项目自定义原生插件。本次把最新 UTS 产物嵌入本地模拟器基座并重新签名安装，安全存储与重启恢复已在该基座通过；后续从 HBuilderX 运行应继续选择包含 `ak-secure-storage` 的自定义基座。
- 未执行 iOS 真机、Android/HarmonyOS 运行、Release 签名、真实邮件 OTP 或生产部署。本轮未 commit、未 push。

## 2026-08-08 AppKernia 开源官网与 Rspress 文档站交付报告

### 交付内容

- 在 `apps/ak-docs` 新建 Rspress 2.0.19 文档站，提供 `zh-CN` / `en-US` 完整镜像路由、响应式首页、明暗主题、本地搜索、SEO metadata、Sitemap、robots、Web Manifest、`llms.txt`、OpenAPI 下载及自定义 404。
- 形成官网、零门槛快速开始、源码开发、Docker、Mobile 开发、项目结构、路线图、故障排查、架构、认证、多租户/权限、i18n、安全、核心 Mobile/Admin API、AK Mobile 组件和社区治理内容，共构建 64 个静态页面。
- 首页写入项目初心、全栈跨端定位、HarmonyOS 支持、真实 Admin/Mobile 界面证据，以及贡献与 GitHub Star 引导；新增 Code of Conduct，并同步根 README、CONTRIBUTING、Makefile、workspace 和 package scripts。
- OpenAPI 的公开 license 元数据改为 MIT；新增 `check-api-docs.mjs`，在每次文档门禁中校验中英文 API 文档引用、接口前缀和协议元数据。构建时自动复制 OpenAPI，避免文档内维护第二份契约。
- 使用 `ui-ux-pro-max` 建立 Master、request、skill output、decisions、review checklist 和截图索引；使用 imagegen 生成不含文字/商标的跨端生态主视觉。Admin 登录图片为本轮真实启动 Vite 后截图，Mobile 图片来自仓库既有 iPhone 16 Pro / iOS 18.6 模拟器证据。
- 新增 `.github/workflows/docs-pages.yml`：main 分支相关路径变更时以 Node 24.18.1 + pnpm 11.18.0 冻结安装，执行完整 Docs check，上传 `apps/ak-docs/doc_build` 并通过 GitHub Pages 官方 Actions 部署。`DEPLOYMENT.md` 记录 Pages Source、`appkernia.com` apex DNS、HTTPS 和 GitHub fallback URL 的一次性设置。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm install --frozen-lockfile` | 0 | 3 个 workspace project 已是最新；pnpm 11.18.0。当前本机 Node 26.5.0 产生项目 Node 24 engine warning。 |
| `pnpm --filter @appkernia/docs check` | 0 | 113 个 API 路径引用通过；RSLint 9 文件 0 error/warning；TypeScript、Prettier、build 全部通过；双语 route parity 通过，Sitemap 生成 64 页。 |
| `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm --filter @appkernia/docs build` | 0 | 模拟默认 project Pages 地址构建成功；HTML 静态资源使用 `/AppKernia/`，Sitemap/LLM 链接使用 `https://payhon.github.io/AppKernia/`。随后恢复默认 custom-domain 根路径构建。 |
| `pnpm check` | 0 | Admin 生成器、ESLint、strict TypeScript、24 文件 / 93 测试、Vite build、bundle budget、Admin blueprint 通过；Docs 全门禁随后再次通过。 |
| Backend / Admin / Mobile blueprint validators | 0 / 0 / 0 | Backend 16 组 migration、74 张表；Admin 45 菜单、55 路由；Mobile 38 路由、33 组件，均为 0 error/warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US`、默认/回退语言和三端 reference packs 契约通过。 |
| GitHub Actions YAML parse | 0 | 本机没有 `actionlint`，使用 Ruby YAML parser 确认语法有效；未运行 GitHub 托管 job。 |
| Rspress production preview + Chromium | 0 | 中英文首页/Quick Start 共 8 张截图；所有页面 HTTP 200、单一 H1、无横向溢出、破图、console error 或失败资源。 |
| axe representative audit | 0 | 首页 375/1440、暗色、英文、Quick Start 中英文、API、暗色组件共 8 个样本，所有 violation 及 serious/critical 均为 0。 |
| `git diff --check` | 0 | 最终文本补丁无空白错误。 |

### 设计与截图证据

- [设计请求与决策](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-001/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-001/review-checklist.md)、[截图索引](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-001/screenshots/INDEX.md)。
- 截图覆盖 `375×812`、`768×1024`、`1024×900`、`1440×900`，包括中文/英文首页、明/暗主题与双语 Quick Start。

### 未完成项、阻塞与风险

- 本轮没有 commit 或 push，未触发远端 GitHub Actions；Pages Source、Custom Domain、DNS A/AAAA/CNAME 及 Enforce HTTPS 都需要仓库所有者在 push 后按 `apps/ak-docs/DEPLOYMENT.md` 完成。因此当前不声明 `appkernia.com` 或 GitHub Pages fallback 已上线。
- `playwright` Skill 的包装脚本与本机 `@playwright/mcp@0.0.79` 命令名不兼容；浏览器取证改用仓库现有 Python Playwright + 真实 Chromium 完成。此差异不影响截图、HTTP、console 或 axe 结果，但没有伪报包装脚本通过。
- Admin 截图证明本轮本地登录界面可渲染，不包含登录后生产数据；Mobile 截图是既有模拟器证据，本轮未重新执行 Android/iOS/Harmony build、安装、真机或发布签名。

## 2026-08-09 GitHub 提交与 Pages 发布报告

### 提交与远端状态

| Commit | 内容 |
|---|---|
| `10f58e2` | `feat(docs): add AppKernia website`，128 个文件，官网、双语内容、MIT/OpenAPI、Pages workflow 与根级接入。 |
| `f2a9534` | `chore(admin): refresh generated artifacts`，单独提交 OpenAPI 生成类型和 bundle budget 证据。 |
| `71f366c` | `ci(docs): update Node 24 actions`，升级 `actions/checkout@v7`、`pnpm/action-setup@v6`。 |
| `31abe86` | `fix(docs): balance hero heading`，修复线上 1440 px 标语末字孤行并同步 UI 决策/检查表。 |

- 四个站点相关提交均直接推送到用户明确授权的 `origin/main`；Pages artifact 的 head SHA 为 `31abe861a19e6c917051450dccb81e1a2f3d4432`。其后只追加本发布报告，根 `docs/` 不在 Pages workflow 的路径触发范围内。
- GitHub Pages 通过 REST API 启用为 `workflow` build type，公开地址为 `https://payhon.github.io/AppKernia/`，`https_enforced=true`。
- 最终 Docs Pages run：`https://github.com/Payhon/AppKernia/actions/runs/31266883357`，build job `93126361063` 用时 40 秒、deploy job `93126433768` 用时 8 秒，结论均为 `success`。

### 线上验收

| 验证 | Exit / 结果 |
|---|---|
| GitHub Pages 首页、中英文、Quick Start、API、Mobile 组件、OpenAPI、Sitemap curl | 7 个 URL 全部 HTTP 200。 |
| Python Playwright + 真实 Chromium | 中文/英文首页、Quick Start、API、Mobile 组件共 5 条路由 HTTP 200；lang 正确、每页一个 H1、无 overflow、broken image、console/page error 或 4xx/5xx resource。 |
| 最终线上首页 1440×900 / 375×812 | 两个 viewport 均 HTTP 200、`titleWrap=balance`、`subtitleWrap=wrap`、一个 H1、无溢出、破图、console error 或失败资源。 |
| `pnpm --filter @appkernia/docs check` | 0；113 个 API 引用、lint、TypeScript、Prettier、双语 parity、64 页 build 和 Sitemap 通过。 |

### 自定义域名边界

- `appkernia.com` 当前没有 A/AAAA 记录，`www.appkernia.com` 没有 CNAME，GitHub Pages 域名验证 TXT 也未配置；本轮没有把可用的 Pages 地址重定向到不可达域名。
- DNS 就绪后需要将 Pages Custom Domain 设置为 `appkernia.com`，等待证书签发并再次验证 HTTPS；现阶段已完成的是 GitHub Pages 默认域名发布，不声明 `appkernia.com` 已上线。
- `playwright` Skill 包装脚本仍因本机 `@playwright/mcp@0.0.79` 不提供 `playwright-cli` 而退出 127；线上验收继续使用已安装 Python Playwright 驱动真实 Chromium，未伪报包装脚本通过。

## 2026-08-09 文档站品牌、实景 Hero 与内容优化交付报告

### 交付内容

- 文档导航、favicon、Apple Touch Icon 与 Web Manifest 全部改用 Admin 品牌资产；移除旧文档 SVG Logo、AI 生态 Hero 和过时登录截图，最终首页生成新的 `1200×630` 社交分享图。
- 新增内部 `HeroShowcase` 类型：一个必需 Admin 对象和两个 Mobile 对象，每项均要求 `src` 与双语 `alt`。桌面使用浏览器主体与双手机交叠，平板改为网格排列，手机仅保留 Admin 与一台手机。
- 修复 Rspress 后加载变量覆盖导致的宽屏偏左：1920px 下形成 1488px 三栏壳与两侧 216px 等距边界；正文 56px 内边距、72ch 行长和中小屏折叠规则已落地。
- 用单层 1px 边框、品牌色顶部标记和稳定 focus/hover 的语义链接卡片替换默认 `features`；补齐首页九类信息区块及中英文五个核心入口页，不虚构 Star 数、用户量、评分或客户背书。
- 新增 Admin 截图脚本、社交图截图脚本和可重复的 Python Playwright production-preview 门禁；`AKDOCS-002` 保存 request、skill output、decisions、review checklist、原始产品截图和响应式截图索引。

### 真实素材与验证边界

- Admin：隔离 `appkernia-acceptance` PostgreSQL/Docker API/Admin，合成 `.example.test` 管理员和验收租户；等待 Dashboard skeleton 消失后截取 1440×900，不含密码、Token、生产数据或个人信息。
- Mobile：`apps/ak-mobile/scripts/build-platform.sh ios` 两次退出 0；HBuilderX 5.06 产物运行于 iPhone 16 Pro / iOS 18.6 模拟器，并通过本地 API 使用合成测试账号登录后截取 Home 与 Profile。
- Mobile 结果只证明本轮 iOS 模拟器编译资源与登录页面，不等同于 iOS 真机、Android、HarmonyOS、Release 签名或上架验收。当前真实素材充足，本轮未调用 imagegen。

### 实际命令与退出码

| 命令 / 阶段                                                                                     |      Exit | 真实结果                                                                                                            |
| ----------------------------------------------------------------------------------------------- | --------: | ------------------------------------------------------------------------------------------------------------------- |
| `apps/ak-mobile/scripts/build-platform.sh ios`                                                  |         0 | HBuilderX 5.06 iOS UTS/UVue 编译完成；执行两次均成功。                                                              |
| `capture-admin-screenshot.py` / Simulator capture                                               |  0 / 通过 | Admin 1440×900 加载态截图与两张 1206×2622 iOS 模拟器登录态截图保存为原始证据。                                      |
| `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm --filter @appkernia/docs check` |         0 | API 文档、RSLint、TypeScript strict、Prettier、双语 parity、64 页 production build 与 Sitemap 全部通过。            |
| `scripts/visual-check.py`                                                                       |         0 | 8 个 Chromium 样本覆盖 375/768/1024/1440/1920、双语、明暗和普通文档页；HTTP/H1/图片/overflow/console/axe 全部通过。 |
| Backend / Admin / Mobile blueprint validators                                                   | 0 / 0 / 0 | 16 组 migration/74 表、45 菜单/55 路由、38 Mobile 路由/33 组件均为 0 error/warning。                                |
| `python3 blueprint/scripts/validate_i18n_contract.py`                                           |         0 | `zh-CN`、`en-US`、默认/回退与三端 reference packs 通过。                                                            |
| `node scripts/check-api-docs.mjs`                                                               |         0 | 113 个文档 API path 引用与 OpenAPI 一致。                                                                           |
| `git diff --check`                                                                              |         0 | 最终文本补丁无空白错误。                                                                                            |

### 设计与截图证据

- [AKDOCS-002 decisions](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-002/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-002/review-checklist.md)、[screenshots](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-002/screenshots/INDEX.md)。
- 1920px 普通文档页实测三栏宽 1488px、左右各 216px；抽样正文 725.625px。8 个 axe 样本 serious/critical 均为 0。

### 发布边界

- `main` 的文档路径变更会触发既有 `Docs Pages` Workflow；远端 run、head SHA、GitHub Pages 页面与静态资源由提交后实查收口。
- `appkernia.com` 只有在 A/AAAA/CNAME、GitHub 域名验证和 Pages Custom Domain 真实生效后才可声明上线；在此之前继续以 `https://payhon.github.io/AppKernia/` 为公开地址。

## 2026-08-09 AKDOCS-003 项目叙事与架构图交付报告

### 交付内容

- 在向导中新增中英文“什么是 AppKernia?”，补全项目由来、开发者初心、品牌命名解释、技术选型、当前边界、未来方向、贡献阶梯与 Star 引导。
- 使用仓库可追溯验收素材增加 4 张 Admin 与 4 张 Mobile 功能截图，覆盖应用、内容、移动发布、存储、文章、通知、登录与个人中心；图片和说明不包含凭据、Token 或生产个人数据。
- 引入固定版本的 Mermaid Rspress 社区插件，将核心契约、受保护请求、总体/三端架构、登录、Refresh Rotation、会话失效、多租户、语言解析、API 与 AK UI 适配流程渲染为可访问 SVG。
- 新增主题级图表、产品画廊和证据边界样式，并用 Layout 包装器让 Rspress 表格和图表的可滚动区域可经键盘聚焦。
- `AKDOCS-003` 保存 Skill request/output/decisions/checklist、22 张验收截图、索引和机器可读 `results.json`。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 tools/validate_blueprint.py`（`blueprint/backend`） | 0 | 16 对 migration、74 张表，0 errors / 0 warnings。 |
| Admin / Mobile blueprint validators | 0 / 0 | Admin 45 菜单/55 路由；Mobile 38 路由/33 组件，0 errors / 0 warnings。 |
| i18n validator / UI Skill check | 0 / 0 | `zh-CN`、`en-US` 和三端 reference pack 一致；Skill 脚本存在。 |
| Node 24.18.1 `pnpm --dir apps/ak-docs check` | 0 | 113 个 API 引用、RSLint 10 文件、TypeScript、Prettier、66 页构建与语言 parity 全部通过。 |
| `apps/ak-docs/scripts/visual-check.py` | 0 | 22 个 Chromium 状态、14 个中英文图表路由、26 张 Mermaid SVG；HTTP/H1/overflow/图片/console/资源/axe 均通过。 |
| GitHub Pages Base Path 静态检查 | 0 | 5 个代表 HTML 与 58 个引用/产品资源存在，未发现误用根路径 `/static`。 |
| `python3 -m py_compile apps/ak-docs/scripts/visual-check.py` / `git diff --check` | 0 / 0 | 验收脚本语法与最终文本补丁通过。 |

### 纠正记录

- 首次 `mise exec -- pnpm ...` 因 mise 尝试联网安装已固定的 pnpm，遇到 GitHub API 403 rate limit 后退出 1；改为把本机已安装的 Node 24.18.1 放入 PATH 并复用 pnpm 11.18.0，最终全量 check 退出 0。
- Node 24 首轮格式检查发现新截图索引与 `results.json` 未按 Prettier 排版，退出 1；格式化后重跑全量 check 退出 0，未隐藏首轮失败。

### 设计、截图与验证边界

- [AKDOCS-003 decisions](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-003/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-003/review-checklist.md)、[screenshots and results](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-003/screenshots/INDEX.md)。
- Admin 图来自仓库记录的本地 API/Admin Chromium 验收；Mobile 图来自仓库记录的 iPhone 16 Pro / iOS 18.6 模拟器验收。本轮没有重新执行 Admin/Mobile 业务运行，不把文档构建或历史素材复用描述为新的应用运行结果。
- Mobile 素材继续受标准基座数据容器刷新边界约束，不代表 HBuilderX 一键同步、自定义基座安全存储、iOS 真机、Android、HarmonyOS、Release 签名或上架验收。

### GitHub Pages 发布收口

- 功能提交 `1f33598a9225f1325bbfeb9b5dd2acb704a9314d`（`feat(docs): enrich architecture and project story`）已推送 `origin/main`；发布前本地与远端 SHA 完全一致。
- [Docs Pages run 31292063763](https://github.com/Payhon/AppKernia/actions/runs/31292063763) 为 `success`：build job `93190822016` 用时 45 秒，deploy job `93190887264` 用时 10 秒，artifact head SHA 为 `1f33598a9225f1325bbfeb9b5dd2acb704a9314d`。
- GitHub Pages API 实查为 `build_type=workflow`、`https_enforced=true`、`cname=null`，公开地址为 `https://payhon.github.io/AppKernia/`。
- curl 实查首页、中英文“什么是 AppKernia?”、中英文架构/认证页、Admin/Mobile 产品图片和 Sitemap 共 8 个 URL，全部 HTTP 200。
- 线上 Python Playwright + Chromium 复核 7 个代表路由：每页单一 H1、无页面级横向溢出、破图、console error 或失败资源；项目介绍各加载 8 张产品图，架构 4 张 SVG、认证 3 张 SVG，其余抽样流程图均按预期渲染；axe serious/critical 为 0。
- `appkernia.com` 当前仍无 A/AAAA，`www.appkernia.com` 无 CNAME，GitHub Pages 验证 TXT 未配置，HTTPS 连接失败；因此本次只声明 GitHub Pages 默认域名上线。

## 2026-08-09 AKDOCS-004 首页内容、产品 Slider 与技术栈交付报告

### 任务目标与交付内容

- 修复首页只有 Hero 的实际渲染问题，并让已编写的完整 MDX 正文进入 Rspress 首页与 Markdown 导出。
- 中英文同步新增 6 张核心能力卡和 9 项技术栈 Logo，包含用户指定的 uni-app x、React 与 Go。
- 使用仓库现有真实截图交付 Admin/Web 与 Mobile 双 Slider；每组 4 张，支持按钮、圆点、键盘左右键和 live region，且不会自动播放。
- 未修改 Go API、OpenAPI、数据库、Admin Client 或 Mobile 公共组件 API。

### 关键实现

- `theme/HomeLayout.tsx` 在保留 Rspress Hero、Footer 与主题扩展点的前提下渲染 `Content`；`SSG_MD` 路径同样输出首页正文。
- `theme/HomeLanding.tsx` 维护双语 Slider 数据、技术栈数据和可访问交互；`index.css` 提供 3/2/1 特性卡、3/2/1 Logo 卡与桌面并排/窄屏堆叠的 Slider 布局。
- 首页 MDX 的 JSX 文本改为显式表达式，最终静态 HTML 中 `heading > p` 和 `p > p` 非法嵌套计数均为 0。
- DCloud 官方 uni-app 图片以本地 SVG 包装资产发布；另外 8 个品牌 mark 来自锁定的 `simple-icons@16.27.0`。来源与非背书说明保存在 `docs/public/tech/README.md`。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm --dir apps/ak-docs check`（首轮） | 1 | RSLint 发现 Slider region 的 2 条 `jsx-a11y` 错误；未隐藏失败。 |
| 同一文档全量 check（修复后） | 0 | 113 个 API 引用、RSLint 12 文件、TypeScript strict、Prettier、66 页 production build、语言 parity 与 Sitemap 全部通过。 |
| Backend / Admin / Mobile blueprint validators | 0 / 0 / 0 | Backend 16 对 migration/74 表；Admin 45 菜单/55 路由；Mobile 38 路由/33 组件，均为 0 error/warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN`、`en-US`、默认/回退和三端 reference pack 通过。 |
| `bash blueprint/admin-frontend/scripts/check_ui_skill.sh` | 0 | 找到并使用仓库 `ui-ux-pro-max` Skill。 |
| `AK_DOCS_SAMPLE=home python3 apps/ak-docs/scripts/visual-check.py` | 0 | 6 个最终 Chromium 状态全部通过；每页 10 区块、6 特性卡、9 Logo 卡、2 Slider。 |
| `python3 -m py_compile apps/ak-docs/scripts/visual-check.py` | 0 | 验收脚本语法通过。 |
| 首轮 Prettier 文件清单 / 去除 `.prettierignore` 后重跑 | 2 / 0 | 首轮仅因 Prettier 无法推断 `.prettierignore` parser 退出 2，其余文件已格式化；去除该非源码文件后完整清单通过。 |
| GitHub Pages Base Path 静态资源核对 / `git diff --check` | 0 / 0 | `/AppKernia/` 引用及本地技术 Logo 资源存在；最终文本补丁无空白错误。 |

### 设计与截图证据

- [AKDOCS-004 decisions](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-004/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-004/review-checklist.md)、[screenshots and results](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-004/screenshots/INDEX.md)。
- 6 个视口状态的 H1、overflow、图片、资源、console 与 axe serious/critical 均通过；Slider 点击、键盘切换、不自动轮播和 2 个 live region 均由脚本断言。

### 验证边界与发布状态

- Admin 图来自仓库已有的本机 Docker/API Chromium 验收；Mobile 图来自已有的 iPhone 16 Pro / iOS 18.6 模拟器验收。本轮没有重新运行两个业务项目。
- 不声称生产部署、iOS 真机、Android 或 HarmonyOS 运行通过；技术 Logo 不代表相关厂商背书。
- 本轮未重新运行 Admin/Mobile 业务项目；发布事实与线上验收记录如下。

### GitHub Pages 发布收口

- `git commit -m "feat(docs): enrich homepage showcase"` 与 `git push origin main` 均退出 0；功能提交 `dcc3a30bfca1268322f6249f60e5fe3b8460cd99` 已推送 `origin/main`，发布时 `git ls-remote` 与本地 SHA 一致。
- [Docs Pages run 31294828773](https://github.com/Payhon/AppKernia/actions/runs/31294828773) 的 `gh run watch --exit-status` 退出 0：build job `93198105500` 用时 45 秒，deploy job `93198175593` 用时 8 秒，artifact head SHA 为 `dcc3a30bfca1268322f6249f60e5fe3b8460cd99`。
- GitHub Pages API 实查为 `build_type=workflow`、`https_enforced=true`、`cname=null`；中英文首页、uni-app Logo、Admin 图片和 Mobile 图片的 curl 请求均退出 0 并返回 HTTP 200。
- 线上 Python Playwright + Chromium 退出 0：`zh-CN` / `en-US` 首页均为单一 H1、10 个首页内容区、6 张特性卡、9 张技术栈卡和 2 个 Slider；点击与键盘切换成功且 1.2 秒内无自动轮播，无页面级横向溢出、破图、console/page error、失败响应或 axe serious/critical 问题。
- 公开地址为 `https://payhon.github.io/AppKernia/`。由于 Pages API 仍为 `cname=null`，本次不声明 `appkernia.com` 已绑定或可访问。

## 2026-08-09 AKDOCS-005 首页作者叙事与横向版式交付报告

### 任务目标与交付内容

- 删除中英文首页 `HONEST MATURITY`，保留对开发者真正有用的选型、上手、能力与贡献信息。
- 从作者身份重写 Hero、初心、三端价值、核心能力、技术栈、源码运行、产品展示、FAQ 和社区 CTA；公开页面不再出现面向项目所有者的验收、证据或交差措辞。
- 以内容优先的 1240px / 12 列连续网格重排首页横向区块，统一浅色白、深色黑、1px 分隔线、单一蓝色强调和无位移交互；这是对 Vercel 通用信息层级原则的借鉴，不复制其视觉资产。
- 保留现有 Admin/Mobile 双 Slider、9 项技术栈、6 张特性卡、路由、搜索、语言切换、暗色模式和 GitHub Pages Base Path。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 tools/validate_blueprint.py`（`blueprint/backend`） | 0 | 16 对 migration、74 张表，0 errors / 0 warnings。 |
| Admin / Mobile blueprint validators | 0 / 0 | Admin 45 菜单/55 路由；Mobile 38 路由/33 组件，0 errors / 0 warnings。 |
| i18n validator / UI Skill check | 0 / 0 | `zh-CN`、`en-US` 与三端 reference pack 一致；仓库 UI Skill 可用且已执行。 |
| `python3 -m py_compile apps/ak-docs/scripts/visual-check.py` | 0 | 首页验收脚本语法通过。 |
| 文档全量 check（首轮） | 1 | 仅新生成的 `AKDOCS-005/screenshots/results.json` 未经过 Prettier；内容、类型与 lint 已通过。 |
| 格式化后 `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm --dir apps/ak-docs check` | 0 | 113 个 API 引用、RSLint 12 文件、TypeScript strict、Prettier、66 页构建、语言 parity 与 Sitemap 全部通过。 |
| `AK_DOCS_SAMPLE=home ... visual-check.py` | 0 | 375/768/1024/1440/1920 中文浅色及 1440 英文深色共 6 个状态通过；9 区块、6 特性卡、9 技术卡、2 Slider、0 maturity。 |

### 纠正记录

- 第一次使用普通 Rspress preview 时，站点未在本机挂载 `/AppKernia`，而构建 HTML 正确引用 Pages Base Path，导致图片请求与 Slider 断言失败；改用只读 Base Path 映射服务器复测后，全部资源、交互与页面断言通过。该问题只属于本地预览挂载方式，没有修改或弱化生产 Base Path。
- 最终视觉结果中 375/768 的能力矩阵保留在自身横向滚动容器内，页面 `scrollWidth` 与 viewport 相等；1024/1440/1920 无溢出元素。
- 首轮线上内联 Chromium 审计脚本重复读取已 `pop` 的文本字段，因 `KeyError: 'text'` 退出 1；修正审计脚本后原样重跑退出 0，未修改站点代码或降低断言。

### 设计证据与发布收口

- [AKDOCS-005 decisions](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-005/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-005/review-checklist.md)、[screenshots and results](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-005/screenshots/INDEX.md)。
- Admin/Mobile 产品图沿用仓库已有验收素材；本轮没有重新运行 Admin、API 或 Mobile 业务环境，也不把文档视觉回归描述为新的终端运行验收。
- `git commit -m "feat(docs): refine homepage voice and layout"` 与 `git push origin main` 均退出 0；功能提交 `f73091635a18ecc6a200279855907157cd8c28dc` 已推送 `origin/main`，发布时本地与远端 SHA 一致。
- [Docs Pages run 31299398867](https://github.com/Payhon/AppKernia/actions/runs/31299398867) 的 `gh run watch --exit-status` 退出 0：build job `93209633691` 用时 45 秒，deploy job `93209706012` 用时 10 秒，head SHA 为 `f73091635a18ecc6a200279855907157cd8c28dc`。
- 发布验收清单更新后，记录提交 `6b1e4bf35ba3c1e3ae6f80650f0c8e7b994d7f54` 触发最终 [Docs Pages run 31299540867](https://github.com/Payhon/AppKernia/actions/runs/31299540867)；build job `93209984665` 用时 50 秒、deploy job `93210062052` 用时 8 秒，均为 success。
- Pages API 为 `build_type=workflow`、`https_enforced=true`、`cname=null`；中英文首页、Admin 图片、uni-app Logo 与 Sitemap 共 5 个 URL 均为 HTTP 200。
- 线上 Python Playwright + Chromium 退出 0：中文 375/1440 与英文深色 1440 均为单一 H1、9 个首页区块、6 张特性卡、9 张技术卡、2 个 Slider、0 个 maturity 区块；交互、overflow、图片、禁用文案、console、请求与 axe serious/critical 全部通过。
- 当前公开地址为 `https://payhon.github.io/AppKernia/`。由于 Pages API 仍为 `cname=null`，不声明 `appkernia.com` 已绑定或可访问。

## 2026-08-10 项目文档治理 Skill 交付报告

### 交付内容

- 新增 `.agents/skills/project-docs-governance/`：`SKILL.md`、Codex UI metadata、可执行初始化脚本、隔离回归脚本，以及新项目/已有项目两份完整提示词。
- 生成框架覆盖 `docs/00-governance` 至 `docs/08-archive`，包含治理原则、工作流、分类层级、命名版本、状态生命周期、七类模板、产品/架构/开发/看板/质量/运维/参考/归档入口。
- 每个工作项使用 `00-feature-spec` 至 `05-release-and-handoff` 六文档闭环；看板包含 Backlog、Ready、In Progress、Review、Blocked、Done。
- `AGENTS.md` 采用 `DOCS_GOVERNANCE` 受管标记块，固化开发前、开发中、开发后、文档新增/更新/归档、DoD、真实性边界和 PR/提交检查项；默认不覆盖已有文件。

### 实际命令与退出码

| 命令 | Exit | 真实结果 |
|---|---:|---|
| `quick_validate.py .agents/skills/project-docs-governance` | 0 | Skill frontmatter、命名和目录结构有效。 |
| `python3 .../test_bootstrap_project_docs.py -v` | 0 | 3/3 隔离回归通过：新项目、已有项目、损坏受管标记。 |
| `python3 .../bootstrap_project_docs.py --help` | 0 | `auto/new/existing`、`dry-run/check/force` 等参数可用。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 38 路由、33 组件、26 任务、3 平台，0 error / warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN`、`en-US` 与三端 reference pack 通过。 |
| `git diff --check` | 0 | 补丁无空白错误。 |

### 验证边界、未完成项与风险

- 本轮测试对象为临时隔离目录，证明脚本生成、保留和幂等行为；未在 AppKernia 根目录运行初始化，因此没有覆盖或迁移当前项目文档。
- 纯 Python/Markdown Skill 不涉及应用构建、设备或可见 UI；截图索引不适用。未执行 GLM5 或 Claude Code 的真实会话验收，两份提示词仅完成内容与脚本契约验证。
- `auto` 根据源码目录、构建清单或 `.git` 保守判定已有项目；无法准确表达用户意图时应显式指定 `new` 或 `existing`。
- `--force` 会覆盖脚本管理的同名文档，只能在用户明确授权且检查差异后使用；自动生成的已有项目基线只是目录级盘点，不是完整架构审计。

## 2026-08-10 APP 管理与 App 升级中心交付报告

### 交付内容

- PostgreSQL migration `000017_app_upgrade_center` 完成 App manifest 身份、描述资料、所有者/团队、文件资产、渠道和应用市场关系化；Down 会在 WGT、多平台、内部包或其他旧模型不可表达数据存在时显式中止。
- `sys.mobile_releases` 成为新旧发布入口共用的版本事实源；实现草稿、上线、部分上线、下线、历史冻结、SemVer 单调发布、native 单平台和 WGT 多平台发布指针。
- Backend 同步应用 CRUD/批量软删除、版本 CRUD/批量删除/发布/下线、跨租户约束、文件类型与扫描校验、操作审计、短期签名下载、Public API 兼容字段和 Public Config manifest 字段。
- Admin 同步新菜单/路由、应用列表和长表单、升级中心选择器/筛选/发布下拉/native 与 WGT Drawer、权限分支、URL Search Params、移动卡片与双语文案；旧系统路由继续挂载同一页面。
- OpenAPI、生成 TypeScript Client、Mobile UTS DTO、权限/菜单种子、Backend/Admin 蓝图和统一 i18n 契约同步完成。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make -C server sqlc-generate` | 0 | sqlc 生成文件与当前查询一致。 |
| `make -C server check` / `make -C server test-race` / `make -C server build` | 0 / 0 / 0 | Go fmt/vet/test、race 与 API/Worker/CLI build 通过；JSON 复核 130 个通过测试事件、34 个通过包。 |
| PostgreSQL 18 `migration-down → migration-up → seed-core` | 0 | 最终 migration `17 dirty=false`；核心种子为 144 permissions、46 menus、8 modules。 |
| 集成测试后首次 `migration-down` | 2 | Down 按设计拒绝测试残留的新模型关系，并把迁移元数据标为 dirty；修正 fixture 清理、确认 v17 表/列仍在后，将元数据恢复为 `17/clean`，最终 `Down → Up` 退出 0。 |
| `go test -count=1 -tags=integration ./internal/modules/appmanagement/application ./internal/modules/mobileprofile/repository ./internal/modules/storageadmin/repository` | 0 | 三个相关 PostgreSQL integration 包通过；覆盖 App 身份不可变、默认保护、批量原子软删、IDOR，以及 WGT 三平台发布、部分上线、冻结、单调版本、下线/重发和文件关系。 |
| `make -C server test-integration` / 串行 integration 重跑 | 2 / 非零 | 全仓组合在 IAM/jobadmin 的 PostgreSQL `40001` serialization failure；未标记通过。 |
| `pnpm --filter @appkernia/admin check` | 0 | API/i18n/route 生成、ESLint、TypeScript、24 文件/94 测试、production build、bundle 和 Admin blueprint 通过。 |
| `python3 apps/ak-admin/scripts/e2e_mobile_releases.py` | 0 | 12 个 Chromium 状态，双语、375/768/1024/1440、应用/版本/Drawer/409；axe serious/critical=0，无 overflow 和意外 console error。 |
| `bash apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile blueprint、i18n Catalog 和静态项目契约通过；不等同于三端编译。 |
| `make check` | 0 | Backend、Admin、Mobile、Docs 与全部蓝图/i18n 聚合门禁通过。 |
| `git diff --check` | 0 | 最终文本补丁无空白错误。 |

### 设计与截图证据

- [App 管理 Skill 决策](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/decisions.md)、[检查清单](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/review-checklist.md)、[截图索引](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/screenshot-index.md)。
- [升级中心 Skill 决策](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/decisions.md)、[检查清单](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/review-checklist.md)、[截图索引](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshot-index.md)。
- UI Skill 的通用鲜艳方案未覆盖 AK 设计系统；实现继续使用既有 Ant Design semantic tokens、系统字体、高密度桌面表格和窄屏卡片。Playwright/axe 检查促成了表单可访问名称、可聚焦滚动区和窄屏完整动作的修复。

### 纠正记录

- 新增 App 生命周期 PostgreSQL integration 首轮暴露批量删除审计 INSERT 对同一参数推断出 varchar/text 冲突；对资源 ID 与路径拼接增加显式 text cast 后复测通过。
- integration fixture 最初仅删除 tenant，但受 tenant membership RESTRICT 和保留内容页删除保护影响，遗留团队/渠道/市场关系并触发 Down 的防丢失保护。清理逻辑改为按测试 tenant 依次删除发布关系、将待删保留页转为测试用 custom、删除 App、membership、tenant 和测试用户；复测残留计数均为 0。

### 未完成项与风险

- 全仓 integration 组合仍受 IAM/jobadmin 的 `40001` 阻塞；相关模块定向 integration 已通过，但二者不能互相替代。
- 浏览器使用 mock-authenticated 本地 production preview，不代表生产部署或真实后端登录链路。Admin 当前无暗色主题，深色系统偏好截图只验证兼容性。
- 未实现真实 App Store、ABM、应用市场或 `uni-stat` 同步；外链由客户端直连。Mobile 不下载、验证或执行 WGT，避免引入在线可执行包能力。
- 未执行 Android/iOS/HarmonyOS compile、模拟器、真机、安装包安装、签名发行或商店上架验收；本轮未 commit、未 push。
- Admin/Docs 检查使用本机 Node 26.5.0，仓库要求 Node 24，存在 engine warning；检查结果为退出 0，但本轮未补做 Node 24 运行时复核。

## 2026-08-10 App 范围页面全局选择器优化交付报告

### 交付内容

- 新增共享 `GlobalAppSelector`，由 `AppShell` 根据路由决定是否渲染，并固定在全屏按钮左侧；覆盖 `/app/users`、`/app/upgrade-center`、`/app/content/*` 与兼容入口 `/system/mobile/releases`，不在 `/app/applications` 等非范围页面显示。
- 新增共享 `AppSelectionRequiredState`。没有 `app_id` 时，App 用户、文章/分类、单页和版本中心的数据 Card 只显示最小高度、水平/垂直居中的灰色提示，不挂载筛选器、表格、移动卡片或它们的 Loading。
- 未选择 App 时隐藏新增用户、创建内容和发布新版等范围动作；选择 App 后保留 URL 筛选参数并重置已有分页。App 内容文章/分类的 URL 同步补齐 `app_id`，避免切换筛选或 Tab 后丢失范围。
- 使用 Zustand 保存非敏感的 `tenant UUID → App UUID` 工作区偏好。URL 显式 App 优先并在当前租户 App 列表确认后刷新记忆；没有 URL 参数的范围页面自动恢复并回写，清空或不存在的 App 同步清理当前租户值。
- 提示、选择器和禁用状态继续使用 `zh-CN` / `en-US` 翻译键；同步 Admin design system、App 管理/升级中心 override、页面规格和独立 UI Skill 证据目录。
- 扩展升级中心 Chromium E2E，断言 5 个范围入口的无选择状态、选择器与全屏按钮几何顺序、最小高度、文字居中、无旧 Alert、无 spinner/table/filter 及无页面级横向溢出。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/features/apps/scope.test.ts src/components/AppSelectionRequiredState.test.tsx` | 0 | 2 文件、15 项路由/helper/共享占位测试通过。 |
| `pnpm --filter @appkernia/admin check`（首轮） | 1 | ESLint 正确拦截 Ant Design 6 已弃用的顶层 `optionFilterProp` 与 `||` 空值规则；改为 `showSearch.optionFilterProp` 和 `??`。 |
| Admin 完整 check（第二轮） | 1 | 生成阶段从 blueprint i18n 恢复旧提示，新增测试因此失败；确认并修正 `blueprint/i18n/admin` 事实源后由生成器同步语言分片。 |
| `pnpm --filter @appkernia/admin check`（最终，Node 24.18.1） | 0 | API/i18n/route 生成、ESLint、strict TypeScript、26 文件/109 测试、Vite production build、bundle budget 和 Admin blueprint 全部通过。 |
| `python3 apps/ak-admin/scripts/e2e_mobile_releases.py` | 0 | Chromium 13 个 axe 状态覆盖双语、375/768/1024/1440、浅色/深色偏好、选中/未选中/Drawer/409；serious/critical=0，无 overflow 和意外 console error。 |
| 真实本地 API 浏览器链路 | 0 | 从 Git 忽略的密码文件读取 seed 管理员凭据；登录、未选提示、选择默认 App、版本表加载、应用清单隐藏选择器全部通过，失败 Admin 响应与 console error 均为 0。 |
| 持久化 follow-up 纯函数测试 | 0 | `scope` 与 `selection-store` 共 19 项通过；覆盖 URL 优先、租户隔离、读取/写入、清空和损坏存储降级。 |
| 持久化 follow-up 首轮 Admin check | 1 | ESLint 拦截清空映射时未使用的解构绑定；改为无副作用键过滤后重新执行完整门禁。 |
| 持久化 follow-up 最终 Admin check（Node 24.18.1） | 0 | 27 个 Vitest 文件、114 项测试、production build、bundle budget 和 Admin blueprint 全部通过。 |
| 持久化 follow-up Chromium E2E | 0 | 真实点击选择器后，不带参数进入 App 用户页自动恢复；新开 Admin 页面并重新登录后仍从 localStorage 恢复并回写 URL。 |
| 持久化 follow-up 真实本地 API | 0 | 选择 App、跨页面、移除 URL 后刷新/重新登录、再次进入无参数 App 用户页均恢复同一 App；失败 Admin 响应和 console error 为 0。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | 两种语言、默认/回退和 Backend/Admin/Mobile reference pack 通过。 |
| `pnpm --filter @appkernia/admin check:ui-skill` / `git diff --check` | 0 / 0 | 仓库 UI Skill 可用；最终文本补丁无空白错误。 |

### 设计与截图证据

- [全局 App 选择器决策与检查清单](../apps/ak-admin/artifacts/ui-ux-pro-max/app-context-selection/decisions.md)、[截图索引](../apps/ak-admin/artifacts/ui-ux-pro-max/app-context-selection/screenshot-index.md)。
- 人工查看 `en-US` 1440/768/375 未选择态及 375 已选择态；全局选择器在桌面保留标签，在窄屏只收起次要标签/状态装饰，控件本身和可访问名称仍保留。
- 人工查看新页面恢复后的中文 1440 App 用户页，以及真实本地刷新/重新登录后的 1440 用户列表；顶部选择器、URL 和表格上下文一致。
- UI Skill 返回的通用鲜艳品牌方案与 AK 已批准的中性 Ant Design token 不一致，因此仅采用其上下文可发现性、前置状态、键盘和响应式建议，没有覆盖现有字体或配色。

### 纠正记录与验证边界

- Playwright Skill 的 CLI wrapper 前置检查发现 `npx` 可用，但 wrapper 解析后缺少 `playwright-cli`；改用仓库现有 Python Playwright + Chromium 运行时，未新增替代测试框架。
- 第一次真实 API 浏览器脚本用硬导航造成内存访问会话丢失并回到登录页；改用与菜单点击一致的 SPA 路由切换后原样重测通过，这不是选择器产品缺陷。
- 当前本地硬刷新会先回登录且登录成功后进入 Dashboard，不自动回原 redirect；持久化验收因此在重新登录后进入无 `app_id` 的用户管理页，确认从浏览器 UUID 记忆恢复。该认证 redirect 行为不在本次 App 选择器范围内。
- 本轮只修改 Admin UI、Admin i18n/规范、E2E 与交付文档；没有 Backend、Migration、OpenAPI 或 Mobile 行为变化，不代表生产部署或移动端设备验收。
- 本轮未 commit、未 push；本地 API 8080 与 Vite Admin 4173 仅作为本机验收环境。

## 2026-08-10 管理员初始化文档交付报告

### 交付内容

- 在文档站“快速开始”“源码开发模式”“故障排查”的中英文正文中补齐管理员密码初始化方法与安全注意事项。
- 文档区分 Docker 交互初始化和源码开发专用密码文件 Seed；明确无固定密码、仅 development 可用、密码文件权限、幂等不改密、已有账号/租户约束及 CORS/数据库实例排查边界。
- 未把本机测试密码、Token、Cookie 或其他 Secret 写入文档或生成产物。

### 实际命令与退出码

| 命令 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/docs format:check` | 0 | 六份中英文 Markdown 均符合 Prettier 格式。 |
| `pnpm --filter @appkernia/docs check` | 0 | 113 个 API 引用、lint、TypeScript、Prettier、Rspress 66 页双语构建、语言 parity 与 Sitemap 全部通过。 |
| `git diff --check -- apps/ak-docs` | 0 | 文档站补丁无空白错误。 |

### 验证边界

- 本轮只修改文档正文，没有改动数据库、CLI、API、Admin 或 Mobile 业务实现，也没有部署 GitHub Pages。
- 完整文档构建在本机 Node 26.5.0 完成；pnpm 对 Admin workspace 的 Node 24 声明产生 warning，但文档包声明支持 Node 24 至 26，命令实际退出 0。

## 2026-08-10 多语言表单 Tab 统一优化交付报告

### 交付内容

- `system.language` 进入核心字典事实源，使用 internal + fixed 策略和锁定全局记录；两种语言分别保存本地化标签，按 sort order 返回，`zh-CN` 为唯一默认项。
- 新增共享语言查询/严格解析 Hook 和 `AkLocalizedFormTabs`。当前 Admin 语言优先激活；Tab 不销毁已访问字段，字典异常显示可重试错误并禁用保存，语言错误具有图标、危险色和可访问名称。
- 原生 App/WGT、内容文章/分类和 App 单页编辑器全部接入共享组件；单值 locale 选择器保持原实现。提交结构、OpenAPI、数据库翻译约束和 `SupportedLocale` 仍为 `zh-CN | en-US`。
- 系统字典页按翻译键展示内置类型名称、说明和 System 分类；Admin/Backend 蓝图、Design System、页面 override、UI Skill request/output/decisions/checklist 与截图索引已同步。

### 实际命令与退出码

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make -C server seed-core` | 0 | 本地 PostgreSQL 18 幂等初始化：144 permissions、46 menus、8 modules、71 dictionary locale records；`development_admin=false`。 |
| `go test ./internal/seed` / `go test -tags=integration ./internal/seed` | 0 / 0 | 核心字典 Catalog、fixed/internal 策略、双语标签与顺序测试通过；无 `AK_TEST_DATABASE_URL` 时 tagged DB 用例按测试约定跳过。 |
| 本地登录后 GET `/admin-api/v1/dictionaries/system.language` | 0 | 真实 API 返回 `fixed`、`zh-CN → en-US`、中文标签及唯一默认 `zh-CN`；凭据和 Token 未输出或写入产物。 |
| `pnpm check:ui-skill` | 0 | 仓库 UI Skill 可用；已保存本任务 request、skill output、decisions、checklist 和截图索引。 |
| `pnpm --filter @appkernia/admin check`（最终，Node 24.14.0） | 0 | API/i18n/route 生成、ESLint、strict TypeScript、29 文件/118 测试、Vite production build、bundle budget 和 Admin blueprint 通过。 |
| `make check` | 0 | Backend vet/test、Admin、Mobile 静态检查、Docs build 及全部蓝图/i18n 聚合门禁通过；其后可访问名称增量由最终 Admin check 再次覆盖。 |
| Chromium 多语言表单验收 | 0 | 5 个状态覆盖 native/WGT、文章、分类、App 单页，`zh-CN/en-US`、375/768/1440、light/preferred-dark；Tab 切换保值、首错定位、无 overflow，axe serious/critical=0，console error=0。 |
| `python3 -m py_compile ...e2e_content_management.py ...e2e_mobile_releases.py` | 0 | 扩展后的既有 Python E2E 脚本语法通过；本机 Python 无 Playwright 模块，未把语法检查描述为脚本运行通过。 |
| `git diff --check` | 0 | 最终源码与文档补丁无空白错误。 |

### 设计与截图证据

- [共享多语言 Tab 决策](../apps/ak-admin/artifacts/ui-ux-pro-max/localized-form-tabs/decisions.md)、[检查清单](../apps/ak-admin/artifacts/ui-ux-pro-max/localized-form-tabs/review-checklist.md)、[截图索引](../apps/ak-admin/artifacts/ui-ux-pro-max/localized-form-tabs/screenshot-index.md)。
- 截图覆盖 WGT 保值、原生包双语错误、文章、分类和英文 App 单页；机器可读结果为 `output/playwright/localized-form-tabs/e2e-results.json`。

### 纠正记录与验证边界

- 首次 i18n 生成从蓝图事实源覆盖了直接编辑的 namespace 文件；改为先更新 `blueprint/i18n/admin` 后重新生成 24 个命名空间，并通过统一 i18n 校验。
- Playwright Skill 的 CLI wrapper 因当前 `@playwright/mcp` 不再暴露 `playwright-cli` 而无法启动；系统 Python 与 Codex Python 也没有 Playwright 模块。最终使用 Codex 工作区自带 Node Playwright + Chromium 完成真实浏览器验收，没有修改项目依赖锁文件。
- 首轮 axe 暴露内容抽屉既有控件缺少可访问名称；补齐后原样重跑，5 个场景全部为 0 serious/critical。
- 浏览器业务数据使用网络 Mock 以稳定覆盖表单状态；字典 Seed 和 API 返回另用本地 PostgreSQL 18 + 真实管理员会话验证。二者不等同于生产部署、外部发布服务或移动端设备验收。
- Admin 目前没有独立暗色主题，`preferred-dark` 仅验证深色系统偏好下的兼容性；本轮未执行 Android/iOS/HarmonyOS 编译、模拟器或真机。

## 2026-08-11 管理端 OpenAPI 文档与 System 底部入口交付报告

### 交付内容

- 新增 Admin 独立 Vite MPA `/openapi/`，精确锁定并自托管 `@scalar/api-reference@1.64.1`。开发和构建直接消费 canonical `server/openapi/openapi.yaml`，产物 `/openapi/openapi.yaml` 与源文件逐字节一致，不创建第二份业务规范。
- 文档按 `?lang=zh-CN|en-US` 初始化，AppKernia 标题、写操作警告、复制等自有文案使用 `openapi` 语义翻译键；`en-US` 仅在 Scalar 内部映射为 `en`。
- 交互请求使用 `credentials: omit` 并覆盖当前 `Accept-Language`，不读取 Admin 内存 Token 或 HttpOnly Cookie，不预填凭据，`persistAuth=false`。关闭默认字体、Agent、遥测、开发者工具、远程代理和插件 URL；运行时移除 Scalar 版本中无公开关闭配置的 MCP 插件层。
- Nginx 对 `/openapi/` 增加自包含 CSP、`nosniff`、`no-referrer`、`X-Frame-Options: DENY` 与 HTML/YAML 重新验证缓存；哈希资源保持长期 immutable。新增同源 `/api/` 和精确 live/ready 健康代理，继续保留 `/admin-api/`，其余 `/internal/` 明确返回 404。
- 菜单树在权限、Feature Flag、实现注册和空目录过滤后拆分。System 仍是数据一级菜单并保留现有 Seed、权限、路由及三级结构，但 UI 只在侧栏底部以齿轮入口出现；文档图标固定在左且始终保留，System 无可见叶子时齿轮隐藏。
- 桌面展开/折叠态使用按钮上方的边框、圆角、阴影限高面板及右侧级联三级菜单；移动 Drawer 使用可滚动内联层级。实现当前路由祖先展开、选中态、导航后关闭、外部/Esc 关闭、方向键、焦点环与焦点回归；折叠态两列各 40 px，移动触控高度各 44 px。
- 同步 Admin 信息架构、路由授权、设计系统 Master/page override、统一 i18n 契约、Scalar MIT 第三方许可、UI Skill request/output/decisions/review checklist、双语截图与机器可读浏览器证据。

### 实际命令、退出码与证据

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin check`（由最终 Node 24 `make check` 调用） | 0 | API/i18n/routes 生成、ESLint、strict TypeScript、30 个 Vitest 文件 / 124 项测试、Vite production build、OpenAPI 产物、bundle budget 与 Admin blueprint 全部通过。 |
| 首轮聚合 `make check` | 2 | i18n validator 发现新增 `openapi` namespace 尚未进入统一契约；补齐 `blueprint/i18n-contract.json` 与文字契约后重跑。 |
| 第二轮聚合 `make check` | 2 | TypeScript 拒绝当前 Scalar 版本不存在的 `showToolbar` 配置；移除无效选项，以受支持配置和 wrapper hardening 收口。 |
| `env PATH=/Users/payhon/.nvm/versions/node/v24.18.1/bin:$PATH make check`（最终） | 0 | Backend blueprint/vet/tests、Admin 完整 check、Mobile blueprint/static checks、统一 i18n、Docs checks/build 与仓库聚合门禁全部通过。 |
| `docker compose -p appkernia-openapi-e2e build admin`（最终） | 0 | Node 24.18.1 / pnpm 11.18.0 production Admin 镜像构建成功。首轮曾因 pnpm 11 拒绝 `vue-demi@0.14.10` build script 失败；将该既有传递依赖精确加入 `onlyBuiltDependencies` 后重建通过。 |
| `docker compose -p appkernia-openapi-e2e up ...` | 0 | 隔离 PostgreSQL/API/Admin 全部 healthy；API 的 secure-origin 首轮使用 `127.0.0.1` 失败，改为浏览器实际来源 `http://localhost:4175` 后通过。 |
| Nginx `curl` 路由/安全头 smoke | 0 | `/openapi/`、YAML、`/healthz`、live、ready 为 200；metrics 和任意其他 `/internal/` 为 404；`/api/v1/public/config` 到达 Backend 并因缺少必需 App header 返回预期 400，`/admin-api/v1/auth/public-config` 为 200。CSP、no-cache、no-referrer、nosniff、DENY 均存在。 |
| `AK_E2E_BASE_URL=http://localhost:4175 ... python apps/ak-admin/scripts/e2e_openapi_system_navigation.py` | 0 | 真实 Chromium 覆盖 1440px 展开/折叠、375px Drawer、`zh-CN`/`en-US`、System 三级导航、公开文档新上下文、健康请求、Cookie omission、内部拒绝、键盘/Esc/焦点、Reduced Motion、无外部请求/横向溢出/console error；稳定表面 axe serious/critical 为 0。 |
| `docker compose -p appkernia-openapi-e2e down -v` | 0 | 精确移除本轮隔离容器、网络和临时数据库/对象存储卷；复核无同 project 容器或 volume。临时测试数据不可恢复，但不包含用户现有 Compose 数据。 |

### 构建与浏览器度量

- canonical 与构建 YAML：362,990 B；SHA-256 均为 `635df57558c4bc95748d2a77e065a1019b65ebe6ed64267cfb18a48edeeedca1`。
- Admin 初始依赖图 gzip 233,372 B，最大 Admin chunk 166,106 B；`admin_scalar_keys=[]`，主 Admin 首屏没有 Scalar。
- OpenAPI 独立初始图 gzip 988,669 B，最大 chunk 686,992 B；均低于独立预算 1,126,400 B / 768,000 B。Scalar 自身仍触发大 chunk 提示，但未进入主 Admin 入口。
- Chromium 证据覆盖 1440px 桌面展开、1440px 桌面折叠、1440px System panel、375px 移动 Drawer及 1440px 中英文文档。375px 是浏览器 viewport，不是 Android/iOS/HarmonyOS 真机。
- 双语文档均为 HTTP 200，`Accept-Language` 分别为 `zh-CN` / `en-US`，Cookie 发送为 false，外部/Agent/遥测请求均为空。证据见 `output/playwright/openapi-system-navigation.evidence.json`。

### 截图索引

- [桌面中文展开](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/admin-navigation-openapi.zh-CN.1440-expanded.png)、[桌面英文折叠](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/admin-navigation-openapi.en-US.1440-collapsed.png)。
- [桌面 System 弹层与三级级联](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/admin-system-popover.zh-CN.1440.png)、[375px 移动 Drawer 内联层级](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/admin-system-drawer.en-US.375.png)。
- [OpenAPI 中文](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/openapi.zh-CN.1440.png)、[OpenAPI 英文](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-system-utilities/screenshots/openapi.en-US.1440.png)。

### 未完成项与风险

- 没有在文档客户端手工输入 Bearer Token 并调用受保护写接口，因此不把“真实受保护接口 Bearer 调用”和“刷新后手工授权清空”标记为 E2E 通过；当前证据覆盖 `persistAuth=false` 配置、公共健康接口、刷新边界、Cookie omission 和无管理端 Token 继承。
- Scalar 内嵌 API client 打开后的第三方瞬态界面仍观测到 ARIA name/required/value 与颜色对比度 axe 问题；稳定文档页和 Admin shell 的 serious/critical 为 0，但前者不能外推为该瞬态客户端状态通过。
- Scalar 包的文档依赖图包含未执行的 Agent component chunk，这是上游打包行为；运行配置、DOM 和网络证据均确认 Agent 不可用且无 Agent 请求。独立文档包虽在预算内，仍明显大于 Admin 首屏。
- 未执行 Firefox/Safari、生产部署、生产数据、物理移动设备或移动 App 验收。未修改业务 OpenAPI 内容、数据库菜单结构、System Seed、后端权限或生成 Client；本轮未 commit、未 push。

## 2026-08-11 管理端 OpenAPI 模块分组与接口标题双语化交付报告

### 交付内容

- canonical `server/openapi/openapi.yaml` 新增 3 个有序接口面、31 个有序业务模块、顶层 `tags` / `x-tagGroups`、英文 `x-displayName` 和语义 i18n key。每个可见 operation 只属于一个稳定 tag，Scalar 原生导航呈现为“接口面 → 业务模块 → 接口”，模块列表默认折叠且不按字母重排。
- canonical `paths` 下实际有 278 个直接 operation，另有 3 个 Scalar 会渲染的 `components.pathItems` 复用 operation；为消除残余未分组英文项，281 个可见接口标题均纳入唯一分组和双语门禁。path、method、`operationId`、Schema、security 和生成 Client 公共签名均未改变。
- 新增仅由独立文档 MPA 导入的 `api_reference` namespace，包含 3 个接口面、31 个模块和 281 个 operation 的双语值。英文标题与 canonical `summary` 精确一致，中文为逐项校订标题；参数、响应、Schema、示例和详细描述继续保留 canonical 英文。
- 独立入口精确使用 `yaml@2.9.0` 解析 canonical YAML，在内存中替换分组名称、模块 `x-displayName` 和 operation `summary` 后交给 Scalar；缺失/多余翻译、重复 `operationId`、未注册/多 tag、错误路径映射及非 OpenAPI 3.1 文档均 fail closed。直接下载仍为唯一原始 YAML。
- 构建门禁确认 canonical/emitted YAML 逐字节一致、没有 locale-specific spec；Scalar、YAML parser 和 `api_reference` 大 catalog 只进入 OpenAPI 文档图，Admin 主 SPA 依赖图没有相关 marker 或 Scalar key。
- 同步 i18n 契约、Admin 信息架构与路由授权说明、Design System Master/OpenAPI override、UI Skill request/output/decisions/review checklist、六张双语桌面/375 视口截图和机器可读 Chromium 证据。

### 实际命令、退出码与证据

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin check`（首轮） | 1 | ESLint 拒绝测试中的两处 `Array<T>`；改为仓库规定的 `T[]` 后重跑。 |
| `pnpm --filter @appkernia/admin check`（最终，Node 24.18.1） | 0 | OpenAPI/i18n/routes 生成、分组契约、ESLint、strict TypeScript、30 个 Vitest 文件 / 128 项测试、Vite build、bundle/OpenAPI 产物与 Admin 蓝图全部通过。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 38 routes、3 tabs、26 baseline APIs、38 API deltas、3 platforms，0 error / 0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py`（首轮 / 最终） | 1 / 0 | 首轮发现文档专用 `api_reference` 未登记统一 namespace；同步 JSON 与文字契约后双语 key、placeholder 和生成 catalog 一致。 |
| `pnpm --package=@redocly/cli@2.12.4 dlx redocly lint ...`（默认 / 跳过既有 `security-defined` 基线） | 1 / 0 | 默认推荐规则暴露仓库既有安全声明基线及一处未加引号的流式 YAML 描述；本轮修复描述并为 31 个 tag 补齐 description，最终明确 `--skip-rule security-defined` 后规范有效，仅保留 3 个既有 file path 歧义 warning。 |
| `env PATH=/Users/payhon/.nvm/versions/node/v24.18.1/bin:$PATH make check` | 0 | Backend blueprint/vet/tests、Admin 完整门禁、Mobile 静态检查、统一 i18n、Docs 113 个 API 引用及 66 页双语构建全部通过。 |
| `docker compose -p appkernia-acceptance build admin && ... up -d --no-deps admin` | 0 | 最终 Node 24.18.1 Admin 镜像构建并替换成功；Admin/API/PostgreSQL healthy，Worker running。 |
| Nginx smoke 首轮 / 最终 | 无效 / 0 | 首轮 smoke 子 Shell 误用 zsh 特殊变量 `path` 导致自身 `PATH` 被覆盖，未产生验收结论；更名后最终 `/healthz`、文档、YAML、live/ready、Admin public config 为 200，metrics/其他 internal 为 404，Mobile public config 到达 Backend 并返回预期 400，安全头完整。 |
| `AK_E2E_BASE_URL=http://127.0.0.1:4174 AK_E2E_SKIP_SHELL=1 ... e2e_openapi_system_navigation.py` | 0 | 真实 Chromium 覆盖 `zh-CN`/`en-US`、1440×900 与 375×812、初始折叠、模块展开、模块/标题搜索、侧栏/正文一致、稳定锚点、健康请求语言头、无 Cookie/外部请求/横向溢出/意外 console error；axe serious/critical 为 0。 |
| `git diff --check` | 0 | 最终补丁无空白错误。 |

### 构建与契约度量

- 3 个接口面、31 个模块、278 个直接 path operation、3 个复用 path-item operation、281 个本地化可见标题；每个 locale 有 315 个 `api_reference` key。
- canonical 与构建 YAML 均为 377,817 B，SHA-256 均为 `efc4a2050a7cbe8f31fa88f23306ebc545783fa2e34dba5622e3ed8f348bd8df`；Nginx 实际下载内容 SHA 同值。
- Admin 初始图 gzip 233,379 B，最大 chunk gzip 166,106 B；OpenAPI 独立初始图 gzip 999,599 B，最大 chunk gzip 686,992 B，均在既定预算内。`admin_docs_only_matches=[]`、`admin_scalar_keys=[]`。
- 浏览器证据位于 `output/playwright/openapi-reference-navigation-i18n.evidence.json`：双语健康请求均为 200，`Accept-Language` 精确对应当前 locale，Cookie 未发送，外部/禁止请求为空，metrics 为 404。

### 截图索引与验证边界

- [中文桌面模块展开](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.zh-CN.1440.module-expanded.png)、[中文搜索](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.zh-CN.1440.search.png)、[中文 375 视口](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.zh-CN.375.module-expanded.png)。
- [英文桌面模块展开](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.en-US.1440.module-expanded.png)、[英文搜索](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.en-US.1440.search.png)、[英文 375 视口](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.en-US.375.module-expanded.png)。
- 375×812 仅为 Chromium viewport，不是 Android/iOS/HarmonyOS 物理设备；未执行 Firefox/Safari、生产部署或受保护写接口手工 Bearer 调用。Scalar 上游 Agent chunk 仍存在于独立文档依赖图，但 Agent 功能、DOM 和网络请求均被关闭；本轮未 commit、未 push。

## 2026-08-11 OpenAPI 顶部交互测试提示可关闭

### 变更

- 文档顶部真实写请求风险提示新增可见的 `×` 关闭按钮，使用双语语义 ARIA 标签、键盘焦点环和移动端适配布局；关闭后不再占据页面空间。
- 关闭状态只保存在当前标签页 `sessionStorage`，刷新当前文档页保持关闭，新建浏览上下文或新标签页重新显示提示，避免永久隐藏重要风险提醒。
- 新增关闭状态 helper 单元测试，并更新 UI Skill request/output/decisions/review checklist、页面 override、截图索引。

### 验证

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin check` | 0 | 30 个 Vitest 文件、129 项测试、lint、strict typecheck、production build、bundle budget 和 OpenAPI 文档检查通过。 |
| `docker compose -p appkernia-acceptance build admin && ... up -d --no-deps admin` | 0 | 包含关闭提示修复的 Admin 镜像重新构建并健康运行。 |
| `AK_E2E_BASE_URL=http://127.0.0.1:4174 AK_E2E_SKIP_SHELL=1 ... e2e_openapi_system_navigation.py`（首次 / CSS 修复后） | 1 / 0 | 首次发现 `.ak-openapi-notice` 的 `display:grid` 覆盖 HTML `hidden` 默认样式；补充 `.ak-openapi-notice[hidden]{display:none}` 后双语 1440/375 Chromium 验收通过，关闭后刷新仍隐藏，axe serious/critical 为 0。 |

截图：[中文关闭态](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.zh-CN.1440.notice-dismissed.png)、[英文关闭态](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-openapi-reference-navigation-i18n/screenshots/openapi.en-US.1440.notice-dismissed.png)。

## 2026-08-11 文档站在线 OpenAPI 与 System 菜单指南发布

### 交付内容

- 在服务端 API 栏目新增 `zh-CN` / `en-US` 双语指南，覆盖 Admin 底部文档与 System 工具入口、桌面/移动导航差异、权限裁剪、接口面/模块/接口三级分组、双语标题、搜索、交互测试凭据与 Cookie 边界、顶部提示关闭、自托管资源、缓存和安全响应头。
- API 首页和总体架构页增加双语交叉链接，栏目 `_meta.json` 将新页面固定在 API 概览之后。文档不复制第二份 API 规范，公开下载仍由构建从 `server/openapi/openapi.yaml` 同步。
- API 文档检查器识别 `/openapi/`、`/api/`、`/admin-api/` 与 `/internal/` 为网关基础路径，不把它们错误拼接为业务 operation；其他内联接口路径继续逐项对 canonical OpenAPI 校验。
- 本次只复用既有 Rspress 页面、callout、Mermaid 和导航机制，没有新增或调整主题组件、CSS、视觉布局或图片资产，因此不产生新的 UI Skill、截图和视觉回归证据。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| Node 24.18.1 `pnpm --filter @appkernia/docs check` | 0 | 121 个文档 API 路径引用、Rslint、TypeScript、Prettier、双语目录 parity、死链/锚点、68 页 Rspress build 与 Sitemap 全部通过。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 38 routes、3 tabs、26 baseline APIs、38 API deltas、3 platforms，0 error / 0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US`、默认/最终回退和 Backend/Admin/Mobile reference packs 一致。 |
| `git diff --check` | 0 | 文档与检查器补丁无空白错误。 |
| `git push origin main && git push gitee main` | 0 | 内容提交 `32fc41e4b05ddbadccd761f17aedbc4f4315c862` 已推送两个远端的 `main`。 |
| GitHub Pages run `31475956211` | 0 | build 56 秒、deploy 45 秒，两个 Job 均成功；Pages artifact 对应 `32fc41e`。 |
| 线上 HTTP / 内容 / Hash smoke | 0 | 中英文新页面、API 首页和 `/openapi.yaml` 均为 HTTP 200；双语标题存在；线上 YAML 与 canonical SHA-256 同为 `efc4a2050a7cbe8f31fa88f23306ebc545783fa2e34dba5622e3ed8f348bd8df`。 |

### 线上地址与验证边界

- 中文：`https://payhon.github.io/AppKernia/api/online-reference`
- English：`https://payhon.github.io/AppKernia/en-US/api/online-reference`
- GitHub Pages：workflow 模式、HTTPS enforcement 开启、`cname=null`；`appkernia.com` 仍未绑定，不能宣称自定义域名已发布。
- 本轮线上验收为 GitHub Actions 状态、HTTP、预渲染标题和 OpenAPI 字节 Hash；没有重新执行 Chromium 视觉、axe、Firefox/Safari 或物理设备验证。既有 Admin/OpenAPI Chromium 证据仍记录在前述交付报告中，不能外推为本轮文档页面的新视觉证据。

## 2026-08-12 移动端自动升级交付报告

### 交付内容

- 新建 `apps/ak-mobile/uni_modules/ak-upgrade`，通过现有 `AkHttpClient` 注入升级策略，不依赖 uniCloud、后台通知或参考模块。模块包含严格 SemVer、启动/手动检查协调器、下载状态机、双语 modal-page 和 Android/iOS/Harmony 安全链接适配器。
- 启动页改为 `onReady` 初始化；隐私同意和公开 App 配置完成后先核对运行时 AppID、执行升级门禁，再恢复登录会话。自动检查网络错误静默放行，AppID 不一致进入配置错误阻断；强制升级禁用关闭/返回，并在从商店或安装器返回时重新读取本地版本。
- Android 内部发布先尝试市场，随后重新获取策略并通过相对签名地址下载 APK；请求只发送 `X-AppID`，不附带用户 Token，404 签名过期最多刷新一次。下载支持进度、取消和失败重试；安装器接管后避免立即删除文件，其余关闭路径清理下载任务、监听器和临时文件。
- About 页面展示真实本地/服务端版本并支持手动检查；新增升级路由、页面/API 映射、`AkProgress`、双语文案、确定性 UTS DTO 生成器和静态契约测试。
- 后端公开版本响应增加 `delivery_mode`、`store_list` 和可用 `upgrade_url`；应用层按 App 类型、包类型、平台和交付方式做最终能力校验，`uni_app_x` 不允许 WGT，内部原生包仅允许 Android APK。新增 `SYS.MOBILE_RELEASE.UNSUPPORTED_PACKAGE_TYPE`、`SYS.MOBILE_RELEASE.UNSUPPORTED_DELIVERY_MODE` 双语 422 映射，OpenAPI 和 Admin/Mobile 生成客户端同步。
- Admin 根据当前 App 类型隐藏 WGT、显示能力说明、限制非 Android 内部包、切换平台时清理内部文件；不兼容历史记录隐藏发布/重新发布并保留下线。既有 Migration `000017_app_upgrade_center` 已覆盖所需字段，本轮未新增 Migration、权限或字典。
- `ui-ux-pro-max` 与 `native-data-fetching` Skill 的有效约束已落到现有 AK 设计系统、44px 触控目标、显式进度/错误、reduced motion、取消清理、类型化失败和有限重试；通用 App Store 视觉建议未覆盖项目 Master。request、output、decisions、review checklist 和截图索引已保存，截图索引明确记录本轮没有真机截图。

### 实际命令、退出码与证据

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 39 routes、3 tabs、26 baseline APIs、39 API deltas、34 components、3 platforms，0 error / 0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | Backend/Admin/Mobile 的 `zh-CN`、`en-US` key 与 placeholder 一致。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | 生成 i18n/API 产物为最新；6 组 SemVer 和模块/契约/启动链静态测试通过。 |
| `go test -tags=integration ./internal/modules/mobileprofile/... -count=1 -v` | 0 | Application、PostgreSQL Repository、HTTP 共 16 个顶层测试通过；能力矩阵子场景另覆盖 6 组。 |
| `make -C server check` | 0 | `go vet ./...` 与 Backend 全量默认测试通过。 |
| Node 24.18.1 `pnpm --filter @appkernia/admin check`（由最终根门禁执行） | 0 | 生成 API/i18n/routes、lint、strict typecheck、30 个 Vitest 文件 / 130 项测试、production build、bundle/OpenAPI/蓝图检查全部通过。 |
| `apps/ak-mobile/scripts/build-platform.sh android` | 0 | HBuilderX 5.06 编译 29 页面，达到 Android class 阶段并输出 `UTS编译完毕`；不是 APK 安装或真机运行。 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | iOS simulator 目标编译 29 页面，`ak-upgrade` 插件与页面进入编译，输出 `UTS编译完毕`；不是 App Store 或物理设备验收。 |
| `apps/ak-mobile/scripts/build-platform.sh harmony` | 0 | 编译 29 页面、输出 `UTS编译完毕`、生成 Harmony 原生工程并安装依赖；当前包名/签名缺失，不是可发布 HAP。 |
| `env PATH=/Users/payhon/.nvm/versions/node/v24.18.1/bin:$PATH make check` | 0 | Backend/Admin/Mobile/Docs 和三套蓝图/i18n 聚合门禁通过；Docs 构建 68 页。 |
| `git diff --check` | 0 | 最终补丁无空白错误。 |

### 截图索引、未完成项与风险

- UI 产物位于 `apps/ak-mobile/artifacts/ui-ux-pro-max/app-upgrade/`；`screenshots/INDEX.md` 明确说明未执行三端运行态截图，不能把空索引解读为视觉验收通过。
- 未执行物理 Android APK 下载/安装权限/取消/失败、市场 fallback，未执行物理 iOS App Store 跳转、Harmony 商店/HTTPS 跳转，也未验证强制升级从安装器/商店返回后的持续拦截；`zh-CN` / `en-US` 真机截图未完成。
- Harmony 编译提示未配置应用包名/签名；只证明 UTS 与原生工程生成链可用。Android/iOS 的退出 0 同样不能外推为签名包、上架链接或真实设备通过。
- 自动检查对普通网络故障采用计划锁定的 fail-open；一旦已展示强制策略，下载或跳转失败仍保持阻断并提供重试。外部链接只接受适配器允许的市场 scheme 和最终 HTTPS fallback。
- 用户原有 `apps/ak-mobile/manifest.json` 版本 `0.2.0` 修改及三个未跟踪参考模块保持原样，没有覆盖、删除或自动纳入本功能依赖。本轮未 commit、未 push。

## 2026-08-15 快学AI微信公众号草稿工作流交付

### 交付内容

- 项目级安装 `wechat-article-writer` Skill，并新增 `quicklearn-ai-wechat` Skill。后者定义快学AI的作者口吻、事实与引用规范、默认封面、草稿 manifest 契约以及“检索 → 写作 → 构建 → dry-run → 投递 → 回读确认”的完整操作流程。
- 新增本地 Markdown 构建器和 SSH 投递器：文章转换为微信兼容的内联 HTML，支持封面、正文图片占位符、摘要生成和结构/大小校验；投递器只向服务器受控暂存目录传输白名单文件，并保证成功或失败后精确清理。
- 在 `1.95.190.254` 部署 `/usr/local/bin/quicklearn-wechat`，使用微信 `stable_token`、永久图片素材、正文图片上传、草稿新增和草稿回读接口。CLI 仅暴露配置、诊断、暂存和创建草稿能力，不包含群发、正式发布或删除能力，也不开放公网 HTTP 服务。
- AppSecret 仅以交互式无回显方式配置到服务器 root-only 文件，未写入 Git、Skill、shell 参数或交付报告；Token 和素材 ID 缓存也位于 root-only 状态目录。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `install-skill-from-github.py --repo staruhub/claudeskills --path skills/Geek-skills-wechat-article-writer ...` | 0 | 目标 Skill 安装到 `.agents/skills/wechat-article-writer`。移除不被 Codex frontmatter 接受的上游 `version` 字段后校验通过。 |
| `quick_validate.py .agents/skills/wechat-article-writer` | 0 | `Skill is valid!`。 |
| `quick_validate.py .agents/skills/quicklearn-ai-wechat` | 0 | 自定义 Skill 的 metadata、说明和目录结构有效。 |
| `python3 -m py_compile`（4 个本地/服务器脚本） | 0 | Markdown 构建器、SSH 投递器、服务器网关和默认封面生成器通过 Python 语法编译。 |
| `build_draft.py`（无正文图片 / 1 张正文图片） | 0 / 0 | 两种草稿包均生成有效 manifest、内联 HTML 和受控图片文件。 |
| `publish_via_ssh.py publish ... --dry-run`（无正文图片 / 1 张正文图片） | 0 / 0 | 本地契约、资源路径和 HTML 安全校验通过，不触发外部写入。 |
| SSH 部署、SHA-256 比对与服务器 `python3 -m py_compile` | 0 | 服务器 CLI 与本地源文件 Hash 一致，远端 Python 编译通过；程序为 root-owned `0755`。 |
| 服务器目录与凭据权限检查 | 0 | `/etc/quicklearn-wechat`、`/var/lib/quicklearn-wechat` 为 `0700`，凭据为 `0600 root:root`。未读取或输出凭据内容。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 39 routes、3 tabs、26 baseline APIs、39 API deltas、34 components、3 platforms，0 error / 0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US`、默认/最终回退和 Backend/Admin/Mobile reference packs 一致。 |
| `git diff --check` | 0 | 当前补丁无空白错误。 |
| `publish_via_ssh.py publish ...`（连通性草稿） | 1 | SSH 文件传输和远端暂存清理通过；微信返回 `40164 invalid ip 1.95.190.254`，未取得 access token、未上传素材、未创建草稿。 |
| `publish_via_ssh.py doctor`（首次配置复测） | 1 | 服务器出网地址为 `1.95.190.254`，当时微信仍返回 `40164`；此历史阻塞已在重新保存白名单后解除。 |

### 未完成项与风险

- 微信后台重新保存 `1.95.190.254` 后，`doctor`、草稿新增与草稿详情回读均已成功；原 `40164` 阻塞已解除。
- 没有执行群发或正式发布，CLI 也没有这些能力；即使草稿创建成功，仍需人工在微信公众号后台预览、校对和决定是否发布。
- 默认封面已本地目视检查；文章排版尚未获得微信草稿编辑器/手机预览证据。当前没有草稿 `media_id`、后台截图或真机阅读证据。
- 本轮未修改 Backend、Admin、Mobile、OpenAPI、数据库或 i18n 业务契约；用户原有 `apps/ak-mobile/manifest.json` 修改保持原样。本轮未 commit、未 push。

## 2026-08-15 快学AI首篇 AppKernia 介绍草稿

### 内容与事实边界

- 文章采用维护者第一人称，以实际推进 OpenAPI 单一事实源、移动端自动升级和安全边界时的经历为主线；技术栈、3 个接口面/31 个模块/281 个操作标题、三平台 29 页面编译等陈述均回查当前 README、三套蓝图、实施状态和交付报告。
- 正文明示“三平台编译通过不等于真机通过”，没有编造用户量、客户案例、收入、性能提升或生产部署结果。远端 `origin/main` 与本地 `HEAD` 均为 `f527b0d`，GitHub 仓库和 GitHub Pages 在本轮均返回 HTTP 200。
- 使用项目自制默认封面，不包含第三方图片或正文外链图片；文章、编辑记录和生成草稿包保存在 Git 忽略的 `tmp/wechat/2026-08-15-appkernia-intro/`。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `publish_via_ssh.py doctor`（白名单重新保存后） | 0 | 公众号为“快学AI”，Token 有效，创建前草稿数为 20。 |
| `build_draft.py ...` | 0 | 主标题 26 字、摘要 70 字、微信安全 HTML 8,048 字符、正文图片 0 张，manifest 构建成功。 |
| `publish_via_ssh.py publish ... --dry-run` | 0 | 标题、摘要、HTML、封面和 manifest 本地安全校验通过，未触发外部写入。 |
| `publish_via_ssh.py publish ...` | 0 | 经 SSH 调用服务器草稿 API，返回 `ok=true`、`verified=true`、`article_count=1`，标题与回读结果一致。 |
| `publish_via_ssh.py doctor`（创建后） | 0 | 草稿总数为 21，Token 仍有效；远端暂存目录检查为空。 |

### 发布边界

- 本轮只创建草稿，没有调用群发或正式发布接口；仍需在微信公众号后台进行编辑器排版和手机预览检查，再由人工决定是否发布。
- 默认封面已在工作流部署阶段完成本地目视检查，但本篇文章没有新的微信后台截图或手机预览证据。
- 用户原有 `apps/ak-mobile/manifest.json` 修改保持原样。本轮未 commit、未 push。

## 2026-08-25 App 启动页与启动介绍交付报告

### 交付内容

- 后端以 `000018_app_startup_experience` 建立双语启动元信息、草稿、不可变 revision、双语排序资产与 published pointer；发布原子校验扫描/MIME/双语完整性、版本递增，并保护 draft/revision 文件引用。
- Admin App Drawer 完成双语元信息、图标预览、最多 10 组双语图片、无障碍说明、键盘排序、启用开关、草稿/发布状态和独立权限发布；保存不会发布，关闭不会删除草稿或历史版本。
- Mobile 首装隐私门禁只使用随包快照；公开配置与强制升级之后才判断 onboarding。当前 published 图片携带 `X-AppID` 预下载为本地临时路径，无跳过且必须全部看完，完成状态按 App UUID 保存最高版本。
- 新增 `ak-cli app-startup export` / `--check`，同步 OpenAPI、生成 Client、权限/契约、双语 Catalog、设计系统和 UI Skill 产物。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| PostgreSQL 18 `migrate up → down 1 → up 1` | 0 / 0 / 0 | 最终 version 18、dirty=false，不是仅静态 SQL 检查。 |
| `go test -tags=integration ./internal/modules/appmanagement/application -count=1 -v` | 0 | 启动草稿/发布/public projection 集成通过；既有 OTP 用例因临时库未 seed 模板 skip 1。 |
| `make -C server check` | 0 | `go vet ./...` 与 Backend 全量默认测试通过。 |
| `ak-cli app-startup export ...` + `--check` | 0 / 0 | 真实数据库/local object store 生成 39,129 字节 PNG 和 650 字节 UTS 快照，漂移检查通过。 |
| `npm run check`（Admin） | 0 | 生成、OpenAPI reference、lint、strict typecheck、30 文件/130 项 Vitest、production build、bundle/OpenAPI/蓝图通过。 |
| bundled Node Playwright `scripts/e2e_app_startup.mjs` | 0 | 真实 Chromium 双语 × 375/768/1440；axe 全量 violation、overflow、console 均为 0，键盘排序通过。系统 Python 缺模块的首次命令真实退出 1，未修改依赖锁。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile blueprint、i18n、生成 Client、升级模块和启动链静态门禁通过。 |
| `build-platform.sh android` / `ios` | 0 / 0 | HBuilderX 5.24 编译 30 页面；仅 compile-only，非安装/真机。 |
| HBuilder Harmony / 无代理 `ohpm install --all` / `hvigorw assembleHap` | 1 / 0 / 0 | UVue/UTS 成功，内置 ohpm 代理 `Invalid URL` 中断；手工依赖安装与 native assemble 成功，生成 15,662,973 字节 unsigned HAP。 |
| blueprint/i18n + `git diff --check` | 0 | Backend/Admin/Mobile 契约与双语 parity 通过，补丁无空白错误。 |
| `env PATH=/Users/payhon/.nvm/versions/node/v24.18.1/bin:$PATH make check`（最终） | 0 | Backend、Admin 30 文件/130 项测试、Mobile 随包快照漂移门禁、蓝图/i18n、Docs 121 个 API 引用与 68 页构建全部通过。 |

### 截图、未完成项与风险

- Admin 证据见 `apps/ak-admin/artifacts/ui-ux-pro-max/app-startup-experience/screenshot-index.md`；Mobile UI 产物见 `apps/ak-mobile/artifacts/ui-ux-pro-max/startup-experience/`。
- 未采集物理设备截图，不能把三平台编译或 unsigned HAP 外推为冷启动、网络、退出、返回键、动态字号和安全区验收。Harmony 仍缺包名/签名，发布流水线需固定可用的 ohpm 网络环境。
- 用户原有 `apps/ak-mobile/manifest.json`、既有文档增量和未跟踪微信公众号 Skills 未覆盖或删除。本轮未 commit、未 push。

## 2026-08-26 AppKernia 三端自定义基座交付报告

### 交付内容

- Android/iOS 原生调试包均使用 custom playground；Android package、iOS bundle identifier 和 HarmonyOS bundle name 统一为 `com.appkernia.mobile`，DCloud 运行时 AppID 为 `__UNI__196F2FC`。
- 从仓库 AppKernia 品牌主图确定性生成并校验 Android/iOS/HarmonyOS 全套图标与启动资源，不使用 DCloud/HBuilderX 默认图标。
- 新增 `build-custom-base.sh`、`generate-native-icons.py`、`prepare-harmony-native.py`、`verify-custom-base.py`、Android native resources、Harmony AppScope overlay、运行手册与 UI Skill 证据。Harmony 签名构建拆为不会重新调用 HBuilderX 的 `harmony-signed`，避免已生成 Signing Config 被覆盖。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `./scripts/build-custom-base.sh android` | 0 | HBuilderX 5.24 云端生成 `unpackage/debug/android_debug.apk`；aapt2 为 `com.appkernia.mobile` / `AppKernia` / `0.2.0`。APK SHA-256 `a556958d1df12eee23fad69155ba451adb1cab11ffa874007e7c2de45c883d1f`。 |
| vivo V2545A `adb install -r` | 先 1，后安装成功 | 首次因设备未授权返回 `INSTALL_FAILED_ABORTED`；后续设备已安装 `com.appkernia.mobile` 0.2.0，`pm path`、`dumpsys package` 和前台 Activity 均可解析。 |
| `./scripts/build-custom-base.sh ios-simulator` | 0 | 云端生成 89 MB `Pandora_simulator_debug.app`；`Info.plist` 为 `com.appkernia.mobile` / `AppKernia` / `0.2.0`，AppIcon 存在，Mach-O 为 x86_64。可执行文件 SHA-256 `01fd984a438a3bb0cf4d42daea405f1fcf1a1a8ece6b87a78117821c44b2df4a`。 |
| `simctl install/launch` + HBuilderX `launch app-ios --playground custom` | 0 | iOS 18.6 iPhone 16 Pro 模拟器安装成功；HBuilderX 明确报告自定义基座安装成功、30 页面同步成功、项目启动，并显示 AppKernia 登录页。 |
| `./scripts/build-custom-base.sh harmony` | 0 | HBuilder UVue/UTS 编译成功；内置 OHPM 代理失败后，脚本在 DevEco 工程内无代理安装依赖，Hvigor `BUILD SUCCESSFUL`。按 DevEco 模板尺寸重建 17,002,087 字节 unsigned HAP，SHA-256 `31f9d8b10b301b6c167c1955135948d72aa85a8bbc03ae9ebb67285bede7d620`。 |
| `./scripts/build-custom-base.sh verify` | 0 | Android APK、iOS simulator App、Harmony unsigned HAP 的原生身份、版本与 AppKernia 图标契约全部通过；Android 四档 launcher icon 逐像素一致，iOS AppIcon 与 1024 品牌主图缩放匹配，Harmony layered/start icon 逐字节一致。HAP 是否签名由官方工具实测，不再依赖文件名。 |
| `prepare-harmony-native.py --require-signing` | 1（预期门禁） | 当前生成工程没有 Signing Config，构建在读取密码/私钥前明确终止。 |
| `./scripts/build-custom-base.sh verify-installable` | 1（预期门禁） | APK/App 身份通过，DevEco `hap-sign-tool.jar` 确认 HAP 无签名块，拒绝把 unsigned HAP 标成真机可安装。 |
| DevEco 自动签名（`com.appkernia.mobile`） | 未生成 | 用户已授权；帐号/Team/Bundle 进入 Managed Profile 流程后，DevEco 明确报缺少在线设备，Signing Config 仍为 0。 |
| vivo V2545A `adb install -r` 再次发起 | 130 | 安装请求等待手机端授权约 60 秒仍无结果，确认包未安装后终止等待进程；未绕过设备安全提示。 |
| HBuilderX `launch app-android --playground custom` | 0 | 明确报告设备端 AppKernia 自定义基座已是最新版本并跳过更新，完成 30 页面资源同步、启动和 `onLaunch`；修复前捕获两处运行错误，修复后进入登录页且进程日志无对应异常。 |
| `apps/ak-mobile/scripts/build-platform.sh android`（运行时修复后） | 0 | HBuilderX 5.24 完成 30 页面 Android class 编译，`ready in 34225ms`。 |
| `apps/ak-mobile/scripts/build-platform.sh ios`（运行时修复后） | 0 | HBuilderX 5.24 完成 30 页面 iOS UTS 编译，`ready in 36724ms`。 |
| `build-platform.sh harmony` / `build-custom-base.sh harmony` | 1 / 0 | 前者 30 页面编译成功后被 HBuilder 内置 OHPM 的代理 `Invalid URL` 中断；后者使用无代理 OHPM/Hvigor 完成 unsigned HAP，`BUILD SUCCESSFUL`。 |
| DevEco Device Manager 协议/模拟器 | 0 / 通过 | 用户分别明确同意 HarmonyOS Software License and Service Agreement 与 HarmonyOS SDK License Agreement；两份协议均完成接受，HarmonyOS 6.0.2（API 22）官方 Phone ARM64 镜像下载完成。六个镜像 detached CMS 签名逐一验证通过，模拟器日志显示系统/用户镜像校验成功和 Guest OS Boot Completed。 |
| API 22 模拟器 `hdc install -r` / `aa start` / `bm dump` | 0 | 新 HAP 返回 `install bundle successfully` 和 `start ability successfully`；`bm dump` 回读 `com.appkernia.mobile`、AppKernia label resource、`$media:layered_image` 与 API 22 编译信息。 |
| `uitest keyEvent Home` + `snapshot_display` + `hdc file recv` | 0 | 保存两张 1080 × 2340 JPEG：Home Screen 完整显示仓库 AppKernia 渐变/绿色轨迹 layered icon 与标签；首次隐私页也完成复测。未代替用户同意应用自身隐私/用户协议。 |
| `bash -n` + `python3 -m py_compile` | 0 | 两个 shell 入口和三个 Python 工具语法通过。 |
| `./scripts/check-project.sh` | 0 | Mobile blueprint/i18n、生成 Client、随包启动快照和升级静态测试通过。 |
| `python3 apps/ak-mobile/scripts/verify-mobile-framework.py` | 0 | 修正静态门禁与当前由 `AkHttpClient` 统一注入设备键的架构漂移后，运行配置逐字段读取、TabBar 生命周期及既有 framework contract 全部通过。 |
| mobile blueprint + cross-platform i18n validators | 0 / 0 | 39 routes、3 platforms，0 error / 0 warning；`zh-CN` / `en-US` parity 通过。 |
| `git diff --check` | 0 | 补丁无空白错误。 |

### 截图、未完成项与风险

- 截图索引位于 `apps/ak-mobile/artifacts/ui-ux-pro-max/custom-native-bases/screenshots/INDEX.md`：包含 vivo 安装授权页、首次未同步诊断页、修复后真机登录页、iOS 模拟器桌面/登录页，以及 HarmonyOS API 22 官方模拟器的 AppKernia 桌面启动器与首次隐私页。
- Android 已完成 vivo V2545A 自定义基座安装、资源同步与匿名登录页启动；未执行登录、后端网络、安全存储、升级、返回键或长时稳定性。iOS 物理机仍需要 `com.appkernia.mobile` 对应开发证书/Profile。
- DevEco Project Structure 已确认 Bundle name 为 `com.appkernia.mobile` 且华为帐号已登录；用户授权后自动签名真实执行到 Managed Profile 创建阶段，但因没有在线物理设备而停止，没有留下空签名配置。两份 HarmonyOS 软件/SDK 许可协议均已按用户即时明确授权接受；API 22 官方 Phone 镜像及 unsigned HAP 模拟器运行已经完成。下一项外部依赖仅为连接实际 HarmonyOS 设备后重试 Managed Profile 自动签名与真机安装。
- Harmony HBuilder 内置 OHPM 会受已启动 HBuilderX 进程的代理环境影响；独立无代理 OHPM/Hvigor 路径已自动化并退出 0。首次生成原生工程若在依赖阶段前失败，应清除代理环境后重启 HBuilderX。
- 用户原有 `manifest.json` 版本 `0.2.0` 保持；两个既有未跟踪微信公众号 Skills 未纳入改动。本轮未 commit、未 push。

## 2026-08-26 多端打包自动化与文档站交付报告

### 交付内容

- 新增 `apps/ak-mobile/scripts/mobile-package.mjs`，在 macOS/Windows 上编排 HBuilderX 云打包、Harmony 原生工程生成、无代理 OHPM 和 Hvigor。Android/iOS/HarmonyOS 自定义基座与正式版均有独立的 preflight、dry-run、单平台、all 和 verify 入口。
- 正式版 Android 固定 AppKernia 包名并使用自有 Keystore；iOS 固定 AppKernia Bundle ID 并使用 Apple Distribution p12/Profile；Harmony 使用“prepare → DevEco Signing Config → release assembleApp”两阶段流程，避免重新生成工程覆盖签名配置。
- HBuilderX 签名配置仅写入系统临时目录的受限 `configure.json`，调用结束后递归删除；命令输出、`package.json` 与文档不保存密码。正式预检缺签名时 fail-closed。
- 新增 `docs/manual/mobile-custom-base-build.md`、`docs/manual/mobile-production-release.md`，并更新 Mobile README。文档站新增中英文打包指南、导航元数据、开始使用索引和移动开发交叉链接。
- `apps/ak-mobile/scripts/check-project.sh` 纳入打包编排器的 Node 单测，后续 Mobile 静态门禁会持续检查 AppKernia custom 配置、Windows 路径和签名脱敏契约。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| Node 24.18.1 `node --check`、`node --test mobile-package.test.mjs` | 0 / 0 | 编排器语法通过；4 项测试全部通过，覆盖参数、AppKernia identity/custom 模式、Windows 可执行路径和秘密值脱敏。 |
| `python3 -m py_compile` + `python3 -m json.tool package.json` | 0 / 0 | 三个 Python 辅助工具与根 `package.json` 语法有效。 |
| `pnpm build:mobile:base:preflight` | 0 | 只读检查原生身份、三端图标尺寸与 Harmony AppScope，未重生成资产或发起打包。 |
| `pnpm build:mobile:base:dry-run` | 0 | 展示 Android/iOS custom pack、Harmony pack/overlay/OHPM/Hvigor 完整编排，未调用云打包或编译。 |
| `pnpm build:mobile:release:dry-run` | 0 | 展示三端 release 编排；7 个签名输入均只输出 `<missing>`，没有输出秘密值。 |
| `pnpm build:mobile:release:preflight` | 1（预期门禁） | 当前首先明确缺少 4 个 Android 发布签名变量并停止；证明无签名不会继续正式打包。 |
| `pnpm build:mobile:base:verify` | 0 | 复核仓库当前已有 Android APK、iOS simulator App 与 Harmony unsigned HAP 的 AppKernia 身份和实际图标；这是既有自定义基座产物复核，不是本轮重新云打包。 |
| Mobile blueprint / cross-platform i18n validators | 0 / 0 | 39 routes、3 platforms，0 error / 0 warning；`zh-CN` / `en-US` parity 通过。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | 生成 Client/i18n/启动快照/升级契约及新增 4 项打包脚本测试全部通过。 |
| 首轮 `pnpm check:docs` | 1 | lint 与 typecheck 通过，Prettier 准确指出英文 guide 索引表列宽格式漂移；未掩盖失败。 |
| Prettier 定向格式化后 `pnpm check:docs` | 0 | 121 个 OpenAPI 引用、rslint 0、TypeScript、Prettier、Rspress 双语构建、language parity 与 70 页 Sitemap 全部通过。 |
| Node 24.18.1 根 `make check`（发布前最终门禁） | 0 | Backend 蓝图与 Go 全量测试、Admin 30 文件/130 项测试及生产构建、Mobile、三套蓝图/i18n 与 Docs 70 页聚合门禁全部通过。 |
| `git diff --check` | 0 | 当前补丁无空白错误。 |

### 未完成项与风险

- 未提供 Android/iOS 正式发布证书，Harmony Managed Profile 仍依赖在线物理设备，因此没有执行三端正式签名出包、真实安装、商店上传或审核；dry-run 与脚本测试不能替代这些外部验收。
- Windows 分支已由单测验证路径选择，并由同一 Node 编排器避免 Bash 依赖；本轮实际执行环境是 macOS，尚无 Windows HBuilderX/DevEco 真实打包证据。
- 文档站源码和本地 production build 已更新；线上发布以本次 `main` 推送触发的 GitHub Pages workflow 及最终 HTTP 验证为准。
- 本项没有新增可视 UI，也没有新的截图；既有自定义基座设备/模拟器截图仍由上一节索引管理。

## 2026-08-27 AKDOCS-006 首页 Hero 视觉优化交付报告

### 交付内容

- 移除 Hero 眉题的胶囊边框、圆角与内边距，并显式重置行高，避免 Rspress 原始大标题行盒把小标签撑至约 93px。
- 以 `.ak-home-main > .rp-home-hero` 覆盖默认 1152px/Flex 规则，采用最大 1488px、`0.88fr / 1.12fr` 两栏网格；产品组合在桌面端超过 Hero 宽度的 45% 门禁。
- Admin 浏览器画布减少右侧保留空间，手机框占比从 22% 增至 23%；不裁改、不替换原始产品截图和 alt 文本。
- 通过 Hero `100vw` 伪元素提供全宽浅色/深色品牌渐变，内容列仍保持居中与响应式堆叠。
- 验收脚本新增标签高度/边框、Hero 渐变、桌面产品图占比断言，并保存 1440px 中英文 Hero 局部截图。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max --design-system` + UX/React 查询 | 0 | 采用 WCAG AA、响应式图片、无页面横向滚动与多视口检查；拒绝与项目阶段不符的客户 Logo、评分和动画社交证明。 |
| `DOCS_ORIGIN=https://payhon.github.io DOCS_BASE=/AppKernia pnpm --dir apps/ak-docs check` | 0 | 121 个 API 引用、RSLint 12 文件 0 error/warning、TypeScript strict、Prettier、70 页双语构建、语言 parity 与 Sitemap 全部通过。 |
| `AK_DOCS_EVIDENCE_ID=AKDOCS-006 AK_DOCS_SAMPLE=home ... visual-check.py` | 0 | 6 个最终状态全部通过；标签 19.5px/0 边框，桌面产品列 53.3%–53.7%，两套渐变存在，H1/图片/overflow/Slider/console/请求/axe 均通过。 |
| Backend / Admin / Mobile blueprint validators | 0 / 0 / 0 | Backend 18 对 migration/89 表；Admin 46 菜单/56 路由；Mobile 39 路由/34 组件，0 error/warning。 |
| i18n validator / UI Skill check / Python compile | 0 / 0 / 0 | `zh-CN`、`en-US` 和三端 reference pack 一致；Skill 存在且已真实执行；视觉脚本语法通过。 |

### 纠正记录与边界

- 第一次把浏览器 URL 也挂到 `/AppKernia` 的只读静态服务器时，Rspress 客户端路由按 404 处理；改用 Rspress preview 后页面路由正确，但 Base Path 资源被返回为 HTML，形成无样式的假失败。最终使用只读映射服务器在根路径呈现页面、同时映射 `/AppKernia/*` 生产资源，6 个状态原断言全部通过；没有为本地预览削弱生产 Base Path。
- [AKDOCS-006 decisions](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-006/decisions.md)、[review checklist](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-006/review-checklist.md)、[screenshots and results](../apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-006/screenshots/INDEX.md)。
- 功能提交 `676304f6e606ceb03681a9158297e0ddaa80c054`（`feat(docs): enhance homepage hero`）已推送 `origin/main`；GitHub Pages run [`33047259705`](https://github.com/Payhon/AppKernia/actions/runs/33047259705) 成功，build job `98434068813` 用时 45 秒，deploy job `98434231357` 用时 9 秒，workflow head SHA 与功能提交一致。
- Pages API 回读 `html_url=https://payhon.github.io/AppKernia/`、`build_type=workflow`、`https_enforced=true`、`cname=null`。中英文首页、Admin/Mobile Hero 静态图及 Sitemap 共 5 个 URL 均返回 HTTP 200；自定义域名尚未绑定，不宣称 `appkernia.com` 可访问。
- 线上 Python Playwright/Chromium 复核 `zh-CN 375 light`、`zh-CN 1440 light`、`en-US 1440 dark` 三个状态：HTTP 200、单一 H1、页面无横向溢出或破图，标签为 19.5px/0 边框，桌面产品图占 Hero 53.7%，两组 Slider 点击和键盘切换成功；axe serious/critical、console error、失败请求/响应均为 0。

## 2026-08-28 Mobile 资讯界面精修交付报告

### 交付内容

- 统一 `ak-icon-button` 和缩小后的 `ak-icon`/返回图标：视觉 14–16px，触控目标保持 44px；首页、详情、资料页和 Sheet 的明确操作改为图标按钮。
- 重做浏览页三类内容卡、右上角筛选浮层和右滑全屏搜索 DialogPage；搜索查询不落盘，避免把潜在敏感内容写入普通存储。
- 重绘四栏 TabBar 语义图标并同步主题/i18n；新增搜索路由及 route/API/permission 契约。
- 建立 `ak-content-viewer` 分派层和文章、图文、视频三个独立查看器。2026-08-28 测试确认原 `uni.getVideoInfo` 方案在缺少可选 `uni-media` 的自定义基座中会运行时崩溃，已改为无该模块依赖的封面自然尺寸判定与稳定横屏回退。
- 更新 Mobile design system、information override、40 项组件兼容矩阵和独立 UI Skill request/output/decisions/review/screenshot index。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max` mobile news / search / video / accessibility 查询 | 0 | 采用 editorial minimalism、44px 触控目标、8px 邻接间距、无自动播放和明确状态反馈。 |
| `python3 blueprint/mobile/scripts/validate_blueprint_specs.py` | 0 | 44 routes、4 tabs、40 components、3 platforms，0 error / 0 warning。 |
| `python3 blueprint/scripts/validate_i18n_contract.py` | 0 | `zh-CN` / `en-US` 及 reference pack parity 通过。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Blueprint、i18n Catalog、API Client、startup snapshot、6 项升级契约、4 项打包脚本测试和静态门禁通过。 |
| `python3 apps/ak-mobile/scripts/verify-mobile-framework.py` | 1 | 额外旧框架检查仍断言 Phase 0 的 `Array<ArticleWireBlock>`、旧 Home/Profile/asset-loader 结构；当前 Phase 1 使用 `ArticleDocument` 与资讯首页，因此在首个过期断言停止。它不在 `check-project.sh` 门禁内，未将失败写成通过。 |
| `apps/ak-mobile/scripts/build-platform.sh android` | 0 | HBuilderX 5.24 完成 35 页面 Android class 编译；修正跨组件反射、视频元数据事件和搜索数组类型后通过。 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.24 完成 35 页面 iOS UVue/UTS 编译。 |
| `apps/ak-mobile/scripts/build-platform.sh harmony` | 0 | 35 页面项目编译成功，OHPM 依赖安装成功并生成未签名调试 HAP。 |
| iOS 18.6 模拟器资源同步 | 未通过 | 已安装旧自定义基座缺少新增 UVue 原生类，新资源启动为白屏；不计作运行通过。同步前资源已备份并恢复，旧环境恢复截图正常。 |

### 截图、未完成项与风险

- UI Skill 索引：[request](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-ui-refinement/request.md)、[skill output](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-ui-refinement/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-ui-refinement/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-ui-refinement/review-checklist.md)、[screenshot index](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-ui-refinement/screenshot-index.md)。
- 新界面仍需重建 35 页面 Android/iOS 自定义基座并在 iOS/Android/HarmonyOS 运行；现有 `restore-check` 仅证明模拟器旧资源已恢复，不代表新界面。
- 视频默认方向现在以封面自然尺寸为无额外原生模块的安全提示；封面与视频方向不一致时可能需要用户手动切换。后续若服务端提供可信视频宽高，应优先透传该元数据。仍需用实际 CDN 视频在三端验证方向、切换过程中的进度保持和离页暂停。
- `verify-mobile-framework.py` 仍需按当前 Information App 架构重写断言；本轮没有通过删除安全检查或伪造旧字符串来让它变绿。
- 未执行物理设备、动态字号、读屏、键盘、真实微信分享、Release 签名或生产部署。本轮未 commit、未 push。

## 2026-08-28 内容查看器测试反馈修复

### 交付内容

- 移除视频详情 mounted/watch 链路中的 `uni.getVideoInfo`，从根源消除未包含 `uni-media` 时的 `UTSSDKModulesDCloudUniMediaIndexSwift` 缺类崩溃；不以 try/catch 掩盖原生模块缺失。
- 视频默认方向改为读取已展示封面的自然尺寸，高大于宽时进入竖屏，否则稳定回退横屏；用户手动切换后封面异步完成不会覆盖选择。
- 竖屏舞台使用直接 UVue class、全屏居中和按当前窗口/封面比例计算的播放器高度；横向视频切到竖屏后采用 `contain`，完整画面位于垂直中心。
- 图文轮播图与回退封面接入 `uni.previewImage`，支持原生全屏预览、双指缩放和按原顺序左右切换。
- 更新 information app 页面设计约束、原设计纠正记录、UI Skill request/output/decisions/review/screenshot index，并在 `check-project.sh` 新增回归门禁。

### 实际命令与结果

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `ui-ux-pro-max --design-system` + UX/SwiftUI 查询 | 0 | 采用 44px 触控、无自动播放、响应式媒体尺寸和标准系统手势；继续使用仓库 AK 蓝与系统字体。 |
| `bash apps/ak-mobile/scripts/check-project.sh`（补丁后最终） | 0 | 44 routes、4 tabs、40 components、3 platforms；Mobile Blueprint/i18n、生成 Catalog/API Client、startup snapshot、6 项升级契约、4 项打包脚本测试及新增媒体查看器门禁全部通过。 |
| `bash apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.24 完成 35 页面 iOS UTS 编译；`UniImageLoadEvent`、`uni.previewImage` 与播放器动态布局类型通过。 |
| `bash apps/ak-mobile/scripts/build-platform.sh android` | 0 | 完成 35 页面 Android class 编译。 |
| `bash apps/ak-mobile/scripts/build-platform.sh harmony` | 0 | 完成 35 页面 HarmonyOS 项目编译并进入原生工程构建阶段。 |
| 生成物符号检查 | 0 | `unpackage/dist/dev/app-ios/app-service.js` 包含 `previewImage`、`video-stage-vertical`、`orientation-probe`，不含 `getVideoInfo` 或 `DCloudUniMedia`。 |
| HBuilderX custom/standard playground 启动尝试 | 0（CLI） | 两次均完成编译后报告“已停止运行”，未形成应用启动/页面交互/控制台运行证据。 |
| `git diff --check` | 0 | 当前工作树补丁无空白错误。 |

### 截图、未完成项与风险

- 设计证据见 [request](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-viewer-feedback/request.md)、[skill output](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-viewer-feedback/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-viewer-feedback/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-viewer-feedback/review-checklist.md) 与 [screenshot index](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-information-viewer-feedback/screenshot-index.md)。索引只登记用户提供的修复前截图，没有把编译或旧资源画面冒充修复后运行证据。
- 需要用包含当前 35 页面 UVue 类的匹配自定义基座复测：进入视频详情无崩溃、横转竖后垂直居中、图集全屏缩放/左右切换、视频进度保持和离页暂停。Android/HarmonyOS 真机与 iOS 真机同样未执行。
- 封面尺寸是兼容旧基座的安全方向提示，不是视频码流元数据；若内容封面与视频方向不一致，默认方向可能需要手动切换。精确自动方向应在上传/内容 API 中提供可信 `video_width`/`video_height` 后再接入，不应重新引入可选客户端原生模块。
- 本轮未 commit、未 push，并保留其他既有未提交修改。

## 2026-08-28 Bottom Sheet 标题栏纠正交付报告

### 交付内容

- 修复 `ak-bottom-sheet` 标题栏的异常 UVue 标签换行，移除被渲染为无作用箭头的独立 `>` 文本节点。
- 标题改为占据剩余宽度，关闭按钮成为最后一个且唯一的标题栏操作，稳定放在最右侧；继续保留至少 44×44px 触控区域和 `common.close` 读屏标签。
- 同步 Mobile 设计 override、UI Skill request/output/decisions/review/screenshot index，并加入静态回归门禁。

### 验证与边界

- `bash apps/ak-mobile/scripts/check-project.sh` 退出 0：44 routes、40 components、3 platforms，Mobile Blueprint/i18n、生成 Catalog/API Client、启动快照、6 项升级契约、4 项打包脚本测试和新增 Sheet 门禁均通过。
- `bash apps/ak-mobile/scripts/build-platform.sh ios` 退出 0：HBuilderX 5.24 完成 35 页面 iOS UVue/UTS 编译。
- `bash apps/ak-mobile/scripts/build-platform.sh android` 退出 0：完成 35 页面 Android class 编译。
- `bash apps/ak-mobile/scripts/build-platform.sh harmony` 退出 0：完成 35 页面 HarmonyOS 编译、依赖安装并生成未签名调试 HAP。
- 编译产物检查确认 iOS/HarmonyOS 的 `ak-sheet-header` 只包含标题文本和关闭按钮两个子节点；`git diff --check` 退出 0。
- 用户附件已登记为修复前证据；修复后仍需匹配当前 35 页面 UVue 类的自定义基座运行截图，编译不能替代实际交互验收。
- 未执行动态字号、VoiceOver/TalkBack 或三端物理设备复测；本轮未 commit、未 push。
## 2026-08-28 App 内容管理文章编辑表单

### 变更

- `ArticleDrawer` 使用 Meta/内容双 Tab 和双语 Markdown 编辑；上传图片、图集、MP4 与 HTTPS MP4/HLS 外链可在表单预览。
- `AkFilePicker` 扩大为资源浏览器，复用现有文件查询、下载和扫描门禁；增加全部/图片/视频/其他筛选与 Blob URL 生命周期清理。
- OpenAPI、Admin adapter/generated types、Backend blocks-to-Markdown 惰性转换、Mobile 真实 body_format 读取及视频上传策略已同步。

### 当前验证边界

- 已执行并通过：`pnpm install`、Admin typecheck、Admin Vitest（130 tests）、`go test ./internal/modules/content/... ./internal/modules/storage/...`、`gofmt`。
- 待继续执行：Admin lint/build、蓝图/i18n/OpenAPI 门禁及可用后端 fixture 下的 Playwright/axe/375/768/1440 截图；未将静态或 Mock 结果当作运行验收。

### 测试反馈修复交付

- 补齐 `system.files.picker.type.*` 与 `system.files.picker.preview.*` 双语键，并同步运行时 namespace Catalog 和蓝图 i18n 事实源。
- 调整 Article Drawer 信息架构：视频来源、视频文件/外链和预览播放器归入内容 Tab；已有图文媒体在 Drawer 打开时主动下载 Blob 缩略图，关闭期间完成的异步下载会撤销 URL，避免泄漏和旧会话回写。
- 新增 `AkFilePicker.test.tsx` 和 `ArticleDrawer.test.tsx`，覆盖原始多语言 key 不泄漏、既有图文缩略图加载和视频字段 Tab 归属。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run ...` | 0 | 3 个文件、8 项针对性测试通过；首轮仅因 Node 26 jsdom 无 `localStorage` 的测试清理代码失败，移除无必要清理后通过。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 路由生成和 TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、133 项测试全部通过。 |
| `pnpm --filter @appkernia/admin generate:i18n` | 0 | 28 个 Admin namespace Catalog 重新生成。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed。 |
| Admin Blueprint / 跨端 i18n / `git diff --check` | 0 | 56 routes、145 permissions、双语 parity 和补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin healthy，`/admin-api/v1/auth/public-config` 可达；宿主和容器 `index.html` SHA-256 一致。 |

- 当前本地服务已经更新，可在 `http://localhost:4174` 强制刷新后复测。尚未使用用户真实账号生成修复后 375/768/1440 双语截图或 axe 报告，因此不把组件测试与静态产物检查表述为真实浏览器视觉验收。

### Meta 排版反馈修复交付

- 封面按钮和缩略图组成独立封面行，发布选项组成第二个响应式网格行；四个选项具有 44px 最小高度、稳定间距和窄屏自动换行。
- 评论、置顶、精选、最新的字段名不再塞入仅选中态可见的 `checkedChildren`；字段名改为常驻外置标签，Switch 内使用现有双语“是/否”，并补齐 accessible name。
- `ui-ux-pro-max` 设计系统与 React accessibility 查询、页面 override、request/output/decisions/review/screenshot index 已同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/ArticleDrawer.test.tsx` | 0 | 1 个文件、3 项测试通过，包含封面/选项分行、四个开关标签与关闭态文案断言。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、134 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed。 |
| Admin Blueprint / 跨端 i18n / `git diff --check` | 0 | 56 routes、145 permissions、双语 parity 和补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin 容器 `running healthy`，`/healthz` 与 `/admin-api/v1/auth/public-config` 可达；宿主和容器 `index.html` SHA-256 均为 `c20b8325f4b3c7b061553b4c681097f878e996729eb6e134fdecac4fc3cff23c`。 |

- 本次使用用户附件作为修复前问题证据；修复后真实账号下的 375/768/1440、双语和 axe 浏览器验证仍待复测，不以 jsdom 或生产构建替代视觉验收。

### 文章编辑保存接口修复交付

- 根因是 Admin API adapter 将表单内部 `version` 与后端要求的 `lock_version` 一起发送；后端 HTTP 解码器启用 `DisallowUnknownFields`，因此请求尚未进入文章业务校验就因未知字段 `version` 返回 `VALIDATION.FAILED`。
- 修复后更新请求只发送 OpenAPI 允许的 `lock_version`，Markdown、媒体与其他文章字段保持不变；新增回归测试验证请求体不再泄漏 `version`。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/features/content/model.test.ts` | 0 | 1 个文件、6 项测试通过，包含 PATCH 乐观锁字段映射回归。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、135 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed。 |
| 根目录执行 `GOTOOLCHAIN=local go test ./internal/modules/content/...` | 1 | 工作目录不包含 `server/go.mod`；属于命令目录错误，未执行模块测试。 |
| `GOTOOLCHAIN=local go test ./internal/modules/content/...`（`server`） | 1 | 本机 local Go 1.24.5 低于仓库要求的 1.26.5；未执行测试。 |
| `go test ./internal/modules/content/...`（`server`，自动工具链） | 0 | application、domain、repository 通过；transport/http 当前无测试文件。 |
| Admin/Mobile Blueprint、跨端 i18n、`git diff --check` | 0 | 56 routes、145 permissions、44 Mobile routes、双语 parity 与补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin/API 容器均 `running healthy`，入口 SHA-256 宿主/容器一致：`868c28b57c1c4dd310c29c4fd9b6d0c00825e031a94eeeced41fe32776c262c2`。 |

- 已复用本地 Chrome 的文章管理页检查部署结果；刷新后原登录会话过期并跳转登录页，因此没有输入用户凭据或提交会改变文章数据的 PATCH。用户重新登录后可直接复测保存。

### 文件选择器缩略图反馈修复交付

- 左侧文件表格将图片内容提升为主信息：每行显示 112×72px、`object-fit: cover` 的真实缩略图，文件名位于其下并在过长时省略；扫描状态仍独立显示，选中逻辑与安全门禁不变。
- 图片缩略图使用现有鉴权下载接口生成 Blob URL，并通过 `IntersectionObserver` 延迟到行接近可视区域时加载；不支持观察器的环境即时加载，关闭弹框或卸载行时统一释放 URL。
- 已按 `ui-ux-pro-max` 流程同步内容管理页面 override、request、skill output、decisions、review checklist 和 screenshot index。用户附件仅作为修复前问题证据，未将组件测试当作真实浏览器截图。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx` | 0 | 1 个文件、2 项测试通过，覆盖双语 key 防泄漏和缩略图下载/排列/URL 回收。 |
| `pnpm --filter @appkernia/admin lint` | 0 | 首轮因测试直接引用未绑定的 `URL.revokeObjectURL` 触发 1 个 lint error；改为断言 mock 引用后复跑通过，0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、136 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed。 |
| Admin/Mobile Blueprint、跨端 i18n、`git diff --check` | 0 | 56 Admin routes、44 Mobile routes、双语 parity 与补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | `/healthz` 与 `/admin-api/v1/auth/public-config` 可达；宿主和容器 `index.html` SHA-256 均为 `361968f9ecd95a9a79fcfdee5f174a6a2e40e81c8f72da7e7bb9a6457c23b877`。 |

- 本地地址为 `http://localhost:4174`。当前浏览器登录会话已过期，未使用用户凭据进入选择器生成修复后视觉截图；登录后的实际缩略图内容、英文长文件名和窄屏布局仍由用户复测确认。

### 文件选择器紧凑列表与上传时间筛选交付

- 任务目标：缩小资源选择器行高与缩略图，增加上传时间/文件大小列，并允许按上传开始、结束日期筛选完整文件集合。
- Admin：缩略图调整为 88×56，表格使用 small 密度和 6px 单元格纵向内边距；上传时间使用当前 locale 格式化，文件大小显示 B/KiB/MiB；搜索、日期范围和类型筛选组成可换行的顶部筛选行。
- API：`GET /admin-api/v1/files` 新增可选 `created_from`、`created_to` RFC 3339 参数，边界均包含。Transport 拒绝无效时间，Application 拒绝倒置区间，Repository 的列表与计数 SQL 使用相同条件和租户过滤。
- 数据库与安全：没有迁移；复用 `idx_files_tenant_time`。权限仍为 `storage.file.read`，只查询认证上下文租户，扫描门禁、下载鉴权、对象存储和文件选择规则均未改变。
- `ui-ux-pro-max` 的 Data-Dense Dashboard 建议用于收紧列表密度和保持筛选可见；营销 Hero、外部字体与推荐配色未用于 Admin。Master、内容页面 override 及 request/output/decisions/review/screenshot index 已同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `make sqlc-generate`（`server`） | 0 | 使用固定 sqlc v1.31.1 生成可空 `created_from`/`created_to` timestamptz 参数。 |
| `pnpm --filter @appkernia/admin generate:api` / `generate:i18n` | 0 | OpenAPI 类型与 28 个 Admin namespace Catalog 已重新生成。 |
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx` | 0 | 最终 1 个文件、3 项测试通过。首轮因 AntD 固定列表生成隐藏辅助表头导致文本匹配到两处，调整断言后通过；第二轮因 hook mock 不是 spy 退出 1，改为 `vi.fn` 后通过。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、137 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed；Node 26.5.0 仍提示项目期望 >=24 <25，但未影响退出码。 |
| `go test ./...`（`server`） | 0 | 后端全仓测试与编译通过，包含新增时间范围校验和 HTTP RFC 3339 解析测试。 |
| `go vet ./internal/modules/storageadmin/...` | 0 | storageadmin 静态检查通过。 |
| OpenAPI reference/docs、bundle、UI Skill | 0 | 321 operations 本地化契约、canonical OpenAPI 字节一致、bundle budgets 和 Skill 安装门禁通过。 |
| Admin/Mobile Blueprint、跨端 i18n、`git diff --check` | 0 | 56 Admin routes、44 Mobile routes、双语 key/placeholder parity 与补丁格式通过。 |
| `docker compose -p appkernia-news-demo build api` 与 `up -d --no-deps api` | 0 | 只重建/替换 API，不重建 PostgreSQL 或对象存储卷；API healthy。 |
| 同步 Admin dist 并重启容器 | 0 | Admin healthy，`/healthz` 与 `/admin-api/v1/auth/public-config` 可达；宿主/容器 `index.html` SHA-256 均为 `b00c6630fe2226d351e727cc09079239bf45c45256fdda99d16a028560ae5588`。 |

- 当前本地地址：`http://localhost:4174`。登录会话未用于生成修复后真实资源列表截图，因此 375/768/1440、暗色模式、英文长文件名和实际日期筛选结果仍由登录态浏览器复测，不以 jsdom 或 build 结果替代视觉验收。

### 文件选择器 footer 操作分组交付

- Modal 使用自定义 footer：上传器及策略提示位于左侧弹性区域，取消/选择位于右侧固定操作组。桌面保持同一行；767px 以下改为上下分组，右侧动作顺序不变。
- 继续复用原 `AkFileUploader`，因此上传权限、image/video 类型限制、进度、暂停/恢复/取消、错误反馈及上传后列表刷新没有复制第二套状态逻辑。
- `ui-ux-pro-max` 用于确认辅助上传操作与主要提交操作的左右分组、窄屏堆叠和键盘行为；未采用与 Admin Master 不一致的玻璃拟态、外部字体或新配色。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx` | 0 | 1 个文件、4 项测试通过，新增 footer 左右分组、body 移除上传器和原生操作按钮断言。 |
| `pnpm --filter @appkernia/admin lint` | 0 | 首轮测试写法先触发 `non-nullable-type-assertion-style`，改用 `!` 后又触发仓库禁用非空断言规则；最终改为显式 guard，复跑 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、138 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed。 |
| Admin/Mobile Blueprint、跨端 i18n、UI Skill、`git diff --check` | 0 | 蓝图、双语 parity、Skill 安装和补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin healthy，`/healthz` 与 API public-config 可达；宿主/容器 `index.html` SHA-256 均为 `965e4483944845b5c6aa83ec27fa8fe569aff4dad7c38e21c78d9340ead7f3e5`。 |

- 本轮只更新 Admin 静态资源，没有重启 API、改写数据库或对象存储。修复后真实登录态桌面/窄屏截图仍待用户复测，不以组件 DOM 测试替代视觉验收。

### 文件选择器整行选择交付

- 对所有满足 `ready + clean/skipped` 门禁的资源行启用整行交互：点击文件、上传时间、大小、扫描状态等任意列都会选中该文件；最左侧 Radio 继续保留为明确的选中状态指示。
- 可选行加入 pointer、键盘焦点和 Enter/Space 操作，并通过 `aria-selected` 向辅助技术同步状态。不可选行保持禁用光标且不会响应点击或键盘，扫描门禁没有放宽。
- `ui-ux-pro-max` 用于核对数据表格整行交互、可见焦点和键盘可达性；保留既有 Admin Master，不引入新的视觉语言。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx` | 0 | 1 个文件、5 项测试通过；新增任意列点击、`aria-selected`、Space 键和确认按钮联动断言。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 32 个测试文件、139 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed；Node 26.5.0 仍提示项目期望 >=24 <25，但未影响退出码。 |
| Admin/Mobile Blueprint、跨端 i18n、UI Skill、`git diff --check` | 0 | 蓝图、双语 parity、Skill 安装与补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | 容器 `running healthy`，`/healthz` 与 API public-config 可达；宿主/容器入口 SHA-256 均为 `963df6c732eeb924766367d5d68a0b32013fe61aa93d4a35f6835b6d3c74803e`。 |

- 本轮只部署 Admin 静态资源，没有重启 API、改写数据库或对象存储。真实登录态视觉与 axe 验收仍待复测，不以组件测试替代浏览器验收。

### 文件选择器选中态与上传图标交付

- 选中行背景从 Ant Design 根据 near-black 主色派生的深色值，改为文件选择器局部浅蓝 `#eff6ff`；hover 使用 `#e8f2ff`。主文字/次级文字对比度实测为 16.47:1/7.77:1，文件名、时间和大小均保持可读。
- Radio、`aria-selected`、整行 click 和 Enter/Space 行为不变，选中状态不只依赖背景色；`ready + clean/skipped` 安全门禁没有放宽。
- `AkFileUploader` 的文字按钮加入已有 `UploadOutlined` SVG 图标，未添加 emoji 或新包，原上传权限、文件类型限制、策略提示、进度与恢复逻辑不变。
- `ui-ux-pro-max` 用于核对 4.5:1 对比度、非颜色状态指示和一致 SVG 图标；OLED 深色推荐与营销比较表结构不适用于本页，继续服从 Admin Master。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx src/components/AkFileUploader.test.tsx` | 0 | 2 个文件、6 项测试通过；覆盖选中行状态类和上传 SVG 图标。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 33 个测试文件、140 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed；Node 26.5.0 仍提示仓库期望 >=24 <25，但未影响退出码。 |
| Admin/Mobile Blueprint、跨端 i18n、UI Skill、`git diff --check` | 0 | 蓝图、双语 parity、Skill 安装与补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | 容器 `running healthy`，`/healthz` 与 API public-config 可达；宿主/容器入口 SHA-256 均为 `1454df53e162698b444d1e359e0c0454c07e664653990f881f43914abe81132f`。 |

- 本轮只部署 Admin 静态资源，没有重启 API、改写数据库或对象存储。用户附件作为修复前证据；真实登录态视觉与 axe 验收仍待复测，不以 jsdom 断言替代浏览器验收。

### 文件选择器可调窗口与分栏交付

- 左侧文件列表与右侧预览使用 Ant Design `Splitter`，桌面端最小宽度为 420px/280px；窄于 900px 时转为上下布局并设置 180px/140px 最小高度。分隔条可使用鼠标拖动，左右内容不会被压缩到不可用。
- 右侧预览新增关闭与重新展开操作。关闭只回收大预览 Blob URL，不清空已选文件；重开后继续通过原鉴权下载接口和扫描安全门禁加载图片或视频。
- Modal 使用现有 `width`、`style`、`styles`、`modalRender` 完成可调窗口：右上角最大化/还原，标题栏 Pointer Events 拖动移动，右下角手柄拖动缩放；几何状态按视口变化重新约束，保留 8px 边距和 800×520px 桌面最小尺寸。
- 键盘路径包括最大化按钮、预览开关、`Alt + 方向键` 移动和方向键缩放；焦点样式与双语 accessible name 已补齐。未引入 react-draggable、react-resizable 等新依赖。
- `ui-ux-pro-max` 用于核对分栏最小尺寸、焦点可见性、拖动手柄和状态恢复；未采用其与 Admin Master 不一致的营销视觉和高饱和配色建议。Master、页面 override、request、skill output、decisions、review checklist 和 screenshot index 已同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx src/components/AkFileUploader.test.tsx` | 0 | 2 个文件、8 项测试通过；覆盖分隔条、预览关闭/展开、最大化/还原、标题栏拖动和右下角缩放。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 33 个测试文件、142 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 生产构建成功，9,936 modules transformed；Node 26.5.0 仍提示仓库期望 >=24 <25，但未影响退出码。 |
| Admin/Mobile Blueprint、跨端 i18n、UI Skill、`git diff --check` | 0 | 蓝图、双语 key/placeholder parity、Skill 安装和补丁格式通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin `running healthy`，`/healthz` 与 API public-config 可达；宿主/容器入口 SHA-256 均为 `9f65d8bf0695d0223d437f80f118edda2827dda4be11d8a8c168281c04c374db`。 |

- 本轮仅更新 Admin 静态资源，没有重启 API、改写数据库或对象存储。真实登录态下的拖拽手感、375/768/1440、暗色模式和 axe 结果仍待浏览器复测，不把 jsdom 或构建结果表述为视觉验收。

### 文件选择器多视图与操作样式交付

- 筛选行最右侧使用 Ant Design Dropdown 切换 `grid/table/thumbnail`，三个菜单项均由语义 SVG 图标和双语文字组成。默认缩略图视图保持 88×56px；网格卡片保持文件名常驻底部并在 hover/focus 覆盖层显示文件类型、上传时间、大小和扫描状态；表格视图使用 16×16px 图片/类型图标与实测 24px 数据行。
- 三种视图不复制数据或安全状态：继续使用同一个 `useAdminFiles` Query、`selected`、预览 Blob 生命周期、确认按钮及 `ready + clean/skipped` 门禁。视图切换卸载旧缩略图时会撤销 Blob URL。
- 折叠后的预览恢复按钮改为右边缘垂直居中的 icon-only 左箭头，Tooltip 与 accessible name 保留完整双语说明；已选提示移除 Alert 边框/彩色表面，使用浅灰底和深灰字；Modal 关闭默认按钮由最大化/关闭等高组合按钮替代。
- 浏览器 axe 首轮暴露资讯筛选区四个既有 Select 缺少 accessible name，已使用现有字段翻译键补齐；文件选择器 axe 严格限定选择器范围，避免把背景 Drawer 的既有表单问题混入本轮结果。
- `ui-ux-pro-max` 用于核对 Data-Dense Dashboard、稳定 Hover、图标一致性、焦点和对比度；未采用其玫红品牌色、外部字体或营销 Portfolio 页面结构。Master、内容页面 override 和五类 artifacts 已同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/AkFilePicker.test.tsx src/components/AkFileUploader.test.tsx` | 0 | 2 个文件、9 项测试通过；覆盖三种视图切换、网格 Meta、16px 表格身份、预览箭头、已选提示和窗口操作组。 |
| `pnpm --filter @appkernia/admin lint` | 0 | ESLint 0 error / 0 warning。 |
| `pnpm --filter @appkernia/admin typecheck` | 0 | 56 routes 生成完成，TypeScript strict 检查通过。 |
| `pnpm --filter @appkernia/admin test` | 0 | 33 个测试文件、143 项测试全部通过。 |
| `VITE_AK_API_BASE_URL=/admin-api/v1 pnpm --filter @appkernia/admin build` | 0 | Vite 8.1.5 生产构建成功，9,936 modules transformed；既有大 chunk warning 和 Node 26.5.0/仓库 Node 24 范围 warning 不影响退出码。 |
| `AK_E2E_BASE_URL=http://127.0.0.1:4174 AK_E2E_FILE_PICKER_ONLY=1 /Users/payhon/.venv/3.12/bin/python apps/ak-admin/scripts/e2e_content_management.py` | 0 | 真实 Chromium + API fixture 通过；选择器 axe serious/critical 0，行高 24px，箭头宽 24px且垂直偏移 0，输出 4 张 1440×900 截图。 |
| Admin/Mobile Blueprint、跨端 i18n、UI Skill | 0 | 蓝图、双语 key/placeholder parity 和 Skill 安装检查通过。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin `running healthy`，`/healthz` 返回 `ok`；宿主/容器入口 SHA-256 均为 `08dbfca1a86eac67146dc7a7ea1b40512b7ac24915d85629b2d13011b62828fb`。 |

- 浏览器证据位于 `apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-content-management/screenshots/`，使用的是受控 API fixture；真实账号、实际对象存储、英文长文件名、暗色模式和 768/375 验收仍需登录态复测。本轮仅更新 Admin 静态资源，API、数据库和对象存储未重启或改写。

### 文件选择器表头、全局下拉与共享缩放能力交付

- 文件列表表头与预览分界处的右上圆角已清零。Select 和 selectable Dropdown/Menu 通过 AntD token 与全局 popup 状态共同使用浅蓝选中背景、深蓝文字，避免 active 状态再次退回暗背景。
- 核实 Ant Design 6 `ModalProps` 无 resize 属性后，新增共享 `AkModal`：调用方通过 `resizable` 传入宽高、最小尺寸和更新回调。右下角 30px 命中区没有图标，弧线仅在 hover/focus/drag 可见，拖动时高亮；支持 Pointer Capture 和方向键调整。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm exec vitest run src/components/AkModal.test.tsx src/components/AkFilePicker.test.tsx src/app/theme.test.ts` | 0 | 3 个文件、11 项测试通过；覆盖共享缩放指针/键盘、禁用态、文件选择器集成和全局主题 token。 |
| `pnpm lint` / `pnpm typecheck` | 0 | ESLint 0 error / 0 warning；56 routes 生成与 TypeScript strict 检查通过。 |
| `pnpm test` | 0 | 35 个测试文件、146 项测试全部通过。 |
| `pnpm build` | 0 | Vite 8.1.5 生产构建成功，9,937 modules transformed；既有大 chunk 与 Node 26/项目 Node 24 范围 warning 不影响退出码。 |
| Admin Blueprint、跨端 i18n、`git diff --check` | 0 | 契约、双语 parity 与补丁格式通过。 |
| `AK_E2E_BASE_URL=http://127.0.0.1:4174 AK_E2E_FILE_PICKER_ONLY=1 ... e2e_content_management.py` | 0 | Chromium + API fixture 通过：表头圆角 0px，下拉背景/文字为 `rgb(220,234,255)`/`rgb(23,62,122)`，窗口 1120×720→1160×744，axe serious/critical 0。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | Admin healthy，`/healthz` 返回 `ok`；宿主/容器入口 SHA-256 均为 `785da6c799011790f965c7fba8cb38acdfb7729b46048b413da8d59a4c45272a`。 |

- 截图证据新增 `file-picker-select-menu-zh-CN.png`、`file-picker-select-open-zh-CN-1440.png` 与 `file-picker-resize-active-zh-CN-1440.png`。本轮未重启 API、改写数据库或对象存储；fixture 证据不冒充真实账号/对象存储联调。

### 微信分享配置申请指引交付

- 在分享配置 Drawer 标题旁增加问号帮助按钮；弹框以五步纵向流程解释微信开放平台申请、三端身份、审核/AppID、AppKernia 绑定/预检/重新打包和真机验收。
- 文案与视图解耦：双语说明进入 `share_configs` Catalog，8 个已核验为 HTTP 200 的微信开放平台/DCloud HTTPS 地址进入 typed Provider registry。所有链接带外链图标并安全打开新浏览上下文。
- 首轮 axe 暴露 waiting Steps 与默认链接对比度不足，已在 guide modal 局部改为深色标题和带下划线深蓝链接；最终桌面/移动端 serious/critical 均为 0。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `pnpm --filter @appkernia/admin exec vitest run src/components/WechatShareApplicationGuide.test.tsx` | 0 | 1 个文件、2 项测试通过；覆盖双语弹框、五步内容和 8 个安全新窗口链接。 |
| `pnpm --filter @appkernia/admin lint` / `typecheck` / `build` | 0 | ESLint、TypeScript strict 和 Vite 8.1.5 生产构建通过。 |
| `pnpm --filter @appkernia/admin check` | 0 | 37 个测试文件/151 项测试、bundle budget、OpenAPI docs/reference 和 Admin Blueprint 全部通过。 |
| 跨端 i18n、Admin Blueprint、`git diff --check` | 0 | 双语 key/placeholder parity、47 menus/57 routes/152 permissions 等契约和补丁格式通过。 |
| 8 个外部文档地址逐一 `curl -L` | 0 | 微信开放平台首页/应用中心/创建指引/三端接入/分享收藏以及 DCloud 构建配置均返回 HTTP 200。 |
| `AK_E2E_BASE_URL=http://127.0.0.1:4174 ... e2e_share_configuration.py` | 0 | Chromium + API fixture：1440/375 中文及 375 英文指引通过；8 个 scoped axe serious/critical 0，控制台错误 0。 |
| 同步 dist、重启 `appkernia-news-demo-admin-1` | 0 | 本地 Admin 页面 HTTP 200，容器 `running healthy`；宿主/容器入口 SHA-256 均为 `1e7af837cbd1b17443c966c80fd9757b3238f883965eadff031efbb7bd813b00`。 |

- 截图新增 `share-guide-zh-CN-1440.png`、`share-guide-zh-CN-375.png`、`share-guide-en-US-375.png`。这是浏览器 UI/契约证据，不替代微信已审核 AppID、发布签名、Universal Link 和安装微信真机上的分享验收。

### Admin 页面 Message 垂直节奏交付

- 交付：为所有 `.ak-page-container` 内相邻的 Ant Alert 与后续内容建立共享 16px 最小间距，覆盖连续提示、卡片、筛选、表格包装和详情内容；Modal/Drawer 与 Ant `Space` 保持原有局部布局职责。
- 范围证据：27 个页面文件、66 个页面 Alert，54 个页面范围非 Space 提示；Master、升级中心 override、UI Skill request/output/decisions/checklist、截图索引及静态回归测试均已同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| Alert AST/source audit | 0 | 27 个页面文件、66 个页面 Alert；54 个属于页面容器且不由 `Space` 托管。 |
| `pnpm --filter @appkernia/admin exec vitest run src/app/page-message-rhythm.test.ts` | 0 | 1 个文件、2 项 CSS 作用域/16px 契约测试通过。 |
| 隔离提交快照 `npm run check` | 0 | ESLint、TypeScript strict、36 个测试文件/148 项测试、Vite production build（9937 modules transformed）、bundle、OpenAPI reference/docs 与 Admin Blueprint 全通过。 |
| 旧 `e2e_mobile_releases.py` 首轮 | 1 | 在已改为操作下拉菜单的应用列表上仍查找旧“升级中心”文本按钮；属于过期测试定位，未冒充通过。 |
| `e2e_page_message_rhythm.py`（Chromium 1440/375） | 0 | 两个视口实测 Alert→Card 均为 16px；无页面溢出、无控制台错误、axe serious/critical 为 0。 |
| 当前聚合工作区 `pnpm --filter @appkernia/admin check` | 0 | 38 个测试文件/153 项测试，generate、OpenAPI reference、lint、typecheck、build、bundle、OpenAPI docs、Admin Blueprint 全通过；未混入本次隔离提交。 |
| `python3 blueprint/scripts/validate_i18n_contract.py`、`git diff --check` | 0 | `zh-CN`/`en-US` 契约及补丁格式通过。 |
| 同步 Admin dist 并重启本地容器 | 0 | `http://localhost:4174/healthz` 返回 `ok`，Admin 容器 healthy。 |

- 截图：[桌面 1440](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-page-message-rhythm/screenshots/upgrade-center.zh-CN.light.1440.png)、[移动 375](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-page-message-rhythm/screenshots/upgrade-center.zh-CN.light.375.png)、[几何与 axe 结果](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-page-message-rhythm/screenshots/e2e-results.json)。受控 API fixture 不代表生产账号联调；Firefox/Safari 未验证。

### 推送渠道申请与对接指引交付

- 页面标题最右侧增加问号帮助按钮，打开后以九个 Provider Tab 呈现申请步骤、表单字段来源、风险检查和官方资源。帮助不依赖当前 App 选择，不会读取或显示任何现有凭据。
- 178 个 guide 翻译键分别进入权威 `zh-CN`/`en-US` Admin Catalog 并生成独立 `push_channels` 语言包；Provider 顺序、字段清单和 URL 由 typed registry 管理，官方域名通过 allowlist 测试。
- `ui-ux-pro-max` 用于核对帮助入口、弹框信息层次、键盘 Tabs、外链辨识、72ch 可读宽度和 375px 布局；设计决策和未完成的登录态截图门禁已记录到 `AKADM-push-channels` artifacts。
- 官方资料核验发现并修复两个服务端漂移：荣耀 OAuth 地址从已失效的 `.com.cn` 改为官方 `.com`；小米四个海外区域改为当前 `sgp/fr/idmb/ru-api.xmpush.global.xiaomi.com`，中国区维持 `api.xmpush.xiaomi.com`。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `vitest run ...provider-application-guides.test.ts ...PushProviderApplicationGuide.test.tsx` + `go test ./internal/modules/push/provider` | 0 | Admin 2 个文件/4 项测试通过；Provider 包通过。首次 jsdom 缺少 `ResizeObserver` 导致 2 项失败，补齐测试环境并迁移到 AntD 6 非废弃属性后复跑通过。 |
| `npm --prefix apps/ak-admin run check` | 0 | generate、OpenAPI reference、lint、TypeScript strict、42 个测试文件/164 项测试、Vite production build（9,961 modules）、bundle、OpenAPI docs 和 Admin Blueprint 全部通过。 |
| `go test ./...` | 0 | Go 全仓所有含测试包通过；Push Provider 地址回归测试包含在内。 |
| i18n、Mobile Blueprint、Backend Blueprint、`git diff --check` | 0 | 双语 key/placeholder parity、44 Mobile routes、21 对迁移及补丁格式通过。Backend 首次误用不存在的 `scripts/validate_blueprint_specs.py` 路径退出 2，随后使用实际 `tools/validate_blueprint.py` 复跑通过。 |
| `docker compose -p appkernia-news-demo build api worker admin` + `up -d --no-deps api worker admin` | 0 | 三个本地镜像重建并替换，PostgreSQL 未重启、未执行数据改写。 |
| 本地 HTTP 与容器检查 | 0 | Admin/API healthy，Worker running；Push SPA route、`/healthz`、`/admin-api/v1/auth/public-config`、API live/ready 均为 HTTP 200。首次探测了不存在的 `/admin-api/v1/config/public` 并得到预期 404，随后按 OpenAPI 更正路径。 |

- 当前用户浏览器仍在登录路由，因此本轮没有伪造登录态截图或 axe 结果。登录后需人工确认九个 Tab、官方外链、英文长文案及 375px 实际视觉；厂商账号审批、凭据预检和真机推送不属于本次 UI 帮助弹框验收。

### App 消息推送现状架构图交付

- 新增 `docs/manual/app-message-push-architecture.md`。架构图以当前实现而非目标蓝图为准，区分同步 API 事务、River 异步边界、站内消息与离线 Push、Provider 受理与设备展示、opened 与已读。
- 文档明确 River 直接使用 PostgreSQL `river_job`，不是 Redis 队列；三类 Job 均在 `notifications` 队列消费，Worker 并发为 8，扇出分页为 500，单设备投递和 River Job 的最大尝试均为 5。
- 运行态检查确认本地 Worker 存活，但 `AK_PUSH_ENABLED` 与 `AK_PUSH_ADAPTER` 均未显式设置：Push Kill Switch 默认为 false，开发 Adapter 默认为 `local-mock`。因此未宣称本地当前会调用真实厂商。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| `rg` / `sed` 审计 notificationadmin、push、bootstrap、ak-worker、迁移与配置 | 0 | 确认发布事务、三类 River Job、分页扇出、单设备发送、归一结果、重试和打开统计调用链。 |
| API/Worker 容器环境白名单检查 | 0 | 未发现 `AK_PUSH_ENABLED` 或 `AK_PUSH_ADAPTER` 显式值；结合 `optionalBool` 和开发默认逻辑，确定当前为 Kill Switch off + local-mock。未输出其他环境变量或秘密。 |
| Mermaid 10.9.8 `parse` | 0 | 架构图语法解析通过。首次直接解析缺少 DOM，随后使用仓库已有 jsdom 提供解析环境后通过；未安装新依赖。 |
| 相对链接存在性检查、`git diff --check` | 0 | 8 个代码索引全部存在，Markdown 代码围栏完整且补丁格式通过。 |

### iOS 启动公开配置向后兼容修复

- 根因：本地运行中的 API 版本落后于当前 Mobile/OpenAPI。实测 `GET /api/v1/public/config` 返回 HTTP 200，但 `data` 只有 `startup`，没有 `share` 和 `push`；UTS 泛型强转不做运行时字段补齐，直接解引用导致 iOS 报 `undefined is not an object`。
- 修复：公开配置 Repository 对 `share`、`push` 及其列表做显式兼容归一化。缺字段时共享和 Push 能力均安全关闭，不影响应用主启动；当前完整响应仍按原协议映射。
- 回归：Mobile 静态门禁禁止直接访问未归一化的 `response.data.share.providers` / `response.data.push.*`，并锁定 Push 默认 `false`。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| 本地 `curl GET /api/v1/public/config` | 0 | HTTP 200；`data_keys` 为 `app_id/app_type/appid/default_locale/name/registration_enabled/registration_verification_mode/startup`，`share`/`push` 均缺失，复现真实版本错配。 |
| `apps/ak-mobile/scripts/check-project.sh` | 0 | Mobile Blueprint 45 routes、跨端 i18n、生成 Client、快照、升级 6 个 SemVer case、打包脚本 4 项测试及新增兼容门禁通过。 |
| `apps/ak-mobile/scripts/build-platform.sh ios` | 0 | HBuilderX 5.24 完成 36 页面和 `ak-permissions`/`ak-push` iOS UTS 编译，`ready in 64976ms`。 |
| HBuilderX custom playground + iPhone 16 Pro / iOS 18.6 | 0 | 自定义基座重签、安装、36 页面资源同步和启动成功；在旧 API 缺字段条件下进入浏览页。 |
| Simulator unified log 定向检查 | 0 | 最近 10 分钟匹配 `TypeError`、`undefined is not an object`、`share.providers`、`push.enabled`、fatal、exception 为 0。 |
| `git diff --check`（本轮文件） | 0 | 补丁格式通过。 |

- 运行截图：`output/playwright/ak-ios-public-config-compatibility.png`，SHA-256 `b69aacd487396ce5072d00cebb130faeea424d481bc3757039321c323c894493`。这是本地 iOS 模拟器证据；未执行 iOS 真机、Archive/TestFlight 或真实 APNs/厂商 Push 验收。

### 移动端扫码、客户端配置与双语文档交付

- Backend：`app.application_scanner_configs` 使用应用级租户联合外键和独立 `lock_version`；Admin 读写端点与 Mobile 公开配置使用同一规范化逻辑。更新审计只记录配置摘要，不包含扫码内容。扫码格式、事件和处理结果保持代码/OpenAPI 稳定枚举，不引入业务字典或服务端可执行处理器。
- Admin：`AppClientConfigurationModal` 以 `ClientConfigTabDefinition` 注册分享/扫码 Tab；分享绑定原能力保持不变，扫码配置支持逐行校验、服务端行错误映射、冲突提示和成功后锁版本刷新。入口对任一客户端配置读取权限可见，编辑能力由各自更新权限控制。
- Mobile：`ak-scanner` 封装相机权限和 `uni.scanCode`，公开 captured/parsed/resolved/cancelled/failed 事件与可释放处理器订阅；协调器按业务处理器、可信网页、结果兜底顺序执行。WebView 路由只接收一次性 token，每次导航都重新校验且无原生消息桥。
- Docs：新增中英文 `mobile-components/scanner` 与 `guide/client-configuration`，补齐导航、首页、权限与安全交叉链接；公开 OpenAPI 由 `apps/ak-docs/scripts/sync-openapi.mjs` 从 `server/openapi/openapi.yaml` 同步。

| 命令 / 阶段 | Exit | 真实结果 |
|---|---:|---|
| Backend `make check` | 0 | gofmt、go vet、全仓 Go 单元与契约测试通过。 |
| PostgreSQL 18 migration `26 -> 25 -> 26` 与 scanner integration | 0 | 表和两项权限随 up/down 正确创建、删除并恢复；扫码配置真实 SQL、乐观锁、租户隔离与审计测试 1/1 通过。临时容器已自动删除。 |
| Admin `npm run check && npm run check:ui-skill` | 0 | lint、strict typecheck、45 个 Vitest 文件/183 项测试、生产构建、Bundle 预算、OpenAPI 字节一致性、Blueprint 与 UI Skill 检查通过。 |
| Mobile `bash scripts/check-project.sh` | 0 | Scanner 契约原交付 6/6；本次补充 iOS 模拟器前置降级后定向 Scanner 契约 7/7，Blueprint、i18n、Catalog/Client/启动快照保持 current。 |
| HBuilderX `build-platform.sh ios` / `android` / `harmony` | 0 / 0 / 0 | 最终 38 页面 iOS UTS、Android class、HarmonyOS UVue/UTS 编译通过；HarmonyOS 为未签名调试 HAP。 |
| Docs `pnpm check` | 0 | OpenAPI 同步、中英文 84 页面语言配对、147 项 API Reference、lint、typecheck、format 和生产构建通过。 |
| Admin/Mobile Blueprint 与跨端 i18n | 0 | Admin 48 menus、59 routes、166 permissions、84 tables、222 APIs + 13 deltas、43 page contracts；Mobile 46 routes、43 components、26 tasks；中英文 key/placeholder 一致。 |
| `git diff --check` | 0 | 当前聚合工作树补丁格式通过。 |
| iOS crash report / 原生符号核对 | 0 | 两份 21:17 `.ips` 均为主线程 `EXC_BAD_ACCESS`，首个 UTS 源码帧 `startScanByJs`；旧基座缺失 `uni-scanCode`，重建后的 `DCloudUTSExtAPI.framework` 包含 `DCloudUniScanCode`、`scanCodeByJs` 与 ML Kit Barcode 符号。 |
| `build-custom-base.sh ios-simulator` + 安装 | 0 | HBuilderX 5.24 云打包 38 页面 `Pandora_simulator_debug.app`，安装并启动 `com.appkernia.mobile` 成功。 |
| `scanner-ios-simulator.yaml` | 0 | iPhone 16 Pro / iOS 18.6，1/1、16 秒：首次隐私确认、点击扫码、安全降级提示和首页存活均通过；修复后无新增 `.ips`。 |
| `build-custom-base.sh android` / `adb install -r` / HBuilderX launch | 0 / 0 / 0 | APK 含 `ak-scanner`、ML Kit 二维码和一维码模型；vivo V2545A / Android 16 安装成功，38 页面资源同步并启动首页。 |
| Android 真机二维码与取消 | 0 | 原生相机实际识别二维码并显示“二维码”结果弹层；再次拉起扫描器后点击关闭，应用无错误返回首页。为遵守扫码内容不持久化约束，含原文的临时截图已删除；取消返回截图保留在 `output/device-tests/android-v2545a/after-cancel.png`。 |
| Harmony 38 页面构建 / 真机安装 | 0 / 语义失败 | 编译和 unsigned HAP 制作成功；`ALN-AL00` / OpenHarmony API 24 安装返回 `code:9568320 / no signature file`。未生成或持久化新的签名凭据。 |

- 静态/编译结果没有被当作真机结果。Android 物理设备二维码识别、结果展示和取消分流已通过；仍待独立验收：Android 一维码、iOS/HarmonyOS 物理设备二维码和一维码、相机首次/拒绝/设置返回、白名单 WebView 与越界重定向关闭、复制反馈、读屏、动态字号、高对比度、减少动效，以及 Admin 双语键盘/窄屏浏览器截图。HarmonyOS 真机继续受 `com.appkernia.mobile` 调试签名阻断。
- 探索失败均保留真实边界：iOS 首次复编译因 HBuilder 缓存目录瞬时 `ENOENT` 退出 1，原命令重跑通过；三次早期 Maestro 分别被遗留文章详情、首次隐私页和未暴露的同意按钮定位阻断，改为清空状态并点击确定坐标后最终通过。Android Maestro 因该双显示 vivo 设备的 `tcp:7001 closed` / gRPC driver 关闭超时退出 130，后续改用 HBuilderX 资源同步和指定显示 ADB 操作取得真机结果，未把自动化驱动失败算作产品失败。
- Docs `pnpm check` 首轮因非 TTY 依赖目录确认退出 1，`CI=true pnpm check` 重跑退出 0：147 个 API 引用、0 lint/type/format 错误、中英文 84 页面构建与语言配对通过。宿主 Node 26.5.0 高于仓库要求的 Node 24，正式 CI 仍应使用 Node 24。含二维码原文的临时真机截图已删除，仅保留不含扫码内容的取消截图和测试条码目标。
