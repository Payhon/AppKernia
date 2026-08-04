# 菜单种子落库规则

`spec/admin-menu-seed.json` 由 `ak seed core` 或版本化 Go Seed 在事务中落入 `sys.menus`，浏览器不得直接写库。

字段映射：`code→code`、`title→title`、`type→menu_type`、`path→route_path`、`component_key→component_key`、`icon→icon`、`sort→sort_order`、`affix→affix`、`feature_flag→metadata.feature_flag`。`permission` 先按 `iam.permissions.code` 解析 `permission_id`，`parent` 按父 code 解析 `parent_id`。

按深度 1→3 upsert，以 `(tenant_id, code)` 为逻辑键；未知权限、未知 Route Registry key、非法图标或循环整批回滚。不要因为新版种子缺少某行就自动删除生产菜单，应使用 `managed_by=core_seed` 与显式 deprecate。更新后增加 menu revision。认证、个人中心和详情页禁止落库。
