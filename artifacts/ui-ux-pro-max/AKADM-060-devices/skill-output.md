# Skill Output

All three repository-local skill commands exited with code `0`.

Relevant returned rules:

- Empty states must guide the user instead of leaving blank space.
- Destructive or irreversible actions require a confirmation dialog.
- Successful actions need explicit feedback; errors must be announced with `role=alert` or an equivalent live region.
- Current/active state and error/success state cannot be communicated by color alone.
- Normal text contrast should be at least 4.5:1.
- React dynamic lists must use stable unique IDs rather than array indexes.
- The generated design-system direction retained the existing enterprise Admin shell and responsive security-card pattern.
