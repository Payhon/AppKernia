# AGENTS.md — AppKernia Repository

本文件是 AppKernia（AK）仓库中所有 AI Coding Agent 的根级强制规则。

## 1. 项目与规范源

AppKernia 是跨平台应用开发基座：

- `apps/ak-mobile`：uni-app x，Android/iOS/HarmonyOS。
- `apps/ak-admin`：React + Vite 管理平台。
- `server`：Go API/Worker/CLI，PostgreSQL。

蓝图位于：

```text
blueprint/backend
blueprint/admin-frontend
blueprint/mobile
```

开始任何工作前先读取本文件，再读取任务所属蓝图的 `AGENTS.md`、主蓝图、相关 docs/spec/integration。跨项目任务必须读取全部三个蓝图。不得删除、重命名或把 blueprint 当生成代码覆盖。

冲突优先级：

```text
用户最新明确指令
> 根 AGENTS.md
> 已批准 ADR
> 子项目 AGENTS.md
> 主蓝图
> 机器可读 spec/integration
> 说明文档
> 现有代码习惯
```

同一蓝图内部，机器可读契约用于一致性校验；若与文字文档冲突，先记录并修正文档/契约，不得暗中任选。

## 2. 固定技术边界

### Backend

GoFrame + pgx/v5 + sqlc + PostgreSQL 18，模块化单体。不得引入 GORM 或复制 HotGo MySQL Schema。

### Admin

React、TypeScript strict、Vite、TanStack Router/Query、Zustand、Ant Design、RHF + Zod。不得改成 Vue/Next.js。

### Mobile

uni-app x、UTS/UVue、VDOM、uView Ultra 固定版本、AK UI 适配层。不得改成传统 uni-app、Flutter、React Native；未经 ADR 不切 Vapor。

### Internationalization

Backend、Admin、Mobile 必须遵守 `blueprint/I18N_CONTRACT.md` 与 `blueprint/i18n-contract.json`。默认和最终回退为 `zh-CN`，首发必须完整实现 `zh-CN`、`en-US` 两套语言包；不得以仅预留接口、只翻译菜单或机器翻译占位作为完成。

## 3. 执行方式

- 先审计 Git 状态、已有代码、工具链和用户未提交修改。
- 直接实现真实代码，不把计划、伪代码或目录空壳当交付。
- 按各蓝图 task backlog 的依赖图连续推进，不在每个 Task 后请求确认。
- 普通技术细节自行采用稳妥生产方案；重大偏离写入 `docs/adr/`。
- 缺少第三方凭据时实现 Port/Adapter、本地 Mock、Feature Flag 和文档，继续其他任务。
- 不执行 `git reset --hard`、`git clean -fd`、强制覆盖用户修改或自动 push。
- 不伪造命令、测试、构建、截图、Skill 产物或真机结果。

## 4. 契约同步

OpenAPI 是前后端 API 的最终事实源。任何接口改动必须同步：

```text
Go Route/Application/Repository
OpenAPI
数据库 Migration/sqlc（如涉及）
权限 Seed
审计与安全事件
Admin/Mobile 生成 Client
Contract/Integration/E2E Test
```

- Admin 只能调用 `/admin-api/v1`。
- Mobile 只能调用 `/api/v1`。
- Token Audience 必须隔离。
- 菜单可见性、页面按钮和客户端权限不能替代后端授权。
- 多租户过滤必须在服务端/SQL 层执行。

## 5. UI 强制流程

任何 Admin 或 Mobile 可视 UI 创建、重构或明显修改：

1. 先读取并运行 `ui-ux-pro-max` Skill。
2. 读取/更新对应 `design-system/MASTER.md` 与页面 override。
3. 保存 request、skill output、decisions、review checklist 和截图。
4. 再写组件和页面。

Skill 不可用时可做非 UI 工作，但 UI 不得标记完成。不得伪造产物。

Mobile 业务页面只能使用 `ak-*` UI；不得直接深度绑定 `up-*`。

## 6. 安全红线

- 不提交真实 Secret、证书私钥、Token、密码、OTP、MFA Secret 或第三方凭据。
- 不在日志、错误上报、截图和 Fixture 中泄漏敏感数据。
- 不关闭 TLS 校验。
- 不信任客户端传入的 user、tenant、role 或 permission。
- 不提供生产 Shell、任意源码写入、在线二进制插件或动态执行代码能力。
- Refresh Token 只存 Hash（服务端）或系统安全存储（移动端）；Admin 使用 Secure HttpOnly Cookie。
- 写请求无幂等保证时不得自动重放。

## 7. 完成与报告

每个 Task 只有满足蓝图 acceptance 并执行真实命令后才能标完成。持续更新：

```text
docs/IMPLEMENTATION_STATUS.md
docs/CODEX_DELIVERY_REPORT.md
```

最终报告必须列出：变更、实际命令及退出码、测试数量、构建平台/设备、截图索引、未完成项、blocked 原因和风险。未执行即为未验证，不能写 passed。

至少运行：

```bash
python3 blueprint/mobile/scripts/validate_blueprint_specs.py
python3 blueprint/scripts/validate_i18n_contract.py
# 以及 backend/admin 蓝图指定的校验和仓库真实 check/build/test 命令
```

## 8. 多语言强制规则

- 所有用户可见文案必须使用语义翻译键；禁止在 Go 响应、React JSX、UVue 模板和业务组件中硬编码中文或英文。
- 规范语言代码只允许 `zh-CN`、`en-US`；平台别名先按统一契约规范化。
- API 请求发送 `Accept-Language`，后端解析并返回 `Content-Language`；业务判断只使用稳定错误码。
- Admin 使用 i18next；Mobile 使用 `AkI18n`；Backend 使用统一 `LocaleResolver` 和嵌入式 Catalog。
- Admin 必须同步 Ant Design、Day.js、HTML lang 和页面标题；Mobile 必须同步导航栏、TabBar、uView Ultra 包装组件和系统格式化。
- 登录用户语言偏好持久化到服务端；匿名语言选择仅保存为非敏感本地偏好。
- `zh-CN` 与 `en-US` key 及占位符必须完全一致。每次 check 都运行 `python3 blueprint/scripts/validate_i18n_contract.py`。
- 关键 E2E、视觉回归、错误态、表单校验和长文本布局必须同时覆盖两种语言。

## 9. 枚举与字典决策

- 新增枚举前必须明确记录它属于“可扩展业务选项”还是“稳定协议状态”，不得仅因前端需要下拉框就创建字典。
- 可扩展业务选项必须提供版本化字典种子、`zh-CN`/`en-US` 标签、后端解析与 active 校验、OpenAPI/生成 Client 和自动化测试；页面不得重新硬编码同一组选项。
- 状态机状态、安全与认证类型、协议值、权限代码、语言代码、任务及持久化状态继续使用 Go/TypeScript 代码枚举、OpenAPI enum 和数据库约束。
- 驱动类字典必须与编译期能力注册表绑定；字典只配置已注册能力及其受控元数据，禁止把字典变成动态代码、SDK、脚本、二进制插件或任意执行入口。
- `fixed` 类型禁止在线扩展，`open` 类型允许业务值扩展，`registered` 类型只允许代码已注册值，`s3_compatible` 类型只允许受控的标准 S3-compatible 存储配置；新增策略需先更新契约和安全审查。
