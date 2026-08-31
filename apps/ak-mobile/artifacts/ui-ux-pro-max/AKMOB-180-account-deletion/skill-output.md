# ui-ux-pro-max output

执行查询：

- `mobile account deletion settings destructive link feature flag accessibility --design-system`
- `destructive account deletion link visibility touch target --domain ux`
- `mobile settings destructive action accessibility --stack vue`

采用：

- 不可逆操作必须二次确认，不能点击入口后直接执行。
- 触控目标至少 44×44px；表单字段具备明确标签。
- 输入错误就地显示并通过文字说明，不能只使用红色边框。
- 请求期间展示 loading、禁用重复提交；成功和失败均提供明确反馈及恢复路径。
- 高对比度、系统字体、长文本换行及读屏可理解状态。
- 根 Tab 页的危险操作入口保持 44px 命中区，并与普通设置列表分离；个人中心内容独立滚动，底部账号操作区位于原生 TabBar 上方。

不采用：

- 搜索结果中的 App Store 营销落地页结构、下载 CTA、评分模块与本功能无关。
- 搜索结果中的深色代码主题、绿色 CTA、Fira Web 字体与 Mobile Master 冲突。
- Vue Web 的 VeeValidate/ARIA 细节不直接套用到 uni-app x；使用项目现有类型化状态、AK UI 与平台无障碍属性。
- 不采用仅增加滚动底部占位的方案；在 390×844 实测中入口仍会落入原生 TabBar 覆盖区，因此改为页面内非滚动账号操作区。
