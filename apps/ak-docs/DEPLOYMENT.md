# GitHub Pages deployment

`.github/workflows/docs-pages.yml` builds `@appkernia/docs` on pushes to `main` that affect the docs site, the server OpenAPI contract, or the lockfile. It uploads `apps/ak-docs/doc_build` as the Pages artifact and deploys through the protected `github-pages` environment.

The workflow reads `origin` and `base_path` from `actions/configure-pages`. Before a custom domain is active it builds assets for `/AppKernia/`; after `appkernia.com` is active it builds for `/`. This keeps both the default project Pages URL and the custom-domain root valid without hard-coding one deployment layout.

Repository setup that a maintainer must complete once:

1. Open **Settings → Pages** and select **GitHub Actions** as the source.
2. In **Custom domain**, enter `appkernia.com` before changing DNS.
3. Configure the registrar DNS using GitHub's current apex-domain instructions, then verify the domain and enable **Enforce HTTPS**.
4. Keep the default Pages URL `https://payhon.github.io/AppKernia/` available during DNS propagation. Once GitHub activates the custom domain, it may redirect to `https://appkernia.com`.

GitHub ignores `CNAME` files when a custom Actions workflow publishes Pages, so the domain is configured in repository settings rather than generated into the artifact.

Local production check:

```bash
pnpm --filter @appkernia/docs check
pnpm --filter @appkernia/docs preview
```

The workflow can be validated locally, but the first real publication and DNS/HTTPS result remain unverified until the repository Pages source and domain records are configured.
