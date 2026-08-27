# AppKernia 实施状态

更新时间：2026-08-26（Asia/Shanghai）

## 总体状态

| Surface             | 当前可交付状态                                                                                                                                                                                                         | 下一依赖                                                                                                  |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Backend             | Admin 蓝图所需 Backend 契约已完成至 `AKADM-310`：认证、自作用域、业务管理、API Client/Webhook、访问规则/服务状态、完整 MFA/OAuth 绑定均形成真实闭环                                                                    | 当前 Admin backlog 已收口；生产 Adapter 联调见风险                                                        |
| Admin               | `AKADM-000`—`AKADM-310` 依赖图内全部 Task 已实现并通过最终硬化门禁；公开 `/openapi/` 与侧栏底部文档/System 工具入口已完成 Node 24、Docker/Nginx 和 Chromium 验收                                                      | 生产部署、真实 Bearer 手工调用、第三方 Scalar 请求客户端瞬态 axe 与跨浏览器验收见风险                     |
| Mobile              | 30 个页面已完成 AK UI 统一刷新；Android/iOS/Harmony 自定义原生产物统一为 `com.appkernia.mobile` 与 AppKernia 品牌，Android vivo 真机、iOS 18.6 模拟器、HarmonyOS API 22 官方模拟器均完成匿名首帧运行验证 | iOS/Harmony 真机签名、三端安全存储/商店跳转/升级/完整交互仍需物理设备验收；Harmony 自动签名需连接在线设备 |
| Docs / Website      | `apps/ak-docs` 已形成 Rspress 2 双语官网与文档站：68 个静态页面、零门槛向导、核心 API、在线 OpenAPI/System 菜单指南、AK Mobile 组件、搜索、暗色与响应式体验通过门禁；线上首页包含 9 个内容区、6 张特性卡、9 项技术栈与 Admin/Mobile 双 Slider | GitHub Pages run `31475956211` 发布成功；`appkernia.com` 仍待 DNS、域名验证和 Custom Domain 绑定 |
| Cross-platform i18n | 蓝图契约通过；Admin 与 Mobile 均有 `zh-CN`/`en-US` 语言包、运行时切换与服务端用户偏好接线                                                                                                                              | Mobile 三端长英文/运行时视觉验收                                                                          |

## 2026-08-15 快学AI微信公众号草稿工作流

- 已从 skills.sh 对应的 `staruhub/claudeskills` 仓库安装并校验项目级 `wechat-article-writer` Skill；另新增 `quicklearn-ai-wechat` Skill，把资料检索与事实核验、作者口吻写作、微信安全 HTML 构建、封面/正文图片处理和草稿箱投递串成单一工作流。
- 发布链路固定为“本地 Skill → 既有 SSH 密钥连接 → 白名单服务器 CLI → 微信公众号 API”。服务器仅安装 root 可执行的草稿 CLI，不开放新公网端口或常驻服务，也不提供群发、正式发布或删除草稿命令。
- AppSecret 只通过服务器交互式无回显输入写入 `/etc/quicklearn-wechat/credentials.json`，文件权限为 `0600 root:root`；仓库、命令参数、日志和草稿包均不保存密钥。默认封面和 Markdown 转微信内联 HTML 工具已纳入项目级 Skill。
- 本地无图片/带正文图片草稿包构建及 dry-run 均通过；SSH 二进制传输、远端临时目录创建/精确清理和微信 API 调用边界已实测。首次连接曾因微信未认可 IP 白名单返回 `40164`；重新保存白名单后，`doctor` 已成功取得有效 Token 并读取草稿数量。
- 首篇 AppKernia 介绍文章已由本地 Markdown 构建器生成微信安全 HTML，经服务器上传默认封面并调用草稿新增接口；随后通过草稿详情回读确认标题和单篇文章数量一致，公众号草稿总数由 20 增至 21。仅完成草稿创建，未调用群发或正式发布接口；微信编辑器与手机预览仍需人工检查。

## 2026-08-12 移动端自动升级

- 新增无 uniCloud、通知栏和第三方运行时依赖的 `uni_modules/ak-upgrade`。启动流程在隐私同意和公开 App 配置完成后、会话恢复前执行版本门禁；严格比较三段式 SemVer，普通网络失败 fail-open，AppID 配置不一致和已发现的强制升级保持阻断。
- Android 内部发布通过五分钟签名地址下载 APK，只携带 `X-AppID`，签名失效最多刷新一次；下载状态支持进度、取消、重试和临时文件清理，完成后使用 `uni.installApk`。Android/iOS/Harmony 外部发布按市场优先级尝试安全 scheme，再回退 HTTPS 地址；拒绝危险协议。
- About 页面展示本地版本和服务端最新版本并可手动检查；新增双语升级 modal-page、`AkProgress` 组件、页面/API/路由蓝图契约和确定性 UTS DTO 生成器。UI Skill 采用现有 AK 设计系统，保留 44px 触控目标、安全区、文本状态和 reduced-motion 约束；未伪造截图。
- 后端公开版本投影新增 `delivery_mode`、过滤排序后的 `store_list` 和签名/HTTPS `upgrade_url`；应用层限制 `uni_app_x` WGT 及 iOS/Harmony 内部原生包，新增两个稳定 422 错误码。已有 `000017_app_upgrade_center` 字段足够，本轮没有数据库 Migration、权限或字典变更。
- Admin 根据应用类型隐藏 `uni_app_x` WGT，限制非 Android 内部包，平台切换会清理内部文件；历史不兼容记录保留查看、下线和草稿删除能力，但隐藏发布/重新发布入口。
- 最终 Node 24.18.1 `make check` 退出 0：Backend vet/tests、Admin 30 个 Vitest 文件 / 130 项测试、production build、Mobile 39 routes/34 components 静态门禁、统一 i18n 和 Docs 68 页构建全部通过。PostgreSQL `mobileprofile` integration 退出 0；HBuilderX 5.06 Android/iOS/Harmony 三端 29 页面 UTS 编译退出 0。
- 验证边界：三端结果是编译证据，不是物理设备安装或应用商店跳转证据；Harmony 仅生成未配置包名/签名的原生工程。Android APK 安装权限/取消/失败、三端市场 fallback、强制升级返回拦截和双语截图仍待真机完成。

## 2026-08-11 文档站发布在线 OpenAPI 与 System 菜单指南

- 服务端 API 栏目新增中英文“在线 OpenAPI 文档与系统菜单”页面，集中说明 Admin 底部文档/System 入口、桌面级联与移动内联层级、接口面/模块/接口三级分组、标题本地化、真实请求安全边界、顶部提示的会话级关闭和自托管构建约束。
- 双语 API 首页和总体架构页已增加交叉入口，`_meta.json` 将新页面放在 API 概览之后；API 文档检查器仅为 `/openapi/`、`/api/`、`/admin-api/` 与 `/internal/` 网关基础路径增加非 operation allowlist，真实 operation 路径仍逐项对 canonical OpenAPI 校验。
- Node 24.18.1 下 `pnpm --filter @appkernia/docs check` 退出 0：121 个 API 路径引用、lint、TypeScript、Prettier、双语 parity、死链/锚点、68 页 Rspress build 和 Sitemap 全部通过。Mobile 蓝图与统一 i18n validator、`git diff --check` 同样退出 0。
- 内容提交 `32fc41e` 已推送 `origin/main` 与 `gitee/main`；GitHub Pages run `31475956211` 的 build/deploy 分别用时 56/45 秒并成功。中英文新页面、API 首页与 OpenAPI YAML 均为 HTTP 200，线上标题匹配，发布 YAML SHA-256 与 canonical 同为 `efc4a2050a7cbe8f31fa88f23306ebc545783fa2e34dba5622e3ed8f348bd8df`。
- 当前公开地址仍为 `https://payhon.github.io/AppKernia/`，Pages 为 workflow 模式且强制 HTTPS，`cname=null`。本轮复用现有文档主题，只新增 Markdown 内容和导航顺序，没有视觉组件或样式变更，因此没有新增 UI Skill 产物、截图、axe 或物理设备证据。

## 2026-08-11 管理端 OpenAPI 模块分组与接口标题双语化

