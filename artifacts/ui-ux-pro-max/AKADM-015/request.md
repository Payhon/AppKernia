# AKADM-015 Request

在 Master 与登录/App Shell override 约束下实现 `zh-CN`、`en-US` 运行时切换，覆盖表单长文本、窄屏、键盘焦点和 reduced-motion。

## 2026-08-03 服务端持久化增量

为已登录用户的语言切换增加服务端偏好保存。请求专项复核异步保存、失败恢复、可访问错误播报和 React 状态处理；匿名选择仍只保存非敏感本地偏好。
