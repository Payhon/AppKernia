---
title: ak-modal and ak-switch
description: API for destructive confirmation and boolean settings.
---

# `ak-modal` and `ak-switch`

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

`ak-modal` accepts `open`, `title`, `message`, `cancelText`, and `confirmText`. It emits `cancel` and `confirm` with no payload. Its confirm button currently uses the danger variant, so reserve it for destructive or security-sensitive actions.

```vue
<ak-switch v-model="pushEnabled" @change="savePushPreference" />
```

`ak-switch` accepts `modelValue: Boolean = false` and emits both `update:modelValue` and `change` with the next boolean. Remote save failures are the page's responsibility: roll back and explain the error so UI and server truth do not drift.
