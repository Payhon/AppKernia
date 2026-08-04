# ui-ux-pro-max 强制工作流

## 1. 适用范围

新页面、Dashboard、表格、表单、Drawer、Modal、登录页，以及布局、主题、色彩、字号、间距、图表、动效、响应式、可访问性、暗色模式和 UI Review，必须使用 `ui-ux-pro-max`。纯 API 类型、Query、权限函数和测试数据无需重复生成设计系统，但必须遵守已有 Master。

## 2. 安装（由项目初始化负责人批准执行）

```bash
npm install -g ui-ux-pro-max-cli
uipro init --ai codex
```

不全局安装可使用：

```bash
npx ui-ux-pro-max-cli init --ai codex
```

Codex 项目路径优先为 `.codex/skills/ui-ux-pro-max/`；旧版初始化器也可能使用
`.agents/skills/ui-ux-pro-max/`。检查脚本必须识别两者。Agent 不得在未获授权时擅自安装宿主机软件。

## 3. 生成并持久化 Master

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py   "enterprise B2B admin dashboard RBAC system management data dense calm professional accessible React"   --design-system --persist -p "AppKernia Admin"
```

必须生成 `design-system/MASTER.md`。页面级 override：

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py   "enterprise user management table filters bulk actions drawer"   --design-system --persist -p "AppKernia Admin" --page "user-management"
```

建议页面组：`dashboard`、`login-auth`、`user-management`、`role-permission`、`system-settings`、`file-storage`、`notification-center`、`job-integration`、`audit-security`、`runtime-monitoring`、`profile-security`。

专项查询示例：

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "form validation accessibility" --stack react
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "data dense dashboard" --domain style
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "dashboard charts" --domain chart
```

## 4. Ant Design 映射

Skill 输出必须整理成 Ant Design semantic tokens 和组件 token，禁止散落硬编码品牌色，也不得因 skill 默认推荐而引入第二套 Tailwind/shadcn 设计系统。Dark Mode 使用 AntD algorithm + AK semantic override。

## 5. UI 证据

每个 UI Task 保存：

```text
artifacts/ui-ux-pro-max/<task>/
├── request.md
├── skill-output.md 或 .json
├── decisions.md
├── review-checklist.md
└── screenshots/{1440x900-light.png,1440x900-dark.png,768x1024.png}
```

检查信息层级、表格密度、Loading/Empty/Error/403、键盘/焦点/对比度、reduced motion、破坏性动作、1440/768 适配、Playwright 截图和 axe。缺少证据不得标记 Done。

本蓝图故意不附带伪造的 `MASTER.md`，目标仓库初始化时由 skill 生成。
