# AppKernia（AK）后端 API 架构与数据库蓝图

> 状态：可作为后续 Coding Agent 的后端开发事实源（Source of Truth）  
> 基线日期：2026-08-02  
> 项目定位：基于 uni-app x、Go、React、PostgreSQL 的生产级跨平台应用基础框架  
> 参考项目：HotGo V2；只借鉴其后端能力、模块边界和管理功能，不采用其 Vue/Naive UI 前端

---

## 1. 结论先行

AppKernia 后端不直接 Fork HotGo，也不逐文件复制 HotGo，而是采用“功能对标、架构重整、PostgreSQL-first、安全加固”的方式实现。

最终方案：

- 架构：**模块化单体（Modular Monolith）**，预留独立 Worker 和后续服务拆分能力。
- Web/API 框架：**GoFrame 2.10.x**，用于 HTTP、路由、中间件、参数绑定、校验、配置、日志和 OpenAPI 输出。
- 数据访问：**pgx/v5 + sqlc**，禁止新业务模块使用 GORM；禁止同时维护 GoFrame gdb 与 sqlc 两套持久层风格。
- 数据库：**PostgreSQL 18.x**，使用 UUIDv7、JSONB、INET/CIDR、部分索引、递归 CTE 等原生能力。
- 认证：短期 JWT Access Token + 不透明、可轮换、只存哈希的 Refresh Token。
- 授权：规范化 RBAC + 稳定权限码 + 组织数据范围；Casbin 不是核心事实源。
- 后台导航：菜单与权限分离。菜单决定“看见什么”，权限决定“能做什么”。
- 后台任务：**River + PostgreSQL**；支持事务内入队、重试、延迟任务和独立 Worker。
- 缓存与在线状态：Redis 可选但推荐；Redis 不保存核心事实数据。
- 文件：S3/MinIO 抽象、预签名上传、分片上传、病毒扫描与隔离状态。
- 代码生成：CLI/模板驱动，不在生产后台开放任意代码生成和写文件能力。
- 插件：编译期模块注册，不动态加载、安装或卸载任意 Go 代码。
- 国际化：统一 LocaleResolver + 嵌入式 Catalog；首发完整提供 `zh-CN`、`en-US`，默认及最终回退为 `zh-CN`。

前端管理平台继续使用已经确定的 React + Vite + TanStack + Ant Design 技术栈；后端只输出稳定的 OpenAPI 契约、菜单、权限、字典和配置接口。

---

## 2. 对 HotGo 的分析结论

HotGo V2 当前后端以 GoFrame 2 为核心，采用 Admin、Home、Api、WebSocket 多入口，内置 JWT、Casbin、动态菜单、角色数据权限、字典、配置、日志、定时任务、消息队列、附件、通知、代码生成、插件、支付和资金模块。其项目规范要求 `api → controller → service → logic → dao` 分层，并大量依赖 GoFrame CLI 生成 DAO 和模型。

这些能力与 AppKernia 的“开箱即用全栈应用基座”高度重合，因此 AK 应借鉴其模块覆盖面和管理功能，但不照搬以下实现：

HotGo 仓库声明采用 MIT License。AK 可以参考设计并依法复用符合许可的代码，但只要复制了其源码、模板或资源，就必须保留适用的版权和许可声明；本蓝图优先采用独立实现，以降低继承历史结构和安全问题的成本。

| HotGo 设计 | AK 决策 | 原因 |
|---|---|---|
| GoFrame 2 HTTP 与路由 | 采用 | 能直接吸收 HotGo 的路由、校验、配置与多入口经验 |
| `api/controller/service/logic/dao` | 调整为模块内分层 | 避免同一业务散落在全局目录；提高 Agent 定位能力 |
| GoFrame gdb 生成 DAO | 改为 pgx + sqlc | PostgreSQL 类型和 SQL 更明确，编译期生成，复杂查询可审计 |
| JWT + Casbin | 保留概念、重做模型 | JWT 只做身份；权限表使用规范化 RBAC，避免 `v0-v5` 泛化表成为业务事实源 |
| 菜单中同时存按钮权限 | 拆分 | 菜单、页面导航、API 权限和按钮权限职责不同 |
| 管理员表包含余额、积分、提现等 | 拆分 | 身份域不应耦合资金域；支付/钱包为可选模块 |
| `pid + level + tree` 冗余树字段 | 使用 `parent_id + recursive CTE` | 避免移动节点时维护冗余路径并引入脏数据 |
| 服务日志写 PostgreSQL | 不采用 | 应用日志进入 OpenTelemetry/Loki；数据库仅保留业务审计和安全事件 |
| 后台在线代码生成 | 改为 CLI | 生产环境写源码和执行模板风险高，也不利于可重复构建 |
| 运行时插件安装/卸载 | 改为编译期模块 | Go 代码动态卸载和依赖治理复杂，存在供应链风险 |
| 多 MQ 一键切换 | 抽象接口，默认 River | 基础框架先提供可靠默认值，不提前承担 Kafka/RocketMQ 全套运维成本 |
| 远程图片转存 | 默认禁用 | HotGo v2.0 的相关实现已有公开 SSRF 漏洞记录，AK 不复制该实现 |

### 2.1 HotGo 表到 AK 表的主要映射

