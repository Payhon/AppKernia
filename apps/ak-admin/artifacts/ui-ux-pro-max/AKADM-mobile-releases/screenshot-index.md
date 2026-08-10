# Screenshot index

Captured from the local production preview with a deterministic authenticated API fixture. These images verify the page implementation, not a deployed environment or live backend data.

- [Upgrade center, en-US, 375px](../../../../../output/playwright/app-upgrade-center/upgrade.en-US.light.375.png)
- [WGT release Drawer, en-US, 1440px](../../../../../output/playwright/app-upgrade-center/upgrade.wgt-drawer.en-US.light.1440.png)
- [Upgrade center, zh-CN, 1440px, dark OS preference](../../../../../output/playwright/app-upgrade-center/upgrade.zh-CN.preferred-dark.1440.png)
- [Browser evidence](../../../../../output/playwright/app-upgrade-center/e2e-results.json): zh-CN/en-US, 375/768/1024/1440, preferred light/dark media environments, axe, overflow, console and 409 input-preservation checks.

The current Admin shell intentionally has one fixed visual theme. The dark-preference capture verifies that the page remains usable when the operating system reports a dark preference; it is not presented as a dark-theme implementation.
