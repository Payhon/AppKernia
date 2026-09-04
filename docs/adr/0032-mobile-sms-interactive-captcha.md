# ADR-0032：移动端短信交互式验证码

状态：按用户批准的 2026-09-04 方案实施。

Mobile 的短信登录、短信注册、手机找回密码、手机号绑定/变更和手机 OTP Step-up 在每次发送或重发前都必须完成一次新的交互式验证码。邮箱 OTP、账号删除邮件验证码和注册邮件重发保持原流程。

Admin 登录之外出现第二个调用方后，验证码生成、AES-GCM Token、Scope、存储、刷新冷却和验证生命周期由 IAM 共享服务统一提供；Admin 的连续失败三次门槛仍只留在 Admin 登录用例。数据库表改名为 `iam.interactive_captcha_challenges`，全局类型配置改为 `iam/security/interactive_captcha.type`，值与权限规则不变。Mobile 使用固定场景枚举，服务端按真实短信请求重新计算绑定范围，不信任客户端场景或验证码类型。

新增 `POST /api/v1/auth/sms-captcha`。匿名场景绑定 Audience、App、场景、规范化手机号、IP 和设备键 Hash；登录态场景额外绑定用户、Session、用途和资源。只有证明验证并单次消费成功后才创建 OTP Challenge 和进入短信投递队列，原短信冷却、限流、防枚举和 Provider Adapter 不变。

客户端新增无网络职责的 `ak-interactive-captcha` uni_modules 组件，以强类型 UTS/UVue 独立实现 `click | slide | drag | rotate`。上游 `go-captcha-uni@1.0.7` 只作为交互和坐标算法参考，不直接引入其 JavaScript、字体、CSS 或运行时包。组件使用原生 image、slider、touch 与 `UniElement.getBoundingClientRect()`，由独立 Repository/Runtime 负责 OpenAPI Wire 映射和网络生命周期。

本次不新增验证码微服务、Redis、运行时插件、独立 Mobile 类型设置或启停开关，也不承诺传统 uni-app、H5 或小程序兼容。
