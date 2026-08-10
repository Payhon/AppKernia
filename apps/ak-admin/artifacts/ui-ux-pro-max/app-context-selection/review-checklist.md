# Review checklist

- [x] Uses existing Ant Design tokens and system font stack.
- [x] Shared selector is directly left of fullscreen and route-gated.
- [x] Missing App state contains centered muted text and no table spinner.
- [x] Unselected pages hide scoped filters and create/publish actions.
- [x] App change preserves URL filters and resets existing pagination.
- [x] `zh-CN` and `en-US` use the same translation keys.
- [x] Keyboard-searchable selector has an accessible name.
- [x] Shared prompt has `role=status` and passes component axe validation.
- [x] Real-browser 1440 px desktop screenshot reviewed.
- [x] Real-browser 768 px tablet screenshot reviewed.
- [x] Real-browser 375 px mobile screenshot reviewed.
- [x] Selected-App transition and non-scoped route visibility verified in a real browser.
- [x] Cross-page navigation restores the active tenant's App and writes `app_id` into the destination URL.
- [x] A newly opened Admin page restores selection from browser workspace memory.
- [x] Stored values contain UUIDs only and are isolated by tenant; corrupt/invalid values fail closed.
- [x] Clearing one tenant's selection does not alter another tenant's remembered App.
