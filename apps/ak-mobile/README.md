# AppKernia Mobile

uni-app x + UTS/UVue + VDOM 的移动端工程。页面与 AK UI 使用可复核的
`ui-ux-pro-max` 证据；平台编译和模拟器验收仍须按下方真实入口独立记录，不能由静态检查替代。

工具链探测：

    ./scripts/detect-toolchain.sh

真实平台编译入口（不能用 Web/Vite 结果替代）：

    ./scripts/build-platform.sh android
    ./scripts/build-platform.sh ios
    ./scripts/build-platform.sh harmony

静态项目门禁：

    ./scripts/check-project.sh
