# Decisions

- Follow the existing AppKernia Mobile grouped-card design system rather than generated landing-page recommendations.
- Separate operating-system notification authorization from the user's Push subscription and server device registration.
- Display only capabilities registered by the current build. The initial registry exposes notifications only.
- Page load and `onShow` only refresh status; the system prompt is invoked only by a labelled user action after legal consent.
- File access will use system pickers when introduced and will not request broad Android storage permission.
