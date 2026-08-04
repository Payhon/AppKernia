# AppKernia Backend — Agent Development Rules

本文件是 AppKernia（AK）后端 Coding Agent 的强制执行规范。任何 Agent 在编写、修改、审查后端代码前必须先阅读：

1. `AK_BACKEND_BLUEPRINT.md`
2. `AGENTS.md`
3. `docs/FEATURE_MATRIX.md`
4. `docs/DATABASE_SCHEMA.md`
5. 当前任务涉及的迁移、OpenAPI、权限种子和模块文档

发生冲突时，优先级为：

```text
用户最新明确指令
> 已批准的 ADR / 变更说明
> AGENTS.md
> AK_BACKEND_BLUEPRINT.md
> FEATURE_MATRIX / DATABASE_SCHEMA
> 现有代码习惯
```

---

## 1. 项目边界

AppKernia 是跨平台应用开发基座：

- 移动端：uni-app x。
- 后台管理端：React + Vite + TanStack + Ant Design。
- 后端：GoFrame + pgx/v5 + sqlc。
- 数据库：PostgreSQL 18。

本后端任务不得：

- 把 React 管理端改成 HotGo 的 Vue/Naive UI。
- 直接 Fork HotGo 后改包名冒充 AK 实现。
- 把 HotGo 的 MySQL 表结构原样迁入 AK。
- 引入 GORM，或在新业务模块中使用 GoFrame gdb。
- 将生产代码生成器、Shell 执行器或任意源码写入能力暴露成管理 API。
- 动态加载、安装、卸载未知 Go 二进制插件。

HotGo 仅作为功能覆盖和工程经验参考。复制任何 HotGo 源码、模板或资源时，必须识别并保留适用的 MIT 版权及许可声明；默认优先独立实现。

---

## 2. 固定技术基线

除非任务明确包含 ADR，不得擅自替换：

```text
Go                    1.26.5 / 1.26.x
GoFrame               2.10.x
PostgreSQL            18.x
pgx                   v5
sqlc                  1.32.x
golang-migrate        4.19.x
River                 0.42.x
Redis client          go-redis/v9
JWT                   golang-jwt/jwt/v5
Password hashing      Argon2id
Object storage        S3-compatible / MinIO
Observability         OpenTelemetry + Prometheus
```

依赖应通过内部 Port/Adapter 隔离；业务模块不得直接依赖 River、Redis、S3 SDK 或具体短信/邮件厂商。

---

## 3. 目标代码结构

```text
server/
├── cmd/
│   ├── ak-api/
│   ├── ak-worker/
│   └── ak-cli/
├── api/
│   ├── app/v1/
│   ├── admin/v1/
│   ├── websocket/v1/
│   └── internal/v1/
├── internal/
│   ├── bootstrap/
│   ├── platform/
│   ├── modules/
│   └── shared/
├── db/
│   ├── migrations/
│   ├── queries/
│   ├── generated/
│   └── seed/
├── openapi/
├── tests/
└── tools/
```

每个模块内采用：

```text
internal/modules/<module>/
├── domain/          # 实体、值对象、状态机、领域错误；不依赖 HTTP/DB SDK
├── application/     # 用例、Port、事务边界、授权需求
├── infrastructure/  # sqlc Repository、外部 Adapter
├── transport/       # GoFrame HTTP/WebSocket 适配
├── seed/            # 权限、菜单、字典、配置定义
└── module.go        # 显式注册
```

允许简化小模块，但不允许把 HTTP、SQL、业务规则全部堆在 Controller/Handler 中。

---

## 4. 调用方向

固定依赖方向：

```text
Transport → Application → Domain
Infrastructure → Application/Domain Port
Bootstrap → 所有模块与平台 Adapter
```

禁止：

- Domain 导入 GoFrame、pgx、sqlc、Redis、River、S3 SDK。
- Transport 直接调用 sqlc Query。
- Repository 决定业务状态流转。
- 一个模块直接查询另一个模块的私有表；必须经对方 Application Port，或使用已批准的只读投影。
- 循环依赖和隐式 `init()` 注册。所有模块在 Composition Root 显式构造。