- canonical OpenAPI 新增 3 个有序接口面、31 个有序业务模块及唯一 operation tag；导航由 Scalar 原生 `tags` / `x-tagGroups` 渲染为“接口面 → 业务模块 → 接口”，模块接口列表默认折叠。路径分配 validator 对 App 子资源优先级和所有已注册模块执行发布门禁。
- canonical 的 `paths` 下有 278 个直接 operation，另有 3 个会被 Scalar 渲染的 `components.pathItems` 复用 operation；为避免残留未分组英文项，本轮实际对 281 个渲染标题全部完成唯一模块归属和双语覆盖。此处理不改变 path、method、`operationId`、Schema、security 或生成 Client 公共签名。
- 新增文档专用 `api_reference` namespace：英文标题与 canonical `summary` 精确一致，中文为逐项校订标题；接口面、模块和标题在文档浏览器内存中本地化，参数、响应、Schema、示例及描述继续保留 canonical 英文。缺失/多余翻译、重复 ID、无效 tag 或非规范文档均 fail closed。
- Scalar 仍加载和直接下载唯一 `/openapi/openapi.yaml`；独立入口用精确 `yaml@2.9.0` 解析展示对象。构建门禁验证 emitted YAML 与源文件逐字节一致、无 locale-specific spec，并确认 YAML、Scalar 和大体量标题 catalog 未进入 Admin 主 SPA 依赖图。
- Chromium E2E 在 1440×900 与 375×812、`zh-CN` 与 `en-US` 下覆盖初始折叠、模块展开、双语模块/标题搜索、侧栏/正文一致、稳定锚点、健康请求及语言头、Cookie omission、无外部请求/横向溢出/意外 console error，稳定文档面 axe serious/critical 为 0。375×812 仅代表 Chromium viewport，不是物理设备证据。
- 最终 Node 24.18.1 `make check` 退出 0：Admin 30 个 Vitest 文件 / 128 项测试、Backend vet/tests、Mobile 静态校验、统一 i18n 及 Docs 66 页构建全部通过。canonical/emitted YAML 均为 377,817 B，SHA-256 同为 `efc4a2050a7cbe8f31fa88f23306ebc545783fa2e34dba5622e3ed8f348bd8df`；Admin/OpenAPI 初始图 gzip 分别为 233,379 B / 999,599 B。
- Redocly 2.12.4 在明确跳过仓库既有 `security-defined` 基线后退出 0，仅保留 3 个既有 files ambiguous-path warning；最终 Admin Docker 镜像、Nginx 路由 smoke 和真实 Chromium E2E 均退出 0。本地验收栈继续运行在 `http://127.0.0.1:4174`，PostgreSQL 映射端口为 55432。
- 后续补充的顶部交互测试提示关闭按钮已同步 `zh-CN`/`en-US` ARIA 文案；关闭状态只写入当前标签页 `sessionStorage`，并已由 Chromium 验证点击隐藏、刷新保持隐藏、新文档上下文重新显示。当前 Admin 单元测试为 129 项。
- UI Skill request/output/decisions/review checklist、页面 override、双语桌面/移动展开与搜索截图已保存。本轮继续基于未提交工作树增量交付，未 commit、未 push。

## 2026-08-11 管理端 OpenAPI 文档与 System 底部入口

- Admin 以独立 Vite 多页面入口公开 `/openapi/`，精确使用自托管 `@scalar/api-reference@1.64.1`；构建和开发服务器直接读取 `server/openapi/openapi.yaml`，没有第二份业务规范。最终产物与 canonical 文件均为 362,990 B，SHA-256 同为 `635df57558c4bc95748d2a77e065a1019b65ebe6ed64267cfb18a48edeeedca1`。
- 文档支持 `?lang=zh-CN|en-US`、双语标题与真实写操作风险提示；交互请求强制 `credentials: omit` 和对应 `Accept-Language`，不读取 Admin 会话，`persistAuth=false`。默认字体、Agent、遥测、开发者工具、远程代理和插件 URL 均关闭。
- 菜单在权限、Feature Flag、实现注册与空目录裁剪后拆分：System 继续保留数据一级菜单、三级结构、路由与后端授权，但不再进入主菜单；文档和齿轮固定在侧栏底部，System 无可见叶子时仅隐藏齿轮。桌面使用上弹面板和右侧级联三级菜单，移动 Drawer 使用可滚动内联层级。
- Nginx 自托管文档资源并增加严格 CSP、`nosniff`、`no-referrer`、禁止 iframe 和分层缓存；同源保留 `/admin-api/`，增加 `/api/`、精确 live/ready 健康代理，其余 `/internal/` 返回 404。文档公开访问不需要登录。
- Node 24.18.1 聚合 `make check` 退出 0：Admin 30 个 Vitest 文件、124 项测试通过，Backend vet/tests、Admin lint/typecheck/build、OpenAPI/蓝图/i18n、Mobile 静态检查和 Docs build 均通过。Admin 初始 gzip 233,372 B，OpenAPI 独立入口初始 gzip 988,669 B；Admin 首屏依赖图未引入 Scalar。
- Admin Docker 镜像构建成功；隔离 Compose 环境完成 Nginx 路由 smoke 和真实 Chromium E2E。1440px 展开/折叠、375px 移动 Drawer、双语文档、System 三级导航、键盘/Esc/焦点回归、Reduced Motion、无横向溢出、无外部请求及控制台错误均有证据；稳定页面 serious/critical axe 为 0。375px 仅为浏览器视口，不代表移动真机。
- 验证边界：未使用手工 Bearer Token 实调受保护写接口，刷新清除授权仅由 `persistAuth=false` 配置与自动化边界覆盖；Scalar 内嵌请求客户端打开后的第三方瞬态界面仍有 ARIA/对比度 axe 问题，不能声明该瞬态状态无障碍通过；未做 Firefox/Safari、生产部署或物理设备验收。本轮未改数据库菜单、权限 Seed、业务 OpenAPI 内容、Backend 业务实现或生成 Client，未 commit、未 push。

## 2026-08-10 App 范围页面全局选择器与前置状态

- App 用户、文章、分类、单页、升级中心及兼容移动发布入口统一使用 `AppShell` 右上角、全屏按钮左侧的全局 App 选择器；应用清单和其他非 App 范围路由不显示该控件。
- 删除旧的页内 App context Card 与 info Alert。未选择 App 时，筛选器、表格、移动卡片和创建/发布动作均不渲染，改用共享的最小高度居中灰色提示，因此禁用查询的 pending 状态不会再表现为持续 Loading。
- 选择仍以 UUID `app_id` 写入 URL；切换 App 保留已有查询参数并把已存在的分页重置为第 1 页，内容管理的文章/分类 URL 同步不再丢失 `app_id`。
- 当前 App 额外以 `tenant_id → app_id` UUID 映射保存为非敏感 Zustand 工作区偏好。显式且属于当前租户的 URL 值优先；跨页面缺少参数时自动恢复并回写 URL，清空选择同步清除当前租户记忆，损坏或已不存在的值 fail closed。
- `zh-CN` / `en-US` 提示明确指向页面右上角；设计系统、页面 override、UI Skill request/output/decisions/checklist 和截图索引已同步。
- Node 24.18.1 下 Admin 完整 check 通过：27 个 Vitest 文件、114 项测试、production build、bundle budget 与 Admin blueprint 均退出 0。Chromium E2E 覆盖 5 个未选择入口、跨页面/新页面恢复、375/768/1024/1440、双语和明暗偏好，13 个 axe 状态 serious/critical 为 0、无页面溢出或意外 console error。
- 当前真实本地 API 与 seed 管理员完成登录、选择默认 App、跨到用户管理、移除 URL 参数后刷新/重新登录、再次进入用户管理并从浏览器记忆恢复的链路；Admin API 失败响应与 console error 均为 0。本轮未修改 Backend、数据库或 Mobile，未 commit、未 push。

## 2026-08-10 本地管理员与安全 Seed 引导

- 当前 `appkernia-acceptance` PostgreSQL 18 的既有 `local` 租户已创建 `admin@appkernia.local`，状态、成员关系和 `super-admin` 角色均为 active；维护者提供的本机密码仅经终端 stdin 输入，没有写入源码、环境变量、日志或交付文档。
- `BootstrapAdmin` 改为单个 Serializable 事务：复用既有 active 租户，首次创建身份/Argon2id 凭据/成员/系统管理员角色，重复执行只补齐角色、权限、菜单和租户配置，不重置已有密码；已存在但不属于目标租户的同邮箱身份会 fail closed。
- `ak-cli seed core` 新增 development-only 管理员引导：默认邮箱为 `admin@appkernia.local`，仅在显式设置 `AK_SEED_ADMIN_PASSWORD_FILE` 时启用；非 development、空文件或缺失文件均拒绝，不存在固定密码回退。
- `.secrets/` 同时从 Git 与 Docker 构建上下文排除，README 提供不进入 shell history 的交互式密码文件创建流程。完整 Seed 实测输出 `development_admin=true`，随后真实 Admin 登录仍返回 HTTP 200，证明重复 Seed 保留原凭据。
- `make -C server check`（133 tests passed / 0 failed）、CLI/Seed 专项 race、3 个 PostgreSQL Seed integration、Backend/Admin/Mobile 蓝图、统一 i18n 和 `git diff --check` 均退出 0；本轮无 Migration、OpenAPI、权限码或可视 UI 变更。

## 2026-08-09 文档站品牌、实景 Hero 与内容优化

- 文档站 Logo、favicon、Apple Touch Icon 与 Manifest 图标统一取自 `apps/ak-admin/public/brand`；删除旧文档 SVG 标志与 AI 生态 Hero，并从最终首页重新生成 `1200×630` 社交分享图。
- 宽屏文档外壳固定为 `280 + 960 + 248 = 1488px` 并居中，正文内边距为 56px、普通段落限制为 72ch。Chromium 在 1920px 实测左右外边距均为 216px，正文样本宽 725.625px；1280–1535px 流式布局、低于 1280px 大纲折叠和低于 768px 侧栏抽屉保持兼容。
- 首页 Hero 采用隔离本地 Docker 环境加载完成的 Admin Dashboard，以及重新编译并登录后的 iPhone 16 Pro / iOS 18.6 模拟器 Home、Profile。图片只含合成测试身份；该证据不代表 iOS 真机、Android 或 HarmonyOS 运行验收。
- 首页移除 Rspress `features` 双层框，改为单边框语义链接卡片；补齐项目初心、三端架构、能力矩阵、三步运行、产品实景、成熟度边界、FAQ、贡献与 Star 引导。Guide、Concepts、Server API、Mobile Components、Community 五个入口同步扩充 `zh-CN` / `en-US`。
- `AKDOCS-002` production preview 共 8 个样本，覆盖 375/768/1024/1440/1920、双语和明暗主题；全部 HTTP 200、单一 H1、无页面横向溢出/破图/console error，axe serious/critical 为 0。
- Docs 完整门禁、Backend/Admin/Mobile 三蓝图与 i18n validator、113 个 API 文档路径引用及 `git diff --check` 均退出 0；站点仍构建 64 个静态页面。发布后状态以 `Docs Pages` Workflow 和线上 URL 实查为最终事实源。

