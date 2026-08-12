# App Upgrade Dialog Override

This modal page inherits `../MASTER.md`. It is a product safety surface rather than a marketing or app-store landing page.

- Use the existing grouped card, system font and AppKernia blue action tokens. Reject generated web fonts, app-store ratings, screenshots, gradients and decorative artwork.
- Show server-localized title and notes, local-to-target version, mandatory status, download progress and text error/retry feedback.
- Optional upgrades expose a 44 px cancel target. Mandatory upgrades suppress dismissal and retain a retry action after delivery failure.
- Progress, success and failure always include text; color is supplementary. Keep the notes scroll area bounded for English expansion and larger text.
- Respect reduced motion and safe-area tokens. The dialog must remain usable at 360×800, 390×844 and 430×932 portrait sizes.
