# Backend 集成说明

移动端只调用 `/api/v1`，不得用 `/admin-api/v1` 代替。

## 契约优先级

```text
server/openapi/app-v1.yaml
> blueprint/mobile/integration/app-api-baseline.json + app-api-delta.json
> 文档示例
```

## 后端增量

详情见 `integration/BACKEND_CHANGES_REQUIRED.md`。实现任何增量必须同步：

- Route/Controller。
- Application Use Case。
- OpenAPI 3.1。
- 权限 Seed。
- sqlc/Repository（如涉及持久化）。
- 审计与安全事件。
- Contract/Integration Test。
- 移动端生成 Client。

## Audience

移动 Token Audience 固定 `ak-mobile`。Admin Token、API Client Token 不得访问移动 Session API；反向也不得成立。
