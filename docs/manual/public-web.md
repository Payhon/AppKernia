# 内置 H5 页面部署与维护

本功能按 ADR-0030 实施。Server 随二进制内置模板、CSS、JS；生产不读取源码目录，也不需要 Node。Admin 继续使用现有 React 工程，Mobile 继续使用 uni-app x。

## 地址与内容

| 地址 | 用途 |
|---|---|
| `/h5/apps/{app_id}/articles/{slug}` | 公众号式资讯阅读 |
| `/h5/apps/{app_id}/download` | 同一发行/下载页，JS 推荐平台，所有下载选项保持可见 |
| `/h5/apps/{app_id}/pages/{slug}` | 当前已发布版本的隐私政策、关于我们等单页 |
| `/s/{slug}?app_id={app_id}` | 原分享地址兼容；canonical 指向新的资讯地址 |
| `/h5/apps/{app_id}/assets/{file_id}` | 已授权公开图片，逐请求检查引用及文件状态 |
| `/h5/apps/{app_id}/apk` | 检查开关/当前 Android 发布/文件状态，302 到短期签名资源 |
| `/h5/apps/{app_id}/packages/{release_id}/{file_id}` | 验签并重新检查发布与开关，输出 APK 文件 |
| `/h5/apps/{app_id}/download?format=qr` | 同一下载页的二维码，不是平台专属页面 |

`app_id` 是已有 App UUID，非 manifest AppID。支持 `?lang=zh-CN`/`en-US`，优先于 Accept-Language，最终回退 zh-CN。正文和分享元信息由 Server 完整输出。Mobile 优先采用详情响应 `share_url`，旧 Server 缺少字段时仍使用原分享地址生成方式。

Markdown 使用 Goldmark GFM，禁用原始 HTML，最终经 Bluemonday 净化。Blocks 支持现有 doc/text、heading、paragraph、列表、引用、代码、分割线、链接、图片、表格节点；不支持任意自定义脚本节点。远程正文图片不直接加载：需上传到文件中心并作为 App/已发布资讯媒体或当前已发布单页的合法图片引用，示例 `![说明](/api/v1/public/content/assets/文件UUID)`。单页读取不写入用户协议同意记录。

平台识别仅在 JS 执行。iPad 桌面 UA 使用触点信号；明确 HarmonyOS/OpenHarmony 标识推荐 HarmonyOS；只有厂商标识、无法区分系统的 UA 保留平台列表。所有情况都允许手选，不在加载时跳转。微信提示使用外部浏览器，没有绕过逻辑。首期没有服务端片段交互，因此不加载 htmx；未来引入时必须固定版本自托管并关闭 eval、脚本执行、历史快照和样式注入，正文保持 `hx-disable`。

## 发布配置

1. 应用迁移 `000028_public_web`，重新执行核心权限种子。读写权限分别为 `app.public_web.read` / `app.public_web.update`；升级迁移按已有 App 读写权限映射已有角色。
2. 设置 `AK_PUBLIC_WEB_BASE_URL=https://public.example.com`。必须是可信 HTTPS origin，不含账号、路径、查询或 fragment。只有 development/test 允许 loopback HTTP。生产未配置时不猜测 Host；H5 返回不可用，后台不能开启发行页。
3. 为 `/h5/` 与 `/s/` 配置反向代理到同一个 API Server；保留原分享域名的这两个路由。代理不要添加按 UA 分片的页面缓存，不覆盖后端 CSP/缓存头。页面内图片使用相对路径以兼容旧分享域名；分享元信息里的图片地址使用可信公开 origin。
4. 开启现有文件存储能力 `AK_FILE_STORAGE_ENABLED=true` 并配置生产对象存储。开发 `local` 适配器要求 `appkernia-local` 桶；对象目录不可充当公开静态目录。APK/图片均通过业务授权接口读取。
5. 在“应用管理 → 操作 → 发行页配置”填写中英名称/介绍、下载推广标题/说明/按钮文字、平台和商店 HTTPS 网页 URL，然后开启发行页。推广文案为空时使用内置双语回退；“显示下载推广”控制资讯/单页页头下载入口和文末推广，关闭后不输出对应 DOM，但不关闭统一下载页。旧市场记录不会猜平台或覆盖 Scheme；名称、启用状态、优先级仍在原 App 编辑器维护。排序沿用 priority 降序，同优先级按 ID 稳定排序。
6. APK 开关默认关闭。启用后仍需升级中心有当前 Android 内部原生包发布、文件 ready 且扫描 clean/skipped。不要用任意 URL 参数代替后台配置。H5 包签名使用独立派生密钥，与原 Mobile 下载签名不可互换。
7. 已发布资讯/单页提供查看、复制公开链接；草稿不提供匿名预览。App 停用阻断整个 App 的公开页面。关闭发行页不破坏仍有效的旧资讯分享，但不会公开尚未启用的发行页双语资料。