| HotGo | AppKernia |
|---|---|
| `hg_admin_member` | `iam.users` + `iam.user_credentials` + `iam.tenant_members` |
| `hg_admin_oauth` | `iam.oauth_accounts` |
| `hg_admin_dept` | `org.units` |
| `hg_admin_post` | `org.positions` |
| `hg_admin_role` | `iam.roles` |
| `hg_admin_member_role` | `iam.user_roles` |
| `hg_admin_role_casbin` | `iam.permissions` + `iam.role_permissions`；可选 Casbin 运行适配器 |
| `hg_admin_menu` | `sys.menus` + `iam.permissions` |
| `hg_admin_role_menu` | `sys.role_menus` |
| `hg_admin_notice` / `notice_read` | `notify.messages` + `notify.recipients` |
| `hg_sys_config` | `sys.config_items` |
| `hg_sys_dict_type/data` | `sys.dict_types` + `sys.dict_items` |
| `hg_sys_attachment` | `storage.files` + `upload_sessions` + `upload_parts` |
| `hg_sys_blacklist` | `iam.block_rules` |
| `hg_sys_cron` / `cron_group` | `jobs.schedules` + River Job + `jobs.schedule_runs` |
| `hg_sys_log` | `audit.operation_logs` |
| `hg_sys_login_log` | `audit.login_events` |
| `hg_sys_serve_log` | OpenTelemetry/Loki，不写业务库 |
| `hg_sys_sms_log` / `ems_log` | `iam.verification_challenges` + `notify.deliveries` |
| `hg_sys_addons_install/config` | `sys.modules` + `sys.config_items` |
| 支付、退款、提现、积分 | 可选 `billing` Schema |

---

## 3. 后端技术选型

### 3.1 核心技术栈

| 类别 | 选型 | 约束与用途 |
|---|---|---|
| Go | Go 1.26.5（1.26.x） | 初始锁定当前安全补丁；CI 与生产镜像版本一致 |
| HTTP 框架 | GoFrame v2.10.x，初始锁定 v2.10.2 | 路由、上下文、中间件、绑定、校验、配置、日志、OpenAPI |
| 数据库 | PostgreSQL 18.x，初始 18.4 | 唯一业务数据库；禁止以 MySQL 兼容为设计目标牺牲 PG 能力 |
| 驱动/连接池 | pgx/v5 | PostgreSQL 原生驱动、事务、批量、COPY、通知 |
| SQL 生成 | sqlc 1.32.x | SQL 为事实源，生成类型安全 Go 代码 |
| 数据库迁移 | golang-migrate v4.19.x | 版本化 Up/Down；生产迁移独立执行 |
| 后台任务 | River v0.42.x | PostgreSQL 原生任务队列；依赖版本通过内部接口隔离 |
| Redis | go-redis/v9 | 缓存、限流、一次性状态、在线心跳、Pub/Sub；P0 可暂不启用 |
| JWT | golang-jwt/jwt/v5 | Access Token，使用 Ed25519/EdDSA 签名 |
| 密码 | `golang.org/x/crypto/argon2` | Argon2id；参数由基准测试确定并可版本化 |
| 对象存储 | S3-compatible 接口；MinIO 为开发默认 | 预签名上传/下载，多云通过 Adapter 扩展 |
| WebSocket | 独立 Realtime Adapter | 只推送事件提示；消息最终状态以数据库为准 |
| 可观测性 | OpenTelemetry + Prometheus + Loki/Tempo | Trace、Metrics、Logs；禁止用业务库代替日志系统 |
| 测试 | Go testing、testify、testcontainers-go | 单元、集成、契约、权限矩阵、迁移测试 |
| 静态检查 | gofmt、go vet、staticcheck、golangci-lint、govulncheck | CI 必须全部通过 |
| API 契约 | GoFrame `g.Meta` + CI 导出 OpenAPI 3.1 | 供 React 管理端和 uni-app x 生成客户端类型 |
| 国际化 | BCP 47 LocaleResolver + embed Catalog | `zh-CN`/`en-US`；错误、通知、字典与法律内容本地化 |

### 3.2 明确不采用

- 不使用 GORM。
- 不在新模块中使用 GoFrame gdb；如果后续迁移 HotGo 代码，只允许放在隔离的 `legacyadapter` 中，并逐步替换。
- 不引入微服务框架、服务注册中心、分布式事务框架作为 V1 前置依赖。
- 不把 Kafka、RocketMQ、NATS 作为基础安装必需项。
- 不允许业务模块直接依赖具体 Redis、对象存储或 River 客户端，必须依赖接口。
- 不使用数据库触发器承载业务流程；触发器仅用于 `updated_at` 等机械性动作。

---

## 4. 总体架构

```mermaid
flowchart TB
    Mobile[AK Mobile / uni-app x]
    Admin[AK Admin / React]
    M2M[API Client]

    Mobile --> AppAPI[/api/v1]
    Admin --> AdminAPI[/admin-api/v1]
    M2M --> PublicAPI[/api/v1 or /partner-api/v1]

    AppAPI --> API[ak-api]
    AdminAPI --> API
    PublicAPI --> API

    API --> IAM[IAM / Auth]
    API --> ORG[Organization]
    API --> SYS[System / Menu / Config]
    API --> FILE[Storage]
    API --> NOTIFY[Notification]
    API --> AUDIT[Audit]

    API --> PG[(PostgreSQL 18)]
    API --> REDIS[(Redis Optional)]
    API --> S3[(S3 / MinIO)]
    API --> RIVER[River Queue]
    RIVER --> WORKER[ak-worker]
    WORKER --> PG
    WORKER --> S3
    API --> OTEL[OpenTelemetry]
    WORKER --> OTEL
```

### 4.1 进程与入口

第一阶段使用三个可执行程序，共享同一代码库：

```text
cmd/
├── ak-api/       # HTTP + WebSocket API
├── ak-worker/    # River Worker、通知、文件处理、Outbox 发布
└── ak-cli/       # migrate、seed、bootstrap-admin、gen、doctor
```

HTTP 路由分组：

```text
/api/v1              移动端与普通用户 API
/admin-api/v1        React 管理平台 API
/ws/v1               实时通知连接
/internal/v1         health、readiness、metrics 等内部入口
```

