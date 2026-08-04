# AKADM-140 Audit and Security Decisions

- 延续既有 Navy/Blue、系统字体、App Shell 与数据密集布局，不引入营销 Hero、外部字体或琥珀主 CTA。
- 操作日志、登录日志、安全事件保持三个静态页面；共享 URL 筛选语义，但不混成难以授权的单页。
- 操作日志详情使用 Drawer 展示稳定元数据及 before/after 字段差异；JSON 只渲染服务端字段级脱敏结果，前端不尝试恢复或猜测秘密。
- 登录日志只展示用户、结果、认证方式、audience、IP、时间和服务端提供的 identifier hint；不返回 identifier hash、完整邮箱/手机号或原始设备 JSON。
- 安全事件列表以文字 + Tag 表达 severity/status；详情使用可深链 URL，resolve 是独立权限动作且显示不可逆影响确认。
- 筛选、页码与排序由 URL Search Params 恢复；375px 收敛非核心列并保留受控内部横向滚动，页面本身不得溢出。
- 非幂等 resolve 不自动 refresh/replay；异步反馈使用 live region，错误使用 `role=alert`，键盘顺序匹配视觉顺序。
