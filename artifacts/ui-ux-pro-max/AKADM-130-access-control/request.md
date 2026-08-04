# AKADM-130 Access Control Request

为 AppKernia Admin 设计并实现真实角色、权限目录、菜单编辑和数据范围。权限与菜单必须分离；角色详情包含基本信息、权限、菜单、数据范围与成员语义；菜单最大三级、拒绝循环，`component_key` 只能从前端静态 Registry 选择。覆盖 URL 状态、逐动作权限、双语、键盘、错误恢复和 375/768/1024/1440。
