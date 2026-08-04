# Review Checklist

- [x] `zh-CN` / `en-US` 可见文案全部来自翻译键。
- [x] 当前会话有非颜色的明确标识。
- [x] 所有撤销动作均有二次确认。
- [x] 当前会话撤销成功后清除本地认证上下文并返回登录页（客户端单测覆盖；E2E 验证警告并取消，以保留主测试会话）。
- [x] 非当前会话撤销后列表真实刷新。
- [x] 失败时保留原状态并显示 `role=alert` 错误。
- [x] 375 / 768 / 1024 / 1440 截图已保存。
- [x] `zh-CN` / `en-US` axe serious/critical 为 0。
- [x] 自作用域与 Refresh Token family 撤销由 Backend 集成测试覆盖。
- [x] 未实现的 Device / MFA 未标记完成；Password 已由独立专项继续交付。
