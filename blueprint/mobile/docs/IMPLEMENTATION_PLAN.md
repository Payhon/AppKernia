# 实施计划

## Phase P0：可登录、可启动、可三端编译的基座

`AKMOB-000` 至 `AKMOB-080`：工程、设计系统、uView 适配、OpenAPI、网络、安全存储、启动、认证、Tab 与权限。

退出条件：至少在可用环境完成三平台 Debug 编译；登录/刷新/退出使用真实后端；普通 Storage 无 Token。

## Phase P1：个人中心和日常安全

`AKMOB-090`、`100`、`110`、`130`、`150`：资料、头像、会话、设备、MFA、消息、设置、法律与版本。

退出条件：个人中心闭环、MFA Secret 安全、消息已读事实源、zh-Hans/en 切换。

## Phase P2：可选平台能力

`AKMOB-120`、`140`、`160`、`170`、`180`、`190`：OAuth、Push、暗色、多租户、注销和弱网性能。

退出条件：Feature Flag 默认安全；缺少 Provider 凭据不阻塞 Core；暗色未通过则保持关闭。

## Phase P3：发布矩阵

`AKMOB-200` 至 `240`：Android、iOS、Harmony 发布验收、文档与 Vapor Spike。

退出条件：三端 Release/真机证据、许可证、完整交付报告。Vapor 不阻塞 VDOM Core 发布。
