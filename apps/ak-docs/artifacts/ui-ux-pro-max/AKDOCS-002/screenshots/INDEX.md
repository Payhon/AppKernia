# Screenshot index

## Source product evidence

| Evidence                                              | Source and boundary                                                                      | Dimensions |
| ----------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------- |
| [Loaded Admin dashboard](source-admin-dashboard.png)  | Isolated local Docker API/Admin, synthetic `.example.test` account and acceptance tenant | 1440×900   |
| [Signed-in Mobile home](source-mobile-home.png)       | Fresh iOS compile, iPhone 16 Pro / iOS 18.6 simulator                                    | 1206×2622  |
| [Signed-in Mobile profile](source-mobile-profile.png) | Local API sign-in, iPhone 16 Pro / iOS 18.6 simulator                                    | 1206×2622  |

The source images contain no password, token, production data, or personal
record. Simulator screenshots are not physical iPhone, Android, or HarmonyOS
runtime acceptance.

## Documentation production preview

Captured from the production Rspress preview at
`http://127.0.0.1:4175/AppKernia/` on 2026-08-09 with Chromium.

| Evidence                                               | Locale / theme  | Viewport  |
| ------------------------------------------------------ | --------------- | --------- |
| [Homepage mobile](home.zh-CN.light.375.png)            | `zh-CN` / light | 375×812   |
| [Homepage tablet](home.zh-CN.light.768.png)            | `zh-CN` / light | 768×1024  |
| [Homepage compact desktop](home.zh-CN.light.1024.png)  | `zh-CN` / light | 1024×900  |
| [Homepage desktop](home.zh-CN.light.1440.png)          | `zh-CN` / light | 1440×900  |
| [Homepage wide desktop](home.zh-CN.light.1920.png)     | `zh-CN` / light | 1920×1080 |
| [English homepage dark](home.en-US.dark.1440.png)      | `en-US` / dark  | 1440×900  |
| [English concepts wide](concepts.en-US.light.1920.png) | `en-US` / light | 1920×1080 |
| [English guide dark](guide.en-US.dark.1440.png)        | `en-US` / dark  | 1440×900  |

All eight samples returned HTTP 200 with one H1, complete images, no horizontal
overflow, no console error, and zero axe serious/critical findings. The wide
Concepts shell measured 1488 px with exactly 216 px on each side; its sampled
paragraph measured 725.625 px. The Rspress build-time dead-link/dead-image and
language-parity checks also passed.
