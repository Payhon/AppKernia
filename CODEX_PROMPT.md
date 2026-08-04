你现在位于 AppKernia 项目根目录。

先完整读取根目录 `AGENTS.md`，再读取 `blueprint/backend`、`blueprint/admin-frontend`、`blueprint/mobile` 中的 `AGENTS.md`、主蓝图、任务依赖图和当前任务相关的 docs/spec/integration。审计现有代码、Git 状态和工具链后，按三个蓝图的任务依赖顺序连续开发真实项目，不要停在计划阶段，也不要逐任务等待确认。

后端、Admin、Mobile 必须以 OpenAPI、权限、数据库契约以及 `blueprint/I18N_CONTRACT.md` 保持一致；默认完整实现 `zh-CN`、`en-US` 两套语言包，所有可见文案使用翻译键，语言切换无需重启或刷新。移动端固定 uni-app x + UTS/UVue + VDOM + uView Ultra，并只通过 AK UI 适配层使用组件。所有 Admin/Mobile UI 任务先运行 `ui-ux-pro-max` 并保存真实产物。缺少第三方凭据时用 Port/Adapter、Feature Flag 和本地 Mock 继续开发。

持续运行蓝图校验、`python3 blueprint/scripts/validate_i18n_contract.py`、静态检查、两种语言的关键 E2E/视觉测试和可用平台的真实构建；Android、iOS、Harmony 未实际执行的结果必须标记 blocked，禁止伪报通过。持续更新 `docs/IMPLEMENTATION_STATUS.md`，完成后生成 `docs/CODEX_DELIVERY_REPORT.md`，列出真实命令、退出码、未完成项和风险。不要自动 push。
