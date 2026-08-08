---
title: Icons, back navigation, and theme
description: Real APIs for ak-icon, ak-back-button, and ak-theme-root.
---

# Icons, back navigation, and theme

`ak-icon` accepts `name: String = 'chevron-right'`, `tone: String = 'primary'`, and `filled: Boolean = false`.

Supported names are `back`, `bookmark`, `search`, `bell`, `home`, `profile`, `security`, `settings`, `device`, `help`, `check`, and `chevron-right`. Unknown names fall back safely to `chevron-right`; the component never loads an arbitrary remote icon.

```vue
<ak-back-button :delta="1" fallback-url="/pages/home/index" />
```

`ak-back-button` uses `delta: Number = 1` and `fallbackUrl: String = '/pages/home/index'`. It calls `uni.navigateBack` when the stack is deep enough, otherwise `uni.reLaunch` to the fallback. Its touch size is 44×44.

```vue
<ak-theme-root>
  <view class="page">…</view>
</ak-theme-root>
```

`ak-theme-root` reads `themeState.isDark`, applies `ak-theme-light` or `ak-theme-dark`, and owns safe-area, page background, and height. Pages continue to use semantic CSS variables instead of hard-coded light/dark colors.
