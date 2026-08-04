# 路由、深链与授权

## 1. 导航 API

建立 `AkNavigator`，页面不得散落直接调用 `uni.navigateTo`、`redirectTo`、`reLaunch`、`switchTab`。Navigator 根据 Route Registry：

- 校验 route key 和参数。
- 检查 access、permission、feature flag。
- 选择正确导航方法。
- 记录脱敏导航 Telemetry。

## 2. 拦截器

对导航 API 注册全局 interceptor，同时在页面 `onLoad/onShow` 再次校验，避免直接 URL、恢复栈或平台差异绕过。

## 3. Return Target

登录前的 return target 只能保存 Route Key + 已验证参数，不能保存任意 URL。登录成功后仅恢复一次，并清除。

## 4. 深链

- Scheme/Universal Link/App Link/Harmony Link 全部映射到 allowlist。
- OAuth Callback 必须校验 provider、state、PKCE verifier 和时效。
- `javascript:`、`file:`、任意 HTTP 跳转和未知 host 直接拒绝。

## 5. 权限

Auth Context 返回权限快照。页面使用：

```text
canView(route)
can(actionPermission)
```

前端拒绝只是一层 UX；API 返回 403 时必须展示 Forbidden，不得刷新 Token 或自动切账号。
