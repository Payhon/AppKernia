# uni-app x AI Rules and MCP source

The mobile subproject uses DCloud's official Codex rules and MCP configuration:

- Documentation: <https://doc.dcloud.net.cn/uni-app-x/tutorial/rules_mcp.html>
- Rules repository: <https://gitcode.com/dcloud/uni-app-x-ai-rules>
- Imported commit: `9ec6ebb2ba57c3634a7be454f2d7c21a02635759`
- Imported on: 2026-08-05
- Rules target: `apps/ak-mobile/AGENTS.md`
- MCP package: `@dcloudio/uni-app-x-mcp` (verified package version `0.0.5`)
- License copy: `apps/ak-mobile/docs/licenses/uni-app-x-ai-rules-LICENSE`

The AppKernia root `AGENTS.md` remains higher priority. The imported DCloud rules add UTS, UVue, UCSS, conditional-compilation, and platform compatibility constraints for the mobile subtree.

The MCP tool accepts an explicit `projectPath`. For this worktree, use:

```text
/Users/payhon/project/AppKernia-mobile-framework/apps/ak-mobile
```

Restarting or reopening Codex from the repository root is required for a newly added project-scoped MCP server to appear in the native tool list. The server can also be verified directly over stdio without restarting the current task.
