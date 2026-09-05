# ADR-0033：akone 首期保持 PostgreSQL 数据层

状态：已被 ADR-0034 替代。

`akone` 首期把 API、Worker、管理端静态资源、迁移、种子与 OpenAPI CLI 合并为一个可发行文件，但不把数据库实现伪装成可替换驱动。现有后端契约固定为 PostgreSQL 18，并直接依赖 pgx/v5、sqlc 生成查询、PostgreSQL schema/约束和 River PostgreSQL 队列；仅加入 SQLite 驱动无法提供等价的迁移、事务、租户隔离与 Worker 语义。

因此当前开发默认值仍连接本机 PostgreSQL，生产必须显式配置 `AK_DATABASE_URL` 或 YAML `database.url`。SQLite 不作为已完成能力，也不加入只覆盖少量接口的降级模式。

后续若要实现真正的 SQLite 默认启动，必须先单独批准数据层 ADR，明确查询生成、迁移、队列、并发写、备份恢复与 PostgreSQL 升级路径，并完成全量契约和集成测试。该工作不阻塞单二进制打包与多渠道发行。
