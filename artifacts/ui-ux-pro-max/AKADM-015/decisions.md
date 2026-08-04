# AKADM-015 Decisions

- 语言选择使用语义原生 select，避免虚拟 listbox 无可访问名称。
- 匿名初始和最终回退固定 `zh-CN`；显式选择保存非敏感 localStorage。
- i18next、AntD、Day.js、HTML lang、document title 与 API locale 同步。
- 已登录切换先完成本地原子切换，再调用本人资料 API 持久化；保存期间禁用选择器，避免乱序写入。
- 保存失败回滚到原语言，并在选择器附近以双语翻译键和 `role="alert"` 提供可访问反馈；匿名切换不调用受认证 API。
