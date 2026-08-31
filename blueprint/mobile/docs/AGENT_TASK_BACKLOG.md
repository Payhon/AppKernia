# Agent 任务积压表

按依赖图连续执行，不得把计划输出当实现。机器契约：`spec/agent-task-backlog.json`。

## AKMOB-000 — 仓库、HBuilderX 工具链与质量底座

- Phase：`P0A`
- 依赖：—
- UI Skill：否
- 产物：apps/ak-mobile；环境探测脚本；CI 分层任务
- 验收：蓝图校验通过；三个平台构建命令/手册可发现；不得把普通 Vite 构建冒充 App 构建

## AKMOB-010 — ui-ux-pro-max 移动端 Master Design System

- Phase：`P0B`
- 依赖：AKMOB-000
- UI Skill：必须
- 产物：design-system/MASTER.md；移动页面 override；UI evidence
- 验收：UI skill 真实产物存在；关键状态和安全区规范完成；浅色手机基线截图


## AKMOB-015 — AkI18n 基础设施与中英文语言包

- Phase：`P0B2`
- 依赖：AKMOB-010
- UI Skill：必须
- 产物：AkI18n；locale registry；`zh-CN/en-US`；语言切换器；key/placeholder 校验脚本
- 验收：默认/回退 `zh-CN`；运行时切换无需重启；同步导航栏和 TabBar；两种语言 key/占位符一致；三端英文长文本 smoke

## AKMOB-020 — 固定 uView Ultra 并建设 AK UI 适配层

- Phase：`P0C`
- 依赖：AKMOB-015
- UI Skill：必须
- 产物：uni_modules/uview-ultra；components/ak-ui；组件画廊
- 验收：版本锁定 4.5.18；保留 Licence；业务示例只使用 ak-*；三端组件矩阵启动

## AKMOB-030 — 应用核心、配置、日志、错误和 Feature Flag

- Phase：`P0D`
- 依赖：AKMOB-000
- UI Skill：否
- 产物：src/core；src/config；src/observability
- 验收：无 any/无静默错误；配置有 schema；敏感字段日志脱敏

## AKMOB-040 — OpenAPI 到 UTS DTO/Endpoint 代码生成

- Phase：`P0E`
- 依赖：AKMOB-030
- UI Skill：否
- 产物：src/generated/api；scripts/generate-mobile-client
- 验收：生成代码不可手改；漂移检查通过；所有 page-api-map endpoint 可解析

## AKMOB-050 — 网络层、Token Vault 与 Refresh Single-Flight

- Phase：`P0F`
- 依赖：AKMOB-040
- UI Skill：否
- 产物：AkHttpClient；ak-secure-storage UTS plugin；auth session store
- 验收：Access Token 仅内存；Refresh Token 仅安全存储；并发 401 只刷新一次；sslVerify 不得关闭

## AKMOB-060 — 启动编排、隐私同意、路由守卫与深链白名单

- Phase：`P0G`
- 依赖：AKMOB-015, AKMOB-030, AKMOB-050
- UI Skill：必须
- 产物：bootstrap page；privacy consent；navigation guard
- 验收：未同意前不调用敏感 API；guest/auth/challenge 路由正确；未知深链拒绝

## AKMOB-070 — 登录、注册、验证码、忘记与重置密码

- Phase：`P0H`
- 依赖：AKMOB-020, AKMOB-050, AKMOB-060
- UI Skill：必须
- 产物：auth pages；auth tests
- 验收：密码和验证码不入日志；错误防账号枚举；倒计时以服务端时间/冷却为准；Android/iOS/Harmony smoke

## AKMOB-080 — App Shell、三 Tab、首页与权限组件

- Phase：`P0I`
- 依赖：AKMOB-020, AKMOB-060
- UI Skill：必须
- 产物：tabBar；home；AkCan/useCan；`ak-scanner` UTS；扫码协调器与受控 WebView
- 验收：Tab 路由静态；权限仅改善 UX；离线/错误/空状态完整；二维码/条形码、事件顺序、处理器优先级、single-flight、相机权限和域名越界三端验证

## AKMOB-090 — 个人中心、基本资料与头像上传

- Phase：`P1A`
- 依赖：AKMOB-080, AKMOB-040
- UI Skill：必须
- 产物：profile pages；avatar flow
- 验收：资料乐观锁；邮箱/手机号不可直接标记验证；头像引用 ready 文件

## AKMOB-100 — 安全中心、密码、会话与设备

- Phase：`P1B`
- 依赖：AKMOB-090, AKMOB-050
- UI Skill：必须
- 产物：security pages；session/device flows
- 验收：不能撤销他人资源；当前会话操作有明确反馈；强制清理本地 Token

## AKMOB-110 — TOTP MFA、恢复码与 Step-up

