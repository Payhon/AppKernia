# AKADM-310 Route Completion Decisions

- Continue using `design-system/appkernia-admin/MASTER.md` as the approved visual source of truth; the newly persisted Skill Master is evidence, not a replacement.
- Add first-class file routes for every registry entry instead of treating a Drawer or catch-all route as equivalent to a declared deep link.
- Scheduled-run history uses a responsive, keyboard-focusable table with URL-backed status/page state and explicit loading/error/empty handling.
- API Client detail reads one tenant-scoped record by UUID and displays only metadata, CIDRs, permission codes, and secret metadata; plaintext credentials are never returned or rendered.
- `/offline` provides a semantic heading plus retry and safe Dashboard navigation. `/404` remains a stable explicit route in addition to the catch-all.
- Reject video/marketing sections, external fonts, new burgundy/gold tokens, logo carousel, transform hover, and scroll-snap because they conflict with the approved Admin system, privacy, performance, and layout stability.
