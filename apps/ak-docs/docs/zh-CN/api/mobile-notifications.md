---
title: 通知与推送 API
description: 使用可信 M2M 接口提交异步通知，并通过 Mobile API 管理偏好、设备绑定、站内消息和打开统计。
---

# 通知与推送 API

本页接口都位于 `/api/v1`，但包含两个不能互换的身份面：

| 身份面          | Audience    | 用途                                        |
| --------------- | ----------- | ------------------------------------------- |
| 可信业务服务    | `ak-api`    | 提交、查询和取消应用通知                    |
| Mobile 当前用户 | `ak-mobile` | 通知偏好、站内消息、Push 设备注册和通知打开 |

字段、枚举和响应 Schema 以[当前 OpenAPI](./online-reference)为最终事实源。

## 可信业务服务准备

1. 在 Admin 创建启用状态的 API Client，设置到期时间和可选 CIDR。
2. 分配 `notify.message.submit`；查询还需要 `notify.message.status.read`。
3. 广播到全部有效 App 成员需要 `notify.message.broadcast`。
4. 提交 `news_operations` 还需要 `notify.operations.publish`。
5. 在 API Client 的应用授权中显式选择目标 App；空 allowlist 默认拒绝。
6. 调用 `/auth/client-token` 换取短期 `ak-api` JWT。Machine Principal 不签发 Refresh Token。

## 提交通知

接口：`/apps/{app_id}/notifications`

```http
POST /api/v1/apps/01900000-0000-7000-8000-000000000001/notifications
Authorization: Bearer AK_API_ACCESS_TOKEN
Idempotency-Key: account-security-event-20260830-0001
Accept-Language: zh-CN
Content-Type: application/json

{
  "source": "account-security",
  "business_event_id": "20260830-0001",
  "category": "service_security",
  "audience": {
    "type": "users",
    "user_ids": ["01900000-0000-7000-8000-000000000099"]
  },
  "content": {
    "type": "inline",
    "inline": {
      "title": {
        "zh-CN": "安全提醒",
        "en-US": "Security alert"
      },
      "body": {
        "zh-CN": "账号安全状态已更新",
        "en-US": "Your account security state changed"
      }
    }
  },
  "push": true,
  "ttl_seconds": 3600,
  "route_key": "notification_detail",
  "resource_id": "opaque-business-id"
}
```

成功返回 `202 Accepted`、`message_id`、`run_id`、当前状态、`status_url` 和创建时间。它表示消息已进入异步流水线，不表示厂商或设备已经接收。

模板内容和受控内联双语内容必须且只能选择一个。请求不接受原始 Token、任意 URL、组件名、脚本或厂商特有载荷。

## 幂等与取消

`Idempotency-Key` 长度为 8–255。当前服务按 tenant 与调用身份（M2M 场景即 API Client）隔离，因此同一 API Client 应在它获准访问的所有 App 和来源之间生成全局唯一键。相同键与相同规范化请求体返回原提交；相同键对应不同请求体时返回 `409 NOTIFY.IDEMPOTENCY.CONFLICT`。

- 查询状态：`/apps/{app_id}/notifications/{message_id}`
- 取消消息：`/apps/{app_id}/notifications/{message_id}/cancel`

状态包含收件人、已评估、设备投递、厂商受理、失败、无效 Token、跳过和打开计数。取消只阻止尚未进入发布或扇出的消息，无法撤回厂商已经受理的通知。

## Mobile 当前用户接口

以下请求使用 `ak-mobile` Bearer Token 和当前 App 上下文：

| 方法     | 路径                                       | 用途                                           |
| -------- | ------------------------------------------ | ---------------------------------------------- |
| `GET`    | `/me/notification-preferences`             | 获取 Push 总开关和分类订阅                     |
| `PATCH`  | `/me/notification-preferences`             | 更新 `push`、`push_service`、`push_operations` |
| `GET`    | `/me/notifications`                        | 分页读取已送达的站内消息                       |
| `GET`    | `/me/notifications/{id}`                   | 读取一条属于当前用户和 App 的消息              |
| `PATCH`  | `/me/notifications/{id}/read`              | 标记站内消息已读                               |
| `GET`    | `/me/push-devices/current`                 | 获取当前安装注册状态，不返回 Token             |
| `POST`   | `/me/push-devices`                         | 幂等注册或刷新不透明厂商 Token                 |
| `DELETE` | `/me/push-devices/{push_device_id}`        | 停用当前用户、当前 App 下的设备绑定            |
| `POST`   | `/me/push-deliveries/{delivery_id}/opened` | 幂等记录通知点击；不自动标记站内消息已读       |

设备注册请求包含 provider、platform、build variant、Token、规范化语言、SDK 版本和 App 版本。tenant、App、user 和业务设备 ID 从已验证会话与应用上下文派生，不接受客户端覆盖；响应永远不返回 Token。

## 内部 Go 调用

同一模块化单体内的业务优先依赖 `server/internal/platform/notification.Service`：

```go
type Service interface {
    Submit(ctx context.Context, scope Scope, cmd SubmitCommand) (Submission, error)
    SubmitTx(ctx context.Context, tx pgx.Tx, scope Scope, cmd SubmitCommand) (Submission, error)
    Status(ctx context.Context, scope Scope, id uuid.UUID) (SubmissionStatus, error)
    Cancel(ctx context.Context, scope Scope, id uuid.UUID) error
}
```

需要与业务事实原子提交时使用 `SubmitTx`。可信 `Scope` 由调用方或认证中间件构造，不能从 HTTP Body 中复制 tenant、App 或 actor。业务模块不应直接写 `notify.*` 表或插入 River Job。

## 常见错误

| HTTP | 稳定错误码                    | 含义                                       |
| ---- | ----------------------------- | ------------------------------------------ |
| 401  | `AUTH.CLIENT.INVALID`         | Client 凭据、JWT、状态或到期时间无效       |
| 403  | `COMMON.FORBIDDEN`            | CIDR、App allowlist 或权限不满足           |
| 409  | `NOTIFY.IDEMPOTENCY.CONFLICT` | 幂等键已绑定到不同请求体                   |
| 422  | `NOTIFY.SUBMISSION.INVALID`   | 受众、内容联合类型、时间、TTL 或路由不合法 |

日志和审计不记录 Access Token、设备 Token 或完整消息载荷。运行状态与失败处理方式见[消息运营工作台](../guide/notification-operations)；异步语义见[消息推送架构](../concepts/notification-architecture)。
