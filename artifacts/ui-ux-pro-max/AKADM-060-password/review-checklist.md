# Review Checklist

- [x] 所有可见文案使用 `zh-CN` / `en-US` 翻译键。
- [x] 显式 label/id，密码输入可显示/隐藏。
- [x] `current-password` / `new-password` autocomplete 正确。
- [x] 长度与确认不一致错误就近展示。
- [x] 服务端稳定错误码映射为本地化提示并用 `role=alert` 播报。
- [x] 成功时清空三个密码字段，不持久化密码材料。
- [x] 写请求不自动 refresh/replay。
- [x] 后端成功、历史复用、旧密码失效、其他会话撤销与脱敏审计由 PostgreSQL 集成测试覆盖。
- [x] Docker 双语 E2E/axe 和截图已在新增表单后重跑。
