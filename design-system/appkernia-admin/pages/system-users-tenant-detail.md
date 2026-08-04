# Tenant Detail Page Overrides

- 顶部提供返回租户、名称/code、状态和权限允许的编辑动作。
- Overview 与 Members 分区；成员表展示服务端授权字段，不还原凭据或身份 Hash。
- 邀请/加入现有用户使用 Drawer；状态修改和移除使用确认 Modal，成功后精确刷新成员与 Auth Context。
- 空态、加载、403、404、409 和可重试 5xx 都有双语可访问反馈。
