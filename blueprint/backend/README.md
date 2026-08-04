# AppKernia Backend Blueprint Package

这是 AppKernia（AK）后端 API、PostgreSQL Schema 和后台管理能力的 Agent 开发资料包。

设计参考 HotGo V2 的后端能力覆盖，但不采用其 Vue/Naive UI 前端；AK Admin 继续使用已确定的 React + Vite + TanStack + Ant Design。后端采用独立实现和 PostgreSQL-first 重构，不直接复制 HotGo 的 MySQL 数据模型。

## 核心结论

```text
Architecture       Modular Monolith
HTTP               GoFrame 2.10.x
Go                 1.26.5 / 1.26.x
Database           PostgreSQL 18.x
Data access        pgx/v5 + sqlc
Migration          golang-migrate
Jobs               River + PostgreSQL
Cache              Redis optional/recommended
Auth               JWT Access + rotating opaque Refresh Token
Authorization      Normalized RBAC + data scope
Storage            S3-compatible / MinIO
Observability      OpenTelemetry + Prometheus + Loki/Tempo
```

## 文件说明

| 文件/目录 | 用途 |
|---|---|
| `AK_BACKEND_BLUEPRINT.md` | 总体技术选型、架构、API、认证、权限、功能范围和 Agent 实施阶段 |
| `AGENTS.md` | Coding Agent 强制开发规则、质量门禁和交付格式 |
| `docs/DATABASE_SCHEMA.md` | 57 张表的数据字典、ER 关系、关键不变量和迁移验收 |
| `docs/FEATURE_MATRIX.md` | P0～P3 功能、API、权限码、后台页面和验收标准 |
| `db/migrations/` | 6 组 PostgreSQL Up/Down 迁移；`000006` Billing 可选 |
| `AppKernia_PostgreSQL_Core_Schema.sql` | `000001`～`000005` 合并版，51 张核心表 |
| `AppKernia_PostgreSQL_Full_Schema_With_Billing.sql` | 含可选 Billing，57 张表 |
| `spec/core-permissions.json` | 97 个核心权限种子定义 |
| `spec/core-menus.json` | 26 个 AK Admin 默认菜单定义 |
| `tools/validate_blueprint.py` | 无第三方依赖的静态一致性检查 |
| `docs/STATIC_VALIDATION_REPORT.txt` | 本资料包当前静态检查结果 |

## 建议交给 Agent 的入口提示

```text
请先完整阅读：
1. AK_BACKEND_BLUEPRINT.md
2. AGENTS.md
3. docs/FEATURE_MATRIX.md
4. docs/DATABASE_SCHEMA.md

严格按 Phase 0 开始开发 AppKernia 后端，不得改用 HotGo 的 Vue 管理端，
不得引入 GORM，不得直接复制 HotGo MySQL 表结构。
每个 Phase 完成后必须给出迁移、OpenAPI、权限种子、测试命令和真实结果。
```

## 推荐实施顺序

```text
Phase 0  仓库、配置、质量门禁、Docker Compose、健康检查
Phase 1  PostgreSQL 迁移、sqlc、Tenant/User/Role Repository
Phase 2  注册、密码登录、Token 轮换、重放检测、会话管理
Phase 3  RBAC、组织、数据范围、菜单和 auth/context
Phase 4  用户后台、字典、配置、文件、通知、任务、审计
Phase 5  OpenAPI 客户端生成、Seed、Module/CRUD CLI
Phase 6  MFA、OAuth、Webhook、API Client、Billing 等可选能力
```

## Schema 使用

真实项目优先使用版本化迁移：

```bash
migrate -path db/migrations -database "$DATABASE_URL" up 5
```

`000006_billing_optional` 不属于 AK Core 默认范围。只有明确需要支付、钱包、积分或提现时才启用。

合并 SQL 主要用于审阅、快速建库和导入数据库工具，不替代生产迁移流程。

## 静态验证

```bash
python3 tools/validate_blueprint.py
```

当前静态验证覆盖：

- Up/Down 文件配对。
- BEGIN/COMMIT 和括号结构。
- 表、索引、触发器重复命名。
- 外键引用表存在性。
- 每个迁移 Down 是否覆盖本迁移创建的表。
- `updated_at` 触发器覆盖。
- 明文凭证型列名检查。
- 权限和菜单 JSON 唯一性、父节点完整性。

静态检查不能替代 PostgreSQL 18 集成测试。后续 Agent 必须在真实 PostgreSQL 18 容器中执行 `Up → Down → Up`，并验证外键、唯一约束和并发行为。

## HotGo 参考边界

借鉴：

- 多入口 API 思路。
- 用户、组织、角色、菜单、字典、配置、日志、任务、附件、通知等功能覆盖。
- GoFrame 路由、参数绑定、校验、配置和 OpenAPI 能力。
- 代码生成提升开发效率的理念。

重构：

- 全部数据模型改为 PostgreSQL-first。
- 身份、租户成员、凭证、会话和资金解耦。
- 菜单与后端权限分离。
- Casbin 不作为主数据事实源。
- GoFrame gdb 改为 pgx/v5 + sqlc。
- 在线代码生成改为本地/CI CLI。
- 动态二进制插件改为编译期模块注册。
- 服务运行日志移出业务库。
- 任意远程 URL 转存默认关闭并按 SSRF 威胁模型设计。

HotGo 声明为 MIT License。复制其实际源码、模板或资源时，应保留对应版权和许可声明；本资料包本身优先描述独立实现方案。