HotGo 的多入口理念保留，但 V1 不部署多个重复 API 服务。未来只有在流量、权限边界或发布周期明确不同后，才拆分独立进程。

---

## 5. 代码目录与分层

```text
server/
├── cmd/
│   ├── ak-api/
│   ├── ak-worker/
│   └── ak-cli/
├── api/
│   ├── app/v1/                 # uni-app x HTTP 契约
│   ├── admin/v1/               # React Admin HTTP 契约
│   ├── websocket/v1/
│   └── internal/v1/
├── internal/
│   ├── bootstrap/              # 组合根、启动和优雅关闭
│   ├── platform/
│   │   ├── database/
│   │   ├── cache/
│   │   ├── authn/
│   │   ├── authz/
│   │   ├── objectstorage/
│   │   ├── jobqueue/
│   │   ├── realtime/
│   │   ├── mail/
│   │   ├── sms/
│   │   └── observability/
│   ├── modules/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── tenant/
│   │   ├── organization/
│   │   ├── accesscontrol/
│   │   ├── navigation/
│   │   ├── dictionary/
│   │   ├── configuration/
│   │   ├── file/
│   │   ├── notification/
│   │   ├── scheduler/
│   │   ├── audit/
│   │   └── region/
│   └── shared/
│       ├── apperror/
│       ├── pagination/
│       ├── transaction/
│       ├── tenantctx/
│       └── validation/
├── db/
│   ├── migrations/
│   ├── queries/                # 按模块分目录的 sqlc SQL
│   └── seeds/
├── gen/
│   └── db/                     # sqlc 生成代码，禁止手改
├── docs/
│   └── openapi/
├── tests/
├── sqlc.yaml
├── go.mod
└── AGENTS.md
```

每个业务模块内部使用：

```text
modules/user/
├── module.go              # 显式注册路由、worker、权限和种子
├── transport/             # HTTP/WebSocket adapter；只做协议转换
├── application/           # 用例、事务边界、权限/数据范围编排
├── domain/                # 领域对象、规则、接口
├── repository/            # sqlc repository 实现
├── queries/               # 或统一位于 db/queries/user
└── tests/
```

调用方向：

```text
transport → application → domain ← repository adapter
```

硬性要求：

- Controller/Handler 不直接调用 sqlc。
- Repository 不读取 HTTP Context，不返回 HTTP 错误。
- 模块之间通过应用服务接口或领域事件交互，禁止跨模块直接查询对方表。
- 使用构造函数显式注入；禁止依赖 `init()` 隐式注册和可变全局 Service Locator。
- 事务由 Application 层开启，并把 `Queries.WithTx(tx)` 传给 Repository。

---

## 6. API 设计规范

### 6.1 响应

