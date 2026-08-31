# Account deletion override

- This is an authenticated, pushed page with the standard 44 px navigation row, a visible back action and no decorative hero artwork.
- Lead with a compact danger callout that names the current App scope and states that deletion is immediate and irreversible. Explain that other App/Admin identities are outside this action.
- Present deletion consequences and legally anonymised retention as short, scannable rows. Do not hide retention details behind another navigation step.
- The verification card uses the server-provided masked email, a labelled six-digit numeric input, inline error text and a resend action governed by the server countdown.
- The acknowledgement is never preselected. Its entire 44 px row is tappable and uses text plus a visible check state, not colour alone.
- The destructive button uses `status.danger`, remains disabled until the code is six digits and acknowledgement is selected, exposes loading state and is followed by a second native confirmation dialog.
- Keep the primary destructive action inside scroll content with bottom safe-area clearance. Support 360 px width, English expansion and keyboard avoidance without horizontal scrolling.
- The Profile entry is a danger-text link directly below sign-out in the root page's non-scrolling action footer. It keeps a 44 px target, never overlaps the native TabBar and remains hidden for guests or when the latest authenticated context disables `account_deletion`.
- A transient authenticated-context refresh failure keeps the last valid feature snapshot; it must not silently remove the entry or clear a still-valid session.
