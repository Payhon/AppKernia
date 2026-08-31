# ui-ux-pro-max output

执行：

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "mobile news bookmarks category tabs filter selected state accessibility" --design-system -p "AppKernia Bookmarks Filter"
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "tabs filter selected state content update loading empty accessibility mobile" --domain ux -n 10
```

与本任务直接相关的建议：

- 当前区段必须有可见的 active state，且不只依赖颜色。
- 异步切换需要 loading 反馈，不能在请求期间保留无反馈的旧内容。
- 每个筛选需要独立 empty state。
- 交互控件应提供屏幕阅读器可理解的语义。

通用搜索返回的红色新闻站配色与 AppKernia 既有设计系统冲突，因此保留 MASTER 中的品牌蓝、系统字体和卡片布局。
