# Security Audit Page Override

本页继承 `../MASTER.md`，仅覆盖以下规则。

- 三类审计数据使用一致的页头、URL 筛选区、分页表格和状态反馈；不要用不可恢复的临时筛选状态。
- `severity`、登录结果与 resolve 状态使用语义 Tag + 明文标签，不能只依赖颜色。
- 操作差异以字段行展示 before/after；空值、删除、增加必须有文字语义。服务端返回的 `[REDACTED]` 保持原样，不提供复制秘密或原始 JSON 的入口。
- 登录页默认聚焦近期异常，但不制造风险评分；展示字段仅限服务端安全响应。
- 安全事件详情是隐藏深链路由；resolve 按钮只在未解决且具有 `audit.security.resolve` 时出现，并使用确认流程与明确异步反馈。
- 375px 隐藏 request/resource 次要列，主要标识、状态、时间与详情动作必须始终可达；表格滚动限制在 card 内。
