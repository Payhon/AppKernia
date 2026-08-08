---
title: AK Mobile 组件
description: AppKernia 移动端真实可用的 ak-* 组件与使用规则。
---

# AK Mobile 组件

业务页面只能使用 `apps/ak-mobile/components/ak-ui` 暴露的 `ak-*` 组件。AK UI 隔离视觉 Token、平台兼容、触控尺寸、事件类型和底层 uView / uni 原生实现。

```text
Feature Page
    ↓
AK UI (ak-*)
    ↓
uView Ultra (up-*) / uni native / platform adapter
```

<div class="ak-doc-callout"><strong>文档范围</strong>本节只记录当前源码中真实存在的组件。兼容矩阵里尚未实现或仍为 conditional 的 AkPicker、AkDatePicker、AkTabs 等，不会在这里写成可直接使用。</div>

## 当前实现

| 组件                                | 用途                               | 状态   |
| ----------------------------------- | ---------------------------------- | ------ |
| [`ak-button`](./button)             | 主、次、危险操作与加载态           | 已实现 |
| [`ak-text-field`](./text-field)     | 标签、输入、密码、禁用与错误       | 已实现 |
| [`ak-card`](./layout-status)        | 通用内容容器                       | 已实现 |
| [`ak-cell-list`](./layout-status)   | 设置/个人中心列表容器              | 已实现 |
| [`ak-empty-state`](./layout-status) | 空状态与下一步动作                 | 已实现 |
| [`ak-loading`](./layout-status)     | 局部加载提示                       | 已实现 |
| [`ak-status-view`](./layout-status) | Loading / Empty / Error 等状态编排 | 已实现 |
| [`ak-modal`](./modal-switch)        | 高风险确认                         | 已实现 |
| [`ak-switch`](./modal-switch)       | 布尔设置                           | 已实现 |
| [`ak-icon`](./icons-theme)          | 受控语义图标                       | 已实现 |
| [`ak-back-button`](./icons-theme)   | 安全返回与 fallback                | 已实现 |
| [`ak-theme-root`](./icons-theme)    | 主题根容器                         | 已实现 |

## 通用规则

- 所有可见文案由页面通过 `AkI18n` 翻译后传入；组件默认值不硬编码展示语言。
- 触控目标至少 44×44。
- Loading / Disabled 时必须阻止重复提交。
- 状态不能只靠颜色表达。
- 组件源码通过 easycom 以 `ak-*` 使用，业务页面不得直接写 `up-*`。
- 真正的平台可用性以 Android、iOS、HarmonyOS 对应构建和设备证据为准。
