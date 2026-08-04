# Design Decisions

1. The selected leaf keeps the existing white background and ink text.
2. Selected/open directory ancestors use `rgba(255,255,255,.92)` instead of inheriting the leaf's ink color. This separates current-location ancestry from leaf selection while retaining strong contrast.
3. The footer hamburger control is removed. A 28 × 48 px white chevron button is anchored 14 px across the sidebar/content boundary at 50% height.
4. The boundary handle is visually hidden until sidebar hover, but appears on keyboard focus and remains visible for `hover: none` pointers.
5. Expanded state points left (collapse); collapsed state points right (expand). Existing translated `aria-label` strings are reused and `aria-expanded` reports state.
6. The control uses opacity/color/shadow transitions only, with reduced-motion fallback and no hover scaling or layout shift.
