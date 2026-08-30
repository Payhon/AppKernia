# ui-ux-pro-max output

命令：

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "mobile news gallery preview pinch zoom video portrait mode centered media viewer accessibility" --design-system -p "AppKernia Mobile Viewer Feedback" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "image preview pinch zoom swipe video orientation safe area touch target" --domain ux -n 8
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "media viewer layout responsive gestures" --stack swiftui
```

采纳结果：

- 内容优先的扁平媒体查看器，不增加装饰性阴影和复杂动效。
- 视频不自动播放；操作叠加层保持高对比度。
- 所有触控目标至少 44×44，邻近操作保留间距。
- 图片适配容器，不以固定设备宽度实现。
- 媒体尺寸仅在确有需要时参与响应式计算。

未采纳：检索结果中的红色新闻站配色、网页字体和网页 hover 建议；本项目继续使用 `design-system/MASTER.md` 的 AK 蓝、系统字体与原生触控反馈。
