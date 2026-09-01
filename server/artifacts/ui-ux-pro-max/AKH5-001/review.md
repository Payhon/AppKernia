# AKH5-001 UI 审阅记录

- 日期：2026-08-31；使用 ui-ux-pro-max Skill 的 request / skill-output / decisions 与 H5 Master、article/download override。
- 资讯：白底单栏正文，标题 24–28px，正文 17px/1.8，移动端 20px 留白，桌面有效正文宽度 680px。没有资讯卡片外壳或遮挡正文的固定下载栏。
- 下载：系统字体、蓝色重点、圆角分组、CSS 横向截图；仅 JS 推荐，不自动导航；所有平台链接持续可见。
- 双语阅读页、下载页、单页的浅/深色 × 390/1440 宽度均进行浏览器 DOM、图片加载、整页横向溢出及 axe WCAG2A/AA 检查。已人工查看中文文章及英文下载页截图；截图中的渐变图是测试素材。
- 额外覆盖：200% 缩放，Tab 跳过导航，无 JS 正文与下载，手动平台覆盖，微信提示；UA 模拟不等于真机。
- Admin 使用既有 Ant Design 风格，不把 H5 Apple 风格套到管理平台。独立读写权限、加载/失败、保存、409 草稿保留、双语抽屉和二维码入口。

截图目录：`output/playwright/public-web/`。

| 文件模式 | 内容 |
|---|---|
| `article-{zh-CN,en-US}-{light,dark}-{390,1440}.png` | 8 张文章阅读布局 |
| `download-{zh-CN,en-US}-{light,dark}-{390,1440}.png` | 8 张下载页布局 |
| `page-{zh-CN,en-US}-{light,dark}-{390,1440}.png` | 8 张单页布局 |
| `article-200-percent.png` | 文本/页面 200% 缩放 |
| `admin-public-web-{zh-CN,en-US}.png` | 管理配置顶部，按可见视口截取 |
| `admin-public-web-{zh-CN,en-US}-downloads.png` | 管理配置下载入口与二维码 |

`evidence.json` 是完整浏览器断言和截图清单；`admin-evidence.json` 是单独重跑的 Admin 断言；`http-evidence.json` 是真实 Server 签名、撤回、资源授权检查。未捏造 App 评分、下载量、用户评价或生产素材。

待外部验收：实际 App 的商店地址与物料；微信 iOS/Android 实机；商店跳转、Android APK 安装；HarmonyOS 实机浏览与原生分享。截图和自动规则检测不是全部可访问性或真实设备认证。
