---
title: 安全模型
description: AppKernia 默认拒绝的风险与必须遵守的安全边界。
---

# 安全模型

AppKernia 的安全默认值不是可选示例：

- 不提交 Secret、Token、密码、OTP、MFA Secret、私钥或真实第三方凭据。
- 不关闭 TLS 校验；生产 Mobile API 不接受 HTTP。
- Refresh Token 服务端只存 Hash，Mobile 只存系统安全存储，Admin 使用 Secure HttpOnly Cookie。
- 文件对象 Key 由服务端随机生成；下载重新校验租户与引用权限。
- V1 不提供任意远程 URL 抓取，避免 SSRF。
- 定时任务只运行编译期注册的 Handler，不执行数据库中的 Shell/SQL/源码。
- 模块是编译期目录，不支持上传、安装或执行未知二进制插件。
- 审计与日志必须字段级脱敏，不能记录 Authorization、Cookie 或预签名 URL。

## 漏洞报告

请使用 GitHub Private Vulnerability Reporting / Security Advisory，不要在公开 Issue 中发布利用细节或真实数据。详见[安全策略](../community/security)。
