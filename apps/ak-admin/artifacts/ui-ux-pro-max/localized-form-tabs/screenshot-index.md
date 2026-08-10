# Screenshot index

所有路径均相对仓库根目录。

| 场景 | 语言 / 视口 / 偏好 | 截图 |
|---|---|---|
| WGT 发布 Tab 与切换保值 | `zh-CN` / 1440 × 900 / light | `output/playwright/localized-form-tabs/upgrade.wgt-tabs.zh-CN.light.1440.png` |
| 原生包双语错误 Tab 与首错定位 | `zh-CN` / 768 × 900 / preferred-dark | `output/playwright/localized-form-tabs/upgrade.native-validation.zh-CN.preferred-dark.768.png` |
| 文章编辑 Tab | `zh-CN` / 1440 × 900 / light | `output/playwright/localized-form-tabs/content.article-tabs.zh-CN.light.1440.png` |
| 分类编辑 Tab | `zh-CN` / 768 × 900 / light | `output/playwright/localized-form-tabs/content.category-tabs.zh-CN.light.768.png` |
| App 单页英文默认 Tab | `en-US` / 375 × 812 / light | `output/playwright/localized-form-tabs/app-page-tabs.en-US.light.375.png` |

机器可读结果位于 `output/playwright/localized-form-tabs/e2e-results.json`：5 个场景的 axe serious/critical 均为 0，console error 为 0，页面级横向溢出断言通过。

说明：AppKernia Admin 当前使用固定视觉主题；`preferred-dark` 表示 Chromium 深色系统偏好兼容性，不等同于已经实现独立暗色主题。