配置采用独立 lock_version；冲突返回 409，界面保留输入，不自动重放更新。配置和市场网页字段在同一事务中写入审计。公开 HTML 忽略管理员会话，不能利用 Cookie 读取草稿或私有文件。

## 缓存与运维

- HTML 与公开图片 `Cache-Control: no-cache`，请求会重新检查发布状态；匿名页面不缓存管理端身份。
- CSS/JS URL 含内容哈希，`public, max-age=31536000, immutable`。升级时新 HTML 使用新 URL。
- 签名链接与 APK 响应 `no-store`；链接默认 5 分钟有效。禁用或撤回后，即使签名尚未过期，后续请求也被阻断。已交付给用户的字节无法远程收回。
- 严格 CSP 限制脚本/样式/图片同源，禁止对象；H5 嵌入仅放行下述可信管理 origin。无需运行时 CDN、在线模板编辑或用户自定义 CSS/JS。
- 多副本滚动发布时，应让旧、新版本静态哈希在发布窗口内都可访问，例如使用代理共享静态缓存或原子切流，避免旧 HTML 的资源请求落到不认识旧哈希的新副本。

## 验证与证据

实际命令和结果见 `docs/CODEX_DELIVERY_REPORT.md` 的 AKH5-001。截图索引：`server/artifacts/ui-ux-pro-max/AKH5-001/review.md`，原图位于 `output/playwright/public-web/`。截图采用测试文字和程序生成的无品牌渐变图片，不代表正式 App 资料。

常规验证：

```bash
make check
make -C server test-race
GOFLAGS=-p=1 make -C server test-integration  # 预先配置隔离的 AK_TEST_DATABASE_URL
```

浏览器脚本：`apps/ak-admin/scripts/e2e_public_web.mjs`。真实 HTTP 撤回/签名检查：`server/tests/verify_public_web_http.py`。两者不能并行操作同一夹具：后者会临时撤回内容并在 finally 恢复。

本地隔离夹具的准备顺序：

1. 在测试 PostgreSQL 创建独立 `ak_h5_*` 数据库，配置 `AK_DATABASE_URL`，以 development 环境执行 `ak-cli migrate up` 与 `seed core`。通过权限 0600 的密码文件设置 `AK_SEED_ADMIN_PASSWORD_FILE`，种子账号 `h5-admin@example.test`、租户代码 `h5-e2e`。
2. `AK_H5_PASSWORD_FILE` 指向同一密码文件；运行 `python3 server/tests/seed_public_web_e2e.py`。设置 `AK_H5_DATABASE` / `AK_H5_PG_CONTAINER` 与实际数据库一致；脚本仅允许专用名称，重复运行复用已有夹具，不重置编辑内容。`AK_H5_E2E_FIXTURE` 为私密 JSON 输出路径，不得提交。
3. 从源码目录之外启动编译好的 Server，loopback 端口建议 18080、公开 origin `http://localhost:18080`、对象目录与 `AK_H5_OBJECT_DIR` 一致。Admin 必须使用同源代理到该 Server，不能误用现有 8080 演示环境。设置 AdminOrigin 与测试 Admin origin 一致。
4. 设置 `AK_H5_E2E_FIXTURE`、`AK_E2E_API_URL`、`AK_E2E_BASE_URL` 和可导入 Playwright 的 `AK_PLAYWRIGHT_MODULE`，执行浏览器脚本，再执行 HTTP 脚本。使用仓库中的 axe-core，未禁用浏览器 CSP。
5. 只清理本次测试数据库、临时对象目录和私密文件，不处理开发者其他环境。

