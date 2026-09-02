# AppKernia 三端自定义基座

Android 和 iOS 使用 HBuilderX 自定义调试基座；HarmonyOS 官方没有“基座”概念，由 HBuilderX 调用本机 DevEco 直接生成 AppKernia HAP。三端统一原生标识为 `com.appkernia.mobile`，均使用仓库内 AppKernia 品牌资产，不接受 `io.dcloud.uniappx` 和 DCloud 默认图标作为交付物。

## 生成与检查资产

```bash
cd apps/ak-mobile
./scripts/build-custom-base.sh assets
```

源图固定为 `apps/ak-admin/public/brand/appkernia-mark.png`。脚本生成 Android 密度图标/Android 12 启动图标、无 Alpha 的 iOS 1024 图标、HarmonyOS layered image 和圆形 Android 原生资源。HarmonyOS layered background/foreground 按 DevEco 模板使用 288 × 288，start icon 使用 144 × 144，避免系统把低分辨率前景放大后丢失品牌细节。禁止手工替换某一个尺寸而不重新运行契约检查。

## 构建

```bash
# Android 自定义调试基座 APK（云打包，调试公共证书）
./scripts/build-custom-base.sh android

# iOS 自定义模拟器基座（云打包）
./scripts/build-custom-base.sh ios-simulator

# HarmonyOS 本地 HAP（先生成 unsigned 工程；DevEco OHPM/Hvigor 构建）
./scripts/build-custom-base.sh harmony

# 三端连续执行
./scripts/build-custom-base.sh all
```

Android/iOS 云打包前必须已登录 HBuilderX。iOS 真机基座另需与 `com.appkernia.mobile` 匹配的开发证书和 Provisioning Profile；不得复用本机其他公司或其他 App 的签名材料。所有密码、证书私钥和 HarmonyOS 本机签名配置都留在用户目录或忽略目录中，不写入仓库。

HarmonyOS 流水线不会信任 HBuilderX 生成工程里的默认 AppScope。它在每次编译后运行 `prepare-harmony-native.py`，把 `harmony-configs/AppScope` 的 `com.appkernia.mobile`、名称和图标覆盖到忽略目录中的原生工程，把运行时资源目录规范化为当前 `manifest.appid`，再调用 DevEco `ohpm`/`hvigorw`。脚本还会读取 `harmony-configs/entry/src/main/oauth-links.generated.json`，仅把 allowlist 内的微信 `weixin` query scheme、`action.system.home`/`wxentity.action.open` action 和 GitHub HTTPS host/path 精确合并进 `EntryAbility.skills`；未知字段、非 HTTPS 回跳和宽泛 host 会直接拒绝。默认移除 HBuilderX 生成目录里的签名引用并生成 `entry-default-unsigned.hap`，避免把属于 `io.dcloud.uniappx` 的旧调试签名错误用于 AppKernia。

HarmonyOS 微信授权边界使用 DCloud uni-app x 的 `uni.login({ provider: 'weixin', onlyAuthorize: true })` 封装；按 DCloud 当前 HarmonyOS 登录能力说明，该实现固定接入腾讯 `@tencent/wechat_open_sdk` 并只向业务层返回一次性授权码。它不是 Web OAuth 或动态插件。静态检查只证明 UTS 调用了该封装、Harmony overlay 精确生成；仍必须使用真实微信开放平台 AppID、签名 HAP 与真机微信客户端验证拉起、取消、回跳和授权码消费，未完成这些门禁时不得写成真机通过。

需要安装到 HarmonyOS 真机时，先执行一次上述 `harmony` 命令生成原生工程，并通过 USB/IP 连接已开启调试的目标设备。再在 DevEco Studio 打开 `unpackage/dist/dev/app-harmony`，进入 **File > Project Structure > Project > Signing Configs**，为 `com.appkernia.mobile` 配置自动或手工调试签名。DevEco Managed Profile 会绑定在线设备；未连接设备时会报 `Unable to create the profile due to a lack of a device`，不会生成可用于真机的 Profile。签名动作会在本机创建或引用证书、私钥和 Provision Profile，必须由有权使用该华为开发者帐号的人确认。

