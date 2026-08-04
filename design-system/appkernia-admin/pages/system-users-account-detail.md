# User Account Detail Page Overrides

## Information architecture

- 隐藏路由保留返回列表及原 URL search；顶部显示脱敏身份、租户成员状态和允许动作。
- Tabs：概览、角色与组织、会话与设备、登录活动。每个 Tab 独立 loading/error/retry。
- 角色、部门和岗位分配使用可搜索多选，保存前展示新增/移除差异；服务端最终校验同租户引用。
- 会话仅显示安全摘要；撤销当前操作者自己的会话沿用自作用域警告，管理他人会话要求独立权限。

## Accessibility and feedback

- Tab/Drawer/Modal 完整键盘可达并在关闭后恢复焦点；长英文标签允许换行。
- 状态同时使用文字、Tag 和语义，不仅依赖颜色；所有后台刷新和批量结果使用 live region。
