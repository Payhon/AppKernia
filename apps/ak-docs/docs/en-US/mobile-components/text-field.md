---
title: ak-text-field
description: v-model, validation errors, and props for AK Mobile text input.
---

# `ak-text-field`

The field unifies label, input, password mode, disabled state, and field errors. Business validation remains in the feature schema.

```vue
<ak-text-field
  v-model="form.email"
  :label="t('auth.fields.email')"
  :placeholder="t('auth.fields.emailPlaceholder')"
  :error="errors.email"
  :disabled="submitting"
/>
```

| Prop          | Type      | Default | Meaning                                 |
| ------------- | --------- | ------- | --------------------------------------- |
| `modelValue`  | `String`  | `''`    | Current value                           |
| `label`       | `String`  | `''`    | Visible field label                     |
| `placeholder` | `String`  | `''`    | Hint; never a replacement for the label |
| `error`       | `String`  | `''`    | Localized error text                    |
| `password`    | `Boolean` | `false` | Native input password mode              |
| `disabled`    | `Boolean` | `false` | Blocks input and updates                |

`update:modelValue(value: string)` emits on native input when enabled. Passwords, OTPs, and MFA input never enter logs, analytics, clipboard monitoring, or normal storage.