## 2026-08-09 GitHub Pages 正式发布

- 官网主体、生成产物、Node 24 Actions 更新和 production hero 排版修复分为 `10f58e2`、`f2a9534`、`71f366c`、`31abe86` 四个可审查提交并直接推送至明确授权的 `main`；Pages artifact 对应的站点 revision 为 `31abe861a19e6c917051450dccb81e1a2f3d4432`。之后仅追加发布报告，不命中 Pages 路径过滤。
- GitHub Pages 已通过 API 启用为 `workflow` 模式，公开地址为 `https://payhon.github.io/AppKernia/`，HTTPS enforcement 已开启。最终 Actions run `31266883357` 的 build 40 秒、deploy 8 秒，全部退出成功。
- 发布后 Chromium 实查中文/英文首页、Quick Start、服务端 API、Mobile 组件及 OpenAPI/Sitemap；7 个 URL 均为 HTTP 200。1440 与 375 首页均为一个 H1、无横向溢出、破图、console error 或失败资源。
- 首轮线上 1440 截图发现 hero 渐变标语末字形成孤行；按 `ui-ux-pro-max` 复核后使用原生 `text-wrap: balance` 修正，仅作用于品牌标语。最终线上 1440/375 重新验证为 balanced title + natural subtitle。
- `actions/checkout` 和 `pnpm/action-setup` 升级至正式 Node 24 runtime 版本 `v7` / `v6`；第二、三轮 Pages run 不再产生原 Node 20 deprecated annotation。
- `appkernia.com` 当前 A、AAAA、`www` CNAME 及 GitHub 验证 TXT 均为空，因此未把 Pages 强制绑定到不可达域名。待 DNS 配置完成后再设置 Custom Domain 并验证证书，当前 GitHub Pages 默认地址不受影响。

## 2026-08-08 AppKernia 开源官网与文档站

- 新增 `apps/ak-docs`，使用 Rspress 2.0.19 构建 `zh-CN` / `en-US` 双语官网、指南、架构与安全概念、服务端核心 API、AK Mobile 组件、贡献指南、安全政策和 MIT 协议说明；构建产出 64 个静态页面并生成本地搜索索引、Sitemap、`llms.txt` 和 OpenAPI 下载资产。
- 官网首页以 AppKernia 的跨端初心为主线，说明 Mobile + Admin + Backend 一体化与 HarmonyOS 支持，提供 Quick Start、GitHub、贡献和 Star 路径；根 README、CONTRIBUTING、Code of Conduct 与 OpenAPI license 元数据同步完成。
- API 文档新增契约检查器，实际核验 113 个路径引用均存在于 `server/openapi/openapi.yaml`；中英文路由启用 build-time parity 门禁。移动组件文档覆盖 button、text field、layout/status、modal/switch、icons/theme，并链接真实 AK UI 源码边界。
- 真实 `ui-ux-pro-max` 产物与截图保存在 `apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-001`。Rspress production preview 覆盖 375/768/1024/1440、明暗主题与中英文；8 个代表页面的 axe serious/critical/all violation 均为 0，控制台错误、失败资源、页面横向溢出和破图均为 0。
- `.github/workflows/docs-pages.yml` 已固定 Node 24.18.1、pnpm 11.18.0，并使用官方 Pages Actions 构建、上传和部署 `apps/ak-docs/doc_build`；`appkernia.com` 的一次性 Pages、DNS 和 HTTPS 操作写入 `apps/ak-docs/DEPLOYMENT.md`。
- 本地 `pnpm check` 通过：Admin 24 个测试文件 / 93 项测试、生产构建和 bundle budget 均通过，随后 Docs lint、TypeScript、Prettier、API 引用、语言对等与 64 页构建通过。Backend/Admin/Mobile 三蓝图及统一 i18n validator 均退出 0。
- 验证边界：本轮未 commit、未 push、未触发 GitHub Actions，也未修改 DNS，因此 `appkernia.com` 和 `payhon.github.io/AppKernia/` 尚未声明线上可访问；本机 Node 26.5.0 产生预期 engine warning，CI 已固定项目要求的 Node 24.18.1。Admin 图片为本轮本地 Vite 登录页实拍，Mobile 图片复用仓库既有 iOS 18.6 模拟器证据，不代表新的真机验收。

## 2026-08-08 Mobile Apple UI refresh

- 全局 `ak-theme-root` 统一状态栏安全区；`ak-button` 对 slot 文本显式应用 primary/secondary/danger 颜色，修复蓝色主按钮黑字；三项原生 TabBar 增加原创 outline/filled PNG 图标及暗色 selected 资产。
- Home、Notifications、Articles、Profile/Settings/Security、Authentication、Legal、Help/About、Error 共 28 页面改为语义 token、分组表面、44 px 触控目标和可滚动高度；移除文章 620/640 px 固定视口。
- `zh-CN` / `en-US` 首页与 TabBar 已在 iPhone 16 Pro 模拟器运行时切换并截图；切回 `zh-CN` 后保留测试账号登录状态。
- iOS 真实运行发现并修复 UTS 字符串桥接把设备 UUID 变成 `[object Object]` / `{}` 的问题：请求头先 JSON 物化原始值，Auth client 使用内存 UUID，安全会话恢复校验 UUID 并在陈旧凭据失败时重建。首次登录及 App 重启 refresh 均验证为合法 36 位 UUID。
- HBuilderX 5.06 iOS、Android、Harmony 三端 28 页面编译均退出 0；Harmony 生成未签名 `.hap`，明确提示未配置包名/证书。Android/Harmony 安装运行、物理设备、暗色/动态字体/VoiceOver 仍未执行。

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

## 2026-08-05 Mobile 四页面 ImageGen 视觉概念（历史设计阶段，后续已实现）

### 状态：视觉稿完成；对应页面已由后续 Mobile Framework 交付实现

- 使用仓库内 `ui-ux-pro-max` 真实执行 design-system、Mobile UX 与 Vue stack 查询，新增 `apps/ak-mobile/design-system/MASTER.md` 及 Home、Profile、Articles 页面 override。
- 使用内置 ImageGen 生成并保存 4 张统一风格的浅色高保真界面：Home、Profile、Article List、Article Detail；成品均为 853×1844 PNG。
- Home/Profile 严格保留 Mobile 蓝图的 `首页 / 消息 / 我的` 三 Tab；文章列表和详情按堆栈页面处理，保留返回导航，不新增第四个 Tab。
- 该设计阶段尚无 Article 路由/API/权限，因此当时只交付视觉概念；后续 Mobile Framework 交付已补齐 Route Registry、OpenAPI、语言包与 UVue 实现。
- request、Skill 输出、决策、最终提示词、review checklist 和截图保存在 `apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-imagegen-four-surfaces/`。
- Mobile Blueprint validator、统一 i18n validator 和 `git diff --check` 均退出 0。Android/iOS/Harmony 构建、模拟器、真机、动态字号、英文运行时和交互可访问性均未执行，不标记通过。

## 2026-08-05 字典驱动化与通知发送完善

### 状态：完成（本机 Docker/PostgreSQL/MinIO/Chromium；外部云凭据联调未验证）

- 仅将 `storage.driver`、`sms.provider`、`notification.sms.template_event`、`notification.email.template_event` 纳入动态字典；状态机、安全、协议、权限、语言等稳定枚举继续使用代码/数据库约束。
- 迁移 000011、核心字典/模板幂等种子、Admin/Public 消费接口、锁定/扩展策略、租户覆盖和 locale 回退已落地；最终数据库为 migration 11 clean，核心 4 类型、40 全局项、22 全局模板。
- 存储编译期能力包含 local、S3、MinIO、腾讯 COS、阿里 OSS、七牛 Kodo 及租户受控 S3-compatible；本机 MinIO 真实完成自定义驱动 Put/Open/Delete。
- 腾讯/阿里短信官方 SDK、供应商隔离 Secret、SMS 模板绑定、HTML/Plain 模板安全渲染、go-mail SMTP、加密 Delivery、River Worker、去重/重试风险和找回密码异步投递已完成。
- Admin 配置/字典/文件/通知模板/投递消费同一字典源并完成双语与风险 UX；最终 Chromium 4 个状态均无 serious/critical axe、无横向溢出、无控制台错误，Email test 为 202 且数据库无目标明文。
- Backend 77 个顶层单元测试、全仓 28 个顶层 PostgreSQL/MinIO integration tests、Admin 14 files / 58 tests 全量 check、Backend/Admin/Mobile/i18n/UI Skill validators 和 `git diff --check` 均退出 0。
- Root `AGENTS.md` 已增加枚举/字典决策规则。Mobile 无现成 OpenAPI→UTS Client 生成管线，本轮同步公开 OpenAPI/蓝图但未伪造 Client；腾讯/阿里/七牛、真实 SMS 与 SMTP 因无隔离凭据均保持未验证。

## 2026-08-05 字典管理分类与行编辑优化

