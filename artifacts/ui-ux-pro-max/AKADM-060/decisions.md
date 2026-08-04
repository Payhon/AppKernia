# AKADM-060 Profile Basic Decisions

- 采用单列设置 Card；账号只读，显示名称/语言/时区可编辑。
- RHF 管理表单，Zod 校验；TanStack Query 读取本人资料，mutation 成功后精确更新缓存与 Auth Context。
- 保存成功同步 i18next/AntD/Day.js/HTML lang；失败保留输入并可访问播报。
- 本子任务不伪造头像、Session、Device 或 MFA：对应 API 未完成前保持后续边界。
