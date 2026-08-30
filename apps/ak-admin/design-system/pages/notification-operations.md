# Notification Operations Override

- Use a compact operational hierarchy: page header, global filters, four summary cards, trend/queue panels, then the selected tab table.
- Tabs are URL-addressable and use text labels. Do not hide critical status in tooltips.
- Trend visualization uses the existing lazy ECharts setup and must be followed by a visually compact, screen-reader-friendly data table.
- Queue age and failure counts are emphasized through typography, semantic text and icons, never color alone.
- Retry actions remain row-level by default. Batch retry appears only when all selected records are server-declared safe.
- Automatic refresh is paused when `document.visibilityState` is not `visible`.
- At 768 px, filters wrap and tables retain keyboard scrolling. At 375 px, summary cards stack and records use the shared responsive table/card behavior.
