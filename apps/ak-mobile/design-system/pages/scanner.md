# Scanner

This surface inherits `../MASTER.md`.

- The Home title bar places one 44 × 44 scan target immediately to the right of Notifications. The 22 px outline glyph keeps the same optical family and the accessible label changes independently of icon color.
- Starting a scan is an explicit user action. While a scanner is active, show the icon-button loading state and reject duplicate taps; a system scanner cancellation returns silently.
- Camera denial opens an `ak-bottom-sheet` with one clear explanation and three ordered actions: Retry, Open system settings, Cancel. The sheet must not imply photo-library access.
- Scan results use `ak-bottom-sheet`, a semantic format label, a wrapping raw-value surface and one explicit Copy action. Long unbroken values wrap within the sheet and never create horizontal overflow.
- A blocked or failed trusted WebView returns to the result sheet with an assertive safety explanation above the untouched original value. Never render scan content as HTML or a route parameter.
- The static WebView page has a persistent 44 px back target and centered title. It accepts only a one-time opaque memory token, exposes no application message bridge, and closes on every non-HTTPS or out-of-allowlist load event.
- Respect platform safe areas, Dynamic Type, screen-reader focus, high contrast and reduced motion. No scan result animation is required.
