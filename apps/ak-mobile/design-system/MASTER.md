# AppKernia Mobile Design System
## Direction

- Style: minimal Swiss-inspired mobile product UI; calm, professional, content-first, and platform-neutral.
- Scope: Android, iOS, and HarmonyOS portrait layouts in light mode. Dark mode remains out of scope until `AKMOB-160`.
- Brand character: trustworthy engineering foundation with restrained blue accents, generous whitespace, clear information hierarchy, and no decorative clutter.
- Typography: use the platform system sans-serif stack. The visual reference may resemble Inter, but the shipped app must not require a runtime font download.

## Semantic tokens

| Token | Value | Use |
|---|---:|---|
| `brand.primary` | `#2563EB` | Primary actions, active navigation, links |
| `brand.primaryPressed` | `#1D4ED8` | Pressed primary state |
| `surface.page` | `#F8FAFC` | App background |
| `surface.card` | `#FFFFFF` | Cards and grouped rows |
| `surface.elevated` | `#FFFFFF` | Floating controls and sticky bars |
| `text.primary` | `#0F172A` | Titles and primary copy |
| `text.secondary` | `#475569` | Supporting copy |
| `text.disabled` | `#94A3B8` | Disabled state only |
| `border.default` | `#E2E8F0` | Card and divider borders |
| `border.strong` | `#CBD5E1` | Focused or emphasized borders |
| `status.success` | `#15803D` | Success, paired with icon/text |
| `status.warning` | `#B45309` | Warning, paired with icon/text |
| `status.danger` | `#B91C1C` | Destructive actions, paired with icon/text |
| `status.info` | `#0369A1` | Informational state |
| `overlay.scrim` | `rgba(15, 23, 42, 0.48)` | Modal overlay |

- Spacing scale: `0, 4, 8, 12, 16, 20, 24, 32, 40, 48` px.
- Radius: `0, 8, 12, 16, 999` px. Content cards default to 16 px; chips and badges use pill radius.
- Type: display 30/38 semibold; title1 24/32 semibold; title2 18/26 semibold; body 16/24 regular; bodyStrong 16/24 semibold; caption 13/18 regular; button 16/22 semibold.
- Elevation: use borders first; cards may use `0 2px 12px rgba(15, 23, 42, 0.06)`; popups may use `0 12px 32px rgba(15, 23, 42, 0.14)`.
- Motion: fast 150 ms, normal 220 ms, slow 320 ms; reduced motion removes non-essential transitions.

## Components and interaction

- Use one consistent outline icon family with 24 px visual boxes; no emoji UI icons.
- Minimum interactive target is 44 x 44 px with at least 8 px between adjacent targets.
- Cards are functional containers, not decoration. Each card has one clear action or a predictable row affordance.
- Status never relies on color alone. Add text and/or an icon.
- Preserve top and bottom safe areas. Sticky actions never overlap the system gesture region.
- Main scrolling direction is vertical; avoid page-level horizontal swipe gestures that conflict with system back navigation.
- Bottom navigation remains exactly `首页 / 消息 / 我的`; pushed article pages use a back affordance and do not introduce a fourth tab.
- All visible copy maps to semantic i18n keys in `zh-CN` and `en-US`; layouts reserve space for English expansion and dynamic type.

## Image-generation constraints

- Generate high-fidelity product UI, not concept art and not a phone-device marketing mockup.
- Canvas baseline: 390 x 844 portrait. Show the screen edge-to-edge, with a realistic status bar and safe areas.
- Use light mode only, white cards on a very light slate background, crisp typography, and restrained blue accents.
- No gradients except one subtle branded home hero surface; no glassmorphism, neon, 3D icons, excessive shadows, or floating decorative blobs.
- Use plausible product data only. Do not show secrets, tokens, MFA recovery codes, real personal data, or third-party marks.