- 左侧字典类型由平铺列表升级为 HotGo 启发的两级分类导航：按字典代码首段命名空间自动归类，分类可折叠并显示稳定 namespace 与数量；一次加载 100 个类型，过期页码会自动回到有效页，避免常规分页切碎分类。
- 右侧表格将“显示标签”和“字典键值”拆为独立列；桌面固定标签/键值与操作列，移动端保持表格内部横向滚动，页面本身无横向溢出。
- 所有可写行均有编辑入口：租户项执行正常 PATCH；可扩展内置项通过 POST 创建租户覆盖，已有覆盖再次编辑仍锁定键值、语言与内置能力元数据。内置行不显示删除，租户覆盖可删除并回退全局项。
- 后端扩展校验允许内置 `local` 等已注册值在完全保留元数据时创建租户覆盖；未知 registered 能力继续拒绝，自定义存储仍只接受 `adapter=s3_compatible`。System Settings integration fixture 已增加字典类型清理，避免测试数据残留到开发字典页。
- Admin 新增 2 个分类/覆盖路由单测；真实 Docker Chromium 验证 zh-CN/en-US 1440 与 en-US 375，覆盖 POST=201、PATCH=200，3 个状态 axe critical/serious=0、页面无溢出、console/HTTP 错误均为 0。
- 本轮未 commit、未 push，并原样保留用户已有 `apps/ak-mobile/artifacts/` 与 `apps/ak-mobile/design-system/`。

## 2026-08-05 字典项颜色与样式类可创建选择器

### 状态：完成（本机 Docker Chromium + PostgreSQL）

- 字典项编辑抽屉的颜色与样式类已从普通文本框升级为复用的单值可创建选择器：空值明确显示默认态，可搜索并选择内置预设，也可输入任意值后按 Enter 创建。
- 颜色提供品牌蓝、成功绿、警告橙、危险红、紫色、青色、中性灰 7 个双语预设；样式类提供主要、成功、警告、危险、信息、中性 6 个双语语义预设。下拉项同时显示效果、名称与实际存储值，不仅依赖颜色表达。
- 管理端只对编译期预设样式应用实时预览；自定义 CSS 类名仅作为代码文本显示，不直接挂载到管理页 DOM。表格外观列会同时展示颜色与样式类，不再二选一隐藏。
- 新增 `AkCreatableSelect`、外观预设/安全预览 helper 及 5 项定向测试；双语事实源、生成语言包、Master/page design-system override 与 `ui-ux-pro-max` request/output/decisions/checklist 已同步。
- 真实写链路验证预设 `#087a68 / ak-dictionary-style-success` 通过 POST 201 保存，自定义 `#123456 / tenant-accent` 通过 PATCH 200 保存；全部 7 个截图状态 axe 0 violations、无页面横向溢出、无 console/HTTP error。
- Admin 全量 17 个测试文件、65 项测试、ESLint、TypeScript strict、4,166 modules production build、bundle budget 和 Admin 蓝图通过；Mobile/i18n/UI Skill、E2E py_compile 与 `git diff --check` 通过。
- 本机 4174 Admin 已使用最终镜像重建且 healthy；临时 E2E 用户、租户和覆盖项均已清理（计数 0）。未部署生产，Firefox/Safari 与移动真机未执行；本轮未 commit、未 push。

## 2026-08-05 Admin 折叠侧栏逐级浮层与折叠控件优化

### 状态：完成（本机 Docker Chromium）

- 修复折叠侧栏仍把展开态 `openKeys` 作为受控值传入 Ant Design Menu 的根因：桌面折叠态交回 Menu 原生 hover/focus 管理，展开桌面与移动 Drawer 继续保留受控祖先展开。
- 二、三级浮层只随对应父级逐级出现，鼠标离开完整层级后全部隐藏；每个目录使用统一 popup class，不修改权限、Feature Flag、路由或菜单数据契约。
- 浮层改为 86% ink 半透明背景、14px backdrop blur、轻边框与双层阴影；连接父级的一侧为直角，外侧使用 10px 圆角，减弱多个独立圆角矩形的割裂感。
- 折叠控件固定于可见视口垂直中点，尺寸 22×40px、无横向 padding；展开时左边缘为 248px，折叠时为 80px，均精确贴合侧栏右边线并增加轻阴影。
- `ui-ux-pro-max` request/output/decisions/checklist、App Shell override、双语截图与结构化 Playwright 证据均已保存。
- 最终 Chromium 在 zh-CN/en-US 下均验证弹层数量 `0 → 1 → 2 → 0`，axe serious/critical=0、页面无横向溢出、控制台错误=0；17 files / 65 tests、lint、strict typecheck、production build、bundle budget 与 Admin/Mobile/i18n/UI Skill 校验均通过。
- 4174 Admin 已用最终源码重建且 healthy；专用 E2E 账号密码哈希/锁定状态已原样恢复、临时会话已退出、临时 4173 容器和凭据文件已删除。未部署生产，Firefox/Safari 未执行；本轮未 commit、未 push。

## 2026-08-05 地区管理可编辑化

### 状态：完成（本机 PostgreSQL 18 + Go API + Chrome）

- 新增 `sys.region.create/update/delete` 并继续以动作权限授权；`super-admin` 经核心权限同步获得全部地区权限，无写权限用户只显示只读树表且不渲染操作列。
- Migration `000012` 为 `sys.regions` 增加乐观锁版本、手工维护标记和软删除时间。管理/公开查询及 `has_children` 均排除软删除；种子 upsert 保留手工新增、修改和删除。
- 新增地区 POST/PATCH/DELETE 管理接口：仅 level 0/1 可新增直接下级，层级由服务端推导；编码、父级、层级不可编辑；更新校验版本；删除仅允许叶子节点且不级联。
- 三类写操作在同一事务内写入 `audit.operation_logs`，记录动作权限、地区编码及前后数据；OpenAPI、sqlc、生成 Client、权限/页面/API/Schema 契约、双语事实源与蓝图说明已同步。
- Admin 保留懒加载树表，增加权限化“新增下级 / 编辑 / 删除”、RHF + Zod 抽屉、锁定关系字段、明确删除确认、父节点拦截反馈与精确分支/搜索缓存刷新。
- 隔离库真实执行 migration `up/down/up`、System Settings PostgreSQL integration、系统管理员省→市→区县 CRUD、父节点删除拒绝、叶子软删除、只读权限、`zh-CN/en-US`、1440/375 与地区主内容 axe；5 个截图状态均 0 violation、无横向溢出、无控制台错误。
- 全页 axe 额外发现既有深色侧栏英文链接对比度不足，本轮地区内容审计限定为 `main` 并在交付报告保留风险；未部署生产，Firefox/Safari 与真实移动设备未执行。本轮未 commit、未 push。

## 2026-08-05 Admin 侧栏三态控制与路由持久化

### 状态：完成（本机 Docker Chromium）

- 侧栏状态统一为 `expanded / collapsed / hidden`，仅由边缘控件修改并持久化为非敏感本地 UI 偏好；折叠态点击级联菜单叶子进入新页面后仍保持 80px，不再由路由导航自动展开。
- 中间展开/折叠控件缩至 18×40px，精确贴合 80/248px 侧栏边界并保持视口垂直居中；hover/focus 完成白底深色图标到深底白色图标的反色，边框与阴影不会消失。
- 折叠态新增独立 18×28px SVG 关闭控件。完全隐藏后保留最左侧 48% 透明恢复条，并在 Header 右侧操作组最左端提供第二个恢复入口；任一入口都直接恢复 248px 完整展开态。
- 桌面控件在小于 1024px 时隐藏，375px 下移动 Drawer 仍可正常打开/关闭；折叠级联继续按 `0 → 1 → 2 → 0` 逐层展示。
- `ui-ux-pro-max` 请求、检索结果、决策、检查清单、双语/移动截图与结构化 Playwright 证据已保存；zh-CN/en-US axe serious/critical=0、控制台错误=0、无页面水平溢出。
- 侧栏定向 ESLint、TypeScript strict、5 项 store 测试、production build、bundle budget、Admin/Mobile/i18n/UI Skill 校验及 `git diff --check` 通过。全量 lint/test 被工作区其他在途改动阻塞：服务状态页 3 个 lint 错误；菜单数已改为 34、旧 icon 测试仍硬编码 35（其余 74/75 通过）。本轮未 commit、未 push。

## 2026-08-05 服务状态收敛与真实模块初始化

### 状态：完成（本机 Docker Chromium + PostgreSQL 18）

