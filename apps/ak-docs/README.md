# AppKernia Docs

AppKernia 的开源官网与文档站，基于 [Rspress](https://rspress.rs) 构建。

```bash
pnpm --filter @appkernia/docs dev
pnpm --filter @appkernia/docs check
```

`check` 会校验手写 API 路径与服务端 OpenAPI 契约、双语目录一致性、lint、TypeScript、格式、死链/锚点、静态构建和 Sitemap。

生产构建输出到 `apps/ak-docs/doc_build`。正式站点部署到 GitHub Pages，并绑定
[`appkernia.com`](https://appkernia.com)。

文档内容同时维护 `zh-CN` 与 `en-US`。服务端接口以
`server/openapi/openapi.yaml` 为最终事实源；移动组件文档必须与
`apps/ak-mobile/components/ak-ui` 的真实实现保持一致。
