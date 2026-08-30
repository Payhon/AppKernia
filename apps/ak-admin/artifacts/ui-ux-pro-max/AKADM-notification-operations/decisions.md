# Decisions

- Keep the existing AppKernia Ant Design semantic tokens and system font; ignore generated external fonts and palette where they conflict with `design-system/MASTER.md`.
- Use one page with four URL-addressable tabs: overview, runs, tasks and failures.
- Use KPI cards plus a lazy ECharts line chart and an accessible data table containing the same trend values.
- Poll every 15 seconds only while the page is visible and unfinished work exists.
- Never expose River args, provider payloads, tokens or stack traces; show stable error code, localized summary, request/trace hint and retry eligibility.
- Batch retry is limited to safe failures. `unknown_after_write` stays a single-item acknowledged action.
