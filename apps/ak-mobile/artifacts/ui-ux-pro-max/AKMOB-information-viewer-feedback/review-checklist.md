# Review checklist

- [x] 不使用 emoji 或文本替代标准操作图标。
- [x] 返回、布局切换、收藏、评论、分享触控区域保持至少 44×44。
- [x] 视频不自动播放，退出页面仍暂停。
- [x] 未将交互控件放到 Dynamic Island 或 Home Indicator 安全区内。
- [x] 竖屏模式使用直接 class，不依赖 UVue 后代选择器。
- [x] 横向视频切换到竖屏时使用 `contain` 并垂直居中。
- [x] 图集按当前顺序传入系统预览，支持左右切换。
- [x] 图片已有替代文本；点击仅增加标准预览手势，不覆盖系统返回手势。
- [x] HBuilderX 5.24 iOS、Android、HarmonyOS 编译通过。
- [x] iOS 生成的 `app-service.js` 不含 `getVideoInfo` / `DCloudUniMedia` 引用。
- [ ] iOS 自定义基座运行复测：进入视频详情无 `uni-media` 缺类崩溃。
- [ ] iOS 模拟器交互复测：图片双指缩放、左右切换、视频竖屏居中。
- [ ] Android 真机复测。
- [ ] HarmonyOS 真机复测。

iOS HIG 快速诊断：本次涉及范围 8/10。安全区、44pt 触控、标准手势和原生导航符合；全局 Dynamic Type 与完整 VoiceOver 流仍属于现有项目级待验收项。
