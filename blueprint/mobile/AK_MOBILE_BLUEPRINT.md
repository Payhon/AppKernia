# AppKernia Mobile（AK Mobile）开发蓝图

**版本：** 1.1  
**验证日期：** 2026-08-02  
**目标目录：** `apps/ak-mobile`  
**平台：** Android / iOS / HarmonyOS NEXT

## 1. 产品定位

AK Mobile 不是某个业务 App，而是可复用的移动端生产基座。首个业务项目应当能在不重写认证、网络、安全、个人中心、权限、通知、设置和三端适配的情况下直接建立 feature。

Core 1.0 交付以下闭环：

- 随包法律快照、隐私同意与启动编排。
- 密码、OTP、注册、找回/重置密码、Refresh Rotation。
- 当前用户、资料编辑、头像、会话、设备、安全记录。
- TOTP MFA、恢复码、OAuth Provider 抽象。
- 静态路由、Feature Flag、页面/动作权限。
- 站内消息、未读计数、通知偏好与可选 Push。
- 多厂商 Push 通过 `ak-push` Port 隔离业务页面：iOS 使用 APNs，Android Google/China 为互斥变体，HarmonyOS NEXT 使用 Push Kit；法定同意和用户主动开启前不得初始化、申请权限或上传 Token。
- 语言、主题基础设施、法律文档、版本检查。
- 多租户切换、账号注销。
- Android、iOS、HarmonyOS 构建与真机验收矩阵。

## 2. 不在 Core 1.0 的范围

- 业务领域页面，如订单、商城、IoT 设备或内容社区。
- 动态下载/执行 UI、JS、UTS 或原生插件。
- 在线热更新可执行业务包。
- 支付、钱包、Billing。
- 小程序、H5、Pad 桌面布局作为正式发布目标。
- 默认 Vapor 发布；Vapor 仅有独立 Spike。

## 3. 技术基线

| 领域 | 选型 |
|---|---|
| 跨端框架 | uni-app x |
| 页面/语言 | UVue + UTS + Composition API |
| 默认渲染 | VDOM / standard mode |
| UI 库 | uView Ultra 4.5.18，仓库内 `uni_modules` 固定版本 |
| AK UI | `components/ak-ui` 适配层，业务只用 `ak-*` |
| 全局状态 | 类型化 Reactive Store |
| Server State | Repository/Query Cache，按资源与租户隔离 |
| 网络 | AK 自建 `AkHttpClient` 包装 `uni.request`/`RequestTask` |
| API 契约 | Backend OpenAPI 3.1 → UTS DTO/Endpoint 生成 |
| 普通存储 | `uni.*Storage`，只放非敏感偏好 |
| 安全存储 | `uni_modules/ak-secure-storage`：Android Keystore、iOS Keychain、Harmony Keystore |
| 短信验证码门禁 | `uni_modules/ak-interactive-captcha`：原生 UTS/UVue `click | slide | drag | rotate`，网络由 Feature Repository/Runtime 管理 |
| 国际化 | `AkI18n` 适配层，首发 `zh-CN`/`en-US`，默认及最终回退 `zh-CN` |
| 任务测试 | 纯函数测试 + uni-automator + 三端真机 Smoke |
| UI 流程 | ui-ux-pro-max → Design Token → AK UI → 页面 |

## 4. 为什么默认 VDOM

uView Ultra 当前作为 uni-app x 标准模式组件提供，并没有在其发布信息中承诺 Vapor、暗色或宽屏完整支持。AK 的基础框架目标优先级是三端确定性，而不是追逐尚未完成生态验证的渲染路径。因此 Core 1.0 固定 VDOM；未来只有在 uView、AK Wrapper、状态、i18n、UTS 插件和全部核心页面完成三端回归后，才可通过 ADR 提议 Vapor。

## 5. 关键设计原则

### 5.1 UI 库是实现细节

```text
Feature Page
    ↓
AK UI Components (ak-*)
    ↓
uView Ultra (up-*) / uni native component / platform adapter
```

业务页面不能依赖 uView 的 props 细节。升级、替换或修补 uView 时，只改 AK UI 与兼容矩阵。

### 5.2 API 是契约，不是手写猜测

```text
server/openapi/app-v1.yaml
    ↓ generate
apps/ak-mobile/src/generated/api/
    ├── models.uts
    ├── endpoints.uts
    ├── errors.uts
    └── schema-hash.json
```

生成层不直接绑定具体 HTTP 实现；`AkHttpClient` 执行 Endpoint 描述。不得手改生成文件。

### 5.3 Token 分层

```text
Access Token   → 仅内存
Refresh Token  → 系统安全存储
偏好/缓存       → uni storage（不敏感）
密码/OTP/MFA    → 永不持久化
```

### 5.4 权限分层

- Route Guard：能否进入页面。
- `AkCan`：能否显示/触发动作。
- Go API：最终授权事实源。
- SQL：租户与数据范围事实源。

