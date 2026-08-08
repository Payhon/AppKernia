# Decisions

1. Retain the existing blue full-width Login button as the only primary action.
2. Replace the recovery and registration secondary buttons with two semantic text-link tap targets in one row below Login.
3. Use the existing brand and text tokens; do not introduce new colors, fonts, icons or decorative effects.
4. Preserve `registration_enabled`: the registration link is absent when the backend disables registration.
5. Fix navigation once in `ak-back-button` by using an explicit click handler and `uni.navigateBack`; guest and legal pages declare a login fallback for empty history.
6. Verify coordinate taps in the real iOS simulator, not only accessibility-element activation, because the reported defect is physical hit behavior.
