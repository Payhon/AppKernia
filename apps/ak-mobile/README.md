# AppKernia Mobile

uni-app x + UTS/UVue + VDOM 的移动端工程。页面与 AK UI 使用可复核的
`ui-ux-pro-max` 证据；平台编译和模拟器验收仍须按下方真实入口独立记录，不能由静态检查替代。

工具链探测：

    ./scripts/detect-toolchain.sh

真实平台编译入口（不能用 Web/Vite 结果替代）：

    ./scripts/build-platform.sh android
    ./scripts/build-platform.sh ios
    ./scripts/build-platform.sh harmony

AppKernia 三端原生身份与自定义基座：

    ./scripts/build-custom-base.sh assets
    ./scripts/build-custom-base.sh android
    ./scripts/build-custom-base.sh ios-simulator
    ./scripts/build-custom-base.sh harmony

Android/iOS 运行入口固定使用 custom playground；HarmonyOS 直接生成本地 HAP。详见 `docs/custom-native-bases.md`。

仓库根目录提供 macOS/Windows 共用的 Node.js 打包入口：

    pnpm build:mobile:base:preflight
    pnpm build:mobile:base:dry-run
    pnpm build:mobile:base
    pnpm build:mobile:release:preflight
    pnpm build:mobile:release:dry-run
    pnpm build:mobile:release

完整自定义基座和正式版签名方法见：

    docs/manual/mobile-custom-base-build.md
    docs/manual/mobile-production-release.md

静态项目门禁：

    ./scripts/check-project.sh

### Feedback client generation

The feedback DTO generator reads the canonical OpenAPI schemas. Install its pinned parser with `python3 -m pip install -r scripts/requirements.txt`, then run `python3 scripts/generate-mobile-client.py --write`. `check-project.sh` verifies the generated client is current.
