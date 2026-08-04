# PostgreSQL Schema 与后台 UI 覆盖说明

本文件对照 AK 后端核心 Schema，明确每张表是否需要后台页面。原则是：**功能必须可追踪，但基础设施表不应为了“每表一个 CRUD”而暴露不安全的管理入口。**

## 覆盖结论

- 核心表总数：51
- 页面直接覆盖：48
- 脱敏聚合覆盖：1
- 仅后端基础设施：2

`spec/schema-ui-coverage.json` 是机器可读事实源，并由 `scripts/validate_blueprint_specs.py` 强制校验全量覆盖。

## 无直接 CRUD 页面的表

| 表 | 分类 | 前端处理 |
|---|---|---|
| `jobs.inbox_events` | `backend_only` | 消息消费幂等与去重基础设施；仅通过运行指标/告警间接观测，不提供逐行管理页面。 关联：无直接页面。 |
| `notify.push_devices` | `indirect_aggregate` | 仅展示推送设备数量、平台、最近活跃与投递状态；Push Token 不进入通用列表或前端日志，也不提供直接 CRUD。 关联：`system.users.accounts`、`system.notifications.deliveries`。 |
| `sys.idempotency_keys` | `backend_only` | HTTP 幂等基础设施；记录由服务端生命周期自动清理，不允许后台人工编辑。 关联：无直接页面。 |

## 设计约束

- `notify.push_devices` 绝不向浏览器返回原始 Push Token；只返回脱敏设备元数据和统计。
- `jobs.inbox_events` 不提供重放/删除按钮；事件补偿应通过受控运维命令或专用 Runbook。
- `sys.idempotency_keys` 不提供人工修改；清理策略由后端任务和 TTL 管理。
- 后续新增 Migration 表时，CI 必须要求同步更新本覆盖清单，否则校验失败。
