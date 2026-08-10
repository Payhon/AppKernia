# Request

Date: 2026-08-10

Optimize all App-scoped Admin pages so that a missing App selection never leaves the table in a permanent loading state. Replace the existing inline tip/Alert with a shared minimum-height prompt whose muted text is centered both vertically and horizontally.

Move App selection out of page content into a reusable shell-level selector, immediately to the left of fullscreen. The selector must only appear on routes that require an App context and must remain usable at desktop, tablet and mobile widths.

Follow-up: make the selected App apply across every App-scoped page and remember it after refresh. The preference must remain tenant-scoped, URL-compatible and explicitly clearable.
