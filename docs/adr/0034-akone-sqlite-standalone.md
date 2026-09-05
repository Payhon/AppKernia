# ADR-0034：akone 默认使用 SQLite 本地运行模式

状态：已实施；替代 ADR-0033 的默认数据库结论。

## 决策

`akone serve` 在没有显式数据库配置时使用 SQLite。数据库文件默认位于运行中二进制同目录的 `data/appkernia.db`；`serve --sqlite FILE`、`AK_SQLITE_PATH` 或 YAML `database.sqlite_path` 可以覆盖。Unix/macOS 新建专用目录和数据库文件分别使用 `0700`、`0600` 权限，已有 group/world-writable 目录会被拒绝；Windows 依赖用户目录 ACL。启动监听前必须完成启用外键、WAL 与 `synchronous=FULL` 的 SQLite 独立迁移，失败即退出。

已有 PostgreSQL 部署保持原行为：显式 `database.driver: postgresql` 或仅提供 `database.url` / `AK_DATABASE_URL` 时继续使用 PostgreSQL 18、pgx、sqlc 和 River Worker。

SQLite 使用独立最终态 schema 和 Repository，不在运行时改写 PostgreSQL SQL。首期本地模式覆盖管理端静态资源、健康检查、管理员初始化、登录/刷新/退出、身份上下文、个人资料/密码/会话及 Dashboard；队列、通知、推送、API Client、App、内容等完整业务模块仍使用 PostgreSQL。SQLite 模式不会启动空操作 Worker，也不会接收后丢弃任务。

## 原因

当前 PostgreSQL 数据层包含 schema、JSONB、数组、行锁、River 队列及大量 pgx/sqlc 查询。将连接字符串替换为 SQLite 或做 SQL 文本转换会产生无法保证的事务、租户隔离和后台任务语义。独立组合根能让单文件本地启动真实可用，同时不改变现有 PostgreSQL 生产路径。

## 后续边界

只有在对应 SQLite Repository、迁移、并发与恢复测试完成后，业务模块才可加入 SQLite 能力表。完整功能等价、SQLite Worker 与 PostgreSQL/SQLite 数据迁移工具分别评审，不作为本 ADR 的隐含承诺。插件系统不在本次范围。
