# AppKernia Mobile Blueprint

这是 AppKernia（AK）移动端的可执行开发蓝图，目标代码目录为 `apps/ak-mobile`。

## 阅读顺序

1. `AGENTS.md`
2. `AK_MOBILE_BLUEPRINT.md`
3. `docs/ARCHITECTURE.md`
4. `docs/UVIEW_ULTRA_INTEGRATION.md`
5. `docs/INFORMATION_ARCHITECTURE.md`
6. `docs/PAGE_SPECIFICATIONS.md`
7. `docs/API_STATE_SECURITY.md`
8. `docs/PLATFORM_COMPATIBILITY.md`
9. `docs/PRIVACY_PERMISSION_MATRIX.md`
10. `docs/UI_UX_PRO_MAX_WORKFLOW.md`
11. `docs/TESTING_QUALITY_GATES.md`
12. `docs/AGENT_TASK_BACKLOG.md`
13. `spec/*.json` 与 `integration/*.json`

## 关键结论

- 核心发布平台：Android、iOS、HarmonyOS NEXT。
- UI：uView Ultra 4.5.18，但业务页面只能使用 `ak-*` 适配组件。
- 默认渲染：uni-app x VDOM/标准模式；Vapor 只做独立可行性 Spike。
- 状态：类型化 Reactive Store；不在 Android VDOM 上强制引入第三方 Pinia 实现。
- i18n：AK 自有适配层，首发 `zh-Hans` 与 `en`。
- Token：Access Token 仅内存；Refresh Token 进入三端系统安全存储 UTS 插件。
- API：以 Backend OpenAPI 3.1 为唯一契约，通过生成器产出 UTS DTO 和 Endpoint 描述。
- 可视 UI：必须先运行 `ui-ux-pro-max`，再写页面代码。

执行前先运行：

```bash
python3 blueprint/mobile/scripts/validate_blueprint_specs.py
```