成功：

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "request_id": "019..."
}
```

分页：

```json
{
  "code": "OK",
  "data": {
    "items": [],
    "page": 1,
    "page_size": 20,
    "total": 0
  },
  "request_id": "019..."
}
```

错误：

```json
{
  "code": "IAM.USER_NOT_FOUND",
  "message": "用户不存在",
  "details": [
    {"field": "user_id", "reason": "not_found"}
  ],
  "request_id": "019..."
}
```

### 6.2 HTTP 约束

- GET：查询；POST：创建或命令；PATCH：局部更新；PUT：完整替换；DELETE：删除/撤销。
- 正确返回 400、401、403、404、409、422、429、500，不允许业务错误全部包装为 HTTP 200。
- 修改型接口默认记录操作审计。
- 支付、钱包、发送通知、批量任务等接口支持 `Idempotency-Key`。
- 列表默认分页；导出使用异步任务，不在 HTTP 请求中生成超大文件。
- API 版本只放在 URL 大版本；小版本保持向后兼容。
- 所有时间均使用 RFC 3339 UTC，数据库类型为 `timestamptz`。
- 外部 ID 使用 UUID；禁止把数据库自增序列暴露为可枚举业务 ID。

### 6.3 错误码命名

```text
<DOMAIN>.<RESOURCE>.<REASON>
IAM.AUTH.INVALID_CREDENTIALS
IAM.SESSION.REFRESH_REUSED
ACCESS.PERMISSION.DENIED
ORG.UNIT.CYCLE_DETECTED
STORAGE.FILE.TYPE_NOT_ALLOWED
SYSTEM.CONFIG.VERSION_CONFLICT
```

---

## 7. 认证与会话设计

### 7.1 Access Token

- 格式：JWT。
- 签名：Ed25519/EdDSA，私钥只在认证服务；支持 `kid` 与轮换。
- 生命周期：默认 15 分钟。
- 必需 Claims：`iss`、`aud`、`sub`、`sid`、`tid`、`ver`、`iat`、`nbf`、`exp`。
- `aud` 必须区分 `ak-mobile`、`ak-admin`、`ak-api`，不同客户端 Token 不互用。
- Token 不携带完整权限列表；权限变化不依赖 Token 过期后才能生效。

### 7.2 Refresh Token

- 生成至少 256-bit 加密安全随机数。
- 客户端持有明文；服务端只保存 SHA-256 哈希。
- 每次刷新必须轮换；旧 Token 标记 `consumed_at`。
- 如果已消费的旧 Token 再次出现，标记 `reuse_detected_at`，撤销整个 Session，产生高风险安全事件。
- Web 管理端：Refresh Token 优先使用 `HttpOnly + Secure + SameSite` Cookie；Access Token 仅保存在内存。
- 移动端：Refresh Token 存入 iOS Keychain、Android Keystore、HarmonyOS 安全存储封装。

### 7.3 密码

- Argon2id；参数必须存入哈希编码并可升级。
- 初始基准可从 64 MiB 内存、3 次迭代、并行度 2 开始，部署时以目标服务器 100–250 ms 验证时间重新标定。
- 禁止 MD5、普通 SHA、手工 salt 拼接。
- 支持密码历史、强制改密、失败次数、临时锁定和管理员撤销会话。
- Bootstrap 不写入默认 `admin/123456`；通过 CLI 交互创建首个管理员并输出一次性临时密码。

### 7.4 登录防护

- 按 IP、账号标识、设备组合限流。
- 登录失败不暴露“账号存在/不存在”差异。
- Admin 登录在同一标识、Audience 与来源的 HMAC 保护范围内连续失败三次后，必须由服务端强制一次性图形验证码；客户端刷新不能清除失败状态。
- 图形挑战使用短时 PNG、一次登录尝试即消费，错误/过期/超限不可复用；成功登录清零对应失败范围。
- 管理端支持 TOTP 和 WebAuthn MFA。
- 验证码只保存哈希；邮箱/手机号等低熵目标只保存服务端 Keyed HMAC-SHA-256 和脱敏 Hint，防止离线字典反查。
- 邮件、短信等实际投递目标在 `notify.deliveries` 中使用信封加密密文、HMAC Hash、Hint 和 Key Version 分离保存。
- 登录审计不保存完整账号标识，只保存 HMAC Hash 和脱敏 Hint。
- 密码修改、MFA 变更、邮箱/手机号变更必须执行近期认证检查。

---

## 8. 权限、菜单与数据范围

### 8.1 权限模型

权限码格式：

```text
<module>.<resource>.<action>
iam.user.read
iam.user.create
iam.user.update
iam.user.disable
iam.user.role.assign
org.unit.read
org.unit.manage
system.config.read
system.config.update
storage.file.delete
jobs.schedule.execute
```

每个受保护路由在 Go 代码中显式声明权限码。数据库 `iam.permissions` 是权限目录与角色授权事实源；`route_pattern` 只用于文档、诊断和后台展示，不允许运行时仅按可编辑 URL 字符串决定授权。

### 8.2 RBAC

```text
用户 ← tenant_members → 租户
用户 N:N 角色
角色 N:N 权限
角色 N:N 菜单
角色 N:N 自定义组织范围
```

Casbin 可在将来作为 `AuthorizationEngine` 的运行时适配器，但规则必须由上述规范化表构建，不能再维护一份互相漂移的 `casbin_rule(v0...v5)` 主数据。

### 8.3 数据范围

角色 `data_scope`：

- `all`：所有租户数据，仅超级管理员或受控角色可用。
- `tenant`：当前租户全部数据。
- `department`：用户主部门。
- `department_tree`：主部门和全部下级组织。
- `self`：仅本人创建或归属数据。
- `custom`：来自 `iam.role_scope_units`。

授权中间件输出：

```go
type AccessScope struct {
    TenantID          uuid.UUID
    UserID            uuid.UUID
    AllTenantData     bool
    SelfOnly          bool
    UnitIDs           []uuid.UUID
    IncludeDescendant bool
}
```

Repository 查询必须把 `AccessScope` 转换为 SQL 条件；不能先查出全部数据再在 Go 内存中过滤。

多个角色合并规则：

1. 任一角色为 `all`，结果为 `all`。
2. 否则任一为 `tenant`，结果为当前租户全部。
3. 其他范围取并集。
4. 显式拒绝规则如未来引入，应优先于允许规则，并单独设计，不在 V1 混入。

### 8.4 菜单

- `sys.menus` 只描述目录、页面和外链。
- 按钮不作为菜单行；按钮显示取决于 `iam.permissions`。
- `component_key` 必须匹配 React 管理端静态 Route Registry，不允许后端返回任意 JS 文件路径并动态执行。
- 用户登录后返回：基本身份、租户、角色、权限码集合、过滤后的菜单树。
- 前端权限只负责可见性；所有 API 必须在后端再次校验。

---

## 9. PostgreSQL 设计规范

### 9.1 基本规则

- Schema：`iam`、`org`、`sys`、`storage`、`notify`、`jobs`、`audit`；可选 `billing`。
- 表名和列名使用 `snake_case`。
- 主键使用 PostgreSQL 18 原生 `uuidv7()`。
- 时间使用 `timestamptz`。
- 结构化扩展数据使用 `jsonb`；能建模成关系的数据不得图省事塞进 JSON。
- IP 使用 `inet`，网段使用 `cidr`。
- 邮箱、用户名、代码等大小写不敏感字段使用 `citext`。
- 金额使用最小货币单位 `bigint` + ISO 4217 `currency`，不使用浮点数。
- 软删除只用于用户、组织、菜单、文件等主数据；审计日志、支付流水和钱包账本不可软删除覆盖历史。
- 所有多租户业务查询必须显式带 `tenant_id`。
- V1 由应用层强制租户隔离；RLS 在查询覆盖和运维机制成熟后再启用。

### 9.2 树结构

组织和菜单使用邻接表 `parent_id`。组织下级查询使用递归 CTE：

```sql
WITH RECURSIVE descendants AS (
    SELECT id, parent_id
    FROM org.units
    WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL

    UNION ALL

    SELECT u.id, u.parent_id
    FROM org.units u
    JOIN descendants d ON u.parent_id = d.id
    WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
)
SELECT id FROM descendants;
```

移动节点前必须检测：新父节点不能是当前节点或当前节点的后代。

### 9.3 乐观锁

用户、配置、钱包账户等有并发修改风险的表使用 `lock_version/version`：

```sql
UPDATE sys.config_items
SET value_json = $1,
    version = version + 1
