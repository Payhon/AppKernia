# 第三方参考与许可证边界

HotGo 仓库为 `https://github.com/bufanyun/hotgo`，以目标 commit 的 MIT LICENSE 为准。AK Admin 优先独立重写；若 Agent 实际复制/改写其代码或资源，必须记录来源文件与 commit，并保留版权/许可声明。

ui-ux-pro-max 仓库为 `https://github.com/nextlevelbuilder/ui-ux-pro-max-skill`。本包未复制其源码、数据或模板，由负责人按官方 CLI 安装。Phase 0 还应生成前端依赖 SBOM 与许可证清单。

Admin 在线 OpenAPI 页面使用精确版本 `@scalar/api-reference@1.64.1`，仅将其构建产物和传递依赖随 Admin 镜像自托管，不加载 Scalar CDN、远程字体、插件 URL 或代理。Scalar 采用 MIT License，完整声明保存在仓库根 `THIRD_PARTY_NOTICES.md`；版本更新必须同步锁文件、许可证记录、独立文档包预算和无外部资源请求验收。

Admin 文章正文编辑使用精确版本 `@uiw/react-md-editor@4.0.4`，从 `@uiw/react-md-editor/nohighlight` 入口按需加载并由应用提供受控 Markdown、媒体选择和安全预览。该组件采用 MIT License，依赖版本、锁文件和仓库根 `THIRD_PARTY_NOTICES.md` 必须同步更新；不得以第三方编辑器能力绕过 Markdown 协议、媒体关系或 URL 安全校验。