- Phase：`P1C`
- 依赖：AKMOB-100
- UI Skill：必须
- 产物：MFA pages；step-up flow
- 验收：Secret/恢复码只显示一次；截图/日志不包含真实 Secret；关闭 MFA 需 Step-up

## AKMOB-120 — OAuth PKCE 与第三方绑定

- Phase：`P2A`
- 依赖：AKMOB-100
- UI Skill：必须
- 产物：OAuth adapter；connections page
- 验收：state/PKCE 校验；回调深链白名单；最后一种登录方式不可误解绑

## AKMOB-130 — 站内消息、未读数与已读状态

- Phase：`P1D`
- 依赖：AKMOB-080, AKMOB-040
- UI Skill：必须
- 产物：notification pages；badge store
- 验收：只读取自己的消息；WebSocket/Push 只作提示；服务端已读为事实源

## AKMOB-140 — Push 注册与通知偏好

- Phase：`P2B`
- 依赖：AKMOB-130, AKMOB-030
- UI Skill：必须
- 产物：push adapter；notification settings
- 验收：权限按需申请；拒绝后可继续使用 App；Token 加密传输和服务端存储

## AKMOB-150 — 设置、语言偏好同步、法律文档、关于与版本检查

- Phase：`P1E`
- 依赖：AKMOB-080, AKMOB-040
- UI Skill：必须
- 产物：language preference sync；settings/legal/about pages；`ak-upgrade` 原生升级模块
- 验收：复用 AKMOB-015；用户偏好跨设备同步；法律文档按 locale/version/hash 可审计；`uni_app_x` 禁止 WGT，Android 可下载签名 APK，三平台仅打开受控商店/HTTPS 链接

## AKMOB-160 — 暗色主题兼容审计

- Phase：`P2C`
- 依赖：AKMOB-020, AKMOB-150
- UI Skill：必须
- 产物：theme tokens；dark screenshots；compatibility report
- 验收：uView 组件逐项审计；不通过则 Flag 保持关闭；三平台深色回归

## AKMOB-170 — 多租户切换和缓存隔离

- Phase：`P2D`
- 依赖：AKMOB-080, AKMOB-050
- UI Skill：必须
- 产物：tenant switch page；tenant-scoped caches
- 验收：切换后轮换 Token；清理旧租户缓存；跨租户 E2E 拒绝

## AKMOB-180 — 账号注销闭环

- Phase：`P2E`
- 依赖：AKMOB-110, AKMOB-150
- UI Skill：必须
- 产物：account deletion page；verified email code flow；immediate scoped erasure
- 验收：当前已验证邮箱验证码必须；仅删除当前 App 且即时生效、不可撤销；删除后仅本地注销 Push 并清理安全 Session、认证上下文和敏感缓存

## AKMOB-190 — 离线、重试、取消、恢复和性能预算

- Phase：`P2F`
- 依赖：AKMOB-130, AKMOB-150
- UI Skill：必须
- 产物：offline state；request cancellation；performance evidence
- 验收：读请求有限重试；写请求无隐式重放；长列表不全量渲染；首屏/内存基线记录

## AKMOB-200 — Android 构建、自动化与真机验收

- Phase：`P3A`
- 依赖：AKMOB-190
- UI Skill：否
- 产物：Android logs；screenshots；test report
- 验收：Debug/Release 编译；Android 8+ 真机 smoke；16KB 兼容检查；关键 uni-automator 通过

## AKMOB-210 — iOS 构建、自动化与真机验收

- Phase：`P3B`
- 依赖：AKMOB-190
- UI Skill：否
- 产物：iOS logs；screenshots；test report
- 验收：Simulator/Archive 编译；iOS 13+ 真机 smoke；Keychain 验证；隐私描述匹配

## AKMOB-220 — HarmonyOS 构建、真机与内存验收

- Phase：`P3C`
- 依赖：AKMOB-190
- UI Skill：否
- 产物：Harmony logs；screenshots；memory report
- 验收：API14+ 编译和真机 smoke；安全存储验证；ArkTS 内存检查；返回键/安全区通过

## AKMOB-230 — 发布硬化、文档、示例模块和交付报告

- Phase：`P3D`
- 依赖：AKMOB-200, AKMOB-210, AKMOB-220
- UI Skill：否
- 产物：release checklist；sample feature；CODEX_DELIVERY_REPORT.md
- 验收：三端矩阵全绿；无真实 Secret；许可证清单完整；全部真实命令和退出码记录

## AKMOB-240 — Vapor 可行性 Spike（不阻塞 Core 1.0）

- Phase：`P3E`
- 依赖：AKMOB-230
- UI Skill：必须
- 产物：ADR；benchmark；wrapper compatibility report
- 验收：默认仍为 VDOM；只有完整三端通过才可提议切换；不修改已发布基线迁移式升级
