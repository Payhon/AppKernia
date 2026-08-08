---
title: 核心概念
description: 理解 AppKernia 的架构、认证、权限、多租户、国际化与安全边界。
---

# 核心概念

AppKernia 的价值不只来自技术栈，更来自三端共同遵守的边界。

- [总体架构](./architecture)：Mobile、Admin、API、Worker 与 PostgreSQL 如何协作。
- [认证与会话](./authentication)：Access Token、Refresh Token 轮换与 Audience 隔离。
- [权限与多租户](./permissions-tenancy)：菜单、权限、数据范围和 SQL 过滤的职责。
- [国际化](./internationalization)：`zh-CN` / `en-US` 的统一契约。
- [安全模型](./security)：秘密、上传、日志、重试与扩展的红线。
