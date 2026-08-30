# ADR-0022：消息运行时、可观测工作台与统一权限中心

- 状态：Accepted
- 日期：2026-08-29
- 范围：Backend、Admin、Mobile、数据库与内部集成

## 背景

多厂商离线推送已经形成“发布、扇出、单设备投递”的 River 链路，但业务模块仍可能直接接触 River，通知任务缺少长期、租户隔离的运行索引，外部可信服务没有稳定的提交接口，Mobile 的系统权限查询也未形成可扩展公共能力。

## 决策

1. 保留 PostgreSQL + River，不引入第二套队列。新增 `server/internal/platform/jobqueue` Port，业务只提交经过编译期注册的任务；River 是基础设施 Adapter。
2. `river_job` 继续作为实时执行状态事实源；`jobs.task_runs` 和 `jobs.task_attempts` 只保存不含 Args、Token、载荷、密钥和原始厂商响应的长期业务投影。Worker 在每次尝试开始和结束时更新投影，维护任务负责对账与保留期清理。
3. `notify.message_runs` 保存一条消息从计划到投递完成的流水线状态，`notify.delivery_daily_metrics` 保存 13 个月按 App、环境、渠道、厂商、分类和结果聚合的数据；任务与尝试明细保留 90 天。
4. 管理端新增应用级“消息运营”工作台。读取必须同时满足 tenant 与 app 隔离；手动重试创建新任务并保留原任务历史，不允许编辑 Args、强制终止运行中任务或操作 River 原始记录。
5. `unknown_after_write` 只允许单条、显式确认重复风险后重试；鉴权配置错误必须先修复渠道并通过预检；批量重试上限 100 条。
6. 新增中立的 `platform/notification.Service`。`SubmitTx` 把业务数据、通知、运行记录和任务入队放在调用方事务内提交，避免双写半完成。可信 Scope 由内部调用方或 Machine Principal 中间件构造，HTTP Body 不能覆盖 tenant、app 或 actor。
7. M2M API 使用短期 `ak-api` JWT。API Client subject 是 Client，自带真实 tenant claim，不签发 Refresh Token；`sys.api_client_apps` 采用显式 App allowlist，空列表默认拒绝，并继续校验 Client 状态、到期时间、CIDR 和权限。
8. Mobile 新增 `ak-permissions` 编译期 Registry。页面加载只查询状态，只有用户主动操作才申请权限；首期展示通知权限，后续相机、照片、文件选择、麦克风、定位和蓝牙只预留定义，未被业务使用时不展示、不申请。
9. `ak-push` 委托 `ak-permissions` 查询、申请与打开系统设置，避免两套 OS 状态源。系统权限不上传服务端，服务端只保存通知偏好和 Push 设备绑定。

## 安全和运营语义

- 管理端展示的是脱敏错误分类和安全摘要，不提供 River Args、堆栈或厂商响应正文。
- `sent` 继续只表示厂商受理；`opened` 表示用户点击进入应用，不等于已读。
- Prometheus 标签不得包含 Token、用户 ID、Trace ID 或消息正文。
- API Client 的广播额外要求 `notify.message.broadcast`；运营消息额外要求 `notify.operations.publish`。
- 取消仅阻止尚未进入发布或扇出的消息，无法撤回已被厂商受理的通知。

## 后果

通知、邮件、短信、Webhook 和后续异步业务可复用同一 JobQueue 边界；Admin 可以从业务语义观察队列而不依赖 River 内部 Schema。代价是需要维护 River 与业务投影的对账，并对数据库明细和聚合执行不同保留策略。

真实厂商凭据、签名包和物理设备权限行为仍是外部门禁；源码编译、Mock Provider 和未签名 HAP 不能替代生产或真机验收。