迁移验收必须在隔离数据库完成 Up/Down/Up。真实安装、商店跳转、微信内置浏览器及三端原生分享需要另行真机检查；UA 模拟不能替代这些结果。

## 回滚

优先按 App 关闭发行页/APK 开关，保留资讯与单页。回滚二进制时同时切回旧 Admin 构建，避免旧 Server 接到新配置写请求；原 JSON 内容与 Scheme 数据不变。

必须回滚 Schema 时，先备份两张 public_web 表及市场 platform/web_url，再停止使用新接口，执行一次迁移 Down。该 Down 会删除新配置和权限；不会删除原资讯、单页、商店 Scheme 或已有发布包。恢复用迁移 Up、权限种子及受控数据恢复。禁止直接对生产运行示例脚本。
## 后台手机预览（AKH5-002）

资讯文章的“更多”菜单、App 发行页配置和已发布单页提供“手机预览”。手机外壳仅模拟外观/尺寸，不修改 User-Agent；商店及 APK 点击后在独立浏览上下文打开。分类、专题、标签、评论操作未调整。刷新重新加载原入口，复制包含后台当前语言的公开 URL。

`AK_ADMIN_ORIGIN` 必须与浏览器地址栏中的后台 origin 完全一致（scheme、host、port）。生产仅接受 HTTPS，development/test 允许 localhost/127.0.0.1/::1 HTTP；无效 origin 会记录不含原值的告警，H5 继续 `frame-ancestors 'none'`。不要填写路径、查询、账号、通配符或域名列表。跨域预览不需要放宽 CORS，不传管理员 Token；公开内容仍忽略管理员会话。

H5 HTML（包括错误页）仅放行此管理 origin；静态、图片及 APK 的原有策略不变。反向代理不得额外给 H5 文档添加 `X-Frame-Options: DENY/SAMEORIGIN` 或另一条禁止嵌入的 CSP；如代理为 Admin SPA 配置 CSP，其 `frame-src` 须包含可信 H5 origin。独立 `/openapi/` 文档策略不得修改。建议公开站点和后台分 origin 部署；同源 `allow-scripts allow-same-origin` 不能视为安全隔离，安全边界仍是正文净化、严格 CSP 和服务器公开数据校验。

预览消息协议固定为 `ak.public-web.preview.v1`（稳定协议，不建业务字典）：父窗口发送 `{channel,type:"init",loadId}`；子窗口返回 `ready`、`unavailable` 或 `close`。双方验证精确 origin、source 和本次 loadId；父窗口不接受 URL、HTML、脚本或其他命令。10 秒未收到确认提示重试/独立打开，不把 iframe load 当作 HTTP 成功。预览脚本随二进制嵌入并按内容哈希发布，无 CDN 和新运行时依赖。

回滚时同时回退 Admin 预览入口与 Server 预览脚本/CSP 变更，恢复 H5 `frame-ancestors 'none'`；不回退已有 H5 内容、发行配置或数据库迁移。旧 Admin 配合新 Server 仍可独立查看公开页面；新 Admin 配合旧 Server 会显示预览超时，可通过保留的独立打开入口访问。

浏览器预览回归脚本：`apps/ak-admin/scripts/e2e_public_web_preview.mjs`，与上述 H5 脚本复用私密夹具。重新运行种子会幂等补齐同租户切换用 App（`switch_app_id`），不重置已有内容或密码。使用 `AK_ADMIN_ORIGIN` 匹配测试 Admin 的 origin，执行前确保 Admin 代理指向同一隔离 API。结果索引见 `apps/ak-admin/artifacts/ui-ux-pro-max/AKH5-002/review-checklist.md`。


## 本地 Compose 更新记录（2026-09-01）

