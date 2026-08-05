# Mobile Releases Page Override

This page inherits `../MASTER.md`. The rules below only define the mobile release-policy workspace.

## Information architecture

- Page heading: mobile release policy title, concise governance description, and the primary create action.
- One filter row for platform. Filtering is client-side because the contracted list endpoint returns the complete small global policy collection and declares no query parameters.
- A single data table shows platform, current version, minimum supported version, active state, upgrade destination, last update, and actions.
- Create and edit use a large right-side Drawer on desktop and a full-width Drawer below the `md` breakpoint.

## Visual and interaction rules

- Reuse the existing Ant Design semantic tokens and AK page/card/table styles. Do not introduce the skill's generic palette or runtime web fonts.
- Keep platform as text plus a neutral tag; active state uses text plus a semantic tag so color is not the only signal.
- Current and minimum versions use tabular/monospace text for scanability.
- Long upgrade URLs truncate visually but retain a safe external link and accessible label.
- The Drawer groups policy fields first, followed by separate `zh-CN` and `en-US` release-note cards.
- Platform becomes read-only while editing because the update contract addresses an existing policy by id; changing platform would make policy identity ambiguous.
- Show a focused 409 conflict message and keep the Drawer open so operators can reload before overwriting newer policy state.

## Responsive rules

- At 1440 px, keep all important columns visible and allow the table region to scroll when necessary.
- At 375 px, stack heading/action/filter, hide upgrade URL and update-time columns, and keep platform/version/status/action operable without page-level horizontal overflow.
- No swipe-only controls. Touch targets and focus states remain visible.

## Validation rules

- Platforms are the stable protocol enum `android | ios | harmony`; they are not configurable dictionaries.
- Current and minimum versions must use the backend's core `x.y.z` SemVer form without a leading `v`, prerelease, or build metadata.
- Minimum version must not be newer than current version.
- Upgrade URL is optional for inactive policies and must be an absolute `https://` URL when present; active policies require it.
- Both release-note locales are required.
