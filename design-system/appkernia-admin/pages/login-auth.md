# Login/Auth Page Override

- Route group: `/login`, `/register`, `/forgot-password`, `/reset-password`, provider callback.
- Desktop 使用 46/54 双栏：左侧 ink 品牌面，右侧 near-white 48px 细网格画布；小于 768px 堆叠。
- 左侧只展示生成的 AppKernia 标志、产品名和既有安全说明，不添加营销 CTA、客户 Logo 或未经确认的品牌口号。
- 品牌背景允许大尺度低透明蓝/青/绿径向氛围光与线性 orbit；不得在小组件内重复渐变。
- 表单卡最大 440px，16px 圆角，白色 96% surface、hairline 与层叠阴影；移动端移除多余悬浮感。
- 标题 600 weight、负字距；表单 label 保持可见；输入 40–48px；错误 `role=alert`，成功 `role=status`。
- 登录保护采用渐进式验证码：默认隐藏；服务端在第三次失败后返回稳定错误码，页面在密码字段与提交按钮之间插入验证码，不以客户端计数决定安全状态。
- 验证码按服务端 Challenge 呈现 `click | slide | drag | rotate`：点击模式提供 44px 标记目标；滑动/拖拽模式同时支持 Pointer 与键盘 Range；旋转模式使用原生 Range。显示尺寸变化时答案坐标必须换算回图片原始尺寸。
- 验证码主图、提示图和拼图块的 alt 只描述用途且不得包含答案；刷新按钮必须有可访问名称与 tooltip。错误紧邻组件并使用 `role=alert`，加载、完成、刷新和过期使用 `role=status`；Challenge 出现或刷新后恢复到首个交互控件焦点。
- 验证码组件无网络职责，由登录页受控传入 Challenge 并接收答案；仅在服务端要求验证码时动态加载。所有动效尊重 `prefers-reduced-motion`。
- 语言入口使用 40px 翻译图标按钮，点击后以右对齐菜单列出 `zh-CN` / `en-US`，当前语言同时通过选中图标和高对比背景表达；图标按钮必须有翻译后的 accessible name。
- Tab 顺序保持语言图标 → 账号 → 密码 → 验证码交互（需要时）→ 刷新 → 辅助动作 → 提交；密码管理器与 autofill 属性不得退化。
- 中英文、375/768/1440、键盘、axe 与无水平滚动必须验证。
