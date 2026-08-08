# Request

Redesign the complete AppKernia uni-app x mobile surface because the existing UI looks like a prototype. The supplied iPhone 16 Pro screenshot specifically shows:

- custom header content overlapping the status bar / Dynamic Island;
- quick-entry cards without dependable spacing;
- a native TabBar without icons;
- primary/hero button labels rendering black on a dark blue background;
- inconsistent page chrome and weak hierarchy across Home, Authentication, Articles, Notifications, Profile, Settings, Security, Legal and Error pages.

The implementation must remain uni-app x + UTS/UVue, use only the AK UI adapter from business pages, preserve `zh-CN` / `en-US`, and follow an Apple HIG-inspired direction without copying Apple assets.
