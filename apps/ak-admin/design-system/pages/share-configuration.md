# Share Configuration Override

## Scope

- Route: `/system/settings/share-configs`
- App binding entry: `/app/applications` row action
- Baseline: `../MASTER.md`; this page does not replace the Admin brand, typography, spacing, or token system.

## Layout

- Use the shared `.ak-page-container` and standard `ak-page-heading ak-org-heading` structure so the page keeps the same centered maximum width and symmetric desktop/mobile gutters as other Admin pages.
- 1440/768: filter card followed by a responsive table; actions wrap inside the action column and the table owns horizontal scrolling.
- 375: replace the table with full-width cards and keep all primary actions at least 44px high.
- Editing and App binding use independent right-side Drawers; widths are 720px and 640px on desktop, full width on narrow screens.

## Safety and state

- WeChat AppSecret is not an input. Android application signature is treated as public native identity but is masked while editing and excluded from audit snapshots.
- `draft`, `active`, and `disabled` combine text and color. Activation validates every enabled platform; disabling states that bound Apps immediately use system sharing.
- Binding requires HTTPS origin, at least one scene, an active provider configuration, and a successful server preflight before save.
- Every save explains that installed binaries do not change until `ak-cli app-share export` and repackaging.

## Accessibility

- Form controls have visible labels; filters and actions retain keyboard access and focus rings.
- Status and preflight results never rely on color alone.
- Long English App names, AppIDs, URLs, and validation copy wrap without page-level overflow.

## WeChat application guidance

- Place one localized question-mark help control beside the configuration Drawer title. The control opens a focused, scrollable modal without leaving or resetting the form.
- Present the application journey as five vertical steps: account qualification, mobile application creation, native identity registration, review/AppID retrieval, and AppKernia binding/rebuild.
- Keep user guidance in the bilingual catalog and provider resource URLs in a dedicated typed registry; do not embed operational copy or changing third-party URLs in the form page.
- Every external resource shows an external-link icon and opens with `target="_blank"` plus `rel="noopener noreferrer"`. Step titles and links must retain WCAG AA contrast on the modal surface.
