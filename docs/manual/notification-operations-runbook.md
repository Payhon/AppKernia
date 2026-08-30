# 消息运营与任务队列运维手册

## 1. 工作台

Admin 路由 `/system/notifications/operations` 包含四个可通过 URL 恢复的页签：概览、发布运行、队列任务、失败中心。所有接口按当前租户和选中 App 强制隔离。

页面可见且存在未完成任务时每 15 秒刷新；浏览器页签隐藏后暂停。运营人员可以随时手动刷新。旧投递记录路由保留兼容跳转。

## 2. 数据事实源

| 数据 | 用途 | 保留 |
|---|---|---:|
| `river_job` | River 实时调度事实源 | 按 River 策略 |
| `jobs.task_runs` | 租户/App/模块/资源维度的任务索引 | 90 天 |
| `jobs.task_attempts` | 每次执行的安全结果、耗时和 Trace ID | 90 天 |
| `notify.message_runs` | 一条消息的发布、扇出、投递流水线 | 90 天明细 |
| `notify.delivery_daily_metrics` | App/环境/渠道/厂商/分类/结果日聚合 | 13 个月 |

任务投影不保存 River Args、完整堆栈、Token、推送载荷、密钥或厂商响应正文。

## 3. 编译期任务策略

通知任务由 `server/internal/modules/notificationadmin/jobdefs` 注册，入队 Adapter 会在写 River 前校验；业务提交的队列或尝试次数不能绕过该策略。

| Task kind | Queue | 最大尝试 | Worker 超时 | 自动重试分类 |
|---|---|---:|---:|---|
| `appkernia-message-publish` | `notifications` | 5 | 30 秒 | `transient` |
| `appkernia-push-fanout` | `notifications` | 5 | 90 秒 | `transient` |
| `appkernia-notification-delivery` | `notifications` | 5 | 90 秒 | `throttled`、`transient` |

`unknown_after_write`、`auth_config_error` 和永久错误不属于自动重试 Registry；它们继续遵守下节的人工风险控制。

## 4. 手动重试规则

1. 只允许终止且服务端判定 `retryable=true` 的任务。
2. `transient`、`throttled` 可在自动尝试耗尽后重试；`permanent`、取消或过期任务禁止重试。
3. `auth_config_error` 必须先修复对应 App/环境/Provider 配置并通过预检。
4. `unknown_after_write` 可能已经送达，只能单条选择并显式确认重复通知风险。
5. 一次批量请求最多 100 条，服务端逐条返回接受或拒绝原因。
6. 重试会创建新的 `jobs.task_runs`，原任务和尝试历史保持不变。

不要直接更新 `river_job`、编辑任务 Args 或强制终止运行中的任务。

## 5. 排障顺序

1. 在“概览”确认队列深度、最老等待时间、P95 排队延迟和故障数量。
2. 在“发布运行”确认消息停在哪一阶段，核对收件人冻结数、已评估数与投递数。
3. 在“队列任务”查看 task kind、尝试次数、下一次重试、错误码和 Trace ID。
4. 厂商鉴权失败时先前往“推送渠道”修复并预检，不要连续重试。
5. 需要完整调用链时，用 Trace ID 到 OpenTelemetry/Loki/Tempo 查询；Admin 只展示脱敏摘要。

## 6. 监控建议

告警至少覆盖队列深度和最老等待时间持续增长、永久失败、重试激增、厂商延迟、无效 Token 比例、渠道连续鉴权失败和发布流水线耗时。Prometheus 标签只使用 App、Provider、Category、Result 等低基数维度，不使用用户 ID、Token 或消息正文。

清理任务只删除已终止且已完成日聚合的数据，不删除 scheduled、queued、running 或 retry_wait 状态。
