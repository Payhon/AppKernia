# Review checklist

- [x] All new visible text uses matching `zh-CN` and `en-US` keys; the repository i18n validator passed.
- [x] Built-in, override, extension policy, disabled, and unsupported states include text.
- [x] Dictionary/config/template forms have visible labels and inline validation.
- [x] Loading, empty, error, retry, success, and permission states are represented.
- [x] SMS test delivery shows billing/duplicate-send risk and requires explicit confirmation before submission.
- [x] Keyboard focus and drawer focus trapping use Ant Design semantics and visible focus styles.
- [x] Desktop 1440 and mobile 375 final screenshots have no page-level horizontal overflow; existing responsive rules cover the intermediate breakpoints.
- [x] `prefers-reduced-motion` and existing Ant Design theme behavior remain intact.
- [x] Final Docker build screenshots and axe evidence are saved; all four audited states have zero serious/critical violations and zero console errors.

## Evidence index

- `screenshots/zh-CN-dictionaries-1440.png`: built-in storage driver dictionary with bilingual items.
- `screenshots/en-US-sms-binding-1440.png`: provider dictionary Select and approved Tencent template binding.
- `screenshots/en-US-sms-test-risk-1440.png`: real SMS test warning and duplicate-charge confirmation.
- `screenshots/zh-CN-templates-375.png`: mobile stacked notification-template layout after encrypted email test queued.
- `../../../../../../output/playwright/dictionary-notification-drivers-e2e-results.json`: axe, overflow, API 202, ciphertext, audit, and console evidence.
