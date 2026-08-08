---
title: 贡献指南
description: 从 Issue 到 Pull Request 的 AppKernia 贡献流程。
---

# 贡献指南

感谢你愿意把时间留给 AppKernia。我们最重视可复核、范围清晰、能够让下一位维护者继续工作的贡献。

## 适合第一次贡献的方向

- 修正文档中不准确、过时或不够清楚的步骤。
- 为 `zh-CN` / `en-US` 补齐等价表达和长文本布局。
- 增加缺失的单元、契约或无障碍测试。
- 在 Android、iOS、HarmonyOS 上复验已有场景并提交完整环境证据。
- 提供最小复现 Issue，而不是只说“不能用”。

## 开始

1. Fork [Payhon/AppKernia](https://github.com/Payhon/AppKernia)。
2. 从自己的 Fork 创建分支。
3. 阅读根 `AGENTS.md` 和所改子项目的 `AGENTS.md`。
4. 运行[源码开发模式](../guide/source-development)并复现问题。
5. 做最小、完整的修改，不顺带重写无关模块。

## 契约同步

API 改动必须同步 Go Route/Application/Repository、OpenAPI、Migration/sqlc（如涉及）、权限 Seed、审计、安全事件、Admin/Mobile 生成客户端与测试。

所有可见文案使用翻译键，并完整维护 `zh-CN` 与 `en-US`。Mobile 业务页面只使用 `ak-*`。

## Pull Request 应包含

- 为什么改、改了什么。
- API / 数据库 / 权限 / 安全影响。
- 实际执行的命令、退出码和测试数量。
- 截图或设备证据索引（UI 变更）。
- 未验证平台、已知风险和回滚方式。

```bash
make check
```

如果无法运行某个平台，请诚实写为“未验证”或“blocked”，不要把静态检查写成真机通过。
