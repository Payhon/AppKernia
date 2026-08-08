---
title: ak-modal 与 ak-switch
description: 危险确认和布尔设置组件 API。
---

# `ak-modal` 与 `ak-switch`

## `ak-modal`

```vue
<ak-modal
  :open="confirming"
  :title="t('sessions.revoke.title')"
  :message="t('sessions.revoke.message', { device: selectedDevice })"
  :cancel-text="t('common.actions.cancel')"
  :confirm-text="t('sessions.actions.revoke')"
  @cancel="confirming = false"
  @confirm="revokeSession"
/>
```

| 属性          | 类型      | 默认值  |
| ------------- | --------- | ------- |
| `open`        | `Boolean` | `false` |
| `title`       | `String`  | `''`    |
| `message`     | `String`  | `''`    |
| `cancelText`  | `String`  | `''`    |
| `confirmText` | `String`  | `''`    |

事件：`cancel`、`confirm`，均无参数。当前确认按钮固定使用 danger 变体，因此只用于危险/不可逆或安全敏感操作。

## `ak-switch`

```vue
<ak-switch v-model="pushEnabled" @change="savePushPreference" />
```

| 属性         | 类型      | 默认值  |
| ------------ | --------- | ------- |
| `modelValue` | `Boolean` | `false` |

| 事件                | 参数                 |
| ------------------- | -------------------- |
| `update:modelValue` | `nextValue: boolean` |
| `change`            | `nextValue: boolean` |

当前实现为立即切换。远端保存失败时，页面负责回滚并显示可理解的错误，不能让 UI 与服务端事实长期不一致。
