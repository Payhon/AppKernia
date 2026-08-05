# Decisions

- 使用既有 near-black 主操作、白灰数据 surface 与系统字体；不引入新的品牌色或外部字体。
- 文章与分类分别由 route permission 阻断，细粒度动作由 `content.*` permission 控制。
- 当前 OpenAPI 尚未进入此 worktree，使用一个局部 typed adapter seam；生成 API 后应替换此 seam，不修改 generated 文件。
- 409 由 UI 告知并精确失效相关 content query，禁止自动覆盖。
