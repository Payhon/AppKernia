# AppKernia Server

GoFrame + pgx/v5 + sqlc + PostgreSQL 18 模块化单体，包含 `ak-api`、`ak-worker` 和 `ak-cli`。数据库迁移事实源位于 `../blueprint/backend/db/migrations`；迁移编号、可选模块边界和回滚脚本以该目录及后端蓝图为准。

从仓库根目录完成环境初始化：

```bash
cp .env.example .env
make dev-deps
make db-setup
```

后端开发入口：

```bash
make -C server dev          # API + Worker
make -C server dev-api      # 仅 API
make -C server dev-worker   # 仅 Worker
```

测试与构建：

```bash
make -C server test
make -C server test-race
make -C server check
make -C server build
```

构建产物位于 `server/bin/`。`make -C server help` 可查看全部入口。

使用 `make -C server bootstrap-admin` 交互创建首个管理员。密码只从 stdin/TTY 读取，禁止放入参数、环境变量、日志或 Git。

发布移动端安装包前，从后台启动元信息导出严格离线的首装隐私页快照与图标：

```bash
go run ./cmd/ak-cli app-startup export --app-id <public-app-uuid> --output ../apps/ak-mobile
go run ./cmd/ak-cli app-startup export --app-id <public-app-uuid> --output ../apps/ak-mobile --check
```

导出要求双语名称/副标题完整，且图标文件属于该 App、处于 `ready` 并已扫描通过。`--check` 不写文件，用于发布门禁检测生成物漂移。

微信分享配置在 Admin 完成激活、App 绑定和预检后，必须导出到 Mobile 原生配置并重新打包：

```bash
go run ./cmd/ak-cli app-share export \
  --app-id <public-app-uuid> \
  --output ../apps/ak-mobile \
  --android-package <release-package> \
  --android-signature <wechat-release-signature> \
  --ios-bundle-id <release-bundle-id> \
  --harmony-bundle-name <release-bundle-name>

# CI 中追加 --check 只读验证生成文件无漂移
```

命令只合并 `uni-share.weixin` 与 iOS Associated Domains，不覆盖其他 Manifest/Entitlements 设置；任一启用平台的构建身份不匹配都会拒绝导出。微信 AppSecret 不由本系统采集或使用。

本地开发可以使用 `.env.example` 中明确标记的 development-only Adapter 和密钥。非 development 环境必须提供独立的 `AK_JWT_PRIVATE_KEY_BASE64`、`AK_JWT_KEY_ID` 与 `AK_CONFIG_MASTER_KEY_BASE64`，并禁用本地 Mock Adapter。
## Multi-provider Push

`AK_PUSH_ENABLED` is the global kill switch. Push provider credentials are managed per tenant, App, environment, and provider through `/admin-api/v1/apps/{app_id}/push-provider-configs`; secrets are write-only and encrypted with the configured settings key. Mobile registration is available only under `/api/v1/me/push-devices` and derives tenant, App, user, and IAM device identity from the authenticated session.

Development can use `AK_PUSH_MODE=mock`. Official mode implements APNs token authentication, FCM HTTP v1, the supported China Android vendor REST APIs, and HarmonyOS Push Kit JWT authorization. Provider acceptance is recorded as `sent`/`accepted`; it is not a device-display receipt. `unknown_after_write` is held for manual review and is never blindly replayed.

## Notification runtime and M2M API

Application modules enqueue asynchronous work through `internal/platform/jobqueue`; River remains the PostgreSQL-backed infrastructure adapter. `jobs.task_runs` and `jobs.task_attempts` contain tenant/App-scoped, payload-free projections for operations and long-term diagnosis. They never contain River Args, device tokens, message payloads, credentials, or raw provider responses.

Reusable notification submission is exposed through `internal/platform/notification.Service`. Trusted API Clients use the same service through:

```text
POST /api/v1/apps/{app_id}/notifications
GET  /api/v1/apps/{app_id}/notifications/{message_id}
POST /api/v1/apps/{app_id}/notifications/{message_id}/cancel
```

Submission requires a short-lived `ak-api` token, explicit `sys.api_client_apps` authorization, `notify.message.submit`, and an `Idempotency-Key`. Broadcast and operations-category messages require their additional permissions. See [`docs/manual/notification-m2m-api.md`](../docs/manual/notification-m2m-api.md).

Admin notification operations APIs are App-scoped under `/admin-api/v1/apps/{app_id}/notification-*`. Safe manual retry creates a new tracked task and preserves the original history. See [`docs/manual/notification-operations-runbook.md`](../docs/manual/notification-operations-runbook.md).