- 已移除“系统设置 → 模块信息”的菜单、路由、页面、查询 Hook、会话客户端方法、`GET /admin-api/v1/modules` OpenAPI/Handler/Application/Repository 链路及 `sys.module.read` 权限；旧页面 URL 走现有 404，旧 API 实测返回 404。
- 新增版本化核心模块目录，初始化按稳定编码精确 upsert `iam/org/sys/storage/notify/jobs/audit/ops`，并删除目录外记录。Migration `000013` 已真实执行 `up → down → up`，最终数据库恰好 8 条、未知模块 0、旧权限 0、旧菜单 disabled 且权限绑定清空。
- API、CLI、Worker、种子与运行摘要统一使用 `buildinfo` 版本；Makefile/Docker 注入版本、提交和构建时间，本地未注入明确为 `dev`。运行摘要模块返回语义名称/说明键、能力、状态及统一版本。
- 服务状态页成为唯一模块入口：桌面区块间距 24px、窄屏 16px，三张状态卡等高并在移动端堆叠；模块展示为双语语义名称、稳定编码、说明和双语能力标签，两张表均有可聚焦横向滚动区域且状态不只依赖颜色。
- `ui-ux-pro-max` request/output/decisions/checklist、Service Status 设计 override、8 张双语多视口截图与结构化 E2E 证据已保存。
- Backend `make check`、全仓 `go test -race ./...`、隔离库 integration、Admin `npm run check`（20 files / 75 tests）、Admin/Mobile/i18n 蓝图校验、`govulncheck`、Docker build/E2E 和 `git diff --check` 均退出 0。
- 最终 Chromium 覆盖 `zh-CN/en-US × 375/768/1024/1440`：8 组 axe violations 均为 0，页面无横向溢出、每组 2 个可聚焦滚动区、控制台错误 0，运行摘要 8 个模块版本均为 `dev`。
- 浏览器视觉验收使用契约等价的 mock auth/data；真实后端数据由 PostgreSQL 查询和隔离库 integration 独立验证。原因是本机 API 对新 bootstrap 账号仍返回 `IAM.AUTH.INVALID_CREDENTIALS`，未把该环境登录问题伪装为真实登录闭环。
- `staticcheck ./...` 与 `golangci-lint run` 仍被工作区既有 notification/storage 文件的 ST1005 错误文案大小写告警阻塞；本轮未修改这些无关在途代码。未部署生产，Firefox/Safari 和真实移动设备未执行；未 commit、未 push。

## 2026-08-06 GitHub Actions CI 修复

### 状态：修复完成并发布前验证通过

- `admin` job 原先在 pnpm 尚未安装时由 `actions/setup-node@v5` 初始化 pnpm cache，导致 `Unable to locate executable file: pnpm`；现改为先由 `pnpm/action-setup@v4` 安装锁定的 pnpm 11.18.0。
- `mobile-blueprint` job 原先缺少 `rg`。显式安装 `ripgrep` 后又暴露最新 Mobile 源码的 9 处禁用 `any`：现将 i18n 参数统一为 `Map<string, string>`，数值在边界转字符串，并使用 `UTSJSONObject.getString` 读取 Catalog/Header。
- 最新 `origin/main` 隔离工作树中，Mobile static check、全蓝图/i18n 校验、workflow YAML/actionlint、Admin 全量 check 均退出 0；Admin 为 16 个测试文件、64 项测试并完成 production build/bundle/blueprint 校验。
- 本机 Admin 检查使用 Node 26.5.0，因仓库要求 Node 24 产生 engine warning；CI 仍固定 Node 24.18.1。此次类型收紧后的 Android/iOS/Harmony compile 与真机未执行，不将静态 CI 修复标记为平台验收。

## 2026-08-06 Mobile Framework 主分支同步复核

### 状态：完成（文件完整性与静态契约）

- 当前本地 `main` 与 `origin/main` 同步，Mobile Framework 提交 `6256b90` 已在主分支历史中；与原独立 worktree 比对后，`apps/ak-mobile` 不缺少任何框架文件，20 个注册页面均存在且 20/20 使用 `ak-theme-root`。
- 主分支在框架提交之后增加了四页面视觉证据，并对 Home、文章页、i18n 和 HTTP Header 类型边界做了后续收紧；没有使用旧 worktree 覆盖这些更新。
- 后续类型收紧把阅读时长插值改为 `Map<string, string>`，原框架 verifier 仍检查旧的动态参数写法。本轮同步更新检查片段为 `minutes.toString()`，不改变运行时业务逻辑或可见 UI。
- Mobile Blueprint、统一 i18n、framework contract、secure-storage contract、refresh policy fake HttpPort 与 `git diff --check` 均退出 0。未重新执行三端 compile、安装或真机验证。

## 2026-08-07 iOS 模拟器国际化启动白屏修复

### 状态：完成（HBuilderX 5.06 + iOS 18.6 / iPhone 16 Pro 模拟器）

- 根因是 `ak-i18n.uts` 把导入的 JSON 仅通过 `as UTSJSONObject` 断言后立即调用 `toMap()`；HBuilderX 5.06 的 iOS app-service 运行时实际得到普通 JavaScript 对象，类型断言不会注入 `UTSJSONObject` 实例方法，因此应用在 Catalog 初始化阶段抛出 `source.toMap is not a function` 并白屏。
- 移动运行时现改为读取由 `locale/zh-CN.json` 与 `locale/en-US.json` 确定性生成的强类型字符串条目，并在 UTS 内构建 `Map<string, string>`；不再依赖 JSON import 的运行时原型或无界动态类型。
- 新增 Catalog 漂移门禁并接入 `check-project.sh`；它同时验证两种语言键集合、非空字符串和生成文件内容，使维护语言包后不能遗漏原生运行时 Catalog。
- HBuilderX 5.06 CLI 已完成 20 页面 iOS UTS compile-only，exit 0；随后对已启动的 iPhone 16 Pro / iOS 18.6 模拟器完成标准基座重签、安装、同步和启动，真实画面进入中文登录页。模拟器日志及生成产物未再出现 `catalogFromJson`、`source.toMap` 或原截图 TypeError。
- 标准基座仍提示 `ak-secure-storage` 的原生配置/SDK不能生效，这是 DCloud 的预期能力边界；本次仅证明标准基座启动和页面渲染，未验证 Keychain 回读。Android/Harmony 运行、iOS 真机、双语切换、签名 Release 与三端安全存储仍未执行。

## 2026-08-07 App 管理、移动注册找回与法律单页交付

### 状态：代码与静态/构建门禁完成；数据库迁移和终端运行验收未完成

- 新增 App 管理边界：`app.applications`、App 用户 membership、内容/通知/推送偏好/移动发布策略、会话、登录与安全事件均按 App ID 隔离；Admin 应用、用户、文章、单页内容和发布策略均使用 `/admin-api/v1/apps/{app_id}/...`，Mobile `/api/v1` 由 `X-AppID` 选择活动 App。
- App 用户管理涵盖列表/详情/创建/编辑、启停、解锁、重置密码和当前 App 会话撤销；写操作使用 `lock_version`，冲突返回稳定 `APP.CONFLICT`。应用、用户、单页列表的 `q/status/page/page_size` 在服务端完成范围校验、tenant/App 过滤、总数统计和 `LIMIT/OFFSET`。
- 文章、单页和法律同意记录均为 App 范围。用户协议、隐私政策、关于我们和自定义页均有 `zh-CN`/`en-US`、draft/published revisions 与原子发布；单页 DTO 保留 `body_format`，markdown 字符串和 blocks 数组不会被串化丢失。
- Mobile 登录流程补齐注册、邮件验证码、忘记/重置密码、用户协议和隐私政策入口；OTP/Session/App membership/Refresh App 一致性、`X-AppID` OpenAPI 参数和 mobile profile 作用域均已同步。外部邮件保持 Port/fake/异步 Adapter 边界，未写入真实凭据。
- 遗留移动版本迁移为每个 App 独立副本，回滚时按默认 App、更新时间、UUID 稳定折叠；遗留 mobile 登录/安全事件回填默认 App，新登录成功/失败和 refresh reuse 事件写入当前 App。

### 设计证据索引

- Admin App 管理：[request](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/request.md)、[skill-output](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/skill-output.md)、[decisions](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/decisions.md)、[review checklist](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/review-checklist.md)、[screenshot index](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/screenshot-index.md)。
- Mobile 认证/法律页：[request](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/request.md)、[skill-output](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/skill-output.md)、[decisions](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/decisions.md)、[review checklist](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/review-checklist.md)、[screenshots](../apps/ak-mobile/artifacts/ui-ux-pro-max/AKMOB-app-management-auth-legal/screenshots/INDEX.md)。

### 验证边界

- 已通过的静态、类型、单元和构建门禁见交付报告；这些结果不等同于 PostgreSQL 迁移、Admin 真实登录浏览器或移动模拟器/真机验收。
- Docker daemon 无响应，PostgreSQL 18 migration `up → down → up` 未实际运行；本次扩展后的 HBuilder X iOS 构建仅推进到 28 页后卡在 `ak-secure-storage`，没有 UTS 完成、模拟器安装或运行验收，因此新增注册、找回和法律页尚未完成模拟器验收。
- 未捕获 Admin 真实登录浏览器截图。OTP 外部邮件通道没有真实凭据，未发送邮件或完成验证码接收验收。
- 本次 App 扩展不重新声明白屏修复验收；既有 iOS 模拟器证据保留在上一节，新 28 页面仍需另行运行验证。

## 2026-08-08 App 管理本地验收修复

### 状态：完成（隔离 PostgreSQL 18 + Go API + Chromium + iOS 模拟器）

