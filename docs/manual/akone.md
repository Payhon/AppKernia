# akone 使用手册

`akone` 同时提供 AppKernia 服务、运维命令和面向 Agent 的 OpenAPI CLI。裸执行只显示帮助，不会意外启动服务。

## 初始化与启动

不提供数据库配置时，`akone` 使用 SQLite，并在二进制同目录自动创建 `data/appkernia.db`。SQLite schema 在打开数据库时幂等升级；首次使用只需初始化管理员并启动：

```bash
akone bootstrap-admin \
  --email admin@example.com \
  --tenant-code local \
  --tenant-name "Local Workspace" \
  --display-name Administrator
akone serve

# 指定 SQLite 文件时，初始化命令也使用同一环境变量或 YAML
AK_SQLITE_PATH=/srv/appkernia/data/appkernia.db akone bootstrap-admin \
  --email admin@example.com \
  --tenant-code local \
  --tenant-name "Local Workspace" \
  --display-name Administrator
akone serve --sqlite /srv/appkernia/data/appkernia.db
```

也可以先运行 `akone config init --output ./akone.yml` 生成完整 YAML。配置文件可能包含数据库凭据与签名密钥：Unix/macOS 下必须保持 `0600`，Windows 下应放在用户配置目录并依赖该目录的用户 ACL；仓库根目录的 `akone.yml` / `akone.yaml` 已被忽略。

`bootstrap-admin` 会从终端隐藏读取密码。服务默认监听 `127.0.0.1:8080`，管理端入口为 `http://127.0.0.1:8080/admin/`；进程接收 `SIGINT`/`SIGTERM` 后会优雅退出。PostgreSQL 模式同时运行 API 与 River Worker，SQLite 模式不启动会丢任务的空 Worker。

配置优先级固定为：命令行非敏感参数 > `AK_*` 环境变量 > YAML > 默认值。`config show` 可查看脱敏后的最终配置：

```bash
akone --config ./akone.yml config show
AK_HTTP_ADDR=127.0.0.1:18080 AK_LOG_LEVEL=debug akone --config ./akone.yml serve
```

常用配置如下；完整模板以 `akone config init` 的输出为准：

```yaml
environment: development
server:
  listen: 127.0.0.1:8080
  public_web_base_url: http://127.0.0.1:8080
  shutdown_timeout: 15s
database:
  driver: sqlite
  sqlite_path: ./data/appkernia.db
admin:
  origin: http://127.0.0.1:8080
  path: /admin
  static_dir: ""
log:
  level: info
  format: text
  file: ""
```

- `admin.path` 可改管理端 URL 前缀；`admin.static_dir` 为空时使用二进制内嵌 Admin，设置后完整使用该目录，目录必须包含兼容的 `index.html`。
- 相对的 `database.sqlite_path`、`admin.static_dir`、日志与本地存储路径以 YAML 所在目录为基准。`--sqlite` 和 `AK_SQLITE_PATH` 的相对路径以当前工作目录为基准。
- Unix/macOS 下 SQLite 会以 `0700` 新建专用目录、以 `0600` 创建数据库，并拒绝 group/world-writable 的已有目录；Windows 依赖用户目录 ACL。通过 npm/Homebrew 更新或移动二进制前，建议用 `AK_SQLITE_PATH` 或 YAML 固定数据位置，避免默认的“二进制同目录”位置随安装目录改变。
- YAML 会拒绝未知字段和多文档内容。生产环境仍必须显式配置签名密钥、登录保护密钥与配置主密钥。
- 对应环境变量包括 `AK_CONFIG_FILE`、`AK_DATABASE_DRIVER`、`AK_DATABASE_URL`、`AK_SQLITE_PATH`、`AK_HTTP_ADDR`、`AK_ADMIN_ORIGIN`、`AK_ADMIN_PATH`、`AK_ADMIN_STATIC_DIR`、`AK_LOG_LEVEL`、`AK_LOG_FORMAT`、`AK_LOG_FILE` 和现有全部 `AK_*` 功能开关。
- 单独设置 `AK_DATABASE_URL` 会选择 PostgreSQL，单独设置 `AK_SQLITE_PATH` 会选择 SQLite；同一层同时填写两种连接值时必须用 `AK_DATABASE_DRIVER` / `database.driver` 明确选择，避免误连数据库。

完整业务与后台任务继续使用 PostgreSQL。只配置 `database.url` / `AK_DATABASE_URL` 会自动选择 PostgreSQL，也可显式设置：

```yaml
database:
  driver: postgresql
  url: postgres://appkernia:password@127.0.0.1:5432/appkernia?sslmode=disable
```

PostgreSQL 首次启动前仍需运行 `akone migrate up` 和 `akone seed core`。SQLite 首期覆盖健康检查、内嵌管理端、管理员认证与个人会话、Dashboard；API Client、App、内容、通知、推送与任务队列等模块尚未宣称 SQLite 等价，详见 [ADR-0034](../adr/0034-akone-sqlite-standalone.md)。

## 配置 CLI 身份

API Client 当前需要 PostgreSQL 模式。先在管理端“API 客户端”页面创建客户端、绑定一个启用用户，并分别配置客户端权限与 App 白名单。实际权限始终是“客户端权限 ∩ 绑定用户当前权限”，令牌 audience 固定为 `ak-api`，不会变成浏览器管理会话。

```bash
akone auth configure \
  --server https://example.com \
  --client-id ak_xxx
```

Secret 从隐藏的终端提示读取，不应放进命令参数。流水线可显式从标准输入读取：

```bash
printf '%s\n' "$AK_CLIENT_SECRET" | akone auth configure \
  --server https://example.com \
  --client-id "$AK_CLIENT_ID" \
  --secret-stdin
```

默认凭据文件位于系统用户配置目录；Unix/macOS 下强制 `0600`，Windows 下依赖用户配置目录 ACL。也可仅设置 `AK_SERVER_URL`、`AK_CLIENT_ID`、`AK_CLIENT_SECRET`、`AK_CREDENTIALS_FILE` 后直接调用；除 localhost 外，服务地址必须使用 HTTPS。

## 调用 OpenAPI

CLI 只列出服务端明确开放给 API Client 的操作，不接受任意 URL，也不会自动重试写请求：

```bash
akone api list
akone api describe getAdminDashboardSummary
akone api call getAdminDashboardSummary --query range=30d

akone api describe createAdminAppContentArticle
akone api call createAdminAppContentArticle \
  --path app_id=00000000-0000-0000-0000-000000000000 \
  --body @article.json
```

参数名与内嵌 OpenAPI 保持一致：路径参数使用可重复的 `--path name=value`，查询参数使用 `--query`，声明过的 Header 使用 `--header`，JSON 请求体可用内联 JSON、`@file` 或 `-` 从 stdin 读取。结果写 stdout，诊断写 stderr，非 2xx 响应返回非零退出码。

## 诊断

```bash
akone version --json
akone --config ./akone.yml doctor --json
```

`doctor` 会实际打开并检查当前选择的 SQLite 或 PostgreSQL。日志支持 `text`/`json`、`debug`/`info`/`warn`/`error` 以及文件输出；日志文件权限为仅所有者可读写，并同步输出到 stderr。
