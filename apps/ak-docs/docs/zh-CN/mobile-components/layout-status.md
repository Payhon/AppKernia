---
title: 布局与状态组件
description: ak-card、ak-cell-list、ak-empty-state、ak-loading 与 ak-status-view。
---

# 布局与状态组件

## `ak-card`

提供统一 surface、border、圆角与 4 级间距，内容由默认 slot 提供。

```vue
<ak-card>
  <text>{{ t('profile.summary') }}</text>
</ak-card>
```

当前无 props 或 events。

## `ak-cell-list`

用于个人中心和设置列表的分组容器，默认 slot 放置具体 cell。当前无 props 或 events。

## `ak-empty-state`

```vue
<ak-empty-state
  :title="t('notifications.empty.title')"
  :description="t('notifications.empty.description')"
>
  <ak-button variant="secondary" :label="t('common.actions.refresh')" @click="reload" />
</ak-empty-state>
```

| 属性          | 类型     | 默认值 |
| ------------- | -------- | ------ |
| `title`       | `String` | `''`   |
| `description` | `String` | `''`   |

默认 slot 用于下一步操作。

## `ak-loading`

`label: String = ''`。适合局部区域；长任务应提供取消，不要用它无限阻塞整个应用。

## `ak-status-view`

把 `loading` 与其他状态统一到同一容器：

| 属性           | 类型     | 默认值  | 说明                                                     |
| -------------- | -------- | ------- | -------------------------------------------------------- |
| `state`        | `String` | `empty` | 等于 `loading` 时显示 AkLoading，其他值显示 AkEmptyState |
| `title`        | `String` | `''`    | 状态标题                                                 |
| `description`  | `String` | `''`    | 状态说明                                                 |
| `loadingLabel` | `String` | `''`    | Loading 文案                                             |
| `retryLabel`   | `String` | `''`    | 非空时展示重试按钮                                       |

事件 `retry` 无参数。页面仍需用稳定状态区分 empty、error、offline、forbidden，不能把所有失败都写成“暂无内容”。
