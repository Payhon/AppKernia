# Review checklist

- [x] 图集输出全部公开媒体、替代文本、计数、前后按钮和缩略图。
- [x] 视频输出真实来源、原生 controls、宽屏/沉浸切换，未启用自动播放。
- [x] 摘要和正文对 article、gallery、video 三类均持续展示。
- [x] 外部视频 CSP 使用精确 HTTPS origin；内部文件继续走 App 归属校验的 H5 资源接口。
- [x] 双语键值同步，模式按钮有可访问名称，reduced motion 下关闭平滑滚动与尺寸过渡。
- [x] 本地容器部署后的双语、390px 窄屏、深色、后台 iframe 与真实视频均已验证。

截图索引：

- `output/playwright/public-web-media/gallery-zh-CN-390.png`
- `output/playwright/public-web-media/gallery-zh-CN-dark-390.png`
- `output/playwright/public-web-media/video-en-US-immersive-390.png`
- `output/playwright/public-web-media/video-en-US-widescreen-390.png`
- `output/playwright/public-web-media/gallery-admin-iframe-390.png`

浏览器断言及 axe 结果：`output/playwright/public-web-media/evidence.json`。本轮为 Chromium 桌面模拟视口，不代表 iOS/Android/HarmonyOS 真机视频播放验收。
