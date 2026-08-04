# Design and implementation decisions

1. `DESIGN.md` is translated into Admin semantics, not copied as a marketing page. Ink primary, near-white surfaces, hairlines, technical labels and stacked shadows are global; marketing Hero/CTA/logo-strip patterns remain excluded.
2. The generated blue/cyan/green mark is the only small-scale gradient element. Authentication may use large, low-opacity atmospheric blue/cyan/green light; feature controls remain flat and semantic.
3. Primary actions move from navy to ink `#171717`; links/info remain `#0070F3`; success/warning/error retain distinct accessible semantics.
4. External Geist/Fira downloads are not added. CSS uses local Geist if installed, then Inter/system fallbacks; technical table labels use local monospace fallback.
5. The existing Dashboard line chart is retained because it matches temporal data and already has an accessible table alternative. The palette, grid, tooltip and first-series area treatment are refined.
6. The generated image is original rather than an exact reproduction. A chroma workflow creates a transparent PNG; hard matte was chosen after visual validation showed the soft-matte pass made the white internal A geometry transparent.
7. Dark mode is not claimed. The existing application exposes no theme selector or dark Ant Design algorithm, so this task delivers and validates the current light product theme instead of saving a mislabeled dark screenshot.
8. Dashboard screenshot data uses a deterministic local mock-contract at the HTTP boundary. It validates rendering only and does not claim real API/PostgreSQL integration; existing application integration behavior is unchanged.
