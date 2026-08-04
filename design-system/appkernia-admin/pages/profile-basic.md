# Profile Basic Page Overrides

> **PROJECT:** AppKernia Admin
> **Generated:** 2026-08-03
> **Page Type:** Authenticated profile settings form

本页继承 `../MASTER.md`；以下规则覆盖 Master 中与资料表单冲突的通用建议。

## Layout

- 使用单列、最大宽度 760px 的设置 Card，避免营销页双栏和重复 CTA。
- 头像上传区置于资料字段之前：当前头像/姓名首字母、文件选择、限制说明和独立上传状态形成一个语义分组；窄屏纵向排列。
- 字段顺序固定为不可编辑账号、显示名称、语言、IANA 时区；375/768/1024/1440 均不得横向溢出。
- App Shell 用户名作为明确入口，当前 URL 为 `/profile/basic`。

## Interaction and accessibility

- 所有输入使用显式 label/id；不可编辑邮箱使用 disabled Input，但仍可读。
- 提交展示 loading，并以 `role="status"`/`role="alert"` 宣告成功或失败。
- 字段校验错误紧邻字段；失败保留用户输入并提供重试路径。
- 资料保存成功后同步 Auth Context 与运行时 locale，不刷新页面。
- 头像只接受 JPEG/PNG/WebP；客户端预检不替代服务端校验。进度使用文字和进度条，成功 `role=status`、失败 `role=alert`，可重试且不清空其他资料字段。
- 本地预览 object URL 在替换或卸载时释放；上传能力、对象存储签名和文件二进制不持久化。

## Visual

- 保持既有 Navy/Blue token、Inter/system font、可见 focus ring 和 reduced-motion。
- 不采用本次检索的红橙紧迫营销色或 Webinar 页面结构；它们不符合企业资料设置语义。
