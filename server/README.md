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

本地开发可以使用 `.env.example` 中明确标记的 development-only Adapter 和密钥。非 development 环境必须提供独立的 `AK_JWT_PRIVATE_KEY_BASE64`、`AK_JWT_KEY_ID` 与 `AK_CONFIG_MASTER_KEY_BASE64`，并禁用本地 Mock Adapter。
