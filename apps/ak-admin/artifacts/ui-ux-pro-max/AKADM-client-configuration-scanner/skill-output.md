# ui-ux-pro-max output

实际执行：

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "admin client configuration modal tabs scanner allowlist unsaved changes validation accessibility" --design-system -p "AppKernia Admin Client Configuration"
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "modal tabs dirty form close confirmation inline validation keyboard focus responsive" --stack react
```

采用：

- 使用 Ant Design Modal/Tabs 的焦点陷阱、键盘导航和关闭后焦点恢复。
- 用 TypeScript `ClientConfigTabDefinition` 固定 Tab 的 ID、标题、读取权限和组件渲染契约。
- 域名错误就近映射到输入行，并用表单错误语义供读屏读取。
- 375、768、1024、1440 px 作为响应式验收断点；窄屏 Modal 全屏，避免域名输入横向溢出。
- 操作维持明确焦点状态，异步保存阻止重复提交。

未采用：

- Skill 返回的 Video-First Hero、超大标题、橙色营销 CTA、外部字体和着陆页布局；它们与既有 Admin MASTER、Ant Design 管理工作流不一致。
- 任意全局配色与营销动效；继续使用仓库 Token，并尊重减少动效设置。
