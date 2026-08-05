# AKADM Login CAPTCHA — UI Request

## User request

在 Web 管理端登录失败 3 次后显示图形验证码，并由服务端强制校验，降低机器人持续调用登录 API 的风险。

## Scope

- Route: `/login`
- Surfaces: Admin Web + `/admin-api/v1/auth/*`
- Locales: `zh-CN`, `en-US`
- States: default, third failure, CAPTCHA loading, CAPTCHA required, CAPTCHA invalid/expired, refresh, successful login

## Constraints

- 保留现有 AppKernia 登录页品牌布局与 Ant Design 表单体系。
- 验证码仅在服务端判定连续失败达到阈值后出现，刷新或重开页面不能绕过。
- 不在图片替代文本、DOM、接口响应或日志中暴露验证码答案。
- 所有新文案使用语义翻译键；键盘、读屏、移动端和高对比度可用。
