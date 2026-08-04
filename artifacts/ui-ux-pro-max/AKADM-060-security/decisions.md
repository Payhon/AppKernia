# Decisions

- 复用现有 Master Design System 和 App Shell，避免安全页引入独立视觉语言。
- 当前会话使用文本 Tag 标识，不只依赖颜色。
- 会话撤销使用二次确认；当前会话的确认文案明确“立即退出”。
- 列表仅展示服务端自作用域返回的 `ak-admin` 活动会话；不在客户端传入 user/tenant。
- 展示 IP、User-Agent、最近活动与过期时间，但不展示任何 Token、Cookie 或敏感凭据。
- 撤销请求不走自动 401 refresh/replay；失败保留原列表并提供重试路径。
- Password、Device、MFA 仍为后续子任务，页面不伪造这些能力。
