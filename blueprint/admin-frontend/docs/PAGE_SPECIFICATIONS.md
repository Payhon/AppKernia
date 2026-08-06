# 页面功能规格

Coding Agent 一次只实现一个 feature，并同时读取本文件和机器可读规格。

## `dashboard` — Dashboard

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| View Permission | `authenticated/self` |
| Schema | `iam.users`, `iam.sessions`, `audit.login_events`, `audit.security_events`, `jobs.schedule_runs`, `notify.messages` |
| 后端状态 | `existing` |

**筛选**：时间范围, 租户（平台管理员）

**主要动作**：无全局主动作

**UX 规范**：Executive + operations dashboard；没有权限的卡片整块隐藏，不能以 0 冒充无数据。

**API**

- `GET /admin-api/v1/dashboard/summary`
- `GET /admin-api/v1/dashboard/trends`
- `GET /admin-api/v1/dashboard/activity`

**页面验收**

- 时间范围写入 URL；卡片按权限裁剪，不把无权限显示为 0。
- 趋势图有表格/文本替代、空态、错误态和 reduced-motion。
- ECharts 路由级动态加载并遵守 bundle budget。
- 所有敏感字段由服务端脱敏。

## `system.settings.configs` — 系统配置

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `sys.config.read` |
| Schema | `sys.config_items`, `storage.upload_sessions`, `storage.upload_parts` |
| 后端状态 | `existing` |

**筛选**：module_code, config_group, config_key, value_type, status, is_public, is_secret

**主要动作**：选择配置分类, 新建配置, 编辑当前值, 轮换 Secret, 使用字典选择存储/短信驱动, 使用当前云存储策略测试上传

**UX 规范**：左侧分类 + 配置表；目录元数据只读，当前值可编辑；Secret 永不回显明文，使用保持不变/替换/轮换语义；分类选择写入 URL。

**API**

- `GET /admin-api/v1/configs`
- `POST /admin-api/v1/configs`
- `PATCH /admin-api/v1/configs/{id}`
- `POST /admin-api/v1/configs/{id}/rotate-secret`
- `GET /admin-api/v1/files/upload-policy`
- `POST /admin-api/v1/files/upload-sessions`
- `PUT /admin-api/v1/files/upload-sessions/{id}/parts/{partNumber}`
- `POST /admin-api/v1/files/upload-sessions/{id}/complete`
- `DELETE /admin-api/v1/files/upload-sessions/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。
- 初始目录包含基本、邮件、短信、登录注册、提现、云存储、地理位置、支付和微信；目录元数据不能通过客户端篡改。
- `storage.driver` 与 `sms.provider` 必须读取 `x-appkernia-dictionary` 对应消费接口，不得在页面维护第二份选项数组；供应商字段按当前选择条件展示。
- 云存储测试上传遵循服务端策略，不展示 Secret、Bucket 或对象键。

## `system.settings.dictionaries` — 字典管理

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `sys.dictionary.read` |
| Schema | `sys.dict_types`, `sys.dict_items` |
| 后端状态 | `existing` |

**筛选**：类型 code/name/status, 条目 label/value/locale/status

**主要动作**：新建类型, 编辑类型, 新建条目, 租户覆盖, 排序, 删除

**UX 规范**：双栏主从布局并在移动端堆叠；内置、租户覆盖、扩展策略、代码能力和不可用状态必须有非颜色单一表达；S3-compatible 扩展使用结构化元数据表单。

**API**

- `GET /admin-api/v1/dictionaries/{code}`
- `GET /admin-api/v1/dict-types`
- `POST /admin-api/v1/dict-types`
- `PATCH /admin-api/v1/dict-types/{id}`
- `GET /admin-api/v1/dict-types/{id}/items`
- `POST /admin-api/v1/dict-types/{id}/items`
- `PATCH /admin-api/v1/dict-items/{id}`
- `DELETE /admin-api/v1/dict-items/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.settings.regions` — 地区管理

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `sys.region.read` |
| Schema | `sys.regions` |
| 后端状态 | `existing` |

**筛选**：code, name, level, parent_code, status

**主要动作**：在省级/市级行新增下级、编辑地区、删除叶子节点

**UX 规范**：树表懒加载；动作按 `create/update/delete` 权限显示；新增与编辑使用抽屉表单，删除必须确认且禁止级联。

**API**

- `GET /admin-api/v1/regions`
- `POST /admin-api/v1/regions`
- `PATCH /admin-api/v1/regions/{code}`
- `DELETE /admin-api/v1/regions/{code}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 编码、父级、层级不可编辑；只有 level 0/1 可新增下级，只有叶子节点可删除。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.users.departments` — 部门

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `org.unit.read` |
| Schema | `org.units`, `org.user_units` |
| 后端状态 | `existing` |

**筛选**：name/code, status

**主要动作**：新建部门, 编辑, 移动, 删除

**UX 规范**：树 + 详情；拖拽仅为增强，必须有键盘可用的移动表单。

**API**

- `GET /admin-api/v1/org/units/tree`
- `POST /admin-api/v1/org/units`
- `PATCH /admin-api/v1/org/units/{id}`
- `POST /admin-api/v1/org/units/{id}/move`
- `DELETE /admin-api/v1/org/units/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.users.accounts` — 用户

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `iam.user.read` |
| Schema | `iam.users`, `iam.tenant_members`, `iam.user_roles`, `org.user_units`, `org.user_positions`, `iam.sessions`, `iam.devices`, `audit.login_events` |
| 后端状态 | `existing` |

**筛选**：关键词, 状态, 部门, 岗位, 角色, 创建时间, 最后登录时间

**主要动作**：新建用户, 编辑, 分配角色/组织/岗位, 启停, 解锁, 重置密码, 导入, 导出

**UX 规范**：部门树筛选 + 数据表；详情用隐藏路由，轻量编辑用 Drawer；破坏性动作展示影响。

**API**

- `GET /admin-api/v1/users`
- `POST /admin-api/v1/users`
- `GET /admin-api/v1/users/{id}`
- `PATCH /admin-api/v1/users/{id}`
- `POST /admin-api/v1/users/{id}/enable`
- `POST /admin-api/v1/users/{id}/disable`
- `POST /admin-api/v1/users/{id}/unlock`
- `POST /admin-api/v1/users/{id}/reset-password`
- `PUT /admin-api/v1/users/{id}/roles`
- `PUT /admin-api/v1/org/users/{user_id}/assignments`
- `GET /admin-api/v1/users/{id}/sessions`
- `DELETE /admin-api/v1/users/{id}/sessions/{session_id}`
- `POST /admin-api/v1/users/import`
- `POST /admin-api/v1/users/export`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.users.positions` — 岗位

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `org.position.read` |
| Schema | `org.positions`, `org.user_positions` |
| 后端状态 | `existing` |

