# Articles override

- Article list is a pushed page with a safe-area navigation bar, grouped search field, category chips and vertically stacked cards.
- List and detail content use flex sizing instead of fixed pixel viewport heights.
- Reading view has strong title hierarchy, readable line-height, sanitized structured blocks and a safe-area-aware bookmark/share bar.
- Article media is optional; text layout must remain useful without it.
- Every comment header uses the shared 36 px circular `ak-avatar` beside the author name. Missing, unavailable or failed images fall back to the author's initial without shifting comment body or actions.
- Comment avatar URLs must be server-derived public asset paths scoped to the selected App; clients never construct storage keys or attach authentication tokens to public comment images.
- The comment sheet uses an expanded, scrollable layout. Existing comments appear before the composer so identity and conversation context remain visible when the sheet opens.
