# Design Decisions

1. Header utility order is fullscreen, language, account to match the supplied reference and keep destructive sign-out inside the account menu.
2. Utility buttons use Ant Design outline SVG icons at 18 px in 40 px targets; hover changes fill/color without scale or layout movement.
3. Only the authenticated App Shell uses icon language selection. Auth pages retain the explicit select for clarity and existing persistence/error behavior.
4. Locale dropdown uses `selectedKeys` for both visible and semantic current-state communication.
5. Account popup has a non-interactive summary separated from its two menu actions. Roles are displayed from Auth Context exactly as supplied; missing roles use a translated fallback.
6. Avatar content uses the authenticated Blob query and revokes generated object URLs. Initial fallback uses the first Unicode grapheme-like code point of display name, then email, uppercased when applicable.
7. Fullscreen uses the standards-based API, reports state through `aria-pressed`, listens to `fullscreenchange`, and disables itself when unsupported.
