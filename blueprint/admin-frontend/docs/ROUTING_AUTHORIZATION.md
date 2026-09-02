# 路由、菜单和授权

## 静态 Route Registry

TanStack Router 在构建期生成全部可执行路由。后端 `sys.menus` 只返回 `component_key`、标题、图标、排序、可见性、重定向和 feature flag。前端用静态 Map 解析；未知 key 被丢弃并记录 telemetry，绝不动态 import 后端字符串。

System 的底部齿轮只是通过静态 Registry 解析后的另一种呈现方式：先执行 Feature Flag 与 View Permission 过滤，再从根菜单集合中提取 `code=system`。它不改变路由注册、直接 URL 守卫、`active_menu_code`、数据库菜单分配或后端授权；没有可访问 System 叶子时不渲染齿轮。

公开 `/openapi/` 是独立 Vite 页面而非 TanStack 管理路由，不读取 Auth Context，不预填管理端凭据。其交互请求强制 `credentials: omit` 并发送当前 `Accept-Language`；受保护 API 仅接受用户在文档页面手动输入的 Bearer Token，且 `persistAuth=false`。

OpenAPI canonical YAML 仍是唯一契约。文档入口加载后只在浏览器内存中替换 `x-tagGroups.name`、tag `x-displayName` 和 operation `summary` 用于双语展示，path、method、`operationId`、security 与原始 tag code 不变；这些分组元数据不参与菜单权限、路由守卫或后端授权。直接下载继续返回原始 YAML，不产生 locale-specific 规范。

## 路由守卫顺序

1. 匿名/受保护策略；2. Auth Context；3. feature flag；4. view permission；5. loader 预取。直接输入 URL 与侧栏点击必须得到相同授权结果。

## 权限语义

- View Permission：能否进入页面。
- Action Permission：按钮、批量动作和行操作。
- Data Scope：由 Go API 转成 SQL 条件，前端不能扩大范围。
- Menu Assignment：只影响导航可见性，不能替代 API 授权。

统一使用 `Can`、`useCan`、`requirePermission`，禁止在 JSX 散落字符串判断。403 不刷新 Token；未登录进入登录页；Feature Disabled 使用 404 或明确不可用页。

第三方登录配置页的 View Permission 为 `sys.login_provider_config.read`；创建、更新/启停、删除、Secret 轮换和预检分别使用稳定的 `sys.login_provider_config.*` Action Permission。应用管理中的第三方登录 Tab 只由 `app.login_provider_binding.read` 显示，并由 `app.login_provider_binding.update` 决定是否可编辑；客户端配置 Modal 的 Tab 注册表集中保存这些权限，新增 Tab 不再扩散硬编码判断。

## URL 状态

列表关键词、筛选、分页、排序、选中 Tab 等可分享状态进入类型安全 Search Params。Drawer 的纯临时编辑状态不进入 URL；独立详情页使用 path param 并设置 `active_menu_code`。
