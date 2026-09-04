# ADR-0031：后台登录交互式验证码

状态：按用户批准的 2026-09-04 方案实施。

Admin 登录在同一邮箱、`ak-admin` Audience 与来源 IP 的 HMAC 范围连续失败三次后，改用 `go-captcha/v2` 生成 `click | slide | drag | rotate` 四类短时挑战；默认类型为 `slide`。Mobile 登录不进入该验证码流程。挑战继续保持 5 分钟有效、最多尝试 5 次、成功后单次消费；同一范围只允许一个活动挑战，刷新会原子失效旧挑战并受 2 秒冷却约束。

服务端启动时初始化并复用生成器。挑战目标、类型、ID、范围和有效期由 AES-GCM 封装为不透明 Token，数据库只保存 Token 的 SHA-256；校验以已签发 Token 和数据库记录为准，不信任客户端声明的类型。配置切换不改变已签发挑战，迁移会消费升级前尚未完成的数字验证码。

验证码类型是稳定安全协议枚举，不创建字典或运行时插件。全局配置项为 `iam/security/admin.login_captcha.type`；缺失或非法值回退 `slide`，读取数据库失败时拒绝生成。只有 `AK_PLATFORM_TENANT_CODE` 指定租户（默认 `local`）中同时具备 `super-admin` 角色和 `sys.platform_config.update` 权限的管理员可修改全局配置，更新继续使用乐观锁并写审计。

Admin 使用自有、无网络职责的 `AkInteractiveCaptcha`，网络请求继续复用匿名请求封装。组件仅在服务端要求验证码时动态加载；交互使用原生 Pointer、Range 与键盘能力，提供坐标缩放、44px 目标、可见焦点、状态/错误播报、焦点恢复和 reduced-motion 支持。不引入 `go-captcha-react`，也不新增公开通用验证码 HTTP API；第二个跨服务调用场景出现后再评估扩展。
