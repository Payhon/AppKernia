# Decisions

- Permission is requested only from an explicit master-switch action after legal consent.
- Category switches are disabled while master push is off. Service/security defaults on; operations defaults off.
- Capability state is decomposed into OS authorization, provider and server registration so a failed layer is diagnosable.
- Registration failures roll the master preference back without changing in-app or email preferences.
- No arbitrary notification URL is displayed or executed; routing remains allowlisted by `route_key`.
