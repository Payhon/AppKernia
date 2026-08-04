# AppKernia Mobile

uni-app x + UTS/UVue + VDOM 的移动端工程。当前完成非 UI 的 `AKMOB-000` 与
`AKMOB-030` 基础；因为本机没有 `ui-ux-pro-max`，页面、AK UI 和可视 App Shell
仍保持 blocked。

工具链探测：

    ./scripts/detect-toolchain.sh

真实平台编译入口（不能用 Web/Vite 结果替代）：

    ./scripts/build-platform.sh android
    ./scripts/build-platform.sh ios
    ./scripts/build-platform.sh harmony

静态项目门禁：

    ./scripts/check-project.sh

