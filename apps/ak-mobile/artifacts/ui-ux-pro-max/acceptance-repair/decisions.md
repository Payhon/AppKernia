# Decisions

- Remove the unsupported percentage `min-height` from the login page root.
- Retain `flex: 1`, page padding, background, and safe-area behavior, so the visual layout does not change.
- Do not replace the page with fixed pixel height or viewport units, which would be less portable across iOS, Android and HarmonyOS.
- Verify with the repository static gate, HBuilderX iOS compiler, and an actual iPhone simulator launch.
