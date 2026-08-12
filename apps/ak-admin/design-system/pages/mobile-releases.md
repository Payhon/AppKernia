# App Upgrade Center Page Override

This page inherits `../MASTER.md`. The rules below only define the mobile release-policy workspace.

## Information architecture

- The new App upgrade-center route and the compatibility System route render the same component and version fact source.
- Page heading: route-specific title, concise governance description, and a dropdown create action for native and WGT packages. The App selector belongs to the shared shell header immediately before fullscreen.
- Keep `app_id`, query, package type, platform, publication state and pagination in URL search parameters.
- When this route is opened without `app_id`, restore the active tenant's remembered App selection and replace the URL before continuing; never share selection across tenants.
- A single data table shows version, package type, targets, localized title, publication state, current platform pointers, upload time and permitted actions.
- Create and edit use a large right-side Drawer on desktop and a full-width Drawer below the `md` breakpoint.
- Before an App is selected, replace the filters and release table/card list with the shared centered prompt; do not expose draft or publish actions and do not show a loading skeleton.

## Visual and interaction rules

- Reuse the existing Ant Design semantic tokens and AK page/card/table styles. Do not introduce the skill's generic palette or runtime web fonts.
- Keep platforms and package type as text plus neutral tags; draft, online, partial and offline always have translated text as well as semantic color.
- Versions use tabular/monospace text for scanability.
- The Drawer groups package identity and targets, one dictionary-driven localized-content Tabs region, one package source, stores and lifecycle switches.
- Locale tab labels/order come from the locked `system.language` dictionary. Switching tabs preserves draft values; validation marks and focuses the first locale with errors.
- Published history opens read-only. Draft edits preserve values across a 409 response.
- Show a focused 409 conflict message and keep the Drawer open so operators can reload before overwriting newer policy state.

## Responsive rules

- At 1440 px, keep all important columns visible and allow the table region to scroll when necessary.
- At 375 px, stack heading/action/filter and use release cards. At 768px, the dense table may scroll inside its own container.
- No swipe-only controls. Touch targets and focus states remain visible.

## Validation rules

- Platforms are the stable protocol enum `android | ios | harmony`; they are not configurable dictionaries.
- Native packages target exactly one platform. WGT targets one or more platforms and requires a minimum native version.
- For `uni_app_x`, the create action exposes native packages only. Android may use an internal APK or HTTPS/store delivery; iOS and HarmonyOS use HTTPS/store delivery only.
- Historical incompatible rows remain readable and may be taken offline, but publish and republish actions are suppressed with a bilingual warning.
- Package and minimum versions use strict core `x.y.z` SemVer without a leading `v`, prerelease or build metadata.
- Publishing requires both locales and exactly one source: a selected internal file or an absolute HTTPS URL.
- Store selections only use enabled store relations of the selected App.
