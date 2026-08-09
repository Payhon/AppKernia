# AKDOCS-003 screenshot index

All captures below were produced from the final Rspress production build with
Chromium. The machine-readable result is in `results.json`.

| File                                           | Route                                  | Locale / theme |  Viewport |
| ---------------------------------------------- | -------------------------------------- | -------------- | --------: |
| `home.zh-CN.light.375.png`                     | `/`                                    | zh-CN / light  |   375×812 |
| `home.zh-CN.light.768.png`                     | `/`                                    | zh-CN / light  |  768×1024 |
| `home.zh-CN.light.1024.png`                    | `/`                                    | zh-CN / light  |  1024×900 |
| `home.zh-CN.light.1440.png`                    | `/`                                    | zh-CN / light  |  1440×900 |
| `home.zh-CN.light.1920.png`                    | `/`                                    | zh-CN / light  | 1920×1080 |
| `home.en-US.dark.1440.png`                     | `/en-US/`                              | en-US / dark   |  1440×900 |
| `what-is-appkernia.zh-CN.light.1440.png`       | `/guide/what-is-appkernia`             | zh-CN / light  | 1440×1000 |
| `what-is-appkernia.en-US.dark.375.png`         | `/en-US/guide/what-is-appkernia`       | en-US / dark   |   375×812 |
| `architecture.zh-CN.light.1920.png`            | `/concepts/architecture`               | zh-CN / light  | 1920×1080 |
| `architecture.en-US.dark.1024.png`             | `/en-US/concepts/architecture`         | en-US / dark   |  1024×900 |
| `authentication.zh-CN.light.768.png`           | `/concepts/authentication`             | zh-CN / light  |  768×1024 |
| `concepts-index.zh-CN.light.1440.png`          | `/concepts/`                           | zh-CN / light  |  1440×900 |
| `concepts-index.en-US.dark.1440.png`           | `/en-US/concepts/`                     | en-US / dark   |  1440×900 |
| `authentication.en-US.dark.1440.png`           | `/en-US/concepts/authentication`       | en-US / dark   |  1440×900 |
| `permissions-tenancy.zh-CN.light.1440.png`     | `/concepts/permissions-tenancy`        | zh-CN / light  |  1440×900 |
| `permissions-tenancy.en-US.dark.1440.png`      | `/en-US/concepts/permissions-tenancy`  | en-US / dark   |  1440×900 |
| `internationalization.zh-CN.light.1440.png`    | `/concepts/internationalization`       | zh-CN / light  |  1440×900 |
| `internationalization.en-US.dark.1440.png`     | `/en-US/concepts/internationalization` | en-US / dark   |  1440×900 |
| `api-index.zh-CN.light.1440.png`               | `/api/`                                | zh-CN / light  |  1440×900 |
| `api-index.en-US.dark.1440.png`                | `/en-US/api/`                          | en-US / dark   |  1440×900 |
| `mobile-components-index.zh-CN.light.1440.png` | `/mobile-components/`                  | zh-CN / light  |  1440×900 |
| `mobile-components-index.en-US.dark.1440.png`  | `/en-US/mobile-components/`            | en-US / dark   |  1440×900 |

## Acceptance result

- 22/22 routes returned HTTP 200 with one H1 and no page-level horizontal overflow.
- Broken images, failed responses, console errors, and serious/critical axe findings: 0.
- All 14 bilingual diagram routes rendered the exact expected diagram count; each
  shell contains one SVG, one accessible title, and one accessible description.
- Both “What is AppKernia?” pages loaded all 8 product-gallery images.
- At 1920px, the documentation shell measured 1488px with 216px on each side;
  sampled prose width was 725.625px.

## Evidence boundary

The Admin images come from the repository's local Chromium acceptance run. The
Mobile images come from the recorded iPhone 16 Pro / iOS 18.6 simulator run.
They do not claim production, iOS device, Android, or HarmonyOS runtime acceptance.
