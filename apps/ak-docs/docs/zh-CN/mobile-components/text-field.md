---
title: ak-text-field
description: AK Mobile 文本输入的 v-model、校验错误与属性。
---

# `ak-text-field`

统一标签、输入、密码显隐、禁用和字段错误。组件发出 `update:modelValue`，业务校验仍由 feature schema 管理。

```vue
<ak-text-field
  v-model="form.email"
  :label="t('auth.fields.email')"
  :placeholder="t('auth.fields.emailPlaceholder')"
  :error="errors.email"
  :disabled="submitting"
/>

<ak-text-field
  v-model="form.password"
  :label="t('auth.fields.password')"
  :password="true"
  :error="errors.password"
/>
```

## Props

| 属性          | 类型      | 默认值  | 说明                      |
| ------------- | --------- | ------- | ------------------------- |
| `modelValue`  | `String`  | `''`    | 当前值                    |
| `label`       | `String`  | `''`    | 非空时显示字段标签        |
| `placeholder` | `String`  | `''`    | 占位提示，不能代替标签    |
| `error`       | `String`  | `''`    | 非空时显示错误文案        |
| `password`    | `Boolean` | `false` | 传给原生 input 的密码模式 |
| `disabled`    | `Boolean` | `false` | 禁止输入且不发送更新      |

## Events

`update:modelValue(value: string)`：原生输入变化且组件未禁用时触发。

## 安全提示

密码、OTP 与 MFA 输入不得进入日志、埋点、剪贴板监控或普通 Storage。服务端字段错误应映射为本地翻译，不直接展示内部错误。
