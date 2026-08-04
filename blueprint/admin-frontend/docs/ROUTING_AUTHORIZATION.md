# 路由、菜单和授权

## 静态 Route Registry

TanStack Router 在构建期生成全部可执行路由。后端 `sys.menus` 只返回 `component_key`、标题、图标、排序、可见性、重定向和 feature flag。前端用静态 Map 解析；未知 key 被丢弃并记录 telemetry，绝不动态 import 后端字符串。

## 路由守卫顺序

1. 匿名/受保护策略；2. Auth Context；3. feature flag；4. view permission；5. loader 预取。直接输入 URL 与侧栏点击必须得到相同授权结果。

## 权限语义

- View Permission：能否进入页面。
- Action Permission：按钮、批量动作和行操作。
- Data Scope：由 Go API 转成 SQL 条件，前端不能扩大范围。
- Menu Assignment：只影响导航可见性，不能替代 API 授权。

统一使用 `Can`、`useCan`、`requirePermission`，禁止在 JSX 散落字符串判断。403 不刷新 Token；未登录进入登录页；Feature Disabled 使用 404 或明确不可用页。

## URL 状态

列表关键词、筛选、分页、排序、选中 Tab 等可分享状态进入类型安全 Search Params。Drawer 的纯临时编辑状态不进入 URL；独立详情页使用 path param 并设置 `active_menu_code`。
