# Decisions

- 公共 GET 支持游客与可选会话；受保护操作统一交给 AuthPromptCoordinator 打开 `ak-bottom-sheet`。
- 登录成功只刷新安全 GET，不重放评论、收藏等非幂等写操作。
- 文章原生渲染受控节点；图文使用横向媒体序列；视频仅详情当前项可播放，离页和切后台暂停。
- 微信三场景仅在 iOS/Android 配置完整时启用；HarmonyOS 使用系统分享降级。
- 分享统一指向公开 HTTPS `/s/{slug}?app_id=...`，不暴露内部页面路径。
