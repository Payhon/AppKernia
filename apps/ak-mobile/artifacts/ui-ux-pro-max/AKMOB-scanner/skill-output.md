# ui-ux-pro-max output

实际执行：

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "mobile qr barcode scanner permission result bottom sheet trusted webview accessibility" --design-system -p "AppKernia Mobile Scanner"
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "scanner camera permission bottom sheet copy result long text focus reduced motion" --stack vue
```

采用：

- 首页入口使用语义 SVG 图标、明确读屏名称和至少 44 × 44px 触控目标。
- 结果和权限恢复使用既有 `ak-bottom-sheet`，输出格式、可换行原文、明确复制按钮和成功/失败反馈。
- 页面行为测试关注输入、事件和可见输出；静态契约检查覆盖事件顺序、订阅释放、single-flight 与 WebView 守卫。
- 正文维持至少 4.5:1 对比度，不只靠颜色传达拦截和错误状态。
- 不新增装饰性动效，沿用 AK UI 安全区和减少动效策略。

未采用：

- Skill 返回的横向滚动旅程、3D 配置器、AI 营销页、外部字体和超大标题；它们不适用于原生扫码任务流，也与 Mobile MASTER 不一致。
- Web hover/cursor 建议；原生端使用触控、读屏名称和平台焦点语义。
