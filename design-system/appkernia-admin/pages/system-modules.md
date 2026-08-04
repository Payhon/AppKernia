# Modules Override

- Read-only catalog cards/table show compile-time module code, name, semantic version, capabilities, status and last registration time.
- State explicitly that modules are compiled and deployed with the application. Never render upload, install, uninstall, enable execution, shell or binary controls.
- Filters: code/name query and status, restored from URL search params.
- Capabilities are text tags with an empty state, not executable actions. Status uses text plus color.
- At 375px stack filters and use an accessible internal table scroll; descriptions remain bounded to readable line length.
