# Profile override

- Use a large page title below the safe-area header, then a compact identity card followed by inset-grouped navigation rows for profile/security/devices and preferences/help.
- Each row uses a distinct leading icon well, primary label, optional secondary value and chevron. Repeated generic icons are not acceptable.
- Destructive sign-out is isolated below the regular groups and requires confirmation.
- Navigation rows expose a 44 px target and a visible chevron; profile data stays server-confirmed.
- Security settings include one distinct “Account and sign-in methods” row. Its page groups password, email, mobile and third-party accounts, with a localized status label and explicit bind, replace or unlink action on every row.
- A passwordless account may add its first password from the password row. The password form validates both entries locally, then obtains a scoped one-time step-up proof before saving.
- Account-method actions respect the bottom safe area and remain reachable at large Dynamic Type sizes. Rows are at least 44 pt high, announce provider/identifier plus status and action to VoiceOver, and never convey connection state by colour alone.
- The server-provided `can_unbind`, `block_reason` and `remaining_login_methods` control unlink availability. Bind, replace and unlink require fresh step-up proof; cancellation preserves the current connection.
- The top-right bell opens the message inbox. The row labelled “Notification settings” opens `settings.notifications`; these destinations must remain distinct because the latter manages in-app, email and push preferences rather than message content.
- Authenticated identity surfaces use the shared circular `ak-avatar`: show the server-confirmed image when available and a localized-safe initial fallback while loading or on failure. Never put bearer tokens in image URLs.
- The edit page places avatar editing before text fields. Tapping it opens an explicit camera/gallery choice; an app-owned crop editor provides the same square pan/zoom flow on Android, iOS and HarmonyOS, followed by a separate circular preview. Upload starts only after preview confirmation.
- The crop editor uses the normalized local path returned by `getImageInfo`, not the picker URI. A native `image` layer owns the visible 320 px preview while Canvas is reserved for the final 512 px export, preventing iOS Canvas timing/path failures from producing a blank preview.
- The crop viewport supports one-finger pan and two-finger pinch zoom from 1× to 3×. Zoom out, zoom in and reset are 44 px icon controls floating at the viewport top left with 8 px spacing and localized accessibility labels; text-button duplicates are not allowed.
- The circular crop boundary uses a dark/light dual stroke so it remains visible on both white and dark photos. Controls may overlap the boundary but must never obscure the central subject area.
- Avatar selection, crop, preview and upload are distinct states. Cancellation preserves the current avatar; upload disables duplicate input, exposes progress, and keeps a retry path on failure.
- Camera/gallery permissions are requested only by the user's direct source action. The system picker is preferred for gallery access where supported so broad photo-library access is not required.
- `albumMode: system` is Android-only and must be guarded by the platform compiler; passing it to iOS or HarmonyOS is not allowed.
- Self-closing identity components must be followed directly by the next element; a standalone template angle bracket is visible text, not a second navigation affordance. The identity card exposes exactly one trailing chevron.
