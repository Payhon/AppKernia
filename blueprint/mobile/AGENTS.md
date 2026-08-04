# AGENTS.md — AppKernia Mobile

本文件对 `apps/ak-mobile` 及移动端相关后端契约的 AI Coding Agent 生效。

## 1. 先读后写

按顺序读取：

1. `blueprint/mobile/AK_MOBILE_BLUEPRINT.md`
2. `blueprint/mobile/docs/ARCHITECTURE.md`
3. `blueprint/mobile/docs/UVIEW_ULTRA_INTEGRATION.md`
4. `blueprint/mobile/docs/INFORMATION_ARCHITECTURE.md`
5. `blueprint/mobile/docs/PAGE_SPECIFICATIONS.md`
6. `blueprint/mobile/docs/API_STATE_SECURITY.md`
7. `blueprint/mobile/docs/PLATFORM_COMPATIBILITY.md`
8. `blueprint/mobile/docs/PRIVACY_PERMISSION_MATRIX.md`
9. `blueprint/mobile/docs/UI_UX_PRO_MAX_WORKFLOW.md`
10. `blueprint/mobile/docs/TESTING_QUALITY_GATES.md`
11. `blueprint/mobile/spec/*.json`
12. `blueprint/mobile/integration/*.json`
13. `blueprint/backend` 中的 App API、OpenAPI、权限和数据库规范

不得猜测 API 字段、路由、权限码或平台能力。

## 2. 固定技术边界

必须：

```text
uni-app x
UTS + UVue
Vue 3 Composition API
VDOM / standard mode（AK Core 1.0 默认）
uView Ultra 4.5.18（仓库内 pin）
HBuilderX stable baseline + Android/iOS/Harmony 工具链
PostgreSQL/Go 后端 OpenAPI 生成的 UTS 契约
```

禁止：

- 把项目改成传统 uni-app、React Native、Flutter 或 WebView 壳。
- 未经 ADR 将默认渲染器改成 Vapor。
- 业务页面直接使用 `up-*`；必须经过 `components/ak-ui` 的 `ak-*` 适配组件。
- 把 Access Token、Refresh Token、密码、OTP、MFA Secret 放入普通 `uni.setStorage`。
- 关闭 TLS 证书校验，或在生产环境接受 HTTP API。
- 用前端权限替代 Go API 授权。
- 从服务端下载并执行页面代码、脚本或动态路由。
- 使用 `any`、无边界 `UTSJSONObject`、静默吞错或无类型 API 响应逃避 UTS 校验。
- 把普通 Node/Vite 构建结果冒充 Android、iOS 或鸿蒙构建成功。

## 3. uView Ultra 规则

- 以 `uni_modules/uview-ultra` 方式纳入仓库；不使用尚不适合 uni-app x 的 npm 集成路径。
- 保留并随代码分发 uView Ultra 的 Licence 文件。
- 修改第三方文件必须在文件中标注，并在 `docs/THIRD_PARTY_PATCHES.md` 记录原因、上游版本和删除条件。
- `uni.scss` 只导入 `theme.scss`；`index.scss` 在 `App.uvue` 的 style 首行导入。
- easycom 使用 `^up-(.*)` 规则，但业务 feature 不直接引用该前缀。
- 组件只有在 `component-compatibility-matrix.json` 标为 approved 或完成 conditional 验收后才能进入 Core 页面。
- uView 不内建暗色/宽屏保证；相关能力必须由 AK Token、Feature Flag 和三端回归提供。

## 4. ui-ux-pro-max 硬门禁

任何创建、重构或明显改变可视 UI 的任务，必须先：

1. 读取 `ui-ux-pro-max` Skill。
2. 读取 `apps/ak-mobile/design-system/MASTER.md` 与页面 override。
3. 输出任务 request、skill output、decisions、review checklist 和三端/多尺寸截图证据。
4. 再写 UVue、SCSS 和交互代码。

Skill 不可用时可以继续 API、类型和非 UI 基础设施，但 UI Task 不得标记完成，也不得伪造产物。

## 5. 架构和状态

- 固定依赖方向：`page/feature → application service → port → platform adapter`。
- `core` 不依赖 feature；feature 不直接调用平台原生 API。
- UTS 原生能力放入显式 `uni_modules/ak-*` 插件。
- Server State 不复制到多个 Store；列表以 Query/Repository Cache 为事实源。
- Session、Feature Flag、Locale、Theme 等少量全局状态使用类型化 Reactive Store。
- 默认不在 Android VDOM 上引入第三方 Pinia；切换需要 ADR 和三端测试。
- 所有请求支持取消、超时、结构化错误和 Request ID。
- 401 Refresh 使用 single-flight，原请求最多自动重试一次；403 不刷新。
- 写操作没有幂等键时不得隐式重试。

## 6. 路由、权限与隐私

- 所有页面静态登记在 `mobile-route-registry.json` 和 `pages.json`。
- 深链和 OAuth Callback 只能进入 allowlist。
- `guest`、`authenticated`、`challenge` 路由必须有自动化测试。
- 页面权限只控制展示与导航；后端再次强制授权。
- App 首次启动先完成隐私同意；同意前不得调用设备标识、Push、定位、相册等非必要能力。
- 权限按功能即时申请；拒绝后提供降级路径，不得循环骚扰用户。

## 7. 三端完成定义

任何 Core 页面完成必须至少有：

- UTS 类型检查无逃逸。
- Android 编译/真机 smoke。
- iOS 编译/模拟器自动化与真机 smoke。
- HarmonyOS 编译/真机 smoke。
- Loading、Empty、Error、Offline、Forbidden 状态。
- UI Skill 证据与关键截图。
- 路由/权限/API 契约测试。
- 日志脱敏检查。

环境无法执行某个平台时，只能标记 blocked，不能写 passed。

## 8. 提交前真实执行

```bash
python3 blueprint/mobile/scripts/validate_blueprint_specs.py
# 以及仓库中实际存在的 mobile lint/typecheck/test/build commands
```

报告必须列出真实命令、退出码、变更文件、未完成项和平台阻塞，禁止虚构测试结果。

## 9. 国际化硬约束

- 先读取 `blueprint/I18N_CONTRACT.md` 与 `docs/I18N_ARCHITECTURE.md`。
- P0 必须实现 `AkI18n`、`zh-CN`、`en-US`；默认/最终回退 `zh-CN`。历史 `zh-Hans/en` 仅作为输入别名，不得作为持久化规范值或资源文件名。
- 所有 UVue 用户可见文案、uView Ultra 包装组件默认文案、导航栏、TabBar、校验、Toast、空状态和错误都必须使用 key。
- 语言切换无需重启/reLaunch；必须同步当前页面标题、全部 TabBar、AK UI/uView 包装组件、日期数字和后续网络请求 Header。
- `pages.json` 可保留 `zh-CN` 静态回退，但运行时必须由 `AkI18n` 覆盖；业务页面不能依赖静态中文标题。
- 登录用户语言选择同步后端；匿名选择只存普通非敏感偏好。Token 和 locale 存储必须分离。
- 两套语言包 key/占位符必须一致，Android/iOS/Harmony 关键页面均在两种语言下测试英文扩展和截断。