- Backend 通知管理集成 fixture 为第二位用户补齐当前 App membership，使查询继续按 tenant + App 双作用域验证；全量 Go check/build 与隔离库 integration 已通过。
- Admin 单页模型允许系统预留页在首个 revision 前没有 translations；编辑器安全初始化双语空值，列表标题按当前语言、另一支持语言、页面类型依次回退。应用状态改用稳定 API 枚举 `active` 的翻译键，并同步 Admin 与 blueprint 双语事实源。
- Mobile 登录页移除 HBuilderX iOS 不支持的百分比 `min-height`；HTTP 响应头不再对 iOS bridge 普通对象调用 `UTSJSONObject.getString()`，改为索引读取并兼容请求 ID 大小写。
- Admin 真实浏览器完成应用列表和单页列表的 `zh-CN/en-US` 验收，4 个状态 axe serious/critical 均为 0，无 console error，相关 API 均为 200。
- HBuilderX 5.06 完成 28 页面 iOS 编译；iPhone 16 Pro / iOS 18.6 模拟器真实进入登录、找回密码、注册、隐私政策和用户协议页面，未再出现白屏、`source.toMap` 或 `headers.getString` 异常。
- 两份法律内容仅在隔离验收库临时发布用于注册流程验证，验收结束后已清理，不属于源码或生产数据。
- HBuilderX 本机 UI 的资源同步阶段仍会停滞；本轮用其最新生成的 `app-ios` 资源覆盖官方标准基座数据容器后完成运行验收。此结果不等同于 HBuilderX 一键同步、自定义基座安全存储、iOS 真机、Android 或 HarmonyOS 验收。

## 2026-08-08 Mobile 登录与访客返回链路修复

### 状态：完成（隔离 PostgreSQL 18 + Go API + iPhone 16 Pro 模拟器）

- 登录页只保留全宽主登录按钮；忘记密码和受 `registration_enabled` 控制的注册账号改为按钮下方的文字链接，使用 44px 点击高度和 8px 相邻间距。
- 认证与法律页自定义导航栏补齐 iOS safe-area 偏移；公共返回控件使用显式点击处理、页面栈返回和空历史登录页兜底，找回密码、注册、隐私政策、用户协议均完成坐标点击往返。
- 修复 iOS UTS 请求边界：设备 UUID 生成后持久化并在请求头对象构造时直接写入；OpenAPI 将设备标识约束为 UUID。安全存储插件接口去除未使用的动态 accessibility 参数，三平台实现保持固定系统安全策略。
- 使用公开注册接口创建本地测试用户；隔离验收库完成邮箱激活后，模拟器真实登录成功。重启应用仍恢复会话并进入 Home，证明本地自定义模拟器基座中的安全存储可读写。
- 本地测试 API、隔离数据库和已登录模拟器保留运行，方便后续手工复测。账号凭据只在对话交付，不写入源码、文档或 fixture。
- iOS 模拟器运行使用包含 `ak-secure-storage` 原生模块的本地重建基座；未执行 iOS 真机、Android/HarmonyOS 运行、真实 OTP 邮件收发或生产环境测试。

## 2026-08-09 文档站项目叙事与架构可视化增强

### 状态：完成并发布（GitHub Pages artifact `1f33598`）

- 向导新增中英文“什么是 AppKernia?”，说明 2026 年项目由来、方便开发者的初心、品牌名的项目内解释、三端技术选型、当前阶段与未来方向，并提供贡献和 Star 路径。
- 使用可追溯仓库证据扩展产品画廊：4 张 Admin 本地 Chromium 界面与 4 张 iPhone 16 Pro / iOS 18.6 模拟器界面；文案明确不等同于生产、iOS 真机、Android 或 HarmonyOS 运行验收。
- Rspress 接入固定版本 `rspress-plugin-mermaid@1.0.1`，核心概念、总体架构、Admin/Web、Server、Mobile、认证会话、多租户、国际化、API 与 AK UI 流程均改为可访问 SVG 图，不再把流程源码直接展示给读者。
- 中英文 14 个图表路由共渲染 26 张 SVG；每张均有标题、描述与紧邻的文字摘要。表格和图表滚动区支持键盘聚焦，配色覆盖明暗主题。
- 最终 production-preview 门禁覆盖 22 个中英文/明暗/多视口状态，HTTP、单 H1、页面级 overflow、图片、console、失败资源、Mermaid 数量与 axe serious/critical 全部通过。
- Backend、Admin、Mobile、跨端 i18n 与 UI Skill 五项校验通过；Node 24.18.1 文档全量 check、113 个 API 引用、66 页构建、GitHub `/AppKernia/` Base Path 静态资源核对及 `git diff --check` 通过。
- Docs Pages run `31292063763` 的 build/deploy 均为 success；线上首页、中英文项目介绍、架构/认证页、产品图片与 Sitemap 均返回 HTTP 200，7 个 Chromium 代表页面无严重/关键可访问性或运行错误。
- 公开地址为 `https://payhon.github.io/AppKernia/`。`appkernia.com` 尚无 A/AAAA、`www` CNAME 或 GitHub Pages 验证 TXT，当前不声明自定义域名可访问。

## 2026-08-09 文档站首页内容、产品 Slider 与技术栈

### 状态：完成并发布（GitHub Pages artifact `dcc3a30`）

- 根因不是正文缺失，而是 Rspress 2.0.19 默认 `HomeLayout` 只渲染 Hero/Features，不渲染首页 MDX body；新增主题 HomeLayout 后，既有初心、三端架构、能力矩阵、源码运行、成熟度、FAQ 与社区内容均进入页面和 SSG Markdown 输出。
- 中英文首页新增 6 张核心特性链接卡，覆盖会话、权限/多租户、国际化、OpenAPI、AK UI 与工程证据；卡片保持单层边框、品牌色顶线和无位移 hover/focus。
- 新增 9 项技术栈：uni-app x、React、TypeScript、Vite、Go、PostgreSQL、OpenAPI、Docker、Ant Design。DCloud 官方 uni-app 图自托管，其余使用锁定的 `simple-icons@16.27.0` SVG 源；不依赖运行时 Logo CDN。
- 新增 Admin/Web 与 Mobile 两个四图 Slider，复用仓库验收截图，支持前后按钮、圆点和键盘左右键，不自动轮播；每张图有双语 alt、caption、计数和 live region。
- 最终 production preview 覆盖 `375/768/1024/1440/1920` 中文浅色与 1440 英文深色：6/6 HTTP 200、单一 H1、无页面级 overflow/破图/console/失败资源，axe serious/critical 为 0；两个 Slider 的点击、键盘与不自动轮播断言全部通过。
- Node 24.18.1 文档全量 check、113 个 API 引用、66 页双语构建、Backend/Admin/Mobile 蓝图、跨端 i18n、UI Skill、Python 语法与 `git diff --check` 均通过。首轮 docs check 曾因 Slider region 直接使用 `tabIndex/onKeyDown` 触发 2 条 JSX a11y lint，已把键盘处理移到真实 button 控件并重跑通过。
- `AKDOCS-004` 保存 request、Skill 输出、决策、页面 override、10 张截图与机器可读结果。功能提交 `dcc3a30` 已推送 `origin/main`，Docs Pages run `31294828773` 的 build/deploy 均为 success。
- GitHub Pages 默认域名的中英文首页、uni-app Logo、Admin/Mobile 图片均返回 HTTP 200；线上 Chromium 复核两种语言均为 10 个内容区、6 张特性卡、9 项技术栈、2 个 Slider，交互、overflow、图片、console、请求与 axe serious/critical 全部通过。
- 本轮没有重新运行 Admin/Mobile 业务环境，不声称生产业务环境、iOS 真机、Android 或 HarmonyOS 运行验收；Pages API 仍为 `cname=null`，因此只声明 `https://payhon.github.io/AppKernia/` 已更新。

## 2026-08-09 文档站首页作者叙事与横向版式优化

### 状态：完成并发布（GitHub Pages artifact `f730916`）

- 中英文首页删除 `HONEST MATURITY` 区块，并逐段移除面向项目所有者的验收、证据与交付口吻；公开文案统一为作者面向开发者的第一人称项目叙事、产品能力说明与贡献邀请。
- 首页改为 1240px 内容轨道与桌面 12 列网格：区块通过连续 1px 分隔线形成统一横向节奏，标题、说明、卡片、能力矩阵、三步运行、产品 Slider、FAQ 与社区 CTA 使用同一视觉骨架。设计只借鉴 Vercel 的内容优先和结构化网格原则，不复制其品牌资产。
- 卡片取消浮层阴影、渐变顶线和位移 hover，改为共享边界、单一强调色与背景色反馈；浅色主页为中性白，深色主页为中性黑，移动端在 768px 以下切换为单列连续结构。
- `AKDOCS-005` 保存 UI Skill request/output、设计决策、页面规范、review checklist、10 张响应式截图与机器可读结果；6 个 Chromium 状态均为 HTTP 200、单一 H1、无页面级横向溢出或破图，axe serious/critical 为 0。
- Slider 点击、键盘切换、2 个 live region 与不自动轮播断言全部通过；公开首页未发现 `HONEST MATURITY` 或面向验收交差的禁用词组。
- 文档全量 check、113 个 API 引用、66 页双语构建、Backend/Admin/Mobile 蓝图、跨端 i18n、UI Skill 与 Python 语法校验均已通过。
- 功能提交 `f730916` 已推送 `origin/main`；Docs Pages run `31299398867` 的 build/deploy 分别用时 45/10 秒并成功。线上中英文首页及 3 项静态资源均为 HTTP 200，Chromium 复核 3 个代表视口的结构、交互、文案、资源、console、网络与 axe serious/critical 全部通过。
- 完整验收清单随记录提交 `6b1e4bf` 推送后，最终 Docs Pages run `31299540867` 再次通过，build/deploy 分别用时 50/8 秒；最终 Pages artifact 与清单均处于已发布状态。
- Pages API 仍为 `build_type=workflow`、`https_enforced=true`、`cname=null`；当前公开地址为 `https://payhon.github.io/AppKernia/`，不声明 `appkernia.com` 已绑定。

