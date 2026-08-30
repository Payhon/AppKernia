# Screenshot index

- 参考图：用户提供的 Apple Developer 六张截图，仅用于设计意图，不计作实现截图。
- 编译证据：Android/iOS/HarmonyOS HBuilderX 5.24 编译通过，HarmonyOS 未签名 HAP 制作成功。
- iOS 运行环境：iPhone 16 Pro、iOS 18.6、HBuilderX 5.24 自定义基座；演示内容来自本地 PostgreSQL 与真实 Go API。
- 首页/导航：`output/playwright/ak-news-ios-home-maestro.png`、`ak-news-ios-browse.png`、`ak-news-ios-topics.png`、`ak-news-ios-profile-guest.png`。
- 认证与个性化：`ak-news-ios-auth-sheet.png`、`ak-news-ios-profile-authenticated.png`、`ak-news-ios-bookmarks.png`、`ak-news-ios-recovered-home.png`。
- 搜索/文章/分享：`ak-news-ios-search.png`、`ak-news-ios-article-detail.png`、`ak-news-ios-share-sheet.png`。
- 评论：`ak-news-ios-comments.png`、`ak-news-ios-comment-pending.png`。
- 图文/视频：`ak-news-ios-gallery-filter.png`、`ak-news-ios-gallery-detail.png`、`ak-news-ios-video-filter.png`、`ak-news-ios-video-ready.png`、`ak-news-ios-video-progress-start.png`、`ak-news-ios-video-progress.png`、`ak-news-ios-video-paused-after-progress.png`。
- 视频证据：初始 Google 示例资源因 Range 403 被替换为白名单内、Range 206 的 W3C HTTPS MP4；`progress-start` 与等待 10 秒后的 `progress` 显示不同实际视频帧，`paused-after-progress` 为原地暂停后的后续帧，证明外链加载、播放推进和暂停闭环。
- 浏览器补充证据：`output/playwright/ak-news-admin-items.zh-CN.{1440,768,375}.png`、`ak-news-admin-categories.zh-CN.1440.png`、`ak-news-admin-comments.zh-CN.1440.png`、`ak-news-public-share.zh-CN.390.png`。
- 未覆盖：物理设备、动态字号、VoiceOver/TalkBack、微信 Provider 真分享和签名商店包；编译、模拟器与系统分享降级不替代这些证据。
