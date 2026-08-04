# Departments Page Overrides

> **PROJECT:** AppKernia Admin
> **Generated:** 2026-08-03
> **Page Type:** Organization tree management

本页继承 `../MASTER.md`；以下规则覆盖 Skill 原始输出中不适用的营销结构和配色。

## Layout

- 桌面使用最大 1400px 的“部门树 + 选中部门详情”双栏；树栏最小 320px，详情栏自适应。
- 768px 以下改为顺序卡片，筛选、树、详情与动作保持同一阅读/Tab 顺序；不得依赖横向滚动完成核心操作。
- 筛选条件使用 URL Search Params；选中节点可使用稳定 UUID 查询参数恢复，但不把租户标识作为可写输入。

## Interaction and accessibility

- 树节点使用语义 Tree/TreeItem，支持方向键；所有编辑动作使用具名 Button。
- 拖拽仅为增强；移动必须提供 Modal 表单与父节点 Select，关闭后焦点返回触发按钮。
- 新建、编辑、移动按钮显示 loading 防重复；成功用 `role=status`，校验/冲突用 `role=alert`。
- 删除确认展示子节点数和直接成员数；占用时提供取消和返回处理成员/子节点的明确路径。

## Visual

- 沿用 Navy/Blue、白色数据卡、8px 圆角和现有高对比辅助色；不使用 Skill 的紫橙营销主题。
- 树选中态同时使用背景、边框和文字权重，不仅依赖颜色；reduced-motion 下关闭非必要过渡。

