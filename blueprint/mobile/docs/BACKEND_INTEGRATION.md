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

## 原生升级

- 自动与手动检查都调用 `GET /api/v1/public/app-version?platform=...&package_type=native_app`，使用运行时三段式 SemVer 在客户端判定是否需要升级。
- `delivery_mode=external_link` 时按 `store_list.priority` 尝试已启用市场 scheme，并以 HTTPS `upgrade_url` 回退；危险 scheme 必须在原生适配器拒绝。
- `delivery_mode=internal_package` 只允许 Android APK。下载短时签名 URL 时仅发送 `X-AppID`，不得附带用户 Access/Refresh Token。
- `uni_app_x` 的 WGT 请求、发布和重新发布由后端拒绝；iOS/HarmonyOS 内部原生包同样拒绝。历史不兼容记录只允许读取和下线。
