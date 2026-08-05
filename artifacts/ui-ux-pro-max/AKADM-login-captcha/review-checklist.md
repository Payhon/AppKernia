# AKADM Login CAPTCHA — Review Checklist

## Interaction

- [x] 前两次失败不显示验证码。
- [x] 第三次失败后显示验证码并聚焦输入框。
- [x] 页面刷新后再次登录仍由服务端要求验证码。
- [x] 刷新按钮生成新图片并清空旧答案。
- [x] 验证码错误/过期后保留区域并取得新挑战。
- [x] 正确验证码和正确凭据登录成功，成功后失败状态被重置。

## Accessibility

- [x] 验证码输入有可见 label 与 autocomplete 语义。
- [x] 图片 alt 不含答案；刷新按钮有 accessible name 与 tooltip。
- [x] 错误使用 `role="alert"`，加载状态使用 `role="status"`。
- [x] 仅键盘可完成输入、刷新与提交；焦点指示清晰。
- [x] `zh-CN`、`en-US` 下 axe 无严重问题。

## Responsive and visual

- [x] 375px 无水平滚动或控件截断。
- [x] 768px 和 1440px 与登录页 Master 一致。
- [x] 验证码图片在高 DPI 下清晰，文字和控件对比度达标。
- [x] 保存双语关键视口截图。

## Security and contract

- [x] 验证码答案未出现在 DOM、接口、日志或存储明文中。
- [x] 未知账号与已知账号使用相同失败/验证码响应路径。
- [x] 挑战绑定登录范围、短时有效、一次性消费、限制尝试次数。
- [x] OpenAPI、生成 Client、Backend i18n 与 Admin i18n 同步。
