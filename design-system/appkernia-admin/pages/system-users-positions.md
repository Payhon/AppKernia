# Positions Page Overrides

> **PROJECT:** AppKernia Admin
> **Generated:** 2026-08-03
> **Page Type:** Position directory CRUD

本页继承 `../MASTER.md`；以下规则覆盖 Skill 原始输出中不适用的营销结构和动效。

## Layout

- 使用最大 1200px 内容区；筛选栏、主操作、结果数量和数据表按信息优先级排列。
- 桌面为表格，375/768 窄屏使用岗位卡片；名称、代码、状态、成员数和动作不得被不可见横向列隐藏。
- 关键词与状态筛选写入 URL Search Params，刷新后恢复。

## Interaction and accessibility

- 新建/编辑使用 RHF + Zod Modal 表单，显式 label/id，提交期间禁用重复写入并返回焦点。
- 删除确认必须显示岗位名和服务端成员占用数；成员数大于零时不可提交删除，并以 `role=alert` 说明原因。
- View/Action permission 分离；Loading、Empty、Error/Retry、保存成功和删除冲突都有双语 live feedback。

## Visual

- 复用现有 status tag 和设置页卡片密度；不采用 block-based 娱乐风格、橙色营销 CTA 或大幅滚动动效。
- 表格/卡片文字对比度至少 4.5:1，触控目标至少 44px，focus ring 清晰。

