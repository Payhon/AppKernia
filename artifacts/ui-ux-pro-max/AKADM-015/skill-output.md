# AKADM-015 Skill Output

复用真实 Skill 输出 `design-system/appkernia-admin/MASTER.md` 与三个 page override；专项查询要求 label 关联、可见焦点、键盘顺序和 reduced-motion。

## 2026-08-03 实际命令

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise SaaS admin language preference persistence error feedback accessibility" --design-system -p "AppKernia Admin" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "language switch save error optimistic update aria-live" --domain ux -n 8
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "locale switch async persistence rollback accessible feedback" --stack react
```

三条命令退出码均为 0。专项输出的相关结论：

- 异步失败不得静默，必须在控件附近给出明确恢复路径。
- 错误使用 `role="alert"` 或 `aria-live` 让辅助技术播报。
- React 异步事件必须捕获异常，避免未处理 Promise rejection。
- 测试优先使用可访问角色和 label 查询。
- 保持 375、768、1024、1440 四个视口、可见焦点和 reduced-motion 验证。

设计系统检索同时返回企业入口、Indigo/Green 配色与 Inter 字体建议；本增量不改变已批准 Master token，避免仅为错误反馈引入无关视觉漂移。
