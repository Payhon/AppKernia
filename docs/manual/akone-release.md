# akone 发行与安装手册

`akone` 是 AppKernia 的单二进制发行名称。GitHub Release 是二进制制品的唯一事实源；Gitee 同步同一个源码 commit 和签名 tag，不重复托管二进制。

## 用户安装

### Shell（macOS / Linux）

一行安装最新稳定版：

```bash
curl -fsSL https://github.com/Payhon/AppKernia/releases/latest/download/install.sh | sh
```

建议先下载再执行，以便检查脚本内容：

```bash
curl -fsSLo install.sh https://github.com/Payhon/AppKernia/releases/latest/download/install.sh
sh install.sh
```

固定版本或安装目录：

```bash
sh install.sh --version 1.0.0-preview.1 --install-dir "$HOME/.local/bin"
sh install.sh --version 1.0.0-preview.1 --prefix "$HOME/.local"
```

安装器支持 macOS/Linux 的 amd64、arm64。它从同一 Release 下载 `checksums.txt` 和平台归档，验证 SHA-256、归档版本及二进制内置版本一致后原子替换 `akone`；不会 sudo、修改 PATH 或启动服务。

### npm（macOS / Linux / Windows）

```bash
npm install --global @appkernia/akone
akone version --json
```

`@appkernia/akone` 通过精确版本的 optional dependency 选择五个平台包：macOS amd64/arm64、Linux amd64/arm64、Windows amd64。安装期间不执行联网下载脚本，`--ignore-scripts`、npm mirror 和离线缓存仍可工作。

预览版使用 `preview` dist-tag：

```bash
npm install --global @appkernia/akone@preview
```

### Homebrew（macOS）

稳定版开放后使用：

```bash
brew install payhon/tap/akone
```

首发不承诺进入 Homebrew Core，因此不承诺无 tap 的 `brew install akone`。

## 维护者发布

两个入口调用同一个 Node 标准库编排器。默认只做完整预检，不创建 tag、不推送、不发布：

```bash
make release VERSION=1.0.0-preview.1
npm run release -- --version 1.0.0-preview.1
```

确认所有输出后，显式启用发布：

```bash
make release VERSION=1.0.0-preview.1 PUBLISH=1
npm run release -- --version 1.0.0-preview.1 --publish
```

编排器会执行以下不可跳过的检查：

1. 版本符合发布 SemVer（不允许 build metadata），工作树干净且当前分支为 `main`。
2. `.env`、`.secrets` 未被 Git 跟踪。
3. fetch 后本地 HEAD、`gitee/main`、`origin/main` 完全相同。
4. 本地及两个远端 tag 不冲突；一致的签名 tag 可用于中断后的安全重试。
5. `make check`、发行单测、Admin/runtime assets staging、GoReleaser v2.17.1 配置检查及五平台 snapshot build 通过。
6. `--publish` 要求本地已配置 `user.signingkey`，创建并验证 annotated signed tag。
7. 先推送并回读验证 Gitee tag，再推送并回读验证 GitHub tag。

GitHub tag 会触发 `.github/workflows/release.yml`：重新运行质量门禁，构建一次 Admin，排除 sourcemap 与 `.vite` 元数据，以 `adminembed` tag 构建五个平台二进制，生成压缩包、SHA-256、SBOM 和 GitHub artifact attestation，并先创建 Draft Release。Draft 完成后发布为 Preview，再调用可单独重跑的渠道 workflow。

渠道 workflow 会：

1. 重新下载并校验全部 GitHub Release 资产。
2. 生成五个 npm 平台包，执行 `npm pack --dry-run` 和 Linux 二进制 smoke。
3. 先发布平台包，最后发布 meta 包；已存在的同版本包只有在 registry `dist.integrity` 与本次本地打包内容一致时才会跳过，否则立即停止发布。
4. 稳定版才生成并更新 `Payhon/homebrew-tap` 的 `Formula/akone.rb`。

## 首次配置

在 GitHub 仓库创建受保护 Environment `akone-release`。npm 的六个公开包都应启用 Trusted Publishing/OIDC：正常 tag 流程登记调用方 `release.yml`；若允许人工重跑渠道 workflow，还需登记 `release-channels.yml`。

npm 不能为尚未存在的包预先配置 Trusted Publisher。第一次发布时在 Environment 中临时放入只允许 `@appkernia/akone*`、短期有效且可发布公开包的 granular token，命名为 `NPM_BOOTSTRAP_TOKEN`；六个包创建成功后立即删除该 secret、逐包启用 Trusted Publishing，并在下一次 Preview 中验证纯 OIDC 发布。不得保留长期 `NPM_TOKEN`。

Homebrew 稳定发布需要只对 `Payhon/homebrew-tap` 有写权限的 fine-grained token，保存为 Environment secret `HOMEBREW_TAP_TOKEN`。

## 稳定版签名门禁

当前流程只允许带 prerelease 后缀的 Preview，例如 `v1.0.0-preview.1`。无后缀稳定 tag 会同时被本地编排器、主发布 workflow 和可人工重跑的渠道 workflow 拒绝；两个 workflow 还会复核签名 tag、检出 commit 与 `origin/main` 的可达关系。

只有在以下原生签名链实现并通过真实安装验收后，才能移除门禁：

- macOS Developer ID 签名及 notarization，覆盖 amd64、arm64。
- Windows amd64 Authenticode 或 Microsoft Trusted Signing。
- 签名后重新归档，再生成最终 checksum、SBOM 与 attestation。

不得通过环境开关绕过该门禁，也不得把未签名的 macOS/Windows 制品发布为 GA。

## 本地验证

```bash
make release-test
sh -n scripts/install-akone.sh
GOTOOLCHAIN=go1.26.5 go run github.com/goreleaser/goreleaser/v2@v2.17.1 check --config .goreleaser.yaml
```

本地检查不会创建 tag 或调用任何外部发布 API。只有 `--publish` 才会推送签名 tag；实际 Release、npm 和 Homebrew 发布均由 GitHub Actions 完成。
