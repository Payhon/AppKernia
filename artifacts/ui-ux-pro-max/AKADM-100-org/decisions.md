# AKADM-100 Organization Decisions

- 继承现有 AppKernia Navy/Blue Master token、系统字体、App Shell 和可见 focus ring；不采用 Skill 的紫橙营销配色、外部 Google Font、Hero/CTA 或滚动特效。
- 部门页采用桌面“树 + 详情/动作”双栏，窄屏改为顺序卡片；拖拽不作为唯一移动方式，始终提供带目标父节点选择的键盘可用移动表单。
- 部门节点由 SQL 按 `tenant_id` 完整隔离并返回，Application 在同一租户结果内组装稳定层级；移动校验使用服务端 Recursive CTE 阻止自身与全部后代，前端候选排除自身但不替代后端防环。
- 删除部门前展示直接成员和子节点占用；占用时服务端返回稳定冲突码，UI 保留上下文并给出下一步。
- 岗位页使用响应式表格并在窄屏提供受控横向滚动，筛选状态进入 URL；删除确认明确成员占用数，成员大于零时由服务端拒绝删除。
- 所有动作同时受路由 view permission、按钮 action permission 与后端授权约束；客户端隐藏按钮不替代服务端校验。
- Loading、Empty、Error、刷新、保存中、成功与冲突状态均使用语义翻译键和 live region；中英文长文本均纳入视觉测试。