**筛选**：name/code, department, status

**主要动作**：新建岗位, 编辑, 删除, 查看成员

**UX 规范**：按部门筛选；删除前展示成员占用数量。

**API**

- `GET /admin-api/v1/org/positions`
- `POST /admin-api/v1/org/positions`
- `PATCH /admin-api/v1/org/positions/{id}`
- `DELETE /admin-api/v1/org/positions/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.users.tenants` — 租户

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `iam.tenant.read` |
| Schema | `iam.tenants`, `iam.tenant_members` |
| 后端状态 | `existing` |

**筛选**：name/code, status, 创建时间

**主要动作**：新建租户, 编辑, 成员管理, 切换租户

**UX 规范**：仅 multi_tenant=true 显示；切换后清空 tenant-scoped Query Cache 并重新获取 Auth Context。

**API**

- `GET /admin-api/v1/tenants`
- `POST /admin-api/v1/tenants`
- `GET /admin-api/v1/tenants/{id}`
- `PATCH /admin-api/v1/tenants/{id}`
- `GET /admin-api/v1/tenants/{id}/members`
- `POST /admin-api/v1/tenants/{id}/members`
- `PATCH /admin-api/v1/tenants/{id}/members/{user_id}`
- `DELETE /admin-api/v1/tenants/{id}/members/{user_id}`
- `POST /admin-api/v1/auth/switch-tenant`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.access.roles` — 角色

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `iam.role.read` |
| Schema | `iam.roles`, `iam.role_permissions`, `sys.role_menus`, `iam.role_scope_units`, `iam.user_roles` |
| 后端状态 | `existing` |

**筛选**：name/code, status, role_type

**主要动作**：新建角色, 编辑, 分配权限, 分配菜单, 数据范围, 删除

**UX 规范**：角色详情分为基本、权限、菜单、数据范围、成员；权限树与菜单树分离。

**API**

- `GET /admin-api/v1/roles`
- `POST /admin-api/v1/roles`
- `PATCH /admin-api/v1/roles/{id}`
- `DELETE /admin-api/v1/roles/{id}`
- `PUT /admin-api/v1/roles/{id}/permissions`
- `PUT /admin-api/v1/roles/{id}/menus`
- `PUT /admin-api/v1/roles/{id}/data-scope`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.access.permissions` — 权限目录

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `iam.permission.read` |
| Schema | `iam.permissions` |
| 后端状态 | `existing` |