## 2026-08-10 项目文档治理 Skill

### 状态：完成（Skill 与隔离脚本回归通过；未对 AppKernia 执行初始化）

- 新增项目级 `.agents/skills/project-docs-governance/`，可在目标项目内建立分层 `docs/`、工作项六文档闭环、全项目 Kanban 看板、质量/运维/归档入口，并以唯一受管块增量写入 `AGENTS.md`。
- 同时支持 `new`、`existing`、`auto` 三种模式以及 `--dry-run`、`--check`、`--force`；默认仅创建缺失文件，保留既有文档和 `AGENTS.md` 非受管内容。受管标记损坏时会在任何写入前失败。
- 两份独立提示词覆盖从 0 开始的新项目，以及已有代码但尚无文档规范的项目；均适用于 Codex、GLM5、Claude Code 等编码智能体。
- 隔离回归 3/3 通过，覆盖新项目预览/创建/检查/幂等、已有项目自动识别和旧内容保留、损坏受管标记的写入前失败。
- 本轮未运行该 Skill 改写 AppKernia 当前 `docs/` 或根 `AGENTS.md`，未修改业务代码、API、数据库、Admin 或 Mobile；没有 UI、构建平台、模拟器、真机或生产验证。

## 2026-08-10 APP 管理与 App 升级中心

### 状态：功能完成（本机 PostgreSQL 18 + Go + Chromium）；全仓 integration 组合仍有非本模块 `40001`

- 新增 `000017_app_upgrade_center` 可逆迁移：`app.applications` 增加 manifest AppID、App 类型、描述/简介/备注、创建者、所有者、软删除和图标引用；团队、资产、渠道、应用市场均落为 tenant 约束的 PostgreSQL 关系表。UUID 继续作为内部关联和 Mobile `X-AppID`，默认 App 回填 `__UNI__APPKERNIA`，其余历史 App 保持待配置。
- `sys.mobile_releases` 扩展为版本历史唯一事实源，新增包类型、目标平台、双语标题/内容、文件或 HTTPS 外链、应用市场和当前发布指针。草稿可编辑/删除，曾发布记录永久冻结；发布、下线、部分上线、多平台 WGT 原子替换及 SemVer 单调校验均在事务中完成。
- 新增应用软删除/批量删除和版本详情、草稿编辑、发布、下线、删除/批量删除 API；全部按 tenant 在 SQL 层过滤。旧 `/admin-api/v1/mobile/releases` 与 `/api/v1/public/app-version` 路径保留，Public Config 和版本 DTO 以加法方式暴露 `appid`、`app_type`、包类型、标题和更新标记。
- 内部安装包只引用 `storage.files`；发布前校验同租户、`ready`、扫描状态、扩展名、MIME 与 ZIP-family 文件头。下载使用绑定 App/版本/文件/有效期的短期签名，外链只接受 HTTPS 且服务端不抓取。
- Admin 菜单增加 `/app/upgrade-center`，与旧系统发布入口复用同一页面和数据。应用页支持 URL 搜索/筛选、移动卡片、桌面表格、多选删除、长 Drawer、文件资产和深链；升级中心支持应用选择、发布下拉、native/WGT 表单、双语内容、内部文件/外链、发布状态和冻结操作。
- 权限、菜单、OpenAPI、生成客户端、Admin/Mobile 契约、蓝图 schema/API/page 映射和 `zh-CN/en-US` 事实源已同步。团队关系仅保存管理资料，不扩大现有 tenant + RBAC 授权。

### 验证与边界

- 根 `make check`、Backend race、Admin 24 文件/94 测试、Mobile `check-project.sh`、全部蓝图/i18n 校验、PostgreSQL 18 `Down → Up → seed`、相关 repository integration、`git diff --check` 均退出 0；Go JSON 复核为 130 个通过测试事件、34 个通过包。
- Chromium mock-authenticated production preview 覆盖 `zh-CN/en-US × 375/768/1024/1440`、12 个页面状态、列表/Drawer/WGT/409 保留输入；axe serious/critical 为 0、页面无横向溢出、无意外 console error。截图与 JSON 证据见 [App 管理索引](../apps/ak-admin/artifacts/ui-ux-pro-max/app-management/screenshot-index.md) 和 [升级中心索引](../apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshot-index.md)。
- `make -C server test-integration` 及串行重跑仍在未改动的 IAM/jobadmin integration 出现 PostgreSQL `40001` serialization failure，退出 2；本模块 `appmanagement/application`、`mobileprofile/repository` 与文件 repository 的定向 integration 退出 0，不能据此宣称全仓 integration 已通过。
- Admin 当前只有固定视觉主题；已验证深色系统偏好环境下可用，但不声称实现了暗色主题。未部署生产，未做真实 App Store/ABM/应用市场/uni-stat 同步，未在 Mobile 下载或执行 WGT，也未执行 Android/iOS/HarmonyOS 编译、模拟器或真机验收。
- Admin/Docs 检查运行于本机 Node 26.5.0，仓库声明范围为 Node 24；命令虽退出 0，但产生 engine warning，本轮没有用 Node 24 重复同一组检查。

## 2026-08-10 管理员初始化文档

### 状态：完成（中英文文档站全量检查通过）

- 快速开始明确 Docker 管理员通过交互终端初始化，Core Seed 不内置固定密码，重复 bootstrap 不会重置已有密码。
- 源码开发正文新增开发专用的 `.secrets/seed-admin-password` 操作流程、`AK_SEED_ADMIN_PASSWORD_FILE`/`AK_SEED_ADMIN_EMAIL` 用法、权限要求、幂等语义、跨租户拒绝和生产环境边界。
- 故障排查新增 `development_admin` 输出、已有密码不覆盖、Vite 同源代理、Compose project 与 PostgreSQL volume 的检查提示；中英文内容同步。
- 文档站全量 check 退出 0：113 个 API 引用、lint、TypeScript、Prettier、66 页双语构建、语言 parity 和 Sitemap 均通过。运行时 Node 26.5.0 高于 Admin 声明的 Node 24，出现 engine warning；文档包自身允许 Node 24 至 26。

## 2026-08-10 多语言表单 Tab 统一优化

### 状态：完成（本地 PostgreSQL 18 Seed + Admin 全量检查 + Chromium）

- 核心字典新增只读 `system.language`，固定包含 `zh-CN`、`en-US`，由字典排序、默认项和当前 Admin 语言共同决定 Tab 顺序、标题与首个激活语言；协议支持范围未扩大。
- 新增共享 `useSystemLanguages` 与 `AkLocalizedFormTabs`。字典加载失败、空值、非 fixed、缺少/重复/未知语言和默认项异常均进入可重试错误态并禁用保存，不静默回退到前端静态列表。
- 原生/WGT 发布、系统与 App 范围的文章/分类，以及 App 单页表单统一使用共享 Tab；切换后保留 RHF 值，校验错误显示在对应语言 Tab，并按字典顺序定位首个错误语言。
- 字典管理页新增系统分类与内置类型的双语名称/说明展示；内容表单补齐本次 axe 发现的 Slug、分类、阅读分钟、精选、排序和状态控件可访问名称。
- 本地 core seed 实际输出 `dictionaries=71`；使用本地管理员会话读取真实 `/admin-api/v1/dictionaries/system.language`，返回 fixed、`zh-CN → en-US` 和唯一默认 `zh-CN`。
- Chromium 覆盖 `zh-CN/en-US`、375/768/1440、浅色和深色系统偏好，共 5 个表单状态；axe serious/critical、console error 与页面级横向溢出均为 0。Admin 当前仍是固定视觉主题，深色系统偏好不等同于独立暗色主题。

## 2026-08-25 App 启动页与启动介绍管理

### 状态：功能完成（PostgreSQL 18、Backend、Admin Chromium、Android/iOS compile-only 与 Harmony unsigned HAP）；三端物理设备验收未执行

- 新增可逆 `000018_app_startup_experience`：启动双语元信息、双语草稿位置/资产、不可变发布版本/资产和 published pointer 均按 tenant + App 规范化持久化；每个位置强制 `zh-CN`/`en-US` 图片与无障碍说明，最多 10 个位置。草稿和不可变版本分别登记 `storage.file_usages`，发布使用 `SERIALIZABLE` 单事务、版本单调递增和 stale-version 409。
- 新增 `app.onboarding.publish` 权限、发布 API、公开配置 startup 投影和受控 startup asset API。公开面只返回当前启用的 published revision；资产读取校验 App/tenant、ready、clean、MIME 与当前发布引用，并返回版本化缓存头。
- `ak-cli app-startup export` 从数据库双语元信息和已扫描 App 图标生成 Mobile 随包 UTS 快照与本地图标；`--check` 只读检测生成物漂移，真实临时目录 `export → --check` 均退出 0。
- Admin App Drawer 新增双语名称/副标题、图标预览、导出提示、启用开关、双语图片与无障碍说明、键盘上/下移动、草稿状态、版本/时间和独立发布。375px 全屏 Drawer 使用表单内 sticky 发布/保存动作，避免挤压标题。
- Mobile 隐私页改为严格离线的 pre-bootstrap 圆角容器；协议保持当前容器 allowlist 路由。Android/Harmony 取消走 `uni.exit()` Port，iOS 保持阻断。新增 `pages/onboarding/index`：强制升级门禁之后携带 `X-AppID` 顺序下载全部当前语言图片到临时本地路径；普通失败 fail-open 且不记录完成，无跳过，访问全部位置并位于末页后才保存 App UUID 对应最高完成版本。
- OpenAPI、生成 Admin/Mobile Client、权限快照、Admin/Mobile route/API/feature 契约、双语 Catalog、设计 override 和 UI Skill 产物均已同步。

