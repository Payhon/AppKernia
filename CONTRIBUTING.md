# Contributing to AppKernia

感谢参与 AppKernia。提交修改前，请先阅读根目录 `AGENTS.md` 和所改子项目的 `AGENTS.md`；跨端修改需要同时核对 Backend、Admin、Mobile 三份蓝图。

参与社区即表示同意遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。完整的贡献向导也会发布在 [AppKernia 文档站](https://appkernia.com/community/contributing)。不确定从哪里开始时，修正文档、补充复现步骤、增加测试和回答 Issue 都是有价值的贡献。

## 本地准备

```bash
cp .env.example .env
make setup
```

推荐使用 pnpm 11 和仓库提交的 `pnpm-lock.yaml`。不要提交 `.env`、密码、Token、私钥、证书或第三方凭据。

## 修改契约

OpenAPI 是前后端 API 的最终事实源。接口变更必须同步 Go route/application/repository、OpenAPI、迁移/sqlc、权限 Seed、审计事件、生成 Client 和测试。Admin 只能访问 `/admin-api/v1`，Mobile 只能访问 `/api/v1`。

所有可见文案使用翻译键，并完整维护 `zh-CN`、`en-US`。两份语言包的 key 和占位符必须一致。

## 提交前检查

```bash
make check
```

若只修改单一子项目，可先运行：

```bash
make -C server check
pnpm check
pnpm check:docs
```

Admin 或 Mobile 的可视 UI 修改必须先运行项目内 `ui-ux-pro-max`，保存真实 request、output、decision、review checklist 与截图，再实现页面。移动业务页面只能使用 `ak-*` UI 适配组件。

## Pull Request

PR 描述应包含变更目的、契约或数据库影响、实际执行的命令与结果、未验证平台，以及已知风险。Android、iOS、HarmonyOS 未实际构建或真机执行时，应明确标为未验证或 blocked。
