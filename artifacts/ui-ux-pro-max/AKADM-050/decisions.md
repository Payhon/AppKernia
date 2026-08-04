# AKADM-050 Decisions

- 桌面双栏、移动单栏；不使用营销轮播与远程字体。
- 账号/密码 label 绑定 id，保留 username/current-password autocomplete。
- 注册和找回默认 fail-closed，由 `/auth/public-config` 控制；未实现 API 不展示可操作入口。
