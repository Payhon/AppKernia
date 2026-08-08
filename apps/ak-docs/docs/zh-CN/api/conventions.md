---
title: 响应、错误与幂等
description: AppKernia API 的通用响应结构、语言协商、分页与安全重试规则。
---

# 响应、错误与幂等

## 成功响应

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "request_id": "01900000-0000-7000-8000-000000000001"
}
```

`request_id` 应贯穿客户端错误上报、服务端日志和审计排障。

## 错误响应

```json
{
  "error": {
    "code": "IAM.AUTH.INVALID_CREDENTIALS",
    "message_key": "errors.iam.auth.invalid_credentials",
    "message": "账号或密码错误",
    "details": {}
  },
  "request_id": "01900000-0000-7000-8000-000000000001"
}
```

客户端只根据稳定 `error.code` 或 `message_key` 判断业务，不解析 `message`。常用 HTTP 状态：

| 状态  | 含义                         | 客户端行为                 |
| ----- | ---------------------------- | -------------------------- |
| `400` | 请求格式错误                 | 修正请求                   |
| `401` | 会话无效或 Access Token 过期 | single-flight Refresh 一次 |
| `403` | 身份有效但无权限             | 不 Refresh，展示无权限     |
| `404` | 资源不存在或不可见           | 不推断其他租户资源是否存在 |
| `409` | 版本/状态冲突                | 重新加载并显式解决         |
| `422` | 字段校验失败                 | 映射到本地表单字段         |
| `429` | 请求过多                     | 尊重 `Retry-After`         |

## 语言协商

请求使用 `Accept-Language: zh-CN` 或 `en-US`，响应使用 `Content-Language`。错误码与原始数据不随语言改变。

## 分页

Admin 资源常用 `page` / `page_size`，移动消息和文章使用 opaque cursor。客户端不得解析 cursor 内容，也不能在同一资源上擅自混用分页模式。

## 幂等与重试

GET / HEAD 可以在网络中断后有限退避；POST / PATCH / DELETE 默认不重试。只有 OpenAPI 明确支持 `Idempotency-Key` 的写操作才可安全重放：

```http
Idempotency-Key: 01900000-0000-7000-8000-000000000001
```
