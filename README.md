# AppKernia

AppKernia（AK）是一个面向生产项目的跨平台应用基座：Go API/Worker、React 管理后台，以及 uni-app x 移动端。后端以 OpenAPI 为 API 事实源，Admin 与 Mobile 共用权限、数据库和 `zh-CN` / `en-US` 国际化契约。

- 开源官网与文档：<https://appkernia.com>
- GitHub Pages 地址：<https://payhon.github.io/AppKernia/>
- 源码仓库：<https://github.com/Payhon/AppKernia>

如果 AppKernia 帮你少走了一些跨端与服务端集成的弯路，欢迎点一个 Star；如果它还缺少你需要的能力，也欢迎从文档、Issue、测试或代码开始贡献。

## 技术栈

- Backend：Go 1.26、GoFrame、pgx/v5、sqlc、PostgreSQL 18、River。
- Admin：React、TypeScript、Vite、TanStack Router/Query、Ant Design。
- Mobile：uni-app x、UTS/UVue、VDOM、uView Ultra，并通过 AK UI 适配层使用组件。

## 环境要求

- Go 1.26.5（Makefile 会使用 `GOTOOLCHAIN=go1.26.5`）。
- Node.js 24.18.1、npm 11+；推荐通过 Corepack 使用 pnpm 11.18.0。
- Docker Desktop 或兼容的 Docker Engine + Compose v2，用于 PostgreSQL 和完整容器开发模式。
- Python 3，用于蓝图与国际化契约校验。

版本基线同时记录在 `.nvmrc`、`.tool-versions` 和根 `package.json`。可先检查本机环境：

```bash
make toolchain
```

HBuilderX、Xcode 和 DevEco Studio 只在开发对应移动平台时需要，doctor 会把它们标为可选工具。

## 快速开始：源码开发模式

这是日常修改 Backend 和 Admin 时推荐的方式：PostgreSQL 运行在 Docker 中，Go 与 Vite 直接运行在宿主机，便于断点调试和即时刷新。

```bash
git clone <your-fork-or-repository-url>
cd AppKernia
cp .env.example .env
make setup
```

`make setup` 会安装 pnpm 依赖、启动 PostgreSQL、执行当前数据库迁移链，并幂等写入核心权限和菜单。之后分别打开两个终端：

```bash
# 终端 1：同时启动 API 与 River Worker
make dev-backend

# 终端 2：启动 Vite Admin，自动代理 /admin-api 到 127.0.0.1:8080
pnpm dev
```

依赖安装完成后，也可以用 npm 调用同一套前端脚本：

```bash
npm run dev
npm test
npm run build
```

仓库以 `pnpm-lock.yaml` 作为锁文件事实源，因此依赖安装和 CI 使用 pnpm；`npm run` 路径用于调用标准 package scripts，避免产生第二份锁文件。

创建首个管理员时，密码仅从交互终端读取，不会进入命令历史或 `.env`：

```bash
make -C server bootstrap-admin
```

开发环境也可以让 `seed core` 幂等创建或补齐管理员授权。邮箱默认是
`admin@appkernia.local`，密码只能从 Git 和 Docker 构建上下文均忽略的文件读取：

```bash
mkdir -p .secrets
chmod 700 .secrets
printf 'Seed administrator password: '
read -r -s AK_LOCAL_SEED_PASSWORD; printf '\n'
printf '%s\n' "$AK_LOCAL_SEED_PASSWORD" > .secrets/seed-admin-password
unset AK_LOCAL_SEED_PASSWORD
chmod 600 .secrets/seed-admin-password
AK_SEED_ADMIN_PASSWORD_FILE="$PWD/.secrets/seed-admin-password" make -C server seed-core
```

出于开源安全考虑，数据库 Migration、源码和 Core Seed **不会内置固定密码**；未设置
`AK_SEED_ADMIN_PASSWORD_FILE` 时 Core Seed 不创建账号，非开发环境则拒绝该选项。管理员密码至少 12 位；本机若由维护者准备了临时测试账号，其凭据只记录在 Git 忽略的位置，不得复制到提交、Issue、日志或截图。

默认访问地址：

- Admin：<http://localhost:4173>
- API readiness：<http://localhost:8080/internal/v1/health/ready>
- Admin public config：<http://localhost:4173/admin-api/v1/auth/public-config>

## akone 单二进制安装

正式制品统一使用 `akone`。可通过 Shell、npm 或 Homebrew 安装：

```bash
# macOS / Linux
curl -fsSL https://github.com/Payhon/AppKernia/releases/latest/download/install.sh | sh

# macOS / Linux / Windows
npm install --global @appkernia/akone

# macOS（稳定版开放后）
brew install payhon/tap/akone
```

