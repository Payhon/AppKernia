# Request

统一优化 AppKernia Admin 中同一业务实体的多语言输入：原生/WGT 发布、内容分类/文章和 App 页面使用共享 Tab 切换，不再纵向平铺语言卡片。Tab 的语言顺序、默认项和标题读取系统字典 `system.language`，当前稳定协议保持 `zh-CN`、`en-US`。

要求保留 React Hook Form 状态、隐藏语言错误提示、键盘可达、窄屏不产生页面级横向溢出，并沿用现有 Ant Design tokens、系统字体和 AK 页面样式。
