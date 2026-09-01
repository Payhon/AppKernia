# 内置公开 H5 展示入口

状态：按用户批准的 2026-08-31 修订方案实施。

Server 在原有 JSON API 之外增加公开 HTML transport；不改变 React Admin SPA 或 uni-app x 技术边界。GoFrame 路由调用 Application Port，使用 html/template 与 go:embed。生产无在线模板编辑、动态脚本或 Node 依赖。

资讯与单页采用公众号式长文排版；App 发行及下载统一 `/h5/apps/{app_id}/download`，平台识别仅由原生 JS 完成，禁止加载时自动跳转。Android APK 为独立资源动作，继续验证发布、归属和文件状态。平台与语言均为稳定协议枚举。

公开数据最小化；存量发行页默认关闭。当前已发布资讯旧 `/s/` 地址保留。Canonical 使用可信 AK_PUBLIC_WEB_BASE_URL；无配置时不猜测 Host。配置更新有 tenant/App 过滤、权限、版本冲突与审计。公开页面不消费管理员身份。

HTML no-cache，包下载 no-store，静态资源按内容哈希缓存。Markdown/Blocks 经受控渲染与净化，正文禁止脚本及 htmx 属性。htmx 只作为将来局部服务端 HTML 交互的可选依赖，首期无此类交互，不加载无用脚本。

2026-08-31 补充（AKH5-002）：后台通过 iframe 文档导航预览真实公开 HTML；这不是 Admin 调用 Mobile JSON API。HTML 仅允许经校验的 AK_ADMIN_ORIGIN 嵌入，公开数据规则不变。预览脚本随 Server 嵌入并使用内容哈希，不修改 UA；只有与可信父窗口握手后才调整外链目标和传递就绪/关闭事件。父子精确校验来源、窗口及加载 ID，消息不得携带管理员凭据、任意跳转/脚本指令。无新 API、权限、Schema 或迁移。

2026-09-01 补充（AKH5-004）：下载推广由 App 发行页配置统一管理，沿用既有读写权限、审计和乐观锁。配置包含显示开关及 `zh-CN`/`en-US` 标题、说明、按钮文字；空文案使用内置双语回退。关闭开关后，资讯和单页不输出顶部下载入口或文末推广 DOM，统一下载页仍可按既有公开开关访问。语言入口移至页头并使用可访问的 SVG 图标；视频布局切换以覆盖播放器右上角的图标按钮呈现。
