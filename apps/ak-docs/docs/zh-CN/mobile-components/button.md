---
title: ak-button
description: AK Mobile 按钮的使用、属性与事件。
---

# `ak-button`

用于主操作、次操作与危险操作。最小高度 44px；`loading` 或 `disabled` 时不会触发 `click`。

```vue
<ak-button
  variant="primary"
  :loading="saving"
  :disabled="!canSubmit"
  :loading-text="t('common.status.saving')"
  :label="t('common.actions.save')"
  @click="submit"
/>
```

也可以用默认 slot：

```vue
<ak-button variant="secondary" @click="cancel">
  {{ t('common.actions.cancel') }}
</ak-button>
```

## Props

| 属性          | 类型      | 默认值    | 说明                                                     |
| ------------- | --------- | --------- | -------------------------------------------------------- |
| `variant`     | `String`  | `primary` | `primary`、`secondary` 或 `danger`；其他值回退到 primary |
| `loading`     | `Boolean` | `false`   | 展示 `loadingText` 并阻止点击                            |
| `disabled`    | `Boolean` | `false`   | 降低透明度并阻止点击                                     |
| `loadingText` | `String`  | `''`      | 加载文案，应传入翻译结果                                 |
| `label`       | `String`  | `''`      | 非空时优先于默认 slot                                    |

## Events

| 事件    | 参数 | 说明                                |
| ------- | ---- | ----------------------------------- |
| `click` | 无   | 仅在非 loading 且非 disabled 时触发 |

## 使用建议

- 一组操作只保留一个 primary。
- 不用 `danger` 表达普通取消；危险操作还需要明确确认与后果说明。
- 网络请求期间保持 `loading=true`，防止重复提交。
