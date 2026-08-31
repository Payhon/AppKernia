# Screenshot index

| 状态 | 文件 | 平台/尺寸 | 说明 | SHA-256 |
|---|---|---|---|---|
| 已验收 | `screenshots/ios-402x874-bookmarks-article.png` | iPhone 16 Pro / iOS 18.6 / 402×874 | 文章 Tab 仅显示原有文章收藏。 | `708ca42a46cc07c52420a7638dc5c5cd1091fe076527bb680d6fd06d2f8b7beb` |
| 已验收 | `screenshots/ios-402x874-bookmarks-gallery.png` | iPhone 16 Pro / iOS 18.6 / 402×874 | 图文 Tab 仅显示临时图文验收收藏。 | `29c76874d8d232100a3f20e98e70f0c1a156b54e881f14e5bbc4db818567df7c` |
| 已验收 | `screenshots/ios-402x874-bookmarks-video.png` | iPhone 16 Pro / iOS 18.6 / 402×874 | 视频 Tab 仅显示临时视频验收收藏。 | `daaa7cf5d64a6e54b384dccaefc2190f09039ed9d9b78682937e339dc53d1170` |

三张图片均为模拟器原始 1206×2622 PNG；端到端结果见 `output/maestro/ak-mobile-bookmark-type-filter.junit.xml`（1/1，20 秒，退出 0）。截图完成后已删除 2 条临时资讯及其收藏，测试用户恢复为原有 1 条文章收藏。
