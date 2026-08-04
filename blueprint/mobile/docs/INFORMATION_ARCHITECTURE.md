# 移动端信息架构

## 1. 启动流

```text
App Launch
  → Bootstrap
  → 读取随包法律快照、本地隐私版本和非敏感偏好
  → 未同意当前版本：使用随包快照展示 Privacy Consent（不初始化第三方 SDK）
  → 同意后获取 public config + app version + 最新法律版本
  → 尝试从 Secure Storage 读取 Refresh Token
  → Refresh 成功：Auth Context → Home
  → Refresh 失败/无 Token：Login
```

不得为了“秒开”先进入受保护页面再闪退到登录页。首次同意前应能离线阅读随包法律快照，避免为了展示隐私政策先初始化网络遥测或第三方 SDK。

## 2. 主导航

```text
首页 Home
消息 Notifications
我的 Profile
```

TabBar 静态注册。消息 Badge 来自服务端未读数；Push/WebSocket 只触发重新获取。

## 3. 认证导航

```text
密码登录
验证码登录（Flag）
注册（Flag）
忘记密码
重置密码
邮箱/手机号验证
MFA Challenge
OAuth Callback（Flag）
```

## 4. 我的

```text
基本资料
编辑资料/头像
安全中心
  ├── 修改密码
  ├── 登录会话
  ├── 登录设备
  ├── MFA
  └── 第三方绑定
设置
  ├── 语言
  ├── 主题（Flag）
  └── 通知偏好（Flag）
切换租户（Flag）
隐私政策
用户协议
关于
账号注销（Flag）
退出登录
```

## 5. Route Registry 原则

- 路由全部静态编译。
- 参数在 Registry 声明并验证。
- 外部深链仅映射到 allowlist route key。
- 未登录访问 authenticated 页面：保存一次安全 return target 后去登录。
- guest 页面在已登录时重定向 Home。
- feature flag 关闭时展示明确不可用状态，不猜测远程页面。

完整路径见 `spec/mobile-route-registry.json`。
