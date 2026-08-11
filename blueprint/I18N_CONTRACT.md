# AppKernia 多语言统一契约

**版本：** 1.0  
**默认语言：** `zh-CN`（简体中文）  
**首发语言：** `zh-CN`、`en-US`  
**最终回退：** `zh-CN`

本文件同时约束 Go Backend、React Admin 和 uni-app x Mobile。任何一端不得自行发明另一套语言代码、回退顺序或翻译键规则。机器事实源为 `blueprint/i18n-contract.json`。

## 1. 语言代码与别名

内部持久化和 API 只使用 BCP 47 规范代码：

| 规范代码 | 名称 | 接受的常见别名 |
|---|---|---|
| `zh-CN` | 简体中文 | `zh`、`zh_CN`、`zh-Hans`、`zh-SG` |
| `en-US` | English | `en`、`en_US` |

浏览器、Android、iOS、HarmonyOS 或旧客户端传入别名后，必须先规范化再保存。未知语言匹配不到时回退到 `zh-CN`，不得把未知原始字符串直接写入用户偏好。

## 2. 解析优先级

匿名状态：

```text
用户在当前客户端显式选择
> Accept-Language / 浏览器或设备语言
> zh-CN
```

登录状态：

```text
服务端持久化的用户 locale
> 当前客户端显式选择（选择后立即同步服务端）
> Accept-Language / 浏览器或设备语言
> zh-CN
```

所有客户端请求发送 `Accept-Language`；后端返回 `Content-Language`。对会因语言变化而变化的可缓存公共响应，后端必须发送 `Vary: Accept-Language`。

## 3. 资源归属

```text
Backend:
server/internal/shared/i18n/locales/zh-CN.json
server/internal/shared/i18n/locales/en-US.json

Admin:
apps/ak-admin/src/locales/zh-CN/<namespace>.json
apps/ak-admin/src/locales/en-US/<namespace>.json

Mobile:
apps/ak-mobile/locale/zh-CN.json
apps/ak-mobile/locale/en-US.json
```

基础命名空间：`common`、`auth`、`navigation`、`validation`、`errors`、`profile`、`settings`、`system`、`notifications`、`content`、`mobile_releases`、`apps`、`openapi`、`api_reference`。其中 `api_reference` 只由独立 OpenAPI 文档入口加载，不得进入 Admin 主 SPA；业务模块使用自己的 namespace，不把所有字符串堆到 `common`。

## 4. 翻译键

- 使用小写、点分隔的语义键，例如 `auth.login.title`、`common.actions.save`。
- 禁止把中文或英文原文直接作为 key。
- 变量使用具名占位符，例如 `{name}`；两种语言的占位符集合必须完全相同。
- Core 共享消息优先使用简单具名占位符。复数和复杂语法必须由各端适配层测试通过，不能依赖某一端独有语法。
- 产品名 `AppKernia`、简称 `AK`、协议错误码、日志字段名等技术标识可以不翻译。

## 5. API 与错误

后端业务判断只能依赖稳定错误码，不能依赖翻译文案。推荐错误结构：

```json
{
  "error": {
    "code": "IAM.AUTH.INVALID_CREDENTIALS",
    "message_key": "errors.iam.auth.invalid_credentials",
    "message": "账号或密码错误",
    "details": {}
  },
  "request_id": "..."
}
```

Admin/Mobile 的显示优先级：

```text
本地 message_key / error.code 翻译
> 后端本次响应 message
> errors.common.unknown
```

表单字段错误的 `field`、`reason` 和参数必须结构化；前端本地化字段标签和原因。后端日志、审计与事件应保存错误码和参数，不只保存某一种语言的成品句子。

## 6. 数据与动态内容

- 时间在 API/数据库始终使用 UTC/RFC 3339，客户端按当前 locale/time zone 展示。
- 金额和数字以原始数值传输，客户端格式化；禁止后端返回带语言和千分位的业务数值字符串。
- `iam.users.locale` 保存规范语言代码。
- 字典项使用 `sys.dict_items.locale`；通知模板使用 `notify.templates.locale`。
- 内置菜单使用稳定 `i18n_key`（通常由 menu code 生成），数据库 `title` 仅是 `zh-CN` 回退文本。
- 通知模板按 `请求语言 → 用户语言 → 租户默认 → zh-CN` 选择；缺少目标语言模板时必须可观测地回退。
- 以后需要本地化的业务内容应使用显式 translation table 或受约束结构，不得把任意多语言对象无规范地塞进 JSONB。

## 7. 语言切换

- Admin 切换语言无需刷新页面；同步 Ant Design、Day.js、图表、页面标题和 HTML `lang`。
- Mobile 切换语言无需重启 App；同步导航标题、TabBar、uView Ultra 包装组件、日期数字和所有可见页面。
- 已登录用户切换语言时写入用户偏好；匿名用户只持久化非敏感本地偏好。
- 切换失败不得让 UI 处于半中半英状态；资源加载必须原子提交或回滚。

## 8. 发布门禁

每次发布必须满足：

1. `zh-CN` 与 `en-US` key 集合完全一致。
2. 两个语言包具名占位符完全一致。
3. 所有用户可见字符串均通过翻译键输出；测试 Fixture、开发画廊除外，但也应优先走语言包。
4. Admin 与 Mobile 均有两种语言的关键 E2E 和视觉截图。
5. 英文文本扩展、换行、按钮宽度、表格列、表单错误、空状态和弹窗均无截断。
6. 后端 Accept-Language、用户偏好、回退、Content-Language、通知模板选择均有自动化测试。
7. 翻译包不得包含 Secret、个人数据或生产凭据。