当前工作树已更新到既有 `appkernia-news-demo`，入口为 Admin `http://localhost:4174`、API `http://localhost:8080`。Compose 明确注入 `AK_PUBLIC_WEB_BASE_URL=http://localhost:8080`，并保留精确的 `AK_ADMIN_ORIGIN=http://localhost:4174`。访问 Admin 必须使用 `localhost`；改用 `127.0.0.1` 属于不同 origin，H5 会按设计拒绝嵌入。

升级前保留旧 API/Worker/Admin 镜像标签和 PostgreSQL custom-format 备份。数据库从 27 升到 28、`dirty=false`，Core Seed 为 182 权限/49 菜单且未创建或重置管理员。原 1 App、2 账号、3 资讯、5 单页保持。发行页配置仍为 0 条，不会因升级自动公开；因此下载页在运营显式启用前返回受控 404，已有已发布资讯 H5、CSS、预览 JS 均返回 200。

新容器 API/Admin/PostgreSQL healthy，Worker running，restart count 均为 0。Chromium 从 Admin origin 嵌入现有公开资讯并完成 `ready` 握手，page error 和失败请求均为 0。实际镜像、备份摘要和 HTTP 证据见 `output/local-deploy-public-web/evidence.json`；敏感容器快照与数据库备份位于 `.secrets/local-deploy-20260901-062117/`，目录不得提交。

回滚必须同时考虑 28 号迁移后新增的发行配置。当前配置表为空，可使用备份与旧镜像恢复；一旦本地运营开始配置发行页，应先另做部署后备份，不得直接 Down 丢弃配置。旧镜像标签为 `before-public-web-20260901-062117`。本次只更新本地 Compose，没有生产部署、提交或推送。


### 本地预览灰屏排查（2026-09-01）

若“查看公开页面”可用但手机预览显示灰色损坏页面并在 10 秒后超时，先检查后台地址栏。`http://localhost:4174` 与 `http://127.0.0.1:4174` 是不同 origin；当前 Compose 的可信 origin 是前者。H5 响应会发送 `frame-ancestors http://localhost:4174`，所以顶层独立打开不受影响，而从 127 地址嵌入会被浏览器 CSP 拒绝。

统一使用 `http://localhost:4174`，不要混用两个别名。切换 hostname 后 Cookie/localStorage 属于另一 origin，可能需要重新登录一次。`.env` 已以 0600 固定 `AK_ADMIN_ORIGIN=http://localhost:4174`，避免后续 Compose 重建漂移。真实 Chromium 对当前已启用下载页的结果：localhost=`ready`、CSP error 0；127=`blocked-or-no-handshake`、明确 frame-ancestors error。证据：`output/local-deploy-public-web/origin-diagnosis.json`。

### 图文/视频资讯与空白单页发布（2026-09-01）

资讯 H5 按服务端 `content_type` 输出对应内容。`gallery` 将全部已发布媒体作为 4:5 主图、CSS Scroll Snap 和缩略图展示，封面只在没有媒体时兜底；标题、元信息、摘要与正文继续显示。`video` 使用真实内部文件或经校验的 HTTPS 外部地址，提供原生 controls、宽屏与沉浸两种布局切换，不自动播放。外部视频只把该页面实际使用的精确 HTTPS origin 加入 `media-src`，不会放宽其他页面 CSP。

AKH5-004 将视频宽屏/沉浸切换改为播放器右上角的 SVG 图标按钮，按钮具有双语 Tooltip、可访问名称和 `aria-pressed` 状态；语言切换从页脚移到全部公开内容页的页头，以地球图标呈现。`000029_public_web_promotion` 增加推广显示开关和双语文案字段；回滚该迁移会删除这些新增字段，因此 Down 前必须备份 public-web 两张表并先切回旧 Admin/Server。

后台预置单页首次创建时可能只有 `content.pages` 行，没有任何 revision。发布接口要求至少有一条已保存的 draft revision；因此界面对这种页面保留“发布”入口作为流程提示，但按钮禁用并要求先编辑、补齐中英内容和保存。已发布页面存在后续草稿时仍显示可用发布入口。服务端校验继续保留，界面禁用不代替授权或状态校验。