WHERE id = $2 AND version = $3;
```

影响行数为 0 时返回 `409 SYSTEM.CONFIG.VERSION_CONFLICT`。

### 9.4 日志与分区

初期审计表为普通表。达到以下任一条件后按月分区：

- 单表超过 5000 万行；
- 日增超过 100 万；
- 审计查询和清理明显影响主业务。

业务库不保存完整服务日志和堆栈；它们进入 Loki/日志平台。审计表中的请求、前后值必须经过字段级脱敏，不记录密码、Token、验证码、Cookie、密钥、完整支付凭据。

---

## 10. Schema 清单

完整可执行草案位于 `db/migrations/*.up.sql`。

### 10.1 IAM

| 表 | 说明 |
|---|---|
| `iam.tenants` | 租户；单租户部署也创建默认系统租户 |
| `iam.users` | 全局身份与个人资料 |
| `iam.user_credentials` | 密码哈希和锁定状态 |
| `iam.password_history` | 密码历史 |
| `iam.oauth_accounts` | 微信、Apple、Google 等外部身份绑定 |
| `iam.tenant_members` | 用户与租户关系 |
| `iam.roles` | 租户角色与数据范围 |
| `iam.permissions` | 全局稳定权限目录 |
| `iam.user_roles` | 用户角色 |
| `iam.role_permissions` | 角色权限 |
| `iam.role_scope_units` | 自定义组织数据范围 |
| `iam.devices` | 已识别设备 |
| `iam.sessions` | 可撤销登录会话 |
| `iam.refresh_tokens` | Refresh Token 轮换链和重放检测 |
| `iam.verification_challenges` | 邮件/SMS OTP、验证和密码重置挑战 |
| `iam.mfa_factors` | TOTP/WebAuthn 因子 |
| `iam.mfa_recovery_codes` | MFA 恢复码哈希 |
| `iam.block_rules` | IP、网段、账号、设备黑名单/挑战/限流规则 |

### 10.2 组织与权限范围

| 表 | 说明 |
|---|---|
| `org.units` | 公司、事业部、部门、团队树 |
| `org.positions` | 岗位 |
| `org.user_units` | 用户组织归属和主部门 |
| `org.user_positions` | 用户岗位 |

### 10.3 系统管理

| 表 | 说明 |
|---|---|
| `sys.modules` | 编译期模块目录与版本 |
| `sys.menus` | 管理端导航元数据 |
| `sys.role_menus` | 角色菜单 |
| `sys.config_items` | 全局/租户配置，支持加密秘密值 |
| `sys.dict_types` | 字典类型 |
| `sys.dict_items` | 字典项 |
| `sys.regions` | 地区编码，可选种子数据 |
| `sys.api_clients` | 机器客户端 |
| `sys.api_client_secrets` | API Client 密钥哈希及轮换 |
| `sys.api_client_permissions` | 机器客户端权限 |
| `sys.idempotency_keys` | 幂等请求结果 |
| `sys.webhook_endpoints` | Webhook 端点 |
| `sys.webhook_deliveries` | Webhook 发送记录和重试 |

### 10.4 文件

| 表 | 说明 |
|---|---|
| `storage.files` | 对象元数据、哈希、扫描和可见性 |
| `storage.upload_sessions` | 预签名/分片上传会话 |
| `storage.upload_parts` | 分片 ETag 和校验 |
| `storage.file_usages` | 文件与业务实体引用关系 |

### 10.5 通知

| 表 | 说明 |
|---|---|
| `notify.templates` | 邮件、SMS、Push、站内信模板 |
| `notify.messages` | 站内通知/公告主体 |
| `notify.recipients` | 收件人、已读和归档状态 |
| `notify.deliveries` | 外部通道发送和重试；目标使用密文 + HMAC Hash + Hint |
| `notify.push_devices` | 加密保存的推送 Token |

### 10.6 任务和事件

| 表 | 说明 |
|---|---|
| `jobs.schedules` | 管理员可维护的定时计划 |
| `jobs.schedule_runs` | 执行历史；关联 River Job |
| `jobs.outbox_events` | 对外发布的事务 Outbox |
| `jobs.inbox_events` | 消费幂等记录 |
| River 自有表 | 由 River 官方迁移管理，不复制到 AK 迁移 |

### 10.7 审计

| 表 | 说明 |
|---|---|
| `audit.operation_logs` | 管理操作和关键业务变更审计 |
| `audit.login_events` | 登录、刷新、MFA、API Client 鉴权事件 |
| `audit.security_events` | Token 重放、权限探测、异常下载等安全事件 |

### 10.8 可选 Billing

| 表 | 说明 |
|---|---|
| `billing.payment_orders` | 通用支付单 |
| `billing.payment_transactions` | 网关交易 |
| `billing.refunds` | 退款 |
| `billing.wallet_accounts` | 余额/积分账户快照 |
| `billing.wallet_ledger_entries` | 不可变账本 |
| `billing.withdrawals` | 提现申请与处理 |

Billing 不属于 AK V1 核心；只有明确存在支付/积分需求时启用 `000006_billing_optional.up.sql`。

---

## 11. 功能清单与优先级

### P0：工程底座与认证闭环

- [ ] GoFrame 服务启动、配置分层、优雅关闭。
- [ ] PostgreSQL 连接池、迁移、sqlc 生成流程。
- [ ] 统一响应、错误码、Request ID、Trace ID。
- [ ] `/health/live`、`/health/ready`、`/metrics`。
- [ ] 密码注册/登录、Access/Refresh Token。
- [ ] Refresh Token 轮换和重放检测。
- [ ] 登出当前会话、登出全部会话。
- [ ] 忘记密码、重置密码、修改密码。
- [ ] 当前用户资料读取和修改。
- [ ] 首个租户、超级管理员、核心权限种子 CLI。
- [ ] OpenAPI 导出和 CI 契约检查。
- [ ] 登录日志、安全事件、基本操作审计。

### P1：后台管理核心

- [ ] 用户管理：查询、创建、编辑、禁用、解锁、重置密码、撤销会话。
- [ ] 租户成员管理和租户切换。
- [ ] 组织机构树、岗位、用户归属。
- [ ] 角色 CRUD、用户分配角色。
- [ ] 权限目录和角色授权。
- [ ] 角色数据范围和查询过滤。
- [ ] 菜单 CRUD、角色菜单、当前用户菜单树。
- [ ] 在线会话列表和强制下线。
- [ ] 操作日志、登录日志、安全事件查询。
- [ ] 批量导入/导出采用异步任务。

### P2：通用平台能力

- [ ] 字典类型和字典项。
- [ ] 动态配置；公开配置和秘密配置隔离。
- [ ] 文件上传、预签名 URL、分片上传、下载授权。
- [ ] 文件类型/大小/哈希校验、恶意文件扫描和隔离。
- [ ] 公告、私信、已读/未读、WebSocket 提示。
- [ ] 邮件/SMS/Push 模板和发送记录。
- [ ] River Worker、任务重试、死信/失败查询。
- [ ] 定时计划 CRUD、暂停、恢复、手动执行、执行历史。
- [ ] Webhook、签名、重试和交付日志。
- [ ] API Client、密钥轮换、CIDR 限制、权限分配。
- [ ] 地区编码查询。
- [ ] 服务指标面板所需聚合 API。

### P3：高级能力与扩展

- [ ] TOTP/WebAuthn MFA。
- [ ] Apple/Google/微信等 OAuth/OIDC 登录。
- [ ] 模块脚手架与 CRUD CLI。
- [ ] 可选 Casbin 适配器和 ABAC 试验，不改变规范化事实表。
- [ ] PostgreSQL RLS 可选加固。
- [ ] 外部 Kafka/NATS 事件总线 Adapter。
- [ ] 支付、退款、钱包、积分和提现模块。
- [ ] 审计表分区、归档与冷存储。
- [ ] 多区域对象存储和 CDN。

---

## 12. 主要 API 面

以下为资源级清单；具体请求/响应由 OpenAPI 定义。

### 12.1 App API `/api/v1`

```text
POST   /auth/register
POST   /auth/login/password
POST   /auth/login/otp
POST   /auth/token/refresh
POST   /auth/logout
POST   /auth/logout-all
POST   /auth/password/forgot
POST   /auth/password/reset
POST   /auth/password/change
POST   /auth/email/send-code
POST   /auth/email/verify
POST   /auth/mobile/send-code
POST   /auth/mobile/verify
GET    /me
PATCH  /me
POST   /me/avatar/upload-session
GET    /me/sessions
DELETE /me/sessions/{session_id}
GET    /me/devices
DELETE /me/devices/{device_id}
GET    /me/notifications
PATCH  /me/notifications/{message_id}/read
POST   /me/notifications/read-all
GET    /public/config
GET    /public/dictionaries/{code}
GET    /regions
```

### 12.2 Admin API `/admin-api/v1`

```text
POST   /auth/login
POST   /auth/token/refresh
POST   /auth/logout
GET    /auth/context                 # 用户、租户、角色、权限、菜单

GET    /users
POST   /users
GET    /users/{id}
PATCH  /users/{id}
POST   /users/{id}/enable
POST   /users/{id}/disable
POST   /users/{id}/unlock
POST   /users/{id}/reset-password
PUT    /users/{id}/roles
GET    /users/{id}/sessions
DELETE /users/{id}/sessions/{sid}
POST   /users/import
POST   /users/export

GET    /tenants
POST   /tenants
PATCH  /tenants/{id}
GET    /tenants/{id}/members
POST   /tenants/{id}/members

GET    /org/units/tree
POST   /org/units
PATCH  /org/units/{id}
POST   /org/units/{id}/move
DELETE /org/units/{id}
GET    /org/positions
POST   /org/positions
PATCH  /org/positions/{id}
PUT    /org/users/{user_id}/assignments

GET    /roles
POST   /roles
PATCH  /roles/{id}
DELETE /roles/{id}
PUT    /roles/{id}/permissions
PUT    /roles/{id}/menus
PUT    /roles/{id}/data-scope
GET    /permissions
GET    /menus/tree
POST   /menus
PATCH  /menus/{id}
POST   /menus/{id}/move
DELETE /menus/{id}

GET    /dict-types
POST   /dict-types
PATCH  /dict-types/{id}
GET    /dict-types/{id}/items
POST   /dict-types/{id}/items
PATCH  /dict-items/{id}

GET    /configs
POST   /configs
PATCH  /configs/{id}
POST   /configs/{id}/rotate-secret
GET    /modules

GET    /files
POST   /files/upload-sessions
POST   /files/upload-sessions/{id}/complete
DELETE /files/{id}
POST   /files/{id}/presign-download

GET    /notices
POST   /notices
PATCH  /notices/{id}
POST   /notices/{id}/publish
POST   /notices/{id}/cancel
GET    /notification-deliveries

GET    /job-schedules
POST   /job-schedules
PATCH  /job-schedules/{id}
POST   /job-schedules/{id}/pause
POST   /job-schedules/{id}/resume
POST   /job-schedules/{id}/execute
GET    /job-schedules/{id}/runs

GET    /online-sessions
DELETE /online-sessions/{id}
GET    /audit/operations
GET    /audit/logins
GET    /audit/security-events
POST   /audit/security-events/{id}/resolve

GET    /api-clients
POST   /api-clients
GET    /api-clients/{id}
PATCH  /api-clients/{id}
POST   /api-clients/{id}/secrets
DELETE /api-clients/{id}/secrets/{secret_id}
PUT    /api-clients/{id}/permissions

GET    /webhooks
POST   /webhooks
PATCH  /webhooks/{id}
POST   /webhooks/{id}/test
GET    /webhooks/{id}/deliveries
```

---

## 13. 后台任务与消息设计

### 13.1 River

River 是 AK 默认的内部任务队列。任务入队应尽量在业务事务内完成：

```go
err := txManager.WithinTransaction(ctx, func(tx pgx.Tx) error {
    if err := repo.CreateUser(ctx, tx, user); err != nil {
        return err
    }
    _, err := riverClient.InsertTx(ctx, tx, WelcomeEmailArgs{UserID: user.ID}, nil)
    return err
})
```

这样事务回滚时任务也不可见，不需要依赖“先写数据库还是先发队列”的脆弱顺序。

### 13.2 定时任务

- 后台只能选择已在代码中注册的 `handler_key`。
- 不允许数据库中保存 shell 命令、Go 源码、SQL 脚本后直接执行。
- Cron 表达式必须校验，时区必须为 IANA 名称。
- 支持重叠策略、错过执行策略、超时、最大尝试次数。
- 手动执行也创建独立 run 记录和审计日志。

### 13.3 Outbox

River 用于 AK 内部异步任务；`jobs.outbox_events` 用于需要发布到外部 Webhook、Kafka/NATS 或其他系统的领域事件。

Outbox 发布器使用 `FOR UPDATE SKIP LOCKED` 批量领取，成功后写 `published_at`，失败递增次数并设置下一次可用时间。

---

## 14. 文件与 SSRF 安全

### 14.1 默认上传流程

1. 客户端请求上传会话。
2. 后端校验权限、扩展名、MIME、大小和租户配额。
3. 后端生成随机对象 Key 和预签名 URL。
4. 客户端直传 S3/MinIO。
5. 客户端提交完成请求。
6. Worker 校验实际大小、SHA-256、MIME 和恶意文件。
7. 仅 `status=ready AND scan_status IN ('clean','skipped')` 的文件可被业务引用。

### 14.2 远程 URL 转存

V1 默认不提供“输入任意 URL，服务端下载并转存”接口。确需开放时必须同时实现：

- 仅允许 `https`；禁止 `file`、`ftp`、`gopher` 等协议。
- 域名 Allowlist，或至少严格 Blocklist 私网、环回、链路本地、云元数据地址。
- DNS 解析后检查全部 IP；连接后再次验证；重定向每一跳重新验证。
- 限制响应体大小、连接/读取超时、内容类型和重定向次数。
- 独立受限 egress 网络或代理。
- 不转发用户 Cookie、Authorization、内部 Header。
- 完整记录安全审计但不记录秘密值。

---

## 15. 模块、种子与代码生成

### 15.1 模块注册

每个模块实现显式接口：

```go
type Module interface {
    Code() string
    RegisterRoutes(*ghttp.Server)
    RegisterWorkers(*river.Workers)
    Permissions() []PermissionDefinition
    Menus() []MenuDefinition
    Seeds() []Seeder
}
```

启动时由 Composition Root 显式构造模块列表。模块启停只控制路由、Worker 和菜单是否注册；不在运行时加载未知二进制。

### 15.2 种子

`ak-cli seed core` 必须幂等，按稳定 code upsert：

- 默认系统租户。
- `super_admin`、`tenant_admin`、`member` 角色。
- 核心权限目录。
- 管理菜单。
- 系统字典。
- 基础配置定义。
- 版本化地区编码目录。

`ak-cli bootstrap-admin`：

- 创建首个管理员。
- 强制设置一次性高强度临时密码或交互输入。
- `force_password_change=true`。
- 不把密码写进 SQL、日志或 Git。

### 15.3 代码生成

```text
ak gen module <name>
ak gen crud --module <name> --table <schema.table>
ak gen permission --module <name>
ak openapi export
sqlc generate
```

生成过程必须：

- 只写工作区内允许目录。
- 先输出 dry-run diff。
- 不覆盖存在的手写文件，除非文件有明确 generated 标记。
- 生成后自动执行 gofmt、sqlc generate、测试和 OpenAPI 检查。
- 生产环境不暴露生成接口。

---

## 16. Agent 实施阶段

### Phase 0：仓库与质量门禁

交付：目录、Go module、Makefile/Taskfile、Docker Compose、CI、AGENTS.md、错误模型、配置加载、健康检查。

验收：

- `go test ./...` 通过。
- `go vet ./...`、`staticcheck ./...`、`govulncheck ./...` 通过。
- PostgreSQL/Redis/MinIO 开发环境可一条命令启动。
- 服务在 SIGTERM 下停止接收请求、等待处理中请求并关闭连接池。

### Phase 1：数据库与 IAM

交付：000001–000003 迁移、sqlc 配置、用户/密码/租户/角色基础 Repository。

验收：

- 空库 Up 成功；回滚后可再次 Up。
- UUIDv7、唯一约束、外键、软删除唯一索引符合预期。
- 并发创建相同邮箱只允许一个成功。

### Phase 2：认证闭环

交付：注册、密码登录、刷新、轮换、重放检测、登出、密码重置、会话管理。

验收：

- 并发使用同一 Refresh Token 只能一个成功。
- 旧 Refresh Token 重用会撤销 Session 并写安全事件。
- Admin/Mobile audience 不能互换。
- 密码、验证码、Token 不出现在日志。

### Phase 3：RBAC、组织、菜单

交付：权限中间件、角色授权、组织树、数据范围、菜单树。

验收：

- 自动化权限矩阵覆盖无角色、单角色、多角色、禁用角色、过期角色。
- 不同租户 UUID 即使已知也不能越权访问。
- 组织节点移动不能形成环。
- 数据范围过滤发生在 SQL 层。

### Phase 4：系统管理

交付：字典、配置、文件、通知、任务、日志、在线会话。

验收：

- 秘密配置加密存储，API 永不返回明文列表。
- 文件绕过扩展名、MIME 欺骗、超限和隔离状态测试通过。
- 定时任务只能执行注册 Handler。
- 审计字段脱敏。

### Phase 5：Agent 开发工具

交付：模块脚手架、CRUD CLI、OpenAPI 客户端生成、种子同步。

验收：

- 新模块生成后无需手改即可编译。
- 重复执行生成器结果幂等。
- 生成代码与手写代码边界清晰。

### Phase 6：可选高级模块

MFA、OAuth、Webhook、API Client、Billing、外部 MQ、RLS、分区归档。

---

## 17. 测试与 CI 清单

### 17.1 测试层级

- Domain 单元测试：规则和状态机。
- Application 用例测试：Mock Port 或真实 PostgreSQL。
- Repository 集成测试：testcontainers PostgreSQL 18。
- HTTP 契约测试：状态码、错误码、校验和 OpenAPI。
- 安全测试：IDOR、租户越权、权限绕过、Refresh 重放、SSRF、上传欺骗。
- 迁移测试：全量 Up、Down、再次 Up；从上一发布版本升级。
- 并发测试：用户唯一性、钱包、Refresh、任务领取。
- E2E：React Admin/uni-app x 只依赖发布后的 OpenAPI 客户端。

### 17.2 CI 必须检查

```text
gofmt -w / gofmt diff
go vet ./...
staticcheck ./...
golangci-lint run
govulncheck ./...
go test -race ./...
sqlc generate && git diff --exit-code
migration smoke test
OpenAPI export && breaking-change check
container image scan
secret scan
```

---

## 18. 安全红线

- 不复制 HotGo 的远程图片转存实现。
- 不写默认弱管理员密码。
- 不把 Refresh Token、密码、验证码、API Secret 明文入库。
- 不信任前端传来的 tenant、role、permission 或 user_id。
- 不允许菜单可见性替代 API 权限校验。
- 不允许对象存储 Key 直接使用用户原始文件名。
- 不允许客户端指定任意本地路径或 Bucket。
- 不允许后台定时任务执行任意命令。
- 不允许生产后台写入项目源码。
- 不允许日志记录 Authorization、Cookie、密码、验证码、私钥、支付凭据。
- 不允许业务 SQL 忘记 tenant 条件；代码评审和测试必须覆盖。
- 不允许删除或修改钱包历史账本；只能写冲正/补偿记录。

---

## 19. 国际化架构

国际化统一契约位于 `blueprint/I18N_CONTRACT.md`，后端详细实现见 `docs/I18N_ARCHITECTURE.md`。P0 即必须完成，不能推迟到前端收尾阶段。

必须交付：

- `server/internal/shared/i18n`：LocaleResolver、Catalog、Formatter、Middleware。
- `server/internal/shared/i18n/locales/zh-CN.json` 与 `en-US.json`。
- `Accept-Language`、用户偏好和默认语言解析。
- `Content-Language`、必要时的 `Vary: Accept-Language`。
- OpenAPI 中的 `SupportedLocale` 枚举、`message_key` 与用户 locale 字段。
- Auth Context / Public Config 返回 `default_locale`、`supported_locales` 和当前 `locale`。
- 用户修改语言偏好接口；移动端和 Admin 共用同一规范值。
- 字典、通知模板、法律文档与版本提示的 locale 回退链。
- 两种语言的契约、回退、模板参数和占位符测试。

数据库 Core Schema 已包含 `iam.users.locale`、`sys.dict_items.locale` 和 `notify.templates.locale`。内置菜单以稳定 `i18n_key` 输出，`title` 仅作 `zh-CN` 回退，不需要为静态 UI 文案建立通用 EAV 翻译表。

## 20. Definition of Done

任一模块只有同时满足以下条件才能标记完成：

1. 数据迁移、sqlc Query、Repository、Application、Transport 分层完整。
2. OpenAPI 契约已更新并通过兼容性检查。
3. 权限码、菜单、字典、配置等种子幂等。
4. 所有写操作定义事务边界、幂等策略和审计策略。
5. 多租户和数据范围测试通过。
6. 单元、集成、契约和关键安全测试通过。
7. 日志无敏感数据；错误响应不泄露内部堆栈和 SQL。
8. `go test -race ./...`、静态检查和漏洞扫描通过。
9. 文档包含 API、错误码、权限码、数据库变更和回滚方式。
10. Agent 输出变更摘要、测试证据、未解决风险，不能只报告“已完成”。

---

## 21. 参考来源

- [HotGo repository](https://github.com/bufanyun/hotgo)
- [HotGo AGENTS.md](https://github.com/bufanyun/hotgo/blob/v2.0/AGENTS.md)
- [HotGo MIT License](https://github.com/bufanyun/hotgo/blob/v2.0/LICENSE)
- [HotGo SQL baseline](https://raw.githubusercontent.com/bufanyun/hotgo/v2.0/server/storage/data/hotgo.sql)
- [NVD CVE-2026-3683](https://nvd.nist.gov/vuln/detail/CVE-2026-3683)
- [Go release history](https://go.dev/doc/devel/release)
- [GoFrame releases](https://github.com/gogf/gf/releases)
- [PostgreSQL 18 documentation](https://www.postgresql.org/docs/current/)
- [sqlc with pgx/v5](https://docs.sqlc.dev/en/latest/guides/using-go-and-pgx.html)
- [River](https://github.com/riverqueue/river)
