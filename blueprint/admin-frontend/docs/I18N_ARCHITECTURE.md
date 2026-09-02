# AK Admin 国际化架构

## 1. 目录

```text
apps/ak-admin/src/
├── shared/i18n/
│   ├── index.ts
│   ├── locale-registry.ts
│   ├── locale-store.ts
│   ├── formatters.ts
│   ├── load-namespace.ts
│   └── types.ts
└── locales/
    ├── zh-CN/{common,auth,navigation,validation,errors,profile,settings,system,notifications,login_providers,...}.json
    └── en-US/{common,auth,navigation,validation,errors,profile,settings,system,notifications,login_providers,...}.json
```

公共首屏 namespace 随主包加载，feature namespace 按路由懒加载。加载完成后原子切换 locale，避免半中半英。

## 2. 适配

统一 `LocaleProvider` 同步：

- i18next language
- Ant Design `ConfigProvider` locale
- Day.js locale
- `document.documentElement.lang/dir`
- document title、Breadcrumb、菜单和工作区标签
- API `Accept-Language`

用户显式选择保存到非敏感本地存储；登录后调用用户偏好 API。服务器偏好与本地冲突时，以最近一次显式选择为准并回写，不能循环覆盖。

## 3. 文案规则

JSX 中只允许产品名、技术 code 和测试 ID 等非翻译内容。字段、按钮、表头、确认文案、Toast、图表、错误和 aria-label 全部使用 key。Menu Seed 的 `i18n_key`、Route Registry 的 `title_key` 必须被两套资源覆盖。

第三方登录的 Provider 名称、平台/构建变体、字段说明、申请步骤、外链标签、失败关闭提示和 409 文案统一归入 `login_providers` namespace；Provider code、Bundle ID、Client ID 等协议标识可保留原文，但不得通过字符串拼接生成面向用户的句子。

日期、数字、百分比、货币和列表格式通过统一 formatter。API 只提供 UTC/原始数值。不要通过字符串拼接生成句子。

## 4. 错误

优先解析：`message_key` → `errors.<normalized error code>` → 后端 `message` → `errors.common.unknown`。Zod 客户端校验也输出 reason key 和参数，不能直接写中文。

## 5. 测试

- key/placeholder parity。
- Namespace 动态加载失败回退。
- 切换后 AntD/Day.js/title/menu/tab 一致。
- 登录前后偏好持久化。
- 两种语言的登录、菜单、CRUD、字段错误、403/404、空状态 E2E。
- 英文长文本、窄屏、表格和弹窗视觉回归。
- HTML lang 和可访问名称正确。
