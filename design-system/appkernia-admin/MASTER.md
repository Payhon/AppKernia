# AppKernia Admin Design System Master

> 所有页面先读取本文件，再读取 `pages/<page>.md`。页面 override 只覆盖明确列出的规则。

## Foundations

- Primary `#1E40AF`；Shell navy `#0F2147`；focus `#60A5FA`。
- App background `#F4F7FB`；card `#FFFFFF`；border `#DCE3EE`。
- Primary text `#172033`；secondary text `#536179`；错误 `#991B1B`；成功 `#166534`。
- 字体使用系统栈 `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`；不请求外部字体。
- 间距基于 4/8/12/16/20/24/32/48px；表单控件和触控目标不小于 40px。

## Components

- Primary button 使用蓝底白字；危险动作使用 AntD danger 语义。hover 不允许 transform/scale 引发布局位移。
- 数据卡使用白底、1px 边框、8–12px 圆角和轻量阴影；非交互卡不显示 pointer cursor。
- 表单必须有可访问名称、可见 focus、字段级错误和 `role=alert`；成功/后台刷新使用 `role=status`。
- 数据表提供空态、加载、错误重试、选择计数和批量操作栏；破坏性批量操作必须显示影响数量。
- Drawer 用于轻量编辑；隐藏详情路由用于可链接、可恢复的完整详情。Modal 仅用于确认或短表单。

## Layout and behavior

- App Shell、菜单、租户和语言控件保持一致；页面最大宽度 1440px。
- 375/768/1024/1440 均需验证；窄屏允许非核心表格受控横向滚动，但操作栏与主要动作不能被遮挡。
- URL Search Params 是筛选、分页和排序的恢复事实源。
- 语言切换不刷新页面；所有可见文本使用翻译键并覆盖 `zh-CN`、`en-US`。
- 尊重 `prefers-reduced-motion`；异步操作禁用重复提交，非幂等写请求不自动重放。

## Forbidden

- 营销 Hero、logo carousel、Contact Sales CTA、紫粉渐变、琥珀主 CTA、WebGL/视差和外部 Google Fonts。
- 仅颜色表达状态、低对比辅助文字、不可见 focus、拖拽唯一交互、无确认破坏操作、前端权限代替后端授权。

## Pre-delivery

- [ ] 可见文案全部由翻译键提供，两种语言 key/placeholder 一致。
- [ ] View/action permission 与后端授权均验证。
- [ ] Loading/empty/error/retry/success/conflict 状态齐全。
- [ ] 键盘、axe、reduced-motion、375/768/1024/1440 通过。
- [ ] 真实 API、PostgreSQL、production build 和 Docker E2E 有证据。
