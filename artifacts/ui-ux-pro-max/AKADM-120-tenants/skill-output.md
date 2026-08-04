# AKADM-120 ui-ux-pro-max Actual Output

实际执行 4 条仓库本地命令，均退出 0。

## Tenant list design-system query

- Pattern: Enterprise Gateway；Corporate Navy/Grey。
- 原始 style: Vibrant & Block-based；颜色 Indigo `#6366F1`、Emerald CTA `#10B981`、背景 `#F5F3FF`。
- Typography: Lexend + Source Sans 3，并建议 Google Fonts。
- Checklist: 4.5:1、visible focus、reduced motion、375/768/1024/1440。

## Tenant detail design-system query

- Pattern: Enterprise Gateway。
- 原始颜色 Purple `#7C3AED`、Orange CTA `#F97316`、背景 `#FAF5FF`。
- Typography: Lexend + Source Sans 3。
- 建议 large sections、animated patterns、scroll snap 和 200–300ms 动效。

## Switcher UX query

- 成功操作提供明确反馈，破坏性操作先确认。
- 错误使用 `role=alert`，不得只靠颜色；正常文字至少 4.5:1。
- 管理 stacking context 和 z-index，避免切换器被 Shell 遮挡。

## React query

- Auth/locale/tenant 属于 app-wide context；不要把表单和列表 server state 放进全局 Context。
- Context value memoize、按 concern 拆分，避免深层 prop drilling。

原始输出中的营销 Hero、logo carousel、Contact Sales、紫橙 CTA、外部字体、scroll snap 和大幅动画与仓库 Master、后台任务语义、隐私和布局稳定性冲突，保留为证据但不采用。