---

## 5. API 规则

路由前缀：

```text
/api/v1          uni-app x / 普通用户
/admin-api/v1    React 管理后台
/ws/v1           实时事件提示
/internal/v1     health、ready、metrics
```

必须遵守：

- REST 资源使用复数名词，路径小写 kebab-case。
- 写操作使用正确的 `POST/PUT/PATCH/DELETE`，不得全部伪装为 POST。
- HTTP 状态码与业务错误码同时正确。
- 所有 API 都有 OpenAPI 契约、示例、错误响应和权限说明。
- 列表统一 cursor 或 page/size；同一资源不得混用两种格式。
- 时间统一 RFC 3339 UTC；金额使用最小货币单位整数。
- UUID 在 JSON 中使用标准字符串。
- 不直接暴露数据库内部错误、SQL、堆栈或供应商错误体。

标准成功响应：

```json
{
  "data": {},
  "meta": {},
  "request_id": "..."
}
```

标准错误响应：

```json
{
  "error": {
    "code": "IAM.AUTH.INVALID_CREDENTIALS",
    "message": "账号或密码错误",
    "details": {}
  },
  "request_id": "..."
}
```

错误码必须稳定，不得用中文文案作为程序判断依据。

---

## 6. 数据库规则

### 6.1 基本要求

- 只支持 PostgreSQL 18，不为 MySQL 兼容牺牲 PostgreSQL 能力。
- SQL 是数据访问事实源，使用 sqlc 生成类型安全代码。
- 所有 Schema、表、列、索引、约束使用 `snake_case`。
- 主键默认 `uuid DEFAULT uuidv7()`；关系表可使用复合主键。
- 时间使用 `timestamptz`；结构化扩展数据使用 `jsonb`。
- IP/CIDR 使用 `inet`/`cidr`；邮箱、用户名等使用 `citext`。
- 不使用数据库枚举类型；有限状态用 `varchar + CHECK`，便于迁移。
- 可变业务表通常带 `created_at`、`updated_at`；需要软删除时带 `deleted_at`。
- 业务编号与主键分离。

### 6.2 多租户

- 请求中的 `tenant_id` 只能来自已验证 Session/Tenant Context，不信任客户端 Body 或 Query。
- 所有租户业务 SQL 必须显式包含 `tenant_id` 条件。
- 列表、详情、更新、删除都必须验证租户边界。
- 关系表优先使用复合外键，防止跨租户关联。
- 超级管理员跨租户访问必须走独立用例并写审计日志，不得通过漏写过滤条件实现。

### 6.3 迁移

- 每个迁移必须同时提交 `.up.sql` 和 `.down.sql`。
- 迁移不可依赖应用启动时自动猜测或自动改表。
- 已发布迁移不可修改；修复必须新增迁移。
- 大表变更需考虑锁时间、回填、分批、索引并发创建和回滚。
- River 自有表由 River 官方迁移管理，不复制到 AK 业务迁移。

迁移验收：

```text
空库 up → down → up
上一发布版本 → 当前版本
约束与索引行为集成测试
并发唯一性测试
```

### 6.4 SQL 查询

- 禁止 `SELECT *`。
- 分页必须有稳定排序和唯一 tie-breaker。
- 数据权限过滤必须在 SQL 层完成，禁止全量读出后在 Go 中过滤。
- 动态排序字段使用白名单，不拼接未验证输入。
- 批量写入使用 pgx Batch/COPY 或集合 SQL，不做 N+1 循环。
- 所有查询必须评估索引；高频查询提交 `EXPLAIN (ANALYZE, BUFFERS)` 证据。

---

## 7. 认证与会话

固定模型：