仓库内可运行 `make build-akone` 构建带内嵌管理端的 `server/bin/akone`。`akone serve` 默认在二进制同目录创建 `data/appkernia.db` 并以 SQLite 启动；可用 `--sqlite FILE`、`AK_SQLITE_PATH` 或 YAML 覆盖。配置 `AK_DATABASE_URL` 时继续使用完整 PostgreSQL 模式。

安装器会在下载后校验 GitHub Release 中的 SHA-256，默认写入用户目录且不会调用 sudo、修改 shell 配置或自动注册系统服务。运行、YAML、环境变量和 Agent CLI 见 [akone 使用手册](docs/manual/akone.md)，制品、预览版和维护者流程见 [akone 发行手册](docs/manual/akone-release.md)。

## 快速开始：全 Docker 模式

如果只想运行完整系统而不在宿主机安装 Go/Node 依赖：

```bash
cp .env.example .env
make docker-up
make docker-bootstrap-admin
```

打开 <http://localhost:4174>。查看日志或停止服务：

```bash
make docker-logs
make docker-down
```

`AK_ADMIN_PORT`、`AK_ADMIN_ORIGIN` 和 `AK_BOOTSTRAP_*` 可以在 `.env` 中覆盖。修改 Admin 端口时需同步修改 `AK_ADMIN_ORIGIN`。不要将管理员密码、生产密钥或第三方凭据写入 `.env` 或提交到 Git。

## 开发、测试与构建命令

前端脚本均可使用 `pnpm <script>` 或 `npm run <script>`：

| 命令                        | 作用                                                  |
| --------------------------- | ----------------------------------------------------- |
| `pnpm dev`                  | 启动 Admin Vite 开发服务器                            |
| `pnpm dev:docs`             | 启动 Rspress 文档站                                   |
| `pnpm test`                 | 运行 Admin Vitest 测试                                |
| `pnpm test:e2e:admin`       | 运行 Admin 双语关键 E2E/视觉测试                      |
| `pnpm build`                | 构建 Admin 与文档站生产静态资源                       |
| `pnpm check:docs`           | 校验并构建双语文档、死链、锚点、OpenAPI 镜像与 Sitemap |
| `pnpm check`                | 生成契约代码并执行 lint、类型、测试、构建和蓝图校验   |

后端命令集中在 `server/Makefile`：

| 命令 | 作用 |
|---|---|
| `make -C server dev` | 从源码同时运行 API 与 Worker |
| `make -C server dev-api` | 仅运行 API |
| `make -C server dev-worker` | 仅运行 Worker |
| `make -C server test` | 运行全部 Go 测试 |
| `make -C server test-race` | 使用 race detector 运行测试 |
| `make -C server build` | 构建 `server/bin/ak-api`、`ak-worker`、`ak-cli` |
| `make -C server db-setup` | 执行迁移和幂等 Seed |
| `make -C server doctor` | 检查当前 SQLite/PostgreSQL 连通性 |
| `make -C server check` | 检查格式、`go vet` 和测试 |

根目录还有跨项目入口：

```bash
make build       # Backend + Admin + Docs
make test        # Backend + Admin
make check       # 蓝图 + Backend + Admin + Mobile + Docs 静态门禁
```

需要数据库集成测试时，显式提供隔离的测试数据库，禁止指向生产库：

```bash
AK_TEST_DATABASE_URL='postgres://appkernia:appkernia-dev-only@localhost:55432/appkernia_test?sslmode=disable' \
  make -C server test-integration
```

## 代码生成与契约

OpenAPI、权限、迁移与国际化契约必须同步修改。生成文件禁止手改：

```bash
make -C server sqlc-generate
pnpm generate:admin
python3 blueprint/scripts/validate_i18n_contract.py
```

Admin 只调用 `/admin-api/v1`，Mobile 只调用 `/api/v1`。所有 API 请求遵守 `Accept-Language` / `Content-Language` 契约，稳定错误码不随语言变化。

## 仓库结构

```text
apps/ak-admin       React 管理后台
apps/ak-docs        Rspress 开源官网与中英文文档站
apps/ak-mobile      uni-app x 移动端
server              Go API、Worker、CLI
blueprint           架构蓝图、机器可读契约与校验器
design-system       Admin/Mobile 设计系统与 UI 产物
docs                ADR、实施状态与交付报告
```

参与开发前请阅读根目录及所属子项目的 `AGENTS.md`。贡献流程见 [CONTRIBUTING.md](CONTRIBUTING.md)，社区行为约定见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)，漏洞报告方式见 [SECURITY.md](SECURITY.md)。

## License

AppKernia 使用 [MIT License](LICENSE)。
