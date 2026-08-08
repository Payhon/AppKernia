---
title: AK Mobile components
description: Real ak-* components currently available in AppKernia Mobile.
---

# AK Mobile components

Business pages use components exported by `apps/ak-mobile/components/ak-ui`. AK UI isolates tokens, platform compatibility, touch sizing, event types, and the underlying uView or native implementation.

```text
Feature Page
    ↓
AK UI (ak-*)
    ↓
uView Ultra (up-*) / uni native / platform adapter
```

<div class="ak-doc-callout"><strong>Scope</strong>This reference only documents components that exist in the current source. Conditional or not-yet-implemented entries such as AkPicker, AkDatePicker, and AkTabs are not presented as available.</div>

| Component                           | Purpose                                 | Status      |
| ----------------------------------- | --------------------------------------- | ----------- |
| [`ak-button`](./button)             | Primary, secondary, danger, loading     | Implemented |
| [`ak-text-field`](./text-field)     | Label, input, password, disabled, error | Implemented |
| [`ak-card`](./layout-status)        | Content container                       | Implemented |
| [`ak-cell-list`](./layout-status)   | Settings/profile list container         | Implemented |
| [`ak-empty-state`](./layout-status) | Empty state and next action             | Implemented |
| [`ak-loading`](./layout-status)     | Local loading label                     | Implemented |
| [`ak-status-view`](./layout-status) | Loading/empty/error state composition   | Implemented |
| [`ak-modal`](./modal-switch)        | High-risk confirmation                  | Implemented |
| [`ak-switch`](./modal-switch)       | Boolean settings                        | Implemented |
| [`ak-icon`](./icons-theme)          | Controlled semantic icons               | Implemented |
| [`ak-back-button`](./icons-theme)   | Safe back with fallback                 | Implemented |
| [`ak-theme-root`](./icons-theme)    | Theme root                              | Implemented |

Visible strings are translated by the page through AkI18n and passed in. Touch targets are at least 44×44. Loading/disabled states prevent repeated submission. Business pages never use `up-*` directly. Platform availability still requires matching Android, iOS, and HarmonyOS build/device evidence.