- JWT Access Token：短期、Ed25519/EdDSA、含 `sub`、`sid`、`aud`、`tenant_id`、版本和过期时间。
- Refresh Token：高熵不透明随机值；数据库只存 SHA-256 哈希。
- Refresh Token 每次使用都轮换。
- 同一旧 Token 并发使用只能一个成功。
- 发现已消费 Token 重用时，撤销整个 Session Family，并写 `audit.security_events`。
- 移动端 Refresh Token 存系统安全存储；管理 Web 使用 Secure + HttpOnly + SameSite Cookie。
- Admin、Mobile、Machine API 的 audience 不得互换。
- 密码使用 Argon2id PHC 字符串，参数版本化并支持登录时升级。
- 密码、Token、验证码、API Secret 永不进入日志和错误响应。

所有认证状态变化必须在事务内完成，并覆盖并发测试。

---

## 8. 授权、菜单与数据范围

- `iam.permissions.code` 是授权事实，例如 `iam.user.read`。
- 后端中间件校验权限码；菜单可见性不能代替 API 授权。
- `sys.menus` 仅描述 React 管理端导航；`component_key` 映射到前端静态路由注册表，不允许服务端下发任意脚本或组件 URL。
- 角色、权限、菜单、组织数据范围分别建模。
- Casbin 只能作为可选运行时 Adapter；不能用 `p_type/v0...v5` 表替代 AK 规范化事实表。
- 数据范围：`all`、`tenant`、`department`、`department_tree`、`self`、`custom`。
- 多角色有效范围按模块定义合并策略；默认权限取并集，显式禁止规则另行 ADR。
- 组织树和角色树移动前必须检测环。

新增管理接口时必须同时提交：

1. Permission Definition。
2. 路由与权限绑定。
3. 角色种子策略。
4. 菜单或 UI Action 定义（需要时）。
5. 权限矩阵测试。

---

## 9. 事务、并发与幂等

- Application 层声明事务边界。
- Repository 不得自行开启不可见事务。
- 外部副作用不得在数据库提交前不可逆执行。
- 内部后台任务通过 River 在同一 PostgreSQL 事务内入队。
- 外部事件使用 Transactional Outbox。
- 支付、创建订单、导入、Webhook 等可重试写接口支持 `Idempotency-Key`。
- 乐观锁使用 `lock_version`；余额等热点写入必须用条件更新或行锁并有并发测试。
- 不能用分布式锁掩盖缺失的唯一约束或事务设计。

---

## 10. 后台任务与定时计划

- Worker Key 必须编译期注册。
- 数据库只保存 `handler_key`、Cron、Payload 和执行策略。
- 禁止存储并执行任意 Shell、SQL 或 Go 代码。
- Worker 必须幂等，定义唯一业务键或去重策略。
- 明确超时、重试、退避、取消和永久失败条件。
- 任务日志只记录必要摘要；大结果写对象存储。
- 管理 API 可暂停、恢复、手动触发和查询历史，但不能注入未知 Handler。

---

## 11. 文件与外部 URL

默认采用预签名上传：

1. 后端创建上传会话。
2. 客户端直传 S3/MinIO。
3. 后端完成确认。
4. Worker 校验大小、哈希、MIME 和恶意文件。
5. 只有 `ready + clean/skipped` 文件可引用。

硬性规则：

- Object Key 由服务端随机生成，不使用用户原始文件名。
- 校验扩展名、声明 MIME、魔数、实际大小和 SHA-256。
- 私有文件下载必须签发短期 URL 并校验租户、用户和业务引用权限。
- V1 不提供任意远程 URL 抓取。
- 确需抓取时必须实现 HTTPS、域名/IP 策略、DNS Rebinding 防护、重定向逐跳验证、私网/元数据地址阻断、响应大小/超时限制和隔离 egress。

---

## 12. 配置、秘密与日志

- 普通配置存 `value_json`；秘密只存加密密文和 Key Version。
- API 列表永不返回秘密明文；创建/轮换时只允许一次性返回必要值。
- 密钥来自 KMS、Vault 或环境注入，不写数据库/Git。
- 应用运行日志进入 OpenTelemetry/Loki；PostgreSQL 只保存业务审计、登录事件和安全事件。
- 审计 `before_data/after_data/request_summary` 必须字段级脱敏。
- 登录标识只保存哈希和展示提示，不保存完整手机号/邮箱。

