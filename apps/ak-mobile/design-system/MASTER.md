# AppKernia Mobile Design System

## Direction

- Minimal Swiss-inspired product UI for Android, iOS and HarmonyOS portrait screens.
- Calm, content-first hierarchy: white cards on a light slate page, restrained blue actions and system sans-serif typography.
- Match the supplied `tmp/ui/mobile` references: 16 px page gutters, grouped cards, outline icons and a fixed three-tab shell.
- Light mode is the Core baseline. Dark mode remains feature-gated until `AKMOB-160` acceptance exists.

## Semantic tokens

| Token | Value | Usage |
|---|---|---|
| `brand.primary` | `#2563EB` | primary action and active tab |
| `brand.primaryPressed` | `#1D4ED8` | pressed primary action |
| `surface.page` | `#F8FAFC` | page background |
| `surface.card` | `#FFFFFF` | cards and rows |
| `text.primary` | `#0F172A` | headings and body |
| `text.secondary` | `#64748B` | metadata and helper text |
| `border.default` | `#E2E8F0` | card and divider border |
| `status.danger` | `#DC2626` | destructive action with text |

Spacing uses 4, 8, 12, 16, 20, 24, 32 and 40 px. Cards use a 16 px radius; controls use 12 px; chips use pill radius. Minimum touch target is 44 px. Use a 0 2 px 12 px rgba(15,23,42,0.06) shadow only when a border cannot establish hierarchy.

## Shared interaction rules

- Every user-visible string is an `AkI18n` key in both catalogs.
- Business pages use only `ak-*` components. The adapter may be implemented on native uni components until the pinned uView module is present.
- Async states are explicit: loading, empty, error, offline, forbidden and mutating. Mutations disable duplicate input and show localised feedback.
- Use outline glyphs or text affordances consistently; do not use emoji as UI icons.
- Reserve safe-area space around the bottom tab/action bars and leave room for English expansion.

## Authentication and legal surfaces

- Authentication pages use one clear primary action, labelled fields, inline validation and a secondary action hierarchy for password recovery and registration.
- Legal links remain visible on the login screen, are never preselected as consent, and open static allowlisted routes.
- Privacy consent is a dedicated pre-bootstrap surface: bundled text is readable offline, primary consent has a 44 px target, and no sensitive SDK or device capability may initialize before acceptance.
- Legal CMS content is rendered as text-only, allowlisted Markdown/blocks; raw HTML, URLs, scripts and remote components are never interpreted by a page.
