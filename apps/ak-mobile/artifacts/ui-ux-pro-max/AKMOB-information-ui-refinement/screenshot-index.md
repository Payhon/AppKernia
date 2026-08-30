# Screenshot index

- 参考图：用户本轮提供的四张截图，仅作为层级与交互参考，不计作实现证据。
- 编译证据：HBuilderX 5.24 Android、iOS、HarmonyOS 均完成 35 页面 UVue/UTS 编译；HarmonyOS 另生成未签名调试 HAP。
- 运行诊断：`output/playwright/ak-news-ios-ui-refinement-home.png` 记录把新编译资源同步到旧自定义基座后的白屏。原因是旧基座不含本轮新增 UVue 原生类，因此该图明确不作为通过证据。
- 环境恢复：`output/playwright/ak-news-ios-ui-refinement-restore-check.png` 仅证明已恢复同步前的旧基座资源和可运行状态，不代表本轮新界面。
- 未完成：需重建包含 35 页面/新增查看器类的自定义基座后，补首页、浏览、筛选、搜索、文章、图文、视频横屏与视频竖屏的新运行截图。
