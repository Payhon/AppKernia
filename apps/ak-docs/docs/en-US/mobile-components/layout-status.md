---
title: Layout and status components
description: ak-card, ak-cell-list, ak-empty-state, ak-loading, and ak-status-view.
---

# Layout and status components

`ak-card` provides the shared surface, border, radius, and spacing around its default slot. `ak-cell-list` is the grouped container for profile and settings cells. Neither currently has props or events.

```vue
<ak-empty-state
  :title="t('notifications.empty.title')"
  :description="t('notifications.empty.description')"
>
  <ak-button variant="secondary" :label="t('common.actions.refresh')" @click="reload" />
</ak-empty-state>
```

`ak-empty-state` accepts `title` and `description` strings and a default action slot. `ak-loading` accepts `label`.

`ak-status-view` accepts `state`, `title`, `description`, `loadingLabel`, and `retryLabel`. `state === 'loading'` renders AkLoading; other values use AkEmptyState. A non-empty `retryLabel` adds a button and emits `retry` with no payload.

Pages still model empty, error, offline, and forbidden as distinct stable states. Do not collapse every failure into “no content.”