---

## 13. 测试要求

每项功能至少覆盖：

- Domain 单元测试。
- Application 用例测试。
- PostgreSQL Repository 集成测试。
- HTTP 契约测试。
- 权限与租户越权测试。
- 关键并发/幂等测试。
- 迁移 Up/Down/Up 测试。

安全关键功能额外覆盖：

- IDOR/BOLA。
- 权限绕过。
- Refresh Token 重放。
- 登录枚举和暴力尝试。
- SQL 注入与动态排序。
- SSRF。
- 上传 MIME/魔数欺骗。
- Webhook 签名与重放。
- 敏感字段日志泄漏。

禁止通过删除测试、弱化断言、跳过 race detector 或 hard-code 测试结果来“修复”失败。

---

## 14. 质量门禁

提交前至少运行：

```bash
gofmt -w .
go vet ./...
staticcheck ./...
golangci-lint run
govulncheck ./...
go test -race ./...
sqlc generate
git diff --exit-code -- db/generated openapi
```

涉及数据库时还要运行：

```bash
migrate up
migrate down 1
migrate up
repository integration tests
```

涉及 API 时还要运行 OpenAPI 导出和 Breaking Change 检查。

---

## 15. Agent 单任务工作流

1. 阅读事实源和目标模块。
2. 写出任务边界、受影响表/API/权限和风险。
3. 先提交或更新迁移与契约，再实现代码。
4. 实现最小闭环，不顺手重写无关模块。
5. 运行测试和质量门禁。
6. 检查多租户、数据权限、日志脱敏和幂等。
7. 更新模块文档、错误码、权限种子和变更记录。
8. 输出可验证结果。

不得在缺少关键事实时凭空发明并隐式固化。可以采用最小安全默认值，但必须在交付摘要中列为假设。

---

## 16. Agent 交付格式

每次交付必须包含：

```text
任务目标
已完成内容
修改文件
数据库变更及回滚
新增/变更 API
新增/变更权限码
安全与多租户影响
测试命令与真实结果
尚未解决的风险/假设
下一阶段可直接执行的入口
```

不接受只有“已完成”“测试通过”的无证据报告。不得伪造命令输出、覆盖率、迁移结果或漏洞扫描结果。

---

## 17. Definition of Done

模块只有在以下条件全部成立时才完成：

- 迁移、SQL、sqlc 生成、Repository、Application、Transport 完整。
- OpenAPI、错误码、权限码和种子同步。
- 所有写操作定义事务、幂等与审计策略。
- 多租户和数据范围过滤在 SQL 层验证。
- 单元、集成、契约、并发和安全关键测试通过。
- 日志无敏感值，错误无内部实现泄漏。
- Up/Down/Up 可执行，回滚方式明确。
- 代码生成结果可重复，工作区无未提交生成差异。
- 文档和交付证据完整。

## 13. 国际化与语言协商

- 开始实现前读取 `blueprint/I18N_CONTRACT.md`、`blueprint/i18n-contract.json` 和 `blueprint/backend/docs/I18N_ARCHITECTURE.md`。
- 必须内置完整的 `zh-CN`、`en-US` Backend Catalog；默认和最终回退均为 `zh-CN`。
- 使用 BCP 47 匹配器规范化语言；请求上下文只能保存规范代码。
- 所有公开 API 接受 `Accept-Language` 并返回 `Content-Language`；本地化缓存响应设置 `Vary: Accept-Language`。
- 错误响应必须包含稳定 `code`，推荐包含 `message_key`；`message` 是本次 locale 的展示回退，不能作为程序判断依据。
- 审计、Outbox、Job 和安全事件保存 code/template key 与结构化参数，不能只保存中文成品句子。
- 通知、字典和法律内容必须按 locale 选择并显式回退；缺失翻译要记录可观测指标。
- API 时间、数字、金额保持语言无关的原始值，禁止后端按 locale 拼成业务字符串。
- 每次后端 check 必须包含两种语言 key/占位符一致性、Accept-Language、用户偏好、回退与通知模板测试。
