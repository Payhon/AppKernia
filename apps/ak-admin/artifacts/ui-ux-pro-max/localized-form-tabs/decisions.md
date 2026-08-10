# Decisions

- 继续使用 `design-system/MASTER.md` 的 Ant Design semantic tokens、系统字体和中性数据密集视觉，不引入 Skill 的靛青/绿色配色与 Inter 网络字体。
- 新增共享 `AkLocalizedFormTabs`，使用 Ant Design line Tabs；不构建万能 JSON 表单引擎。
- `system.language` 是锁定的 fixed 系统字典，只提供现有稳定语言的标签、顺序和默认值，不允许后台在线增加协议语言。
- 字典异常时显示可重试错误并禁用保存，禁止静态数组静默降级。
- Tab 错误使用危险色文字语义、错误图标和可访问名称；字段下继续显示具体校验错误。
- `destroyOnHidden=false`，已访问的语言面板保持挂载，RHF 值和 dirty 状态不因切换丢失。
- 使用 Ant Design Tabs 自带的窄屏触控/溢出导航，不覆盖内部 transform/overflow 实现，页面容器不产生横向滚动。
