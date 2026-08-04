# Backend 国际化架构

## 1. 组件

```text
HTTP Middleware
  → LocaleResolver
  → RequestLocale Context
  → Catalog/Translator
  → Application Error Presenter / Notification Renderer
```

推荐目录：

```text
server/internal/shared/i18n/
├── locale.go
├── resolver.go
├── middleware.go
├── catalog.go
├── formatter.go
├── errors.go
├── locales/
│   ├── zh-CN.json
│   └── en-US.json
└── tests/
```

LocaleResolver 使用 BCP 47 匹配。实现库必须被内部接口隔离，业务模块只依赖 `Translator`，不得在各模块重复解析 Header。

## 2. 解析与响应

匿名请求解析 `Accept-Language`；登录请求优先 `iam.users.locale`。规范化后写入 Request Context，所有 Application Use Case 可以读取，但领域状态机不得因语言不同产生不同业务结果。

响应规则：

- `Content-Language: zh-CN|en-US`
- 本地化且可缓存的公共接口：`Vary: Accept-Language`
- OpenAPI 声明 `SupportedLocale = zh-CN | en-US`
- Auth Context/Public Config 返回支持语言和默认语言

## 3. 错误

领域层返回稳定错误 code 和参数；Transport Presenter 才进行翻译：

```go
DomainError{Code: "IAM.AUTH.INVALID_CREDENTIALS", Params: map[string]any{}}
```

客户端显示时优先本地语言包，以便离线和即时切换；Server message 是兼容旧客户端和未命中 key 的安全回退。

## 4. 通知与动态内容

模板选择顺序：显式请求 locale → 用户 locale → 租户默认 locale → `zh-CN`。模板变量先通过 Schema 校验，再渲染。通知 Job Payload 保存 template code、locale、变量和版本，禁止只保存已经翻译完成且不可重放的字符串。

字典 API 接受/推导 locale，优先返回目标语言；缺失时返回 `zh-CN`。内置菜单返回 `i18n_key` 和 fallback `title`。法律文档必须按 document type + locale + version 管理，用户同意记录保存实际 locale/version/hash。

## 5. 验收测试

- Header alias 规范化。
- q-value 匹配与不支持语言回退。
- 登录用户偏好覆盖 Header。
- 稳定错误 code 在两种语言一致。
- key/占位符一致性。
- 通知模板选择、变量错误和回退。
- 字典、法律文档和菜单语言响应。
- `Content-Language`/`Vary`。
- 审计与日志不存敏感参数，也不依赖翻译文案检索。
