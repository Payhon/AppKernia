# AKADM-210 UI Decisions

- Regions use a read-only tree table. Root and child requests are separate, children load only when expanded, and the API returns `has_children` without recursively materializing the full dataset.
- Search is server-side and returns a flat match list with stable code/full-name context; clearing search restores root lazy-tree mode.
- Module information is a read-only catalog of compiled registrations: code, version, capabilities and status. No upload/install/uninstall/execute affordance exists.
- Both pages persist filters in URL search params, use visible text plus status color, provide loading/empty/error/retry states, and collapse filters vertically at 375px.
