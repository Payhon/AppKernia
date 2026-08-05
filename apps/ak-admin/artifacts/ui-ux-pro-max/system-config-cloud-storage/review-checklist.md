# Review checklist

- [x] `zh-CN` and `en-US` keys and placeholders match (`validate_i18n_contract.py`).
- [x] Desktop light-mode screenshots captured at the available Chrome viewport (1800 x 952) in both locales.
- [ ] Dark-mode screenshot captured (the current Admin shell has no dark-mode control).
- [ ] 768 px responsive screenshot captured (the attached Chrome window could not be resized safely).
- [x] Category navigation is keyboard reachable and URL-persistent.
- [x] Loading, empty, failure, secret-unconfigured, and permission-denied states are explicit in the component implementation.
- [x] Upload policy, provider, file-size/type validation, progress, pause/resume/cancel, and retry are implemented.
- [x] File selection and download remain scan-gated.
- [x] No secret, object key, token, or credential is returned or rendered; the bucket field appears only in its authorized configuration editor.
- [x] No horizontal page overflow at the verified desktop viewport.
- [x] Reduced-motion CSS and visible focus behavior are present.

Evidence:

- `screenshots/system-config-desktop-zh-CN.jpg`
- `screenshots/cloud-storage-desktop-zh-CN.jpg`
- `screenshots/cloud-storage-desktop-en-US.jpg`
- `screenshots/catalog-config-edit-protection-en-US.jpg`
