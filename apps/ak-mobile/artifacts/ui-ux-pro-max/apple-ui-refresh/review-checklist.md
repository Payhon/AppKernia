# Review checklist

- [x] Every custom-navigation page starts below the status-bar safe area through `ak-theme-root`.
- [x] No scroll surface uses a fixed 620/640 px viewport height; the project gate rejects regressions.
- [x] Primary, secondary and destructive button labels have explicit readable colors.
- [x] Home quick entries have dependable two-column and 12 px row separation.
- [x] Native TabBar shows paired outline/filled icons and retained localized text labels.
- [x] Root Tab pages leave content clear of the TabBar in iPhone 16 Pro screenshots.
- [x] Pushed pages expose a 44 × 44 px back control and predictable history/fallback; privacy, registration, recovery and article-list back paths were exercised.
- [x] Cards, grouped rows, fields, states and modals follow the Master tokens.
- [x] Authentication, Home, Notifications, Articles, Profile, Settings, Legal and Error families are visually updated.
- [x] `zh-CN` and `en-US` remain key/placeholder aligned; both Home variants and localized native TabBar labels were visually checked.
- [x] HBuilderX 5.06 iOS compilation succeeds for all 28 pages with no UTS/UCSS error.
- [x] iPhone 16 Pro simulator screenshots cover login, Home, Notifications, Profile and Articles; privacy/recovery/registration/settings were interactively inspected.
- [x] Android and Harmony compilation both succeed; physical-device smoke remains explicitly unexecuted and is not inferred from compilation.
