# Login/Auth Page Override

- Route group: `/login`, `/register`, `/forgot-password`, `/reset-password`, provider callback.
- Desktop 使用 46/54 双栏：左侧 ink 品牌面，右侧 near-white 48px 细网格画布；小于 768px 堆叠。
- 左侧只展示生成的 AppKernia 标志、产品名和既有安全说明，不添加营销 CTA、客户 Logo 或未经确认的品牌口号。
- 品牌背景允许大尺度低透明蓝/青/绿径向氛围光与线性 orbit；不得在小组件内重复渐变。
- 表单卡最大 440px，16px 圆角，白色 96% surface、hairline 与层叠阴影；移动端移除多余悬浮感。
- 标题 600 weight、负字距；表单 label 保持可见；输入 40–48px；错误 `role=alert`，成功 `role=status`。
- Tab 顺序保持语言 → 账号 → 密码 → 辅助动作 → 提交；密码管理器与 autofill 属性不得退化。
- 中英文、375/768/1440、键盘、axe 与无水平滚动必须验证。