### 5.5 多语言是启动级基础设施

- 统一规范语言代码为 `zh-CN`、`en-US`；设备返回的 `zh-Hans`、`zh_CN`、`en` 等先规范化。
- App 启动在首个用户可见页面前完成 locale 解析与 Catalog 加载。
- 语言切换不重启 App，并同步 `uni.setNavigationBarTitle`、`uni.setTabBarItem`、AK UI/uView 包装组件和请求 `Accept-Language`。
- 所有文案使用语义 key；`pages.json` 中文只作为无法执行运行时代码时的静态回退。
- `zh-CN`/`en-US` key、占位符、页面覆盖必须由脚本校验。

### 5.6 平台能力通过 Port 隔离

至少定义：

- `SecureStoragePort`
- `DeviceInfoPort`
- `NetworkStatusPort`
- `PushPort`
- `OAuthPort`
- `BiometricPort`
- `FilePickerPort`
- `AppVersionPort`
- `TelemetryPort`

Feature 不得直接出现 Android/iOS/Harmony 原生类型。

## 6. 目标代码结构

```text
apps/ak-mobile/
├── App.uvue
├── main.uts
├── manifest.json
├── pages.json
├── uni.scss
├── locale/
│   ├── zh-CN.json
│   └── en-US.json
├── design-system/
│   ├── MASTER.md
│   └── pages/
├── artifacts/ui-ux-pro-max/
├── components/
│   └── ak-ui/
├── pages/
├── src/
│   ├── core/
│   │   ├── bootstrap/
│   │   ├── config/
│   │   ├── errors/
│   │   ├── feature-flags/
│   │   ├── i18n/
│   │   ├── logging/
│   │   ├── navigation/
│   │   ├── network/
│   │   ├── permissions/
│   │   ├── query/
│   │   ├── security/
│   │   └── stores/
│   ├── features/
│   │   ├── auth/
│   │   ├── home/
│   │   ├── notifications/
│   │   ├── profile/
│   │   ├── settings/
│   │   └── tenant/
│   ├── generated/api/
│   └── shared/
├── uni_modules/
│   ├── uview-ultra/
│   ├── ak-secure-storage/
│   ├── ak-push/
│   └── ak-oauth/
├── tests/
│   ├── unit/
│   ├── contract/
│   ├── automator/
│   └── fixtures/
└── scripts/
```

## 7. 页面信息架构

默认 Tab：

```text
首页
消息
我的
```

隐藏/堆栈页面包括：隐私同意、登录、注册、OTP、找回/重置密码、MFA Challenge、个人资料、安全中心、会话、设备、MFA、第三方绑定、设置、租户切换、法律文档、账号注销和错误页。完整定义见 `spec/mobile-route-registry.json`。

## 8. API 与后端依赖

- `integration/app-api-baseline.json`：Backend Blueprint 已定义的 API。
- `integration/app-api-delta.json`：为移动端完整页面建议新增的 API。
- `integration/app-permissions.delta.json`：建议增加的自助权限。

后端改动必须同步：Go Route、OpenAPI、Application Use Case、sqlc、权限 Seed、审计、测试和移动端生成 Client。

## 9. UI 质量要求

每个页面至少覆盖：

- 正常、Loading、Empty、Error、Offline、Forbidden。
- 键盘弹出、返回键、安全区、长文本、动态字号。
- 360×800、390×844、430×932 视觉基线。
- Android、iOS、HarmonyOS 真机核心交互。
- 浅色基线；暗色只有完成 AKMOB-160 后开启。

## 10. 安全红线

- 不在普通存储保存 Token、密码或验证码。
- 不关闭 `sslVerify`。
- 不记录 Authorization、Cookie、Refresh Token、OTP、MFA Secret、恢复码。
- 不允许任意 URL 深链或 OAuth 回调。
- 不把客户端 `tenant_id` 当可信来源。
- 不自动重放非幂等写请求。
- 不在隐私同意前初始化 Push、统计、OAuth、定位、相册或广告 SDK；首次法律文本使用随包快照。
- 不允许外部配置下发可执行脚本、页面或二进制。

## 11. Definition of Done

Core 1.0 只有在以下条件均满足时完成：

1. 三平台 Debug/Release 构建证据存在。
2. Android/iOS/Harmony 真机 Smoke 通过。
3. 注册、登录、刷新、退出、资料、会话、设备和消息使用真实后端。
4. Access/Refresh Token 存储策略经测试验证。
5. Route/API/Permission/Feature Flag 契约校验通过。
6. uView 第三方许可和补丁记录完整。
7. ui-ux-pro-max 产物及关键截图完整。
8. 敏感日志扫描通过。
9. 生产构建不包含 Mock Server、Dev Gallery 或真实 Secret。
10. 未执行的平台不能被标记为通过。
11. `zh-CN`/`en-US` 语言包完整，key/占位符校验通过，三端关键流程均完成双语验收。

实施顺序以 `spec/agent-task-backlog.json` 为准。
