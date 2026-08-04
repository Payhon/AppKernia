# AKADM-300 Review Checklist

- [x] TOTP secret and recovery codes are returned once, acknowledged, then removed from component state.
- [x] Server stores encrypted TOTP material and only hashed recovery codes.
- [x] TOTP verification covers clock skew without allowing replay-prone enrollment reuse.
- [x] Disable and recovery rotation require a valid, recent step-up proof.
- [x] OAuth state, PKCE verifier/challenge, code and expiry are validated and single-use.
- [x] Missing provider credentials use an explicit development-only Adapter and feature flag.
- [x] OAuth callback removes code/state from browser history and offers recoverable localized errors.
- [x] All controls have translated accessible names and keyboard-visible focus.
- [x] `zh-CN`/`en-US` switch without refresh; no visible hardcoded copy.
- [x] Axe, secret scans, and 375/768/1024/1440 screenshots are verified.

Evidence: `output/playwright/admin-identity-security-e2e-results.json`; seven final screenshots are indexed in `screenshots.md`. The final E2E run exited 0 with zero axe violations, zero overflow, no unexpected console errors, expected wrong-proof 403, callback replay 422, and no persisted one-time values.
