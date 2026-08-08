# ui-ux-pro-max output

Query executed: `mobile authentication flex layout safe area form --stack react-native`.

Relevant guidance:

- Preserve controlled form state and type-safe navigation.
- Keep touch targets at least 44 px and preserve safe-area spacing.
- Use the platform-supported flex layout rather than browser-only sizing assumptions.

The repository Mobile Master and `auth-legal` override remain authoritative. The repair only removes the unsupported redundant minimum height; the page already uses `flex: 1`.
