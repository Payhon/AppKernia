# Decisions

- `tmp/ui/mobile` is the visual source of truth; skill output supplies accessibility and interaction constraints.
- Keep the existing AppKernia blue token family and system sans-serif typography.
- Business pages use only `ak-*` components; uView Ultra remains behind the adapter layer.
- Article cards and detail use API-backed localized content. No hard-coded sample article is treated as production data.
- Home composes current-user, unread-count, and featured-article queries; failure of one secondary card does not blank the whole page.
- Profile child destinations cover basic profile, security center, devices, notification settings, language/appearance, help/about, and logout.
- Bookmark state is server-confirmed; share uses a platform port and never injects arbitrary executable content.
- Dark mode stays feature-gated and is not claimed complete in this task.
