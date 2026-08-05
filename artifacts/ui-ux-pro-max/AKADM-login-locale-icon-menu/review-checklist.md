# Review Checklist

- [x] Login/auth language control is an icon button rather than a native select.
- [x] Activating the icon opens a two-language menu.
- [x] Current locale has a semantic and visible selected state.
- [x] `zh-CN` / `en-US` switching occurs without page refresh.
- [x] Trigger has a translated accessible name and visible focus.
- [x] Keyboard activation and menu navigation work.
- [x] Anonymous preference behavior and authenticated rollback remain intact.
- [x] 375/768/1024/1440 layouts have no page-level horizontal overflow.
- [x] Real-browser axe and console checks pass.
- [x] Unit tests, lint, typecheck, build, i18n and blueprint validators pass.
- [x] Runtime screenshots are stored under `screenshots/`.

## Runtime evidence

- Docker Chromium against `http://127.0.0.1:4174/login`.
- Native select absent; 40px translation icon present; menu opened and English selected entirely by keyboard.
- `performance.timeOrigin` remained unchanged across the locale switch.
- Current `zh-CN` and `en-US` items both exposed the Ant selectable-menu selected state.
- axe returned zero violations for the open Chinese desktop menu and open English mobile menu; console errors were empty.
