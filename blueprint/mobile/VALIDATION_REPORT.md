# Validation Report

验证日期：2026-08-02

## 集成一致性校验

```text
AppKernia Mobile Blueprint Validation
Routes: 35
Tabs: 3
Baseline APIs: 26
API delta: 28
Permission delta: 9
Components: 33
Privacy capabilities: 11
Tasks: 25
Platforms: 3
PASSED: 0 errors, 0 warning(s)
```

## 已检查

- 全部 JSON 使用标准解析器加载。
- Route Key、Path、File 唯一。
- 三个 Tab 与静态 Route Registry 双向一致。
- 页面 API 均属于 Backend 基线或 Mobile API Delta。
- 页面权限均属于 Backend Core Permission 或 Mobile Permission Delta。
- Feature Flag 引用完整。
- uView/AK UI 组件准入状态和三端字段完整。
- 11 项隐私能力均有 Android、iOS、Harmony 处理策略。
- Android、iOS、Harmony 平台矩阵完整。
- 25 个 Agent Task 的依赖图无环。
- 必需文档、Root AGENTS 和 Codex Prompt 存在。
- ZIP 可独立解压，并可在包含 `blueprint/backend` 的模拟项目根目录再次通过校验。

## 边界

该报告仅证明蓝图和机器契约的静态一致性，不代表实际 uni-app x 项目、Android、iOS 或 Harmony 构建已执行。真实构建、真机 Smoke 和 UI 截图必须由 Coding Agent 按 `AKMOB-200`～`AKMOB-230` 完成；未执行的平台必须标记 `blocked`，不能标记 `passed`。
