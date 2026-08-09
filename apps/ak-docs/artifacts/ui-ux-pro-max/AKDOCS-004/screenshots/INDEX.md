# AKDOCS-004 screenshot index

All captures use the final Rspress root-path production preview in Chromium.
`results.json` is the machine-readable result.

| File                                     | Route                    | Locale / theme |  Viewport |
| ---------------------------------------- | ------------------------ | -------------- | --------: |
| `home.zh-CN.light.375.png`               | `/`                      | zh-CN / light  |   375×812 |
| `home.zh-CN.light.768.png`               | `/`                      | zh-CN / light  |  768×1024 |
| `home.zh-CN.light.1024.png`              | `/`                      | zh-CN / light  |  1024×900 |
| `home.zh-CN.light.1440.png`              | `/`                      | zh-CN / light  |  1440×900 |
| `home.zh-CN.light.1920.png`              | `/`                      | zh-CN / light  | 1920×1080 |
| `home.en-US.dark.1440.png`               | `/en-US/`                | en-US / dark   |  1440×900 |
| `home.zh-CN.light.1440.technology.png`   | `/` stack section        | zh-CN / light  |   section |
| `home.zh-CN.light.1440.product-tour.png` | `/` slider section       | zh-CN / light  |   section |
| `home.en-US.dark.1440.technology.png`    | `/en-US/` stack section  | en-US / dark   |   section |
| `home.en-US.dark.1440.product-tour.png`  | `/en-US/` slider section | en-US / dark   |   section |

## Acceptance result

- 6/6 viewport states returned HTTP 200 with one H1 and no page-level horizontal overflow.
- Every state rendered 10 homepage sections, 6 feature cards, 9 technology cards, and 2 sliders.
- Click and keyboard slide changes passed; no automatic advancement occurred.
- Broken images, failed resources, console errors, and serious/critical axe findings: 0.

## Evidence boundary

Admin images come from the repository's local Docker/API Chromium acceptance.
Mobile images come from the recorded iPhone 16 Pro / iOS 18.6 simulator run.
No production or physical-device acceptance is claimed.