若改用本地 HarmonyOS 模拟器，可跳过设备绑定签名并安装 unsigned HAP；首次打开 DevEco **Tools > Device Manager** 可能要求用户阅读并接受 HarmonyOS Software License and Service Agreement，下载系统镜像时还可能单独要求接受 HarmonyOS SDK License Agreement，两项法律协议都必须分别获得用户即时明确授权。本仓库 2026-08-26 的本机操作已分别获得两份协议的明确授权并完成接受，随后下载 HarmonyOS 6.0.2（API 22）官方 Phone ARM64 镜像。创建官方模拟器还要求目标磁盘至少有 10 GB 可用空间；空间不足时不要把 Device Manager 可打开误报为模拟器已安装。

配置完成后只构建现有 DevEco 原生工程：

```bash
./scripts/build-custom-base.sh harmony-signed
```

`harmony-signed` 不再调用 HBuilderX 重新生成工程，因此不会覆盖刚生成的 DevEco 签名配置。构建前会检查 default product 确实绑定 Signing Config、三个签名文件存在，并使用 DevEco 官方 `hap-sign-tool.jar` 验证 Provision Profile 绑定的是 `com.appkernia.mobile`；检查过程不输出密码、Alias 或材料路径。所有签名材料只保留在本机忽略目录/用户目录，不读取到报告、不复制、不提交。

HBuilderX 内置 OHPM 若受代理影响失败，只要原生工程已生成，脚本会用清除大小写代理变量后的独立 OHPM/Hvigor 命令继续，并保留真实退出码。若首次生成原生工程前就失败，应清除 HBuilderX 进程的代理环境后重新启动 HBuilderX，再执行同一命令。

构建完成后运行：

```bash
./scripts/build-custom-base.sh verify
```

检查会读取 APK/IPA/HAP 的原生标识；只有实际产物中不再出现 DCloud 默认包名，并且 AppKernia 名称/Bundle ID 通过，才能称为自定义基座产物。图标检查不是“文件存在”占位：Android APK 的四档 launcher icon 必须与生成资产逐像素一致，iOS 打包 AppIcon 必须与 1024 品牌主图缩放结果匹配，Harmony HAP 的 layered/start icon 必须与 AppScope overlay 逐字节一致。普通 `verify` 会明确区分 unsigned/signed HAP；真机可安装交付必须额外执行：

```bash
./scripts/build-custom-base.sh verify-installable
```

该命令用 DevEco 官方签名工具验证 HAP 签名块和内嵌 Provision Profile，而不是根据文件名推断。也可在单个平台构建后执行 `python3 scripts/verify-custom-base.py --artifacts --platform android|ios|harmony`。

## 运行

```bash
./scripts/build-platform.sh android
./scripts/build-platform.sh ios
./scripts/build-platform.sh harmony
```

Android/iOS 脚本已固定 `--playground custom`，不会静默退回标准基座。HarmonyOS 每次修改 ArkTS/UTS 后重新构建、签名和安装，不把 Android/iOS 的热刷新基座语义套用到 HarmonyOS。

## 验收边界

- 云端/本地打包成功：只证明产物生成与原生配置可解析。
- HarmonyOS unsigned HAP：证明 AppKernia 原生包已完成编译和打包；可安装到官方模拟器，真机仍需匹配 `com.appkernia.mobile` 的调试或发布签名。
- 模拟器安装：本机 API 22 官方 Phone 镜像已验证 unsigned HAP 安装、`EntryAbility` 启动、首次隐私页渲染，以及桌面启动器完整显示仓库 AppKernia layered icon 与标签；该结果不替代物理设备签名、Asset Store、通知或硬件能力。
- 真机 Smoke：需分别记录设备、系统/API、安装包 SHA-256、冷启动、AppKernia 桌面图标/启动窗口和安全存储读写清理。
