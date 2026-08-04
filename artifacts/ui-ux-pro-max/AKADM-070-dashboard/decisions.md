# Decisions

- Keep the existing AppKernia system typography and enterprise shell; do not load the suggested external Google Fonts.
- Render only metrics/series returned by the permission-pruned Backend. Missing permission is never displayed as a zero KPI.
- Store the selected 7/30/90-day range in Dashboard URL search params and include it in all three GET requests.
- Lazy-load the ECharts renderer as a dedicated `DashboardTrendChart` chunk; disable its animation under reduced motion.
- Add a keyboard-operable details/table alternative with horizontally scrollable Ant Design Table.
- Keep activity payloads intentionally sparse: stable action/event/error codes and timestamps only. Raw audit details, job output, payloads and error messages are excluded by the Backend.
- Provide per-section loading, retryable error and guided empty states so one failing aggregate does not blank the entire Dashboard.
