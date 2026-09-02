# Third-party Sign-in Provider Configuration

This page inherits `../MASTER.md`.

- Route: `/system/settings/login-providers`; view permission: `sys.login_provider_config.read`.
- Keep the page heading, one-sentence scope, question-mark guide and primary create action on one row at desktop widths. Below 768 px, stack the heading and keep actions aligned without page-level horizontal overflow.
- The question-mark trigger has a tooltip, semantic accessible name, visible focus and at least a 40 px target. It opens one bounded 1040 px Modal; use start-side tabs on desktop and scrollable top tabs on narrow screens.
- Provider guide tabs stay in compiled order: WeChat, GitHub, Apple, Google. WeChat and GitHub cover Android, iOS and HarmonyOS; Apple is iOS-only and Google is Android-Google-only. Each guide follows summary/account type, ordered application steps, exact Admin field mapping, before-save warning, then official HTTPS links opened with `noopener noreferrer`.
- Desktop configuration inventory uses a contained table; below the tablet breakpoint use full-width cards. Status and preflight labels combine text with semantic color, and long client identifiers wrap safely.
- Editor fields are compiled per Provider and checked against the server capability catalog. A catalog/schema mismatch is a blocking error that disables create/edit/bind instead of attempting a generic form.
- Separate basic/public configuration from Secret rotation. Existing Secret values are never returned or represented as fake password dots. The one-time Secret form resets on cancel and success; Apple `.p8` uses a multiline control.
- Lifecycle order is draft or disabled → write Secret if required → preflight → active. Activation is available only after successful preflight. Disable and delete use confirmations that identify the configuration and binding impact; bound configurations cannot be deleted.
- The form exposes GitHub's computed callback URL as read-only. Native build identity fields explain that Admin persistence does not change an installed App and a rebuild/export remains required.
- Preserve URL search state for query, Provider, status and page. Loading, empty, error, catalog mismatch, 403 and 409 are explicit non-overlapping states. A conflict retains edits and never automatically replays an unsafe write.
- Validate both locales at 1440, 768 and 375 px. Include keyboard navigation, modal focus, long English labels, link safety, reduced motion and no plaintext Secret in screenshots, fixtures, telemetry or logs.
