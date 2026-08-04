# 页面任务提示词模板

任务 ID：`<AKADM-xxx>`；页面：`<component_key>`。

1. 读取 AGENTS、Master/页面 override、页面规格、Route Registry、权限矩阵、API/Schema 映射和 OpenAPI。
2. UI Task 先运行 ui-ux-pro-max 并保存证据。
3. 实现类型安全 Search Params、Query/Mutation、权限分支、Loading/Empty/Error/403、响应式和键盘可用性。
4. 不手写重复 DTO，不使用 localStorage Token，不动态 import 后端组件路径。
5. 增加 unit/component/E2E/visual/axe 测试，运行真实质量命令并报告结果。