**筛选**：module_code, resource_name, action_name, permission_kind, status, 关键词

**主要动作**：无全局主动作

**UX 规范**：只读目录；按 module/resource 分组并可复制权限码，权限来自代码与 Seed。

**API**

- `GET /admin-api/v1/permissions`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.access.menus` — 菜单

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `sys.menu.read` |
| Schema | `sys.menus`, `sys.role_menus` |
| 后端状态 | `existing` |

**筛选**：title/code, type, status, hidden

**主要动作**：新建目录/页面/外链, 编辑, 移动, 删除, 预览

**UX 规范**：树表；component_key 从静态 Registry 选择；最大三级，禁止任意组件路径。

**API**

- `GET /admin-api/v1/menus/tree`
- `POST /admin-api/v1/menus`
- `PATCH /admin-api/v1/menus/{id}`
- `POST /admin-api/v1/menus/{id}/move`
- `DELETE /admin-api/v1/menus/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.storage.files` — 文件管理

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `storage.file.read` |
| Schema | `storage.files`, `storage.upload_sessions`, `storage.upload_parts`, `storage.file_usages` |
| 后端状态 | `existing` |

**筛选**：文件名, media_type, provider, status, scan_status, visibility, owner, 上传时间

**主要动作**：上传, 下载, 查看引用, 删除

**UX 规范**：表格/缩略图切换；只有 ready 且 clean/skipped 文件可预览或被选择。

**API**

- `GET /admin-api/v1/files/upload-policy`
- `GET /admin-api/v1/files`
- `GET /admin-api/v1/files/{id}`
- `POST /admin-api/v1/files/upload-sessions`
- `GET /admin-api/v1/files/upload-sessions/{id}`
- `PUT /admin-api/v1/files/upload-sessions/{id}/parts/{partNumber}`
- `POST /admin-api/v1/files/upload-sessions/{id}/complete`
- `DELETE /admin-api/v1/files/upload-sessions/{id}`
- `POST /admin-api/v1/files/{id}/presign-download`
- `GET /admin-api/v1/files/{id}/content`
- `GET /admin-api/v1/files/{id}/usages`
- `DELETE /admin-api/v1/files/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.notifications.notices` — 公告管理

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `notify.notice.read` |
| Schema | `notify.messages`, `notify.recipients` |
| 后端状态 | `existing` |

**筛选**：title, status, 发布时间, 过期时间

**主要动作**：新建公告, 编辑, 预览, 发布, 取消

**UX 规范**：message_type=notice 专用管理面；HTML 服务端净化，发布确认展示预计收件人数。

**API**

- `GET /admin-api/v1/notices`
- `POST /admin-api/v1/notices`
- `PATCH /admin-api/v1/notices/{id}`
- `POST /admin-api/v1/notices/{id}/publish`
- `POST /admin-api/v1/notices/{id}/cancel`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.notifications.messages` — 站内消息

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `notify.message.read` |
| Schema | `notify.messages`, `notify.recipients` |
| 后端状态 | `existing` |

**筛选**：message_type, title, status, sender, 发布时间

**主要动作**：新建消息, 编辑, 发布, 取消, 查看收件情况

**UX 规范**：支持 system/private/marketing/security；用户、部门、角色收件条件提交前显示去重人数。

**API**

