# AppKernia Admin Design System Master

> 2026-08-04 依据用户提供的 `DESIGN.md` 与品牌参考图更新。所有页面先读取本文件，再读取 `pages/<page>.md`；页面 override 只覆盖明确列出的规则。

## Brand direction

- 品牌语义是「跨平台基座 + 智能加速」：几何字母 A、向右上方的运动弧线、蓝 → 青 → 绿光谱。
- 主品牌图使用 `/brand/appkernia-mark.png`；侧栏与小尺寸界面使用 `/brand/appkernia-icon-64.png`；浏览器与设备图标使用同目录派生资产。
- 品牌渐变仅用于 Logo、身份识别面和大尺度低透明氛围层，不在按钮、状态标签或小图标上滥用。
- AppKernia 是独立品牌；视觉方向可借鉴 `DESIGN.md` 的工程感与克制感，但不得出现第三方商标或逐像素复制。

## Foundations

- Ink primary `#171717`；primary hover `#333333`；on-primary `#FFFFFF`。
- Canvas `#FFFFFF`；app canvas `#FAFAFA`；inset canvas `#F5F5F5`；hairline `#EBEBEB`；strong hairline `#D8D8D8`。
- Primary text `#171717`；body `#4D4D4D`；muted/description 不浅于 `#666666`；link/info `#0070F3`。
- Success `#087A68`；warning `#A65A00`；error `#D40000`。状态不得只依赖颜色。
- 字体使用本地系统栈 `Geist, Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`；不得请求外部字体。
- 技术标签与表头可使用 `Geist Mono, ui-monospace, SFMono-Regular, Menlo, Monaco, monospace`，正文不得使用等宽字体。
- Display 字重上限 600，使用紧凑负字距；正文 400，按钮 500。基础间距为 4px，常用 8/12/16/20/24/32/48px。

## Components

- Primary button 使用 ink 底白字；危险动作使用 AntD danger 语义；hover 只改变颜色/边框/阴影，不 transform 或 scale。
- 数据卡使用白底、1px hairline、8–12px 圆角与多层轻阴影：inset hairline + 1px/2px 小偏移，禁止单层重阴影。
- 表头使用 `#F7F7F7`、12px 技术标签；行 hover 使用 `#FAFAFA`，窄屏受控横向滚动。
- 表单控件最小 40px，必须有可访问名称、明显 focus ring、字段级错误与提交反馈。
- Drawer/Modal/Popover 使用白色 elevated surface、hairline 和 Level-5 式层叠阴影；Modal 仅用于确认或短表单。
- 安全交互不能只依赖拖拽；验证码等 Pointer 操作必须提供等价键盘控件、状态播报与焦点恢复。
- Loading、Empty、Error、Forbidden、Offline/Stale、409 conflict 状态遵循同一 surface 与文本层级。

## Layout and behavior

- App Shell：桌面侧栏支持展开 248px、折叠 80px、完全隐藏 0px 三态，ink 黑色；三态是非敏感本地 UI 偏好，路由跳转不得改写。顶部 64px 半透明白色并带 blur；内容最大宽度 1440px。
- 认证页：左侧品牌氛围面 + 右侧近白网格画布；移动端堆叠。品牌面不是营销 Hero，不出现 CTA 或客户 Logo。
- 页面标题使用 30–42px/600/负字距；分区标题 18–20px/600；数据密度优先，分区之间保持 32–40px。
- 375/768/1024/1440 均需验证；小于 1024px 侧栏变 Drawer；主要动作与操作列不可被遮挡。
- URL Search Params 是筛选、分页和排序恢复事实源；语言切换不刷新；所有可见文本使用翻译键。
- 尊重 `prefers-reduced-motion`；异步操作禁用重复提交；非幂等写请求不自动重放。

## Accessibility

- 正文至少 4.5:1；大文本至少 3:1；不能使用过浅灰色表达主要信息。
- 所有点击目标有明确 hover/focus；focus ring 使用 `#0070F3` 的高可见混合色。
- 图表提供表格/文本替代；图像有 alt 或明确标记为 decorative；抽屉/对话框关闭后恢复焦点。
- 颜色不是状态、严重度或选择的唯一表达方式。

## Forbidden

- 营销 Hero、logo carousel、Contact Sales CTA、WebGL/视差、外部 Google Fonts。
- 把品牌渐变缩成状态色、按钮底色或大量装饰；在同一屏幕混用营销胶囊按钮和管理端 6px 控件。
- Emoji 图标、动态执行图标路径、重阴影、700/800 display 字重、正文等宽字体。
- 低对比辅助文字、不可见 focus、拖拽唯一交互、无确认破坏操作、客户端权限替代后端授权。

## Pre-delivery

- [ ] 可见文案全部由翻译键提供，两种语言 key/placeholder 一致。
- [ ] View/action permission 与后端授权均验证。
- [ ] Loading/empty/error/retry/success/conflict 状态齐全。
- [ ] 品牌资产透明通道、24/32/64/180/512 尺寸可读性经过检查。
- [ ] 键盘、axe、reduced-motion、375/768/1024/1440 通过。
- [ ] 真实 API、PostgreSQL、production build 和 Docker E2E 的验证边界如实记录。
