# 信息架构、菜单与隐藏路由

## 1. 最终菜单树

```text
Dashboard
系统
├── 系统设置
│   ├── 系统配置
│   ├── 字典管理
│   ├── 地区管理
│   └── 模块信息
├── 用户管理
│   ├── 部门
│   ├── 用户
│   ├── 岗位
│   └── 租户（multi_tenant）
├── 权限设置
│   ├── 角色
│   ├── 权限目录
│   └── 菜单
├── 文件存储
│   └── 文件管理
├── 通知中心
│   ├── 公告管理
│   ├── 站内消息
│   ├── 通知模板
│   └── 投递记录
├── 任务集成
│   ├── 定时任务
│   ├── API 客户端
│   └── Webhook
├── 审计安全
│   ├── 操作日志
│   ├── 登录日志
│   ├── 安全事件
│   └── 访问控制
└── 运行监控
    ├── 在线会话
    └── 服务状态
```

系统一级菜单下的二级项是能力域，三级项才是实际页面。这样既满足用户指定的布局，又承载后端 51 张核心表对应的管理能力。

## 2. 菜单页面表

| 菜单码 | 标题 | 路由 | View Permission | 阶段 | 后端状态 |
|---|---|---|---|---|---|
| `dashboard` | Dashboard | `/dashboard` | `authenticated` | P0 | existing |
| `system.settings.configs` | 系统配置 | `/system/settings/configs` | `sys.config.read` | P2 | existing |
| `system.settings.dictionaries` | 字典管理 | `/system/settings/dictionaries` | `sys.dictionary.read` | P2 | existing |
| `system.settings.regions` | 地区管理 | `/system/settings/regions` | `sys.region.read` | P2 | existing |
| `system.settings.modules` | 模块信息 | `/system/settings/modules` | `sys.module.read` | P2 | existing |
| `system.users.departments` | 部门 | `/system/users/departments` | `org.unit.read` | P1 | existing |
| `system.users.accounts` | 用户 | `/system/users/accounts` | `iam.user.read` | P1 | existing |
| `system.users.positions` | 岗位 | `/system/users/positions` | `org.position.read` | P1 | existing |
| `system.users.tenants` | 租户 | `/system/users/tenants` | `iam.tenant.read` | P1 | existing |
| `system.access.roles` | 角色 | `/system/access/roles` | `iam.role.read` | P1 | existing |
| `system.access.permissions` | 权限目录 | `/system/access/permissions` | `iam.permission.read` | P1 | existing |
| `system.access.menus` | 菜单 | `/system/access/menus` | `sys.menu.read` | P1 | existing |
| `system.storage.files` | 文件管理 | `/system/storage/files` | `storage.file.read` | P2 | existing |
| `system.notifications.notices` | 公告管理 | `/system/notifications/notices` | `notify.notice.read` | P2 | existing |
| `system.notifications.messages` | 站内消息 | `/system/notifications/messages` | `notify.message.read` | P2 | existing |
| `system.notifications.templates` | 通知模板 | `/system/notifications/templates` | `notify.template.read` | P2 | existing |
| `system.notifications.deliveries` | 投递记录 | `/system/notifications/deliveries` | `notify.delivery.read` | P2 | existing |
| `system.integrations.schedules` | 定时任务 | `/system/integrations/schedules` | `jobs.schedule.read` | P2 | existing |
| `system.integrations.api-clients` | API 客户端 | `/system/integrations/api-clients` | `sys.api_client.read` | P2 | existing |
| `system.integrations.webhooks` | Webhook | `/system/integrations/webhooks` | `sys.webhook.read` | P2 | existing |
| `system.security.operation-logs` | 操作日志 | `/system/security/operation-logs` | `audit.operation.read` | P1 | existing |
| `system.security.login-logs` | 登录日志 | `/system/security/login-logs` | `audit.login.read` | P1 | existing |
| `system.security.events` | 安全事件 | `/system/security/events` | `audit.security.read` | P1 | existing |
| `system.security.block-rules` | 访问控制 | `/system/security/block-rules` | `iam.block_rule.read` | P2 | existing |
| `system.monitoring.sessions` | 在线会话 | `/system/monitoring/sessions` | `iam.session.read` | P1 | existing |
| `system.monitoring.health` | 服务状态 | `/system/monitoring/health` | `ops.health.read` | P2 | existing |

## 3. 隐藏路由

```text
个人中心（头像菜单进入）
├── /profile/basic        基本设置
├── /profile/security     安全设置
└── /profile/connections  第三方绑定

认证（无后台布局）
├── /login
├── /register
├── /forgot-password
└── /reset-password
```

另外，用户/租户/角色/文件/公告/消息/任务运行/API Client/Webhook 投递/安全事件详情均为静态隐藏路由，通过 `active_menu_code` 高亮所属列表页面。

## 4. 菜单管理规则

- 类型仅允许目录、页面、受控外链。
- 页面 `component_key` 从静态 Registry 下拉选择，不允许输入任意文件路径。
- 最大三级；移动时同时校验循环、深度和租户边界。
- 图标来自静态 Ant Design Icons 白名单。
- 外链必须服务端 allowlist，默认新标签；iframe 默认禁用。
- 认证、个人中心和详情页严禁插入 `sys.menus`。
