# ui-ux-pro-max 移动端工作流

uView Ultra 提供组件实现，`ui-ux-pro-max` 负责信息层级、交互、视觉系统、可访问性和验收；二者不能互相替代。

## 每个 UI Task 的顺序

1. 读取 Route/Page Spec、API 状态和平台矩阵。
2. 调用并读取 `ui-ux-pro-max` 结果。
3. 更新 `apps/ak-mobile/design-system/MASTER.md` 或页面 override。
4. 将语义 Token 映射到 AK UI，不直接散落到 feature。
5. 编写 UVue 页面。
6. 生成多状态、多尺寸、三端截图。
7. 完成 checklist 和缺陷修复。

## 证据目录

```text
apps/ak-mobile/artifacts/ui-ux-pro-max/<task-id>/
├── request.md
├── skill-output.md
├── decisions.md
├── review-checklist.md
└── screenshots/
    ├── android-360x800-light.png
    ├── ios-390x844-light.png
    ├── harmony-430x932-light.png
    └── dark/  # 仅 AKMOB-160 后
```

## 设计原则

- 触控目标、错误反馈和键盘体验优先于装饰。
- 高风险操作必须明确对象、后果和取消路径。
- 状态不能只靠颜色表达。
- 兼容动态字号和中英文扩展。
- 认证页面不使用诱导授权、默认勾选或隐藏条款。
- 安全区和系统手势区域不可被主要操作遮挡。

Skill 不可用时，UI Task 保持 blocked；禁止人工编造 Skill 输出文件。
