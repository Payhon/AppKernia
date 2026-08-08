---
title: 路线图
description: AppKernia Core 的当前边界与推进方向。
---

# 路线图

AppKernia 正在走向第一个稳定的 Core 版本。路线图表达方向，不代表承诺日期；每项能力只有在代码、契约、测试与对应平台验收齐全后才会标记完成。

## Core 1.0

- 安全认证、Session/Refresh Token 轮换与账号恢复
- 用户、租户、组织、RBAC 与数据范围
- Admin 菜单、配置、字典、文件、通知、任务与审计
- Mobile 登录、资料、安全、消息、设置与三端适配
- `zh-CN` / `en-US` 完整契约
- OpenAPI 生成客户端与 GitHub Actions 质量门禁

## 后续方向

- MFA / OAuth 的生产 Adapter 与更多真实平台验证
- Webhook、API Client、外部通知与对象存储生态
- 可选 Billing，而不污染 Core 身份模型
- 更完善的组件画廊、示例业务与版本化迁移指南

当前最准确的完成状态见仓库中的 `docs/IMPLEMENTATION_STATUS.md` 和 `docs/CODEX_DELIVERY_REPORT.md`。