- `GET /admin-api/v1/messages`
- `POST /admin-api/v1/messages`
- `GET /admin-api/v1/messages/{id}`
- `PATCH /admin-api/v1/messages/{id}`
- `POST /admin-api/v1/messages/{id}/publish`
- `POST /admin-api/v1/messages/{id}/cancel`
- `GET /admin-api/v1/messages/{id}/recipients`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.notifications.templates` — 通知模板

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `notify.template.read` |
| Schema | `notify.templates`, `notify.sms_template_bindings`, `notify.deliveries` |
| 后端状态 | `existing` |

**筛选**：code/name, channel, locale, status

**主要动作**：新建模板, 编辑, 变量校验, 预览, SMS 供应商绑定, 真实测试发送

**UX 规范**：模板元数据 + 编辑/预览；channel 对应用途来自字典；variables_schema 生成样例输入；短信测试明确呈现计费与重复发送风险。

**API**

- `GET /admin-api/v1/notification-templates`
- `POST /admin-api/v1/notification-templates`
- `PATCH /admin-api/v1/notification-templates/{id}`
- `GET /admin-api/v1/notification-templates/{id}/sms-bindings`
- `PUT /admin-api/v1/notification-templates/{id}/sms-bindings/{provider}`
- `DELETE /admin-api/v1/notification-templates/{id}/sms-bindings/{provider}`
- `POST /admin-api/v1/notification-templates/{id}/test`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.notifications.deliveries` — 投递记录

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `notify.delivery.read` |
| Schema | `notify.deliveries` |
| 后端状态 | `existing` |

**筛选**：channel, provider, status, target_hint, 时间范围

**主要动作**：查看详情, 重试失败投递

**UX 规范**：目标只显示脱敏 hint；错误可复制 Request/Provider ID；重试二次确认。

**API**

- `GET /admin-api/v1/notification-deliveries`
- `GET /admin-api/v1/notification-deliveries/{id}`
- `POST /admin-api/v1/notification-deliveries/{id}/retry`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.integrations.schedules` — 定时任务

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `jobs.schedule.read` |
| Schema | `jobs.schedules`, `jobs.schedule_runs` |
| 后端状态 | `existing` |

**筛选**：name/handler_key, status, 时区

**主要动作**：新建计划, 编辑, 暂停/恢复, 立即执行, 执行记录

**UX 规范**：handler_key 来自编译期注册表；Cron 提供人类可读解释和下 5 次执行预览。

**API**

- `GET /admin-api/v1/job-handlers`
- `POST /admin-api/v1/job-schedules/preview`
- `GET /admin-api/v1/job-schedules`
- `POST /admin-api/v1/job-schedules`
- `PATCH /admin-api/v1/job-schedules/{id}`
- `POST /admin-api/v1/job-schedules/{id}/pause`
- `POST /admin-api/v1/job-schedules/{id}/resume`
- `POST /admin-api/v1/job-schedules/{id}/execute`
- `GET /admin-api/v1/job-schedules/{id}/runs`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.integrations.api-clients` — API 客户端

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `sys.api_client.read` |
| Schema | `sys.api_clients`, `sys.api_client_secrets`, `sys.api_client_permissions` |
| 后端状态 | `existing` |

**筛选**：name/client_id, status, 过期时间

**主要动作**：新建客户端, 编辑, 创建/撤销 Secret, 分配权限

**UX 规范**：Secret 仅创建时展示一次，关闭前要求确认已安全保存。

**API**

- `GET /admin-api/v1/api-clients`
- `POST /admin-api/v1/api-clients`
- `GET /admin-api/v1/api-clients/{id}`
- `PATCH /admin-api/v1/api-clients/{id}`
- `POST /admin-api/v1/api-clients/{id}/secrets`
- `DELETE /admin-api/v1/api-clients/{id}/secrets/{secret_id}`
- `PUT /admin-api/v1/api-clients/{id}/permissions`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.integrations.webhooks` — Webhook

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `sys.webhook.read` |
| Schema | `sys.webhook_endpoints`, `sys.webhook_deliveries`, `jobs.outbox_events` |
| 后端状态 | `existing` |

**筛选**：name, event_type, status

**主要动作**：新建端点, 编辑, 发送测试, 查看投递

**UX 规范**：端点 URL 明示 SSRF 限制；投递显示事件 ID、响应摘要和重试轨迹。

**API**

- `GET /admin-api/v1/webhooks`
- `POST /admin-api/v1/webhooks`
- `PATCH /admin-api/v1/webhooks/{id}`
- `POST /admin-api/v1/webhooks/{id}/test`
- `GET /admin-api/v1/webhooks/{id}/deliveries`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.security.operation-logs` — 操作日志

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `audit.operation.read` |
| Schema | `audit.operation_logs` |
| 后端状态 | `existing` |

**筛选**：actor, action, resource_type/id, result, IP, request_id, trace_id, 时间范围

**主要动作**：查看详情, 复制 Request/Trace ID

