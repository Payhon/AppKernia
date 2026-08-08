---
title: ak-button
description: Usage, props, and events for the AK Mobile button.
---

# `ak-button`

The button covers primary, secondary, and dangerous actions. It has a 44px minimum height and emits no click while loading or disabled.

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

| Prop          | Type      | Default   | Meaning                                                                  |
| ------------- | --------- | --------- | ------------------------------------------------------------------------ |
| `variant`     | `String`  | `primary` | `primary`, `secondary`, or `danger`; unknown values fall back to primary |
| `loading`     | `Boolean` | `false`   | Shows `loadingText` and blocks click                                     |
| `disabled`    | `Boolean` | `false`   | Reduces opacity and blocks click                                         |
| `loadingText` | `String`  | `''`      | Localized loading label                                                  |
| `label`       | `String`  | `''`      | Takes precedence over the default slot when non-empty                    |

`click` emits with no payload only when the control is enabled and not loading. Keep one primary action in a group; use danger only for destructive actions with an explicit consequence and confirmation.
