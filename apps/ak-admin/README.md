# AppKernia Admin

React + TypeScript strict + Vite 管理平台。OpenAPI 类型从 `server/openapi/openapi.yaml` 生成，禁止手改；认证只在内存保存 Access Token，并通过 HttpOnly Cookie 刷新。

在仓库根目录执行：

```bash
corepack enable
pnpm install --frozen-lockfile
pnpm dev
```

Vite 默认监听 <http://localhost:4173>，并将 `/admin-api` 代理到 <http://127.0.0.1:8080>。也可用 npm 调用相同脚本：`npm run dev`、`npm test`、`npm run build`。

常用命令：

```bash
pnpm generate:admin
pnpm lint:admin
pnpm typecheck:admin
pnpm test
pnpm build
pnpm check
```

可视 UI 创建或修改前，必须先运行项目内 `ui-ux-pro-max` 并保存真实产物；所有用户可见文案使用翻译键，同时维护 `zh-CN` 和 `en-US`。
