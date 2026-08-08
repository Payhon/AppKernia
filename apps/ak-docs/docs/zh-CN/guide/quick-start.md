---
title: 从零运行 AppKernia
description: 面向第一次使用者的源码拉取、环境准备、启动与验证步骤。
---

# 从零运行 AppKernia

这条路径只要求你安装 Git 和 Docker Desktop。Go、Node 与 PostgreSQL 都由容器提供，适合第一次体验。

## 1. 安装基础工具

- [Git](https://git-scm.com/downloads)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) 或带 Compose v2 的 Docker Engine

确认 Docker 已启动：

```bash
git --version
docker version
docker compose version
```

三条命令都应正常返回版本号。

## 2. 拉取源码

```bash
git clone https://github.com/Payhon/AppKernia.git
cd AppKernia
```

如果你准备贡献代码，推荐先在 GitHub Fork，再克隆自己的 Fork。

## 3. 创建本地配置

```bash
cp .env.example .env
```

仓库提供的值只用于本机开发。不要把 `.env`、管理员密码、Token、私钥或第三方凭据提交到 Git。

## 4. 启动数据库、迁移、API 与 Admin

```bash
make docker-up
```

第一次运行需要构建镜像，耗时取决于网络和机器性能。命令会按顺序：

1. 启动 PostgreSQL 18。
2. 执行数据库迁移。
3. 幂等写入核心权限、菜单和配置种子。
4. 启动 Go API 与 React Admin 静态站。

## 5. 创建第一个管理员

```bash
make docker-bootstrap-admin
```

按照终端提示输入至少 12 位的密码。密码只从交互终端读取，不会写入 Git 或命令历史。默认开发邮箱来自 `.env` 中的 `AK_BOOTSTRAP_EMAIL`，你可以在执行前改成自己的本地测试邮箱。

## 6. 打开系统

- Admin：<http://localhost:4174>
- API readiness：<http://localhost:8080/internal/v1/health/ready>（源码模式可直接访问；Docker Admin 通过内部 API 容器通信）

如果 Docker 模式没有把 API 端口发布到宿主机，使用下面的命令检查容器健康：

```bash
docker compose ps
docker compose logs api --tail=100
```

看到 `postgres`、`api`、`admin` 为 healthy，且浏览器能打开登录页，就完成了第一次启动。

## 7. 停止系统

```bash
make docker-down
```

数据库数据保存在 Docker volume 中，普通 `docker compose down` 不会删除它。需要重建空数据库时请先备份；不要在不理解影响的情况下删除 volume。

## 下一步

- 需要热更新和断点调试：进入[源码开发模式](./source-development)。
- 需要运行移动端：进入[移动端开发](./mobile-development)。
- 遇到问题：查看[故障排查](./troubleshooting)。