### 验证与边界

- PostgreSQL 18 空库 migration up/down/up 成功；定向 integration 覆盖草稿不外泄、发布、stale 409、locale 投影、文件引用和禁用开关。`make -C server check` 退出 0。
- Admin 全量 check 为 30 个 Vitest 文件 / 130 项测试；真实 Chromium 覆盖 `zh-CN/en-US × 375/768/1440`，6/6 状态 axe 全量 violation、页面横向溢出和 console error 均为 0，键盘排序通过。Mobile blueprint/i18n/generated/static checks 退出 0。
- HBuilderX 5.24 Android/iOS 30 页面 compile-only 均退出 0。Harmony UVue/UTS 输出项目编译成功；HBuilder 内置 ohpm 受代理解析影响退出 1，随后无代理 `ohpm install --all` 和 DevEco 6.0.2 `hvigorw assembleHap` 退出 0，生成 15,662,973 字节 unsigned HAP。包名/签名未配置。
- 未执行三端物理设备冷启动、首装零网络抓包、动态字号、安全区、系统返回、退出行为、版本升级/回滚/重装或真实后台图片下载；Mobile 截图索引明确未完成。用户原有 `manifest.json`、既有文档内容和未跟踪微信公众号 Skills 均保留。本轮未 commit、未 push。
- 最终 Node 24.18.1 根 `make check` 退出 0：Backend、Admin 30 文件/130 项测试、Mobile（含随包快照 App ID/本地图标漂移门禁）、全部蓝图/i18n 和 Docs 121 个 API 引用/68 页构建通过。

## 2026-08-26 AppKernia 三端自定义基座

### 状态：Android 自定义基座已完成 vivo 真机匿名启动，iOS 与 HarmonyOS 官方模拟器均已运行 AppKernia 自定义原生产物

- 将 DCloud AppID 从占位值切换为已登记的 `__UNI__196F2FC`，保留用户原有 `0.2.0` 版本修改；三端原生标识统一为 `com.appkernia.mobile`。Android/iOS 打包和运行入口强制 custom playground，不允许回退到 `io.dcloud.uniappx` 标准基座。
- 以 `apps/ak-admin/public/brand/appkernia-mark.png` 为唯一源，确定性生成 Android 密度/Android 12 启动图、无 Alpha 的 iOS 1024 图标、Android round icon，以及 DevEco 模板尺寸的 HarmonyOS 288 × 288 layered icon / 144 × 144 start icon；产物门禁进一步核对 Android 四档 launcher icon 逐像素一致、iOS AppIcon 与 1024 主图缩放匹配、Harmony layered/start icon 逐字节一致，并同步设计系统、UI Skill request/output/decisions/review 与截图索引。
- 新增 `build-custom-base.sh`、原生资产生成器、Harmony overlay/签名隔离和 APK/App/HAP 产物检查器，并新增三端运行手册。HarmonyOS 官方没有 Android/iOS 式基座，交付物为带 AppKernia 原生身份的 HAP；签名采用“先生成原生工程，再只重跑 DevEco 工程”的二阶段流程，避免 HBuilderX 覆盖签名配置。
- Android 云端自定义基座生成 16 MB `android_debug.apk`，包名 `com.appkernia.mobile`、名称 `AppKernia`、版本 `0.2.0`。vivo V2545A 后续已完成安装；HBuilderX 5.24 明确识别设备端 `com.appkernia.mobile` `0.2.0` 自定义基座为最新版本、跳过基座替换、同步 30 页面并启动。首次真实启动暴露并修复 manifest `UTSJSONObject` 结构强转和 TabBar 尚未创建时同步原生主题两处运行错误；复编译/复启动进入真实登录页，进程定向日志未再匹配对应 `ClassCastException`、`IndexOutOfBoundsException` 或 TabBar fail。
- iOS 云端生成 89 MB x86_64 `Pandora_simulator_debug.app`；`Info.plist`、图标和 Mach-O 架构检查通过。在 iOS 18.6 iPhone 16 Pro 模拟器安装成功，HBuilderX 报告 custom playground 安装/30 页面同步/启动成功，并采集桌面图标与登录页截图。
- HarmonyOS 使用 DevEco 6.0.2：HBuilder 内置 OHPM 仍受进程代理影响报 `Invalid URL`，一键脚本随后在生成工程内以无代理 OHPM 安装依赖并由 Hvigor `assembleHap` 成功生成 17,002,087 字节 `entry-default-unsigned.hap`，SHA-256 为 `31f9d8b10b301b6c167c1955135948d72aa85a8bbc03ae9ebb67285bede7d620`。HAP 的 `module.json` 为 `com.appkernia.mobile` / `0.2.0` / layered icon，运行时资源目录为 `__UNI__196F2FC`。
- 当前无 HarmonyOS 物理设备。用户已授权自动签名；DevEco 在 `com.appkernia.mobile`、已登录帐号和 Managed Profile 流程中真实返回 `Unable to create the profile due to a lack of a device`，因此没有 Signing Config 被持久化。用户已分别明确同意 HarmonyOS Software License and Service Agreement 与 HarmonyOS SDK License Agreement；两份协议均完成接受，HarmonyOS 6.0.2（API 22）官方 Phone ARM64 镜像已下载。镜像六个 detached CMS 签名逐一校验通过；修正本地 HVD 的官方 `imageSubPath` 格式并丢弃失败磁盘层后，模拟器日志显示 `USERimage/SYSimage ... verify succeed` 与 `Guest OS Boot Completed`。重建后的 unsigned HAP 经 `hdc install -r` 安装成功，`aa start` 启动 `EntryAbility`，`bm dump` 回读 bundle、label 和 `$media:layered_image`；1080 × 2340 桌面截图显示完整 AppKernia 渐变/绿色轨迹图标与标签，首次隐私页也完成复测。应用自身隐私/用户协议未代用户点击。物理设备可安装包仍需连接设备完成 Managed Profile 自动签名；iOS 物理设备同样需匹配 Bundle ID 的开发证书与 Provisioning Profile。
- Mobile framework contract、39 routes/3 platforms 蓝图、跨端 i18n、`check-project.sh`、脚本语法、三产物检查和 `git diff --check` 均退出 0；HBuilderX 5.24 Android/iOS/Harmony 30 页面 UTS 编译通过，Android vivo 真机自定义基座资源同步/匿名启动退出 0。Harmony 的 HBuilder 内置 OHPM 仍因代理 `Invalid URL` 中断，随后脚本以无代理 OHPM 和 DevEco Hvigor 完成 `BUILD SUCCESSFUL` 并重建 unsigned HAP。`verify-installable` 按预期退出 1 并明确 unsigned HAP 不能作为真机可安装交付；官方模拟器安装与首帧运行已单独通过。本轮未 commit、未 push，两个既有未跟踪微信公众号 Skills 保持未改动。
- 已采集 vivo 匿名登录页、iOS 模拟器桌面/登录页，以及 HarmonyOS API 22 桌面启动器和首次隐私页；未执行 Harmony 应用协议同意、登录、网络、退出、返回键、动态字号、安全区或安全存储生命周期，不能把三端 smoke 外推为完整物理设备验收。Harmony 真机仍缺签名/在线设备，发布流水线还需固定可用的 ohpm 网络环境。

## 2026-08-26 多端打包自动化与操作文档

### 状态：脚本、根命令、双语手册和文档站构建完成；正式签名出包仍待发布凭据

- 新增跨平台 Node.js 编排器，统一 Android/iOS/HarmonyOS 自定义基座与正式版构建入口；支持 macOS/Windows 常见 HBuilderX、DevEco、Python 路径和显式环境变量覆盖。
- 根 `package.json` 新增自定义基座、正式版、预检、dry-run、单平台、产物校验和脚本测试命令。Android/iOS 签名通过退出即清理的受限临时配置交给 HBuilderX，日志和命令行只显示签名变量配置状态，不显示秘密值。
- 自定义基座预检只读校验 AppKernia 原生身份和图标契约；正式版预检严格要求 Android Keystore、Apple Distribution p12/Profile 与 Harmony Signing Config，缺少任一项即失败。
- `docs/manual` 新增自定义基座和正式版三端操作手册，覆盖 macOS/Windows、工具链、签名变量、Harmony 两阶段 release、产物路径、安全边界与发布验收。
- 文档站新增中英文“移动端自定义基座与正式版打包”页面，并接入 guide 导航、开始使用索引和移动端开发交叉链接；Rspress 最终构建 70 页且双语 parity、Sitemap、121 个 OpenAPI 引用全部通过。
- Node 24.18.1 脚本测试 4/4 通过；三端 dry-run、自定义基座只读预检、现有 APK/iOS App/Harmony unsigned HAP 产物身份与图标校验、Mobile 蓝图、跨端 i18n、Mobile 静态门禁和 `git diff --check` 通过。
- 当前没有 Android/iOS 正式发布证书，Harmony 也仍缺在线物理设备形成的 Signing Config；正式预检按设计退出 1，未执行正式版真实出包、商店上传或审核。Windows 兼容性已由路径单测和无 Bash 依赖的编排覆盖，尚未在 Windows 主机实际调用 HBuilderX/DevEco。
