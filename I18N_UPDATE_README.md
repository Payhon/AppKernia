# AppKernia i18n 蓝图增量包

将本 ZIP 解压到项目根目录并允许覆盖同名蓝图文档。它不会包含或覆盖业务源码，只更新根规则、Codex 提示词和三个蓝图的 i18n 约束，并新增统一契约、参考语言包和校验脚本。

新增核心要求：

- 首发完整支持 `zh-CN`、`en-US`。
- 默认/最终回退为 `zh-CN`。
- Backend、Admin、Mobile 使用同一语言代码和解析链。
- Admin/Mobile 运行时切换无需刷新/重启。
- API 支持 `Accept-Language`/`Content-Language`。
- 两套语言包 key 和占位符强一致。
- i18n 从 P0 开始，不再推迟到收尾阶段。

解压后运行：

```bash
python3 blueprint/scripts/validate_i18n_contract.py
```
