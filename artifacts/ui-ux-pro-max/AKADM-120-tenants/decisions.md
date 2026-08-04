# AKADM-120 Tenant Management Decisions

- 保持既有 Navy/Blue、系统字体与 App Shell；不采用紫橙主题、外部字体和营销结构。
- `multi_tenant=false` 时隐藏租户菜单和切换器，直达页面由路由层 fail-closed。
- 租户列表只展示服务端授权范围；普通租户管理员不能通过 query/body 指定其他 tenant。
- 切换器展示 `available_tenants`，当前租户显式标记；切换成功先替换 Token/Auth Context，再清理所有 tenant-scoped Query Cache，保留 global catalog cache。
- 租户详情使用隐藏路由，成员管理按 `iam.tenant.member.*` 逐动作授权；停用、移除成员说明 Session 影响并可取消。
- 375px 将数据表收敛为主信息 + 状态 + 更多动作；375/768/1024/1440 与双语均保存截图。
