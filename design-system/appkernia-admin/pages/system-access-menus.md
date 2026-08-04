# Access Menus Page Overrides

## Tree editor

- 树表展示标题、code、type、path、component key、status 与 sort；目录、页面、外链字段按类型渐进显示。
- 新建/编辑使用 Drawer；父级只能从合法候选中选择，最大三级，当前节点及后代不得成为父级。
- `component_key` 必须从前端静态 Registry Select 选择，禁止自由文本和动态 import 路径。

## Movement and safety

- 移动通过父级 Select + sort 提供完整键盘路径，拖拽仅可作为增强。
- 删除与移动展示子节点/角色菜单映射影响；循环、超深和占用冲突由服务端稳定错误码返回并本地化。
- 权限仅作为页面访问约束引用，菜单树中不创建“按钮权限”节点。
