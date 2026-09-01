# Review checklist

- [x] 视频两种模式仅使用 SVG 图标，并浮动在播放器右上角。
- [x] 图标支持键盘、可见焦点、双语可访问名称和非颜色选中状态。
- [x] 语言图标位于全部可用的文章、图文、视频、单页及下载模板顶部，底部不再出现语言切换。
- [x] 推广双语内容、开关、回退、关闭后不输出 DOM 均通过单元、集成和真实后台保存验证。
- [x] Admin 双语配置、既有权限、乐观锁、审计和保存反馈保持有效。
- [x] 375/390/1440、浅深色、200% 和可信 Admin origin iframe 截图完成。

## Evidence

- 视频图标与顶部语言：`output/playwright/public-web-controls/video-icons-zh-CN-390.png`、`video-en-US-dark-375.png`。
- 图文/下载：`gallery-top-language-en-US-390.png`、`download-top-language-zh-CN-1440.png`。
- 200% 与 iframe：`article-top-language-200-percent.png`、`video-controls-admin-origin-iframe.png`。
- 结构化断言：同目录 `browser-evidence.json`、`accessibility-evidence.json`、`iframe-evidence.json`、`admin-promotion-evidence.json`。
- axe-core：视频、图文、下载的 375px 深色 WCAG 2 A/AA violation 均为 0；Admin en-US 768px Drawer violation 为 0。
