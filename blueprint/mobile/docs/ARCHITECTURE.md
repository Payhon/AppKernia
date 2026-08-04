# AK Mobile 架构设计

## 1. 分层

```text
pages/*.uvue
  ↓
features/<feature>/presentation
  ↓
features/<feature>/application
  ↓
core ports / feature repositories
  ↓
network, secure storage, push, OAuth, native adapters
```

### Presentation

只处理页面生命周期、输入、视图状态和导航，不直接拼 URL、不直接读写 Token、不直接调用平台 SDK。

### Application

实现登录、刷新、资料更新、撤销会话、切换租户等用例。用例决定事务式状态流转和错误映射。

### Infrastructure/Adapter

- `AkHttpClient`：请求、取消、超时、Request ID、错误转换。
- `AkSecureStorage`：三端系统安全存储。
- `AkPush`：不同厂商 Push 的可选 Adapter。
- `AkOAuth`：系统浏览器/SDK 与 PKCE 回调。

## 2. Store 边界

建议 Store：

```text
AppStore       启动阶段、网络状态、前后台、公开配置
SessionStore   内存 Access Token、用户、活动租户、权限快照
PreferenceStore 语言、主题、非敏感设置
NotificationStore 未读计数与短期事件提示
```

列表、详情和 Mutation 状态由 Query Repository 管理，不复制进 SessionStore。

## 3. Query Cache

Query Key 必须包含：

```text
tenantId + resource + parameters + authSubjectVersion
```

切换租户、退出或会话撤销后，删除全部受保护 Cache。敏感 Cache 不持久化。

## 4. 错误模型

统一错误类型至少包括：

- `NetworkUnavailable`
- `Timeout`
- `Cancelled`
- `Unauthorized`
- `Forbidden`
- `Validation`
- `Conflict`
- `RateLimited`
- `ServerError`
- `Maintenance`
- `Unknown`

页面只根据稳定错误类型/业务码渲染，不解析中文 message。

## 5. Feature 模板

```text
src/features/<name>/
├── application/
│   ├── use-cases.uts
│   └── ports.uts
├── domain/
│   ├── models.uts
│   └── errors.uts
├── infrastructure/
│   └── repository.uts
├── presentation/
│   ├── state.uts
│   └── mapper.uts
└── index.uts
```

页面文件只组装 feature 和 AK UI。

## 6. 编译模式

Core 1.0 固定 VDOM/标准模式。Vapor 不作为“优化开关”随意打开，因为它会改变组件、样式隔离、状态库和原生调用边界。升级必须通过 `AKMOB-240`、ADR 与全量三端回归。