**UX 规范**：数据密集表格 + 详情 Drawer；before/after JSON 差异视图，服务端先脱敏。

**API**

- `GET /admin-api/v1/audit/operations`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.security.login-logs` — 登录日志

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `audit.login.read` |
| Schema | `audit.login_events` |
| 后端状态 | `existing` |

**筛选**：用户标识 hint, 结果/原因, IP, 平台, 设备, 时间范围

**主要动作**：查看详情, 关联用户/安全事件

**UX 规范**：成功、失败、阻断状态语义一致；不显示完整邮箱、手机号或凭据。

**API**

- `GET /admin-api/v1/audit/logins`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.security.events` — 安全事件

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `audit.security.read` |
| Schema | `audit.security_events` |
| 后端状态 | `existing` |

**筛选**：severity, event_type, status, 用户, IP, 时间范围

**主要动作**：查看详情, 标记处理中, 解决

**UX 规范**：优先级队列；解决必须填写结论并保留审计，严重事件不使用闪烁。

**API**

- `GET /admin-api/v1/audit/security-events`
- `GET /admin-api/v1/audit/security-events/{id}`
- `POST /admin-api/v1/audit/security-events/{id}/resolve`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.security.block-rules` — 访问控制

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `iam.block_rule.read` |
| Schema | `iam.block_rules` |
| 后端状态 | `existing` |

**筛选**：subject_type, subject_hint, scope, status, 过期时间

**主要动作**：新建规则, 编辑, 启停, 撤销

**UX 规范**：使用中性术语；创建前展示影响范围，账号/IP/设备标识只显示脱敏值。

**API**

- `GET /admin-api/v1/block-rules`
- `POST /admin-api/v1/block-rules`
- `PATCH /admin-api/v1/block-rules/{id}`
- `DELETE /admin-api/v1/block-rules/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.monitoring.sessions` — 在线会话

| 项目 | 定义 |
|---|---|
| 阶段 | P1 |
| View Permission | `iam.session.read` |
| Schema | `iam.sessions`, `iam.devices`, `iam.refresh_tokens` |
| 后端状态 | `existing` |

**筛选**：用户, 租户, 平台, IP, 最后活跃时间, 状态

**主要动作**：查看会话, 强制下线

**UX 规范**：强制下线确认展示用户、设备和影响；当前会话有额外警告。

**API**

- `GET /admin-api/v1/online-sessions`
- `DELETE /admin-api/v1/online-sessions/{id}`

**页面验收**

- 刷新后筛选、分页和排序从 URL Search Params 恢复。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `system.monitoring.health` — 服务状态

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `ops.health.read` |
| Schema | `sys.modules`, `jobs.schedule_runs` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：刷新, 复制诊断摘要

**UX 规范**：这是编译模块的唯一展示入口。展示 API、PostgreSQL、Redis、对象存储、Worker/队列健康和统一构建版本；模块使用双语语义名称、稳定编码、说明和能力标签，不暴露连接串。

**API**

- `GET /admin-api/v1/ops/health`
- `GET /admin-api/v1/ops/runtime-summary`

**页面验收**

- 提示、状态卡、依赖卡和运行摘要卡在桌面保持 24px、窄屏保持 16px 垂直间距。
- 三张状态卡等高且在移动端依次堆叠；依赖和模块表格提供可聚焦横向滚动容器，页面本身不产生横向溢出。
- 模块未知编码安全回退为原始 code；状态同时提供可读文本，不只依赖颜色。
- View Permission 在路由层阻断；Action Permission 控制动作展示。
- Loading、Empty、Error、403 和数据刷新状态完整。
- 敏感字段只使用服务端脱敏值，不在前端“还原” Hash/密文。

## `profile.basic` — 基本设置

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| View Permission | `authenticated/self` |
| Schema | `iam.users`, `iam.tenant_members`, `storage.files` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：选择并在浏览器裁剪头像、通过配置的对象存储上传、更新昵称/姓名/语言/时区

**UX 规范**：头像菜单进入；窄列布局；头像裁剪同时支持拖动与键盘控制，上传状态不影响资料表单；保存后同步 Auth Context。

**API**

- `GET /admin-api/v1/me`
- `PATCH /admin-api/v1/me`
- `POST /admin-api/v1/me/avatar/upload-session`
- `PUT /admin-api/v1/me/avatar/upload-sessions/{id}/content`
- `GET /admin-api/v1/me/avatar/content`

**页面验收**

- 只允许当前已登录用户访问，不得通过 URL 参数切换为他人资料。
- 保存、Step-up、撤销和第三方跳转状态完整，防止重复提交。
- 成功变更后精确刷新 Auth Context 与当前用户 Query，敏感信息不持久化。
- Loading、Empty、Error、403、键盘操作和窄屏布局完整。

## `profile.security` — 安全设置

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| View Permission | `authenticated/self` |
| Schema | `iam.user_credentials`, `iam.sessions`, `iam.devices`, `iam.mfa_factors`, `iam.mfa_recovery_codes` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：修改密码, 撤销会话, 管理设备, 配置 MFA

**UX 规范**：高风险动作 Step-up；恢复码仅展示一次。

**API**

- `POST /admin-api/v1/me/password/change`
- `GET /admin-api/v1/me/sessions`
- `DELETE /admin-api/v1/me/sessions/{id}`
- `GET /admin-api/v1/me/devices`
- `DELETE /admin-api/v1/me/devices/{id}`
- `GET /admin-api/v1/me/mfa`
- `POST /admin-api/v1/me/mfa/totp/enroll`
- `POST /admin-api/v1/me/mfa/totp/verify`
- `DELETE /admin-api/v1/me/mfa/totp`
- `POST /admin-api/v1/me/mfa/recovery-codes/rotate`

**页面验收**

- 只允许当前已登录用户访问，不得通过 URL 参数切换为他人资料。
- 保存、Step-up、撤销和第三方跳转状态完整，防止重复提交。
- 成功变更后精确刷新 Auth Context 与当前用户 Query，敏感信息不持久化。
- Loading、Empty、Error、403、键盘操作和窄屏布局完整。

## `profile.connections` — 第三方绑定

| 项目 | 定义 |
|---|---|
| 阶段 | P3 |
| View Permission | `authenticated/self` |
| Schema | `iam.oauth_accounts` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：绑定, 解绑

**UX 规范**：显示 Provider、账号 hint 和绑定时间；解绑最后登录方式前确认已有密码/MFA。

**API**

- `GET /admin-api/v1/me/oauth-accounts`
- `POST /admin-api/v1/me/oauth/{provider}/start`
- `POST /admin-api/v1/me/oauth/{provider}/callback`
- `DELETE /admin-api/v1/me/oauth/{provider}`

**页面验收**

- 只允许当前已登录用户访问，不得通过 URL 参数切换为他人资料。
- 保存、Step-up、撤销和第三方跳转状态完整，防止重复提交。
- 成功变更后精确刷新 Auth Context 与当前用户 Query，敏感信息不持久化。
- Loading、Empty、Error、403、键盘操作和窄屏布局完整。

## `auth.login` — 登录页

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| Route Policy | `anonymous-only` |
| Schema | `iam.sessions`, `iam.refresh_tokens`, `iam.login_failure_states`, `iam.login_captcha_challenges`, `audit.login_events` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：密码登录, 三次失败后图形验证码, 可选 MFA

**UX 规范**：登录成功只接受同源 redirect；错误不泄露账号是否存在；图形验证码由服务端失败状态触发，刷新页面不可绕过。

**API**

- `GET /admin-api/v1/auth/public-config`
- `POST /admin-api/v1/auth/login`
- `POST /admin-api/v1/auth/login/captcha`
- `POST /admin-api/v1/auth/token/refresh`

**页面验收**

- 已登录用户访问匿名页时安全跳转 Dashboard，不形成重定向循环。
- Redirect 只接受同源白名单路径；错误不泄露账号是否存在。
- 提交、限流、服务错误和离线状态完整；密码、验证码、Token 不写日志或持久化。
- 键盘、自动填充、密码管理器和 768px 窄屏流程可用。

## `auth.register` — 注册页

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| Route Policy | `anonymous-only` |
| Schema | `iam.users`, `iam.user_credentials`, `iam.tenant_members` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：注册

**UX 规范**：默认配置关闭；关闭时路由不可用，不只是隐藏入口；协议勾选不可预选。

**API**

- `GET /admin-api/v1/auth/public-config`
- `POST /admin-api/v1/auth/register`

**页面验收**

- 已登录用户访问匿名页时安全跳转 Dashboard，不形成重定向循环。
- Redirect 只接受同源白名单路径；错误不泄露账号是否存在。
- 提交、限流、服务错误和离线状态完整；密码、验证码、Token 不写日志或持久化。
- 键盘、自动填充、密码管理器和 768px 窄屏流程可用。

## `auth.forgot-password` — 找回密码页

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| Route Policy | `anonymous-only` |
| Schema | `iam.verification_challenges` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：发送找回链接/验证码

**UX 规范**：无论账号是否存在均返回相同提示，并展示冷却/限流状态。

**API**

- `POST /admin-api/v1/auth/password/forgot`

**页面验收**

- 已登录用户访问匿名页时安全跳转 Dashboard，不形成重定向循环。
- Redirect 只接受同源白名单路径；错误不泄露账号是否存在。
- 提交、限流、服务错误和离线状态完整；密码、验证码、Token 不写日志或持久化。
- 键盘、自动填充、密码管理器和 768px 窄屏流程可用。

## `system.content.articles` — 文章管理

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `content.article.read` |
| Schema | `content.articles`, `content.article_translations`, `content.categories` |

**筛选**：q、category_id、status、featured、page、page_size、sort。

**主要动作**：新建、编辑、删除、发布、撤回发布、归档。

**API**

- `GET /admin-api/v1/content/articles`
- `POST /admin-api/v1/content/articles`
- `GET /admin-api/v1/content/articles/{id}`
- `PATCH /admin-api/v1/content/articles/{id}`
- `DELETE /admin-api/v1/content/articles/{id}`
- `POST /admin-api/v1/content/articles/{id}/{transition}`

**页面验收**

- URL Search Params 恢复筛选、分页和排序；草稿/发布/归档清晰可见。
- 两种语言内容、封面文件引用、阅读时间、锁版本冲突与权限动作完整。

## `system.content.categories` — 文章分类

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `content.category.read` |
| Schema | `content.categories`, `content.category_translations` |

**筛选**：q、status、page、page_size、sort。

**主要动作**：新建、编辑、删除。

**API**

- `GET /admin-api/v1/content/categories`
- `POST /admin-api/v1/content/categories`
- `GET /admin-api/v1/content/categories/{id}`
- `PATCH /admin-api/v1/content/categories/{id}`
- `DELETE /admin-api/v1/content/categories/{id}`

**页面验收**

- 双语名称与说明、排序、状态和 409 并发冲突状态可用。

## `system.mobile.releases` — 移动端发布策略

| 项目 | 定义 |
|---|---|
| 阶段 | P2 |
| View Permission | `mobile.release.read` |
| Schema | `sys.mobile_releases` |

**筛选**：platform（OpenAPI 列表无查询参数，前端对全局小集合进行 URL 驱动的本地筛选）。

**主要动作**：新建、编辑。

**API**

- `GET /admin-api/v1/mobile/releases`
- `POST /admin-api/v1/mobile/releases`
- `PATCH /admin-api/v1/mobile/releases/{id}`

**页面验收**

- Android、iOS、HarmonyOS 使用稳定协议枚举；平台筛选可从 URL 恢复。
- 当前版本与最低版本只接受核心 `x.y.z` SemVer，最低版本不得高于当前版本；升级地址只接受 HTTPS，生效策略必须配置升级地址。
- 两种语言发布说明均为必填，生效状态同时使用文字和语义样式表达。
- PATCH 携带最新 `lock_version`；409 时保留 Drawer 输入并提示刷新，不静默覆盖。
- 创建、编辑分别受 `mobile.release.create`、`mobile.release.update` 控制。

## `auth.reset-password` — 重置密码页

| 项目 | 定义 |
|---|---|
| 阶段 | P0 |
| Route Policy | `anonymous-only` |
| Schema | `iam.verification_challenges`, `iam.user_credentials`, `iam.password_history` |
| 后端状态 | `existing` |

**筛选**：无

**主要动作**：设置新密码

**UX 规范**：短期 Token 尽快交换；完成后清理 URL、撤销旧会话并跳转登录。

**API**

- `POST /admin-api/v1/auth/password/reset`

**页面验收**

- 已登录用户访问匿名页时安全跳转 Dashboard，不形成重定向循环。
- Redirect 只接受同源白名单路径；错误不泄露账号是否存在。
- 提交、限流、服务错误和离线状态完整；密码、验证码、Token 不写日志或持久化。
- 键盘、自动填充、密码管理器和 768px 窄屏流程可用。
