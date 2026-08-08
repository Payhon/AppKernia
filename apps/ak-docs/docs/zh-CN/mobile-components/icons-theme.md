---
title: 图标、返回与主题
description: ak-icon、ak-back-button 与 ak-theme-root 的真实 API。
---

# 图标、返回与主题

## `ak-icon`

```vue
<ak-icon name="security" tone="primary" />
```

| 属性     | 类型      | 默认值          | 说明                                      |
| -------- | --------- | --------------- | ----------------------------------------- |
| `name`   | `String`  | `chevron-right` | 受控图标名                                |
| `tone`   | `String`  | `primary`       | `muted` 对 home/bell/profile 使用弱化图标 |
| `filled` | `Boolean` | `false`         | 对 home/bell/profile 使用 filled 版本     |

当前支持：`back`、`bookmark`、`search`、`bell`、`home`、`profile`、`security`、`settings`、`device`、`help`、`check`、`chevron-right`。未知名称回退到 `chevron-right`，不会加载任意远程图标。

## `ak-back-button`

```vue
<ak-back-button :delta="1" fallback-url="/pages/home/index" />
```

| 属性          | 类型     | 默认值              |
| ------------- | -------- | ------------------- |
| `delta`       | `Number` | `1`                 |
| `fallbackUrl` | `String` | `/pages/home/index` |

页面栈足够时调用 `uni.navigateBack`；否则通过 `uni.reLaunch` 进入 fallback。它的触控尺寸为 44×44。

## `ak-theme-root`

```vue
<ak-theme-root>
  <view class="page">…</view>
</ak-theme-root>
```

组件读取 `themeState.isDark`，切换 `ak-theme-light` / `ak-theme-dark`，并统一安全区、页面背景和高度。业务页面继续使用语义 CSS 变量，不直接写明暗模式颜色。
