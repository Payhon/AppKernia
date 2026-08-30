# 信息架构、菜单与隐藏路由

## 0. Shell 呈现规则

- `sys.menus` 与权限上下文继续把 `system` 保存为数据一级菜单，既有二级能力域、三级页面、排序、Feature Flag 和权限语义均不改变。
- Admin 在完成 Feature Flag、View Permission 和静态 Route Registry 过滤后，才把已解析树拆成“普通主菜单”和“System 工具菜单”。普通主菜单进入左侧独立滚动区；System 不再出现在该滚动区，而是由侧栏底部右侧齿轮打开。
- 底部工具区固定为 OpenAPI 文档在左、System 在右。若过滤后 System 没有可访问页面，则整个 System 数据节点被裁掉且齿轮隐藏；公开文档入口始终保留。完全隐藏侧栏时工具区随侧栏隐藏。
- 桌面端齿轮上方展示 System 二级能力域，三级页面通过右侧级联菜单访问；移动 Drawer 使用可滚动的内联展开层级。选择页面后关闭面板，移动端同时关闭 Drawer。
- OpenAPI 文档不是 `sys.menus` 项，不增加权限码。它通过 `/openapi/?lang=zh-CN|en-US` 在新的浏览上下文公开打开，页面规范仍以 `server/openapi/openapi.yaml` 为唯一事实源。
- OpenAPI 文档内部导航固定为“接口面 → 业务模块 → 接口”三级：3 个接口面包含 31 个有序模块，模块接口列表默认折叠。该层级由 canonical YAML 的单一 operation tag、顶层 `tags` 与 `x-tagGroups` 驱动，不使用前端自定义侧栏或字母重排。
- 文档独立入口仅在内存中本地化接口面、模块名称和接口标题；参数、响应、Schema、示例及详细说明保留 canonical 英文。`api_reference` namespace 只进入 OpenAPI MPA，不进入 Admin 主 SPA；直接下载 `/openapi/openapi.yaml` 始终返回逐字节不变的 canonical YAML。

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

系统一级菜单下的二级项是能力域，三级项才是实际页面。上面的树描述数据结构；实际 Shell 视觉入口位于固定底部齿轮，而不是左侧滚动主菜单。

## 2. 菜单页面表

| 菜单码 | 标题 | 路由 | View Permission | 阶段 | 后端状态 |
|---|---|---|---|---|---|
| `dashboard` | Dashboard | `/dashboard` | `authenticated` | P0 | existing |
| `system.settings.configs` | 系统配置 | `/system/settings/configs` | `sys.config.read` | P2 | existing |
| `system.settings.share-configs` | 分享配置 | `/system/settings/share-configs` | `sys.share_config.read` | P2 | existing |
| `system.settings.dictionaries` | 字典管理 | `/system/settings/dictionaries` | `sys.dictionary.read` | P2 | existing |
| `system.settings.regions` | 地区管理 | `/system/settings/regions` | `sys.region.read` | P2 | existing |
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
