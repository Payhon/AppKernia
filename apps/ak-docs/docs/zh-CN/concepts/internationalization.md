---
title: 国际化
description: AppKernia 在 Backend、Admin 与 Mobile 之间共享的语言契约。
---

# 国际化

首发规范语言只允许 `zh-CN` 和 `en-US`，默认与最终回退都是 `zh-CN`。`zh`、`zh-Hans`、`zh_CN`、`en` 等输入别名会先规范化。

## 解析顺序

登录用户：

```text
服务端用户 locale
> 当前客户端显式选择
> Accept-Language / 设备语言
> zh-CN
```

所有请求发送 `Accept-Language`，后端返回 `Content-Language`；会因语言变化的缓存响应同时返回 `Vary: Accept-Language`。

## 错误与数据

- 业务逻辑只判断稳定错误码，不解析展示文案。
- 时间保持 RFC 3339 UTC，金额保持最小货币单位整数。
- Admin 同步 i18next、Ant Design、Day.js、HTML `lang` 与标题。
- Mobile 同步 AkI18n、导航标题、TabBar、AK UI 与后续请求。
- 两套语言包的 key 与占位符必须完全一致。
