# 测试与质量门禁

## 测试层

- Unit：Zod、formatter、permission、menu resolver、query key、Search Params。
- Component：Testing Library + MSW 覆盖 loading/empty/error/403、action permission、表单错误映射。
- Contract：OpenAPI 生成无漂移；页面 API 均存在于后端快照或 delta。
- E2E：登录/刷新、用户 CRUD、角色授权、租户切换、强制下线、Secret 一次性展示、文件上传、定时任务。
- Visual/a11y：1440 light/dark、768，Playwright screenshot + axe；键盘、焦点、reduced motion。

## 合并门禁

```bash
pnpm lint
pnpm typecheck
pnpm test --run
pnpm test:e2e
pnpm build
python3 scripts/validate_blueprint_specs.py
```

建议 bundle budget：首屏业务 JS gzip ≤ 300KB（不含按需图表），单页面 chunk gzip ≤ 180KB；以真实 CI 基线调整。严重/关键 axe 问题为 0。不得仅报告“理论上通过”。

## 多语言质量门禁

```text
zh-CN/en-US key 集合一致
具名占位符集合一致
Route Registry title_key 全部存在
Menu Seed i18n_key 全部存在
无未允许的用户可见硬编码字符串
两种语言关键 Playwright E2E
两种语言视觉截图与 axe
英文长文本不截断
```

语言切换测试必须覆盖：无刷新切换、Ant Design/Day.js、document title、已打开标签、表单错误、服务端 error code 回退、持久化用户偏好和重新登录恢复。
