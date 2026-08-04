# AK Mobile 国际化架构

## 1. 目录和接口

```text
apps/ak-mobile/
├── locale/
│   ├── zh-CN.json
│   └── en-US.json
└── src/core/i18n/
    ├── ak-i18n.uts
    ├── locale-registry.uts
    ├── locale-normalizer.uts
    ├── formatter.uts
    ├── navigation-sync.uts
    └── types.uts
```

`AkI18n` 至少提供：`t(key, params)`、`setLocale(locale)`、`getLocale()`、`formatDate`、`formatNumber`、`formatRelativeTime`。业务页面不得直接读取 JSON，也不得直接依赖平台语言 API。

## 2. 启动解析

```text
登录用户服务端偏好
> 用户本地显式选择
> device/browser locale
> zh-CN
```

首次启动匿名状态可先使用本地/设备语言；完成登录后拉取 Auth Context 并同步。规范值只保存 `zh-CN`/`en-US`，`zh-Hans/en` 等只作为输入别名。

## 3. 运行时切换

语言切换事务：

1. 验证并加载目标 Catalog。
2. 检查 key/placeholder manifest。
3. 原子更新 locale store。
4. 调用 `uni.setNavigationBarTitle` 更新当前页。
5. 逐项调用 `uni.setTabBarItem` 更新三 Tab。
6. 通知 AK UI/uView Wrapper 重渲染。
7. 更新 `Accept-Language`。
8. 登录状态异步写入用户偏好；失败显示可重试提示但不破坏当前 Catalog。

`pages.json` 的静态 title/tab text 使用 `zh-CN` 仅作冷启动兜底，不能作为运行时事实源。

## 4. uView Ultra

业务层只用 `ak-*`。所有 uView 内建按钮、加载、空状态或校验文案都由 Wrapper 显式传入本地化字符串，避免组件默认中文泄漏。升级 uView 后，组件兼容矩阵必须在两种语言和三端重新执行。

## 5. 格式化

日期、数字、百分比、相对时间使用 `AkI18n` Formatter。API 始终传 UTC/原始数值。禁止 `value + '条'` 等字符串拼接；使用 `{count}` 参数和完整翻译句子。

## 6. 测试

- 两套 key 和占位符一致。
- 别名规范化与未知语言回退。
- 切换无需重启，当前标题和 TabBar 即时变化。
- 登录前后偏好同步。
- Android/iOS/Harmony 双语登录、主页、消息、个人中心、设置、错误页 smoke。
- 英文长文本、动态字号、窄屏、安全区和键盘布局。
- API Header、错误回退、法律文档 locale/version/hash。
