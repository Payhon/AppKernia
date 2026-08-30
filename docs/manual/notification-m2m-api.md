# 通知 Go 服务与 M2M HTTP API

## 1. 内部 Go 调用

内部业务统一依赖 `server/internal/platform/notification.Service`，不得直接写 `notify.*` 或插入 River Job。

```go
submission, err := notifications.SubmitTx(ctx, tx, trustedScope, notification.SubmitCommand{
    IdempotencyKey:  event.ID,
    Source:          "account-security",
    BusinessEventID: event.ID,
    Category:        "service_security",
    Audience:        notification.Audience{Type: "users", UserIDs: []uuid.UUID{userID}},
    Content: notification.Content{Inline: &notification.LocalizedContent{
        Title: map[string]string{"zh-CN": "安全提醒", "en-US": "Security alert"},
        Body:  map[string]string{"zh-CN": "账号安全状态已更新", "en-US": "Your account security state changed"},
    }},
    Push:    true,
    RouteKey: "notification_detail",
})
```

`SubmitTx` 使用调用方的 `pgx.Tx`，因此业务事实、消息、`notify.message_runs`、`jobs.task_runs` 和 River Job 一起提交或一起回滚。`Scope` 必须由可信调用方构造，禁止把 HTTP Body 中的 tenant、app 或 actor 透传进来。

## 2. API Client 准备

1. 在 Admin 创建启用状态的 API Client，配置到期时间和可选 CIDR。
2. 分配 `notify.message.submit`；查询状态还需 `notify.message.status.read`。
3. 如果 audience 为所有有效 App 用户，额外分配 `notify.message.broadcast`。
4. 如果 category 为 `news_operations`，额外分配 `notify.operations.publish`。
5. 在 API Client 的“授权应用”中显式选择 App；空 allowlist 默认拒绝。
6. 使用 `/api/v1/auth/client-token` 获取短期 `ak-api` Access Token。该流程不签发 Refresh Token。

## 3. 提交消息

```http
POST /api/v1/apps/{app_id}/notifications
Authorization: Bearer <short-lived-ak-api-jwt>
Idempotency-Key: account-security-event-20260829-0001
Content-Type: application/json

{
  "source": "account-security",
  "business_event_id": "20260829-0001",
  "category": "service_security",
  "audience": {"type": "users", "user_ids": ["<user-uuid>"]},
  "content": {
    "type": "inline",
    "inline": {
      "title": {"zh-CN": "安全提醒", "en-US": "Security alert"},
      "body": {"zh-CN": "账号安全状态已更新", "en-US": "Your account security state changed"}
    }
  },
  "push": true,
  "ttl_seconds": 3600,
  "route_key": "notification_detail",
  "resource_id": "opaque-business-id"
}
```

成功返回 `202 Accepted`，包含 `message_id`、`run_id`、当前状态、`status_url` 和创建时间。相同 Client、相同 `Idempotency-Key`、相同请求体返回原提交；相同键但请求体不同返回 `409 NOTIFY.IDEMPOTENCY.CONFLICT`。

模板内容和受控内联双语内容必须且只能选择一个。请求不接受原始 Token、任意 URL、组件名、脚本或厂商特有载荷。

## 4. 查询与取消

```http
GET /api/v1/apps/{app_id}/notifications/{message_id}
POST /api/v1/apps/{app_id}/notifications/{message_id}/cancel
```

查询返回收件人、已评估、投递、厂商受理、失败、无效 Token、跳过和打开计数。取消只对尚未进入发布或扇出的任务生效；已发送到厂商的通知无法撤回。

## 5. 错误与审计

- `401 AUTH.CLIENT.INVALID`：凭据、JWT、Client 状态或到期时间不合法。
- `403 COMMON.FORBIDDEN`：CIDR、App allowlist 或权限不满足。
- `409 NOTIFY.IDEMPOTENCY.CONFLICT`：幂等键对应另一请求体。
- `422 NOTIFY.SUBMISSION.INVALID`：内容联合类型、受众、时间、TTL 或受控路由不合法。

提交与取消均记录安全审计；日志和审计不记录 Access Token、完整消息载荷或设备 Token。
