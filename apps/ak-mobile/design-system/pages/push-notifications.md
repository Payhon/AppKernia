# Push Notifications Override

- Route: `/pages/settings/notifications/index`; baseline: `../MASTER.md` and `notifications-settings.md`.
- Keep the in-app and email preferences independent. The push master switch owns two indented category rows: service/security and news/operations; operations remains off on first enable.
- The master switch sequence is legal consent, OS authorization, native channel registration, server device registration, then preference persistence. A failed step restores the previous switch state and exposes one localized recovery action.
- Show OS authorization, selected provider and server registration as three text-labelled states. Status must not rely on color and unavailable channels must explicitly preserve in-app notifications.
- Permanently denied authorization exposes a 44px `open system settings` action. Do not prompt during bootstrap, login, or page entry.
- Security notification previews use a protected summary. Opening a notification and marking it read remain separate actions.
- Long English status text wraps at 375px without squeezing switches; category rows remain at least 52px high.
