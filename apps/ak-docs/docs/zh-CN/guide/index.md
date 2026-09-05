---
title: 开始使用
description: 选择最适合你的 AppKernia 启动路径。
---

# 开始使用

第一次接触 AppKernia，不需要先理解所有蓝图。你可以先把完整系统跑起来，再按需要深入 Mobile、Admin 或 Server。

如果你还不确定它是否适合自己的项目，先读[什么是 AppKernia？](./what-is-appkernia)：那里介绍了项目为什么出现、名字的意义、技术选择、真实产品画面与未来方向。

| 你的目标                       | 推荐入口                                    | 大约需要                          |
| ------------------------------ | ------------------------------------------- | --------------------------------- |
| 体验 akone 单二进制与 SQLite   | [akone 安装与部署](./akone)                 | 当前从源码构建；公开发行后可安装  |
| 通过源码看到完整系统           | [Docker 一键运行](./quick-start)            | Docker Desktop、Git               |
| 准备修改 Go 或 React 源码      | [源码开发模式](./source-development)        | Go 1.26、Node 24、pnpm 11、Docker |
| 开发 Android / iOS / HarmonyOS | [移动端开发](./mobile-development)          | HBuilderX 与目标平台工具链        |
| 构建自定义基座或正式发布包     | [移动端打包](./mobile-packaging)            | HBuilderX、DevEco 与平台签名      |
| 配置多厂商离线推送             | [推送渠道配置](./push-channels)             | 厂商账号、App 身份与凭据          |
| 观察消息队列并处理失败         | [消息运营工作台](./notification-operations) | Admin 通知运营权限                |
| 接入或检查 App 系统权限        | [移动端权限中心](./mobile-permissions)      | 目标平台工具链                    |
| 配置应用分享和扫码域名         | [客户端配置](./client-configuration)        | Admin 客户端配置读取权限          |
| 先理解整个项目                 | [仓库结构](./project-structure)             | 5–10 分钟                         |
| 先判断项目是否适合自己         | [什么是 AppKernia？](./what-is-appkernia)   | 8–12 分钟                         |

<div class="ak-doc-callout"><strong>当前阶段</strong>AppKernia 尚未发布公开的 akone Release、npm 包或 Homebrew Formula，也未发布稳定版本。接口、迁移和组件 API 仍可能调整；用于生产前，请完成与你的部署环境和目标设备一致的安全、性能与平台验收。</div>

## 第一次运行的成功判定

不要只以“命令没有报错”作为成功。下面是 Docker 源码路径的判定；单二进制路径请按 [akone 安装与部署](./akone)验证版本、Readiness、登录、数据路径和备份。完整的 Docker 本机启动至少应满足：

1. `docker compose ps` 中 PostgreSQL、API、Admin 的健康检查通过，Migration 与 Seed 以 `0` 退出。
2. `http://localhost:4173/healthz` 返回成功，Admin 登录页可以加载且浏览器控制台没有阻断性错误。
3. 使用你自己通过 `bootstrap-admin` 创建的本机账号登录；仓库不会提供固定默认密码。
4. Dashboard 的摘要和趋势请求完成，而不是一直停留在 Skeleton 或错误态。
5. 只有在目标平台真实编译、安装或运行后，才把 Mobile 对应平台记为通过。

如果任何一项不满足，先进入[故障排查](./troubleshooting)，保留首个失败命令和退出码，不要从后续连锁错误反推原因。

## 运行之后怎么继续

| 你准备做的事               | 下一步                                                                                             |
| -------------------------- | -------------------------------------------------------------------------------------------------- |
| 新增业务 API               | 先读[服务端约定](../api/conventions)与 OpenAPI 事实源                                              |
| 新增后台页面               | 先确认权限、菜单、API Client 和双语键                                                              |
| 新增 App 页面或组件        | 进入[移动端开发](./mobile-development)与 AK UI 文档                                                |
| 配置分享绑定或扫码网页     | 使用[客户端配置](./client-configuration)，再阅读[扫码能力](../mobile-components/scanner)           |
| 接入业务消息与离线 Push    | 阅读[消息推送架构](../concepts/notification-architecture)和[通知 API](../api/mobile-notifications) |
| 配置、预检和测试厂商渠道   | 按[推送渠道配置](./push-channels)准备账号和构建身份                                                |
| 排查积压或失败投递         | 使用[消息运营工作台](./notification-operations)                                                    |
| 打包自定义基座或正式版 App | 按[移动端打包](./mobile-packaging)配置工具链与签名                                                 |
| 调整认证、权限或多租户边界 | 先读[认证](../concepts/authentication)与权限职责                                                   |
| 准备提交改动               | 按[贡献指南](../community/contributing)完成验证与证据                                              |

## 推荐阅读顺序

1. [安装与部署 akone](./akone)
2. [从源码仓库运行](./quick-start)
3. [什么是 AppKernia？](./what-is-appkernia)
4. [总体架构](../concepts/architecture)
5. [认证与会话](../concepts/authentication)
6. [消息推送架构](../concepts/notification-architecture)
7. [服务端 API](../api/)
8. [移动组件](../mobile-components/)
9. [移动端打包](./mobile-packaging)
10. [参与贡献](../community/contributing)
