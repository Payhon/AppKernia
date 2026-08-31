# Review checklist

- [x] 入口仅在已登录且 `account_deletion` 开启时显示，并位于退出登录下方。
- [x] 页面明确当前 App 范围、立即生效、不可恢复以及匿名化留存边界。
- [x] 脱敏邮箱来自服务端；客户端请求不含邮箱或用户 ID。
- [x] 六位验证码、服务端倒计时、错误就地反馈和重发状态完整。
- [x] 确认项默认未勾选，整行触控区域至少 44px且状态不只靠颜色。
- [x] 删除按钮具备 disabled/loading，并有第二次原生确认。
- [x] 删除成功后仅执行本地 Push 注销，再清理 Session/Cache 并跳转登录页。
- [x] 所有用户可见文案使用 `zh-CN` / `en-US` 对等翻译键。
- [ ] 360×800 Android 浅色截图。
- [ ] 390×844 iOS 浅色截图。
- [x] 402×874 iPhone 16 Pro / iOS 18.6 浅色截图。
- [ ] 430×932 HarmonyOS 浅色截图。
- [x] Android、iOS、HarmonyOS 编译。
- [ ] Android、iOS、HarmonyOS 真机/模拟器交互验收。
- [ ] 最大动态字号、VoiceOver/TalkBack 和英文长文本验收。

验证状态：代码审查、双语契约、静态检查和三平台编译已完成。iPhone 16 Pro / iOS 18.6（402×874 逻辑视口）已验证入口首屏可见、可进入删除页，并连续 3 次进入/返回基本资料；定向统一日志未出现原已销毁实例错误。截图文件沿用原计划的 `390x844` 产物名，但实际像素为 1206×2622（3×，即 402×874）：`screenshots/profile-delete-entry-fixed.zh-CN.390x844.png`、`screenshots/account-deletion-page.zh-CN.390x844.png`、`screenshots/profile-basic-request-lifecycle.zh-CN.390x844.png`。英文、Android/HarmonyOS 运行截图、动态字号及 VoiceOver/TalkBack 仍未验证。
