# AppKernia Mobile Design System

## Direction

- Apple HIG-inspired cross-platform product UI for Android, iOS and HarmonyOS portrait screens. It adopts clarity, deference and depth without copying Apple assets or replacing native platform behavior.
- Calm, content-first hierarchy: grouped system backgrounds, elevated white surfaces, restrained AppKernia blue actions, system sans-serif typography and generous breathing room.
- The visual signature is a precise blue focus state, softly grouped cards and consistent line/filled icon pairs. Avoid marketing artwork, excessive glass, gradients, heavy shadows and decorative motion.
- Match supplied mobile references where compatible: 16–20 px page gutters, grouped cards, outline icons and the contracted four-tab information shell.
- Light mode is the Core acceptance baseline. Semantic dark tokens remain available, but dark-mode completion is still gated by `AKMOB-160`.

## Screen chrome and safe areas

- Every custom-navigation page reserves `var(--status-bar-height)` before its 44 px navigation/header row. Backgrounds may extend behind hardware; text and controls may not.
- Root Tab pages use a 44 px header after the safe-area inset and reserve at least 16 px of content clearance above the native TabBar.
- Pushed pages use a 44 px navigation row with a 44 × 44 px back target, centered title and an equal-width trailing slot.
- Scroll content uses flex sizing. Do not use fixed 620/640 px viewport heights; the page must adapt to iPhone SE-class widths, Dynamic Island devices and larger text.
- Native TabBar keeps four semantic destinations (home, browse, topics and profile), text labels and paired outline/filled local assets. Selected state is communicated by icon form and label color, not color alone.

## Semantic tokens

| Token | Value | Usage |
|---|---|---|
| `brand.primary` | `#246BFE` | primary action and active tab |
| `brand.primaryPressed` | `#1557DC` | pressed primary action |
| `surface.page` | `#F2F2F7` | grouped page background |
| `surface.card` | `#FFFFFF` | cards and rows |
| `surface.secondary` | `#F7F7FA` | input and secondary-control background |
| `surface.tertiary` | `#EAF1FF` | selected icon well and quiet callout |
| `text.primary` | `#1C1C1E` | headings and body |
| `text.secondary` | `#6C6C70` | metadata and helper text |
| `text.tertiary` | `#8E8E93` | placeholders and inactive tabs |
| `border.default` | `#E5E5EA` | card and divider border |
| `status.danger` | `#FF3B30` | destructive action with text |
| `status.success` | `#34C759` | success state paired with an icon/label |

Spacing uses 4, 8, 12, 16, 20, 24, 32 and 40 px. Root gutters are 20 px on iOS-sized screens and may reduce to 16 px on the narrowest supported width. Cards use an 18 px radius, controls use 12–14 px, hero surfaces use 22 px and chips use pill radius. Minimum touch target and list row height is 44 px. Use a subtle `0 4px 18px rgba(15,23,42,0.06)` visual elevation only for floating/hero surfaces; grouped lists normally use borders and background layers.

## Typography

- Use the platform system font. Never download a web font for Core UI.
- Large screen title: 30–34 px / bold; navigation title: 17 px / semibold; section title: 20–22 px / semibold; body and row label: 16–17 px; supporting text: 13–15 px.
- Body line height is at least 1.35×. Long content is left aligned and never justified.
- Prefer weight, spacing and semantic color over extreme size changes. English expansion must wrap without hiding actions.

## Components

- Primary buttons are filled AppKernia blue with explicit white label text inside the component. Secondary actions use a quiet surface or text style; destructive actions use semantic red.
- Text fields use a subtle grouped fill, visible focus/error border and explicit label/error text. Forms group related controls inside a single elevated surface.
- Grouped rows use a leading icon well, 16–17 px label, optional secondary value and trailing chevron/switch. Adjacent rows are separated by a hairline inset from the leading content.
- Empty/loading/error states live in a bounded content area with a clear title, optional explanation and one 44 px recovery action. They must never be clipped by the TabBar or action bar.
- Modals use a dim backdrop, rounded surface and horizontally balanced actions with at least 8 px separation.

## Icons

- Use original geometric icons with a 24 × 24 viewBox, rounded caps/joins and consistent 1.8 px optical stroke.
- Standard tabs use outline icons when inactive and filled icons when selected. Do not redistribute SF Symbols or use emoji.
- Page icon buttons are hosted by a 44 × 44 px target. Decorative icons are not separate actions.
- Top-bar and operation glyphs use a 14–16 px optical size inside the unchanged 44 × 44 px target. Form submits, destructive confirmations and ambiguous actions keep text labels.

## Native application identity

- Android, iOS and HarmonyOS launcher/start-window assets are derived from the canonical `apps/ak-admin/public/brand/appkernia-mark.png`; do not ship DCloud/HBuilder default artwork.
- iOS App Store artwork is an opaque 1024 × 1024 PNG. Android launcher artwork uses the platform density matrix and keeps the mark inside the Android 12 splash safe circle.
- HarmonyOS uses an AppKernia blue layered-image background plus the AppKernia foreground mark. The native label is `AppKernia` on all three platforms.
- Native package identity is `com.appkernia.mobile` for the current reusable base. Changing it requires synchronized signing profiles, build scripts, verification and delivery documentation.

## Shared interaction rules

- Push permission and channel state follows `pages/push-notifications.md`; SDK authorization is always initiated by an explicit user action after legal consent.

- The official information-app surfaces additionally follow `pages/information-app.md` for four-tab discovery, controlled content detail, authentication sheets and sharing.

- Every user-visible string is an `AkI18n` key in both catalogs.
- Business pages use only `ak-*` components. The adapter may be implemented on native uni components until the pinned uView module is present.
- Async states are explicit: loading, empty, error, offline, forbidden and mutating. Mutations disable duplicate input and show localised feedback.
- Use outline glyphs or text affordances consistently; do not use emoji as UI icons.
- Reserve safe-area space around the bottom tab/action bars and leave room for English expansion.
- Respect native swipe-back, pull-to-refresh and modal-dismiss gestures; do not bind conflicting horizontal gestures.

## Authentication and legal surfaces

- Authentication pages use one clear primary action, labelled fields, inline validation and a secondary action hierarchy for password recovery and registration.
- Legal links remain visible on the login screen, are never preselected as consent, and open static allowlisted routes.
- Privacy consent is a dedicated pre-bootstrap surface: bundled text is readable offline, primary consent has a 44 px target, and no sensitive SDK or device capability may initialize before acceptance.
- Legal CMS content is rendered as text-only, allowlisted Markdown/blocks; raw HTML, URLs, scripts and remote components are never interpreted by a page.
