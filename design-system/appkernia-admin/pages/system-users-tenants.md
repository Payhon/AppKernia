# Tenant Management Page Overrides

## Layout

- 标题区显示租户数量、筛选与新建动作；筛选、状态、分页和排序写入 URL。
- 桌面使用租户表格；小于 768px 收敛为租户主信息、状态和“更多”动作。
- 仅 `multi_tenant=true` 且具备 `iam.tenant.read` 时可见；隐藏不替代后端授权。

## Tenant switch

- App Shell 切换器显示当前工作区与可用租户；当前项不可重复提交。
- 切换中禁用控件并公告状态；成功后更新 Access/Refresh session、清空 tenant-scoped cache、重新获取 Auth Context 和当前路由数据，不整页刷新。
- 失败保持原租户、原 Token 和原缓存可用，并使用 `role=alert` 给出双语恢复提示。

## Actions

- 创建/编辑使用 Drawer + RHF/Zod；成员管理使用隐藏详情路由。
- 停用租户、移除成员必须说明新 Session/当前访问影响并允许取消。
