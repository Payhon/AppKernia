# Login and authentication override

This page inherits `../MASTER.md`.

- Keep login CAPTCHA progressive and server-driven. It is absent by default and appears only after the API returns `IAM.AUTH.CAPTCHA_REQUIRED`; the browser never owns the failed-attempt threshold.
- Render the server challenge as one of `click`, `slide`, `drag`, or `rotate`. Scale pointer coordinates from the rendered viewport back to the image's intrinsic dimensions before submission.
- Every pointer flow has an equivalent keyboard path: arrow keys plus Enter/Space for point selection, native range controls for slide and rotate, and arrow keys plus X/Y ranges for free drag.
- Load the CAPTCHA component only after it is required. The reusable component is controlled and has no network responsibility: its caller supplies the challenge, answer, error, disabled state, refresh action, and focus lifecycle.
- Keep the challenge within the 440 px authentication card, preserve its source aspect ratio, stack range controls on narrow screens, and avoid page-level horizontal overflow at 375 px.
- Use purpose-only image alternative text that does not disclose the answer. Refresh and undo are 44 px icon controls with translated names and tooltips; errors use `role=alert`, while loading and completion use `role=status`.
- A new or refreshed challenge focuses its first interaction control. Do not remove account/password autofill semantics or change the login tab order.
- The only motion is short visual feedback; remove it under `prefers-reduced-motion: reduce`.
- Validate both locales at 1440, 768, and 375 px, all four challenge modes, pointer and keyboard completion, invalid/expired refresh, axe, and no horizontal overflow.
