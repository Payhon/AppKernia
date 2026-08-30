# Mobile 统一权限中心

## 1. 使用方式

“我的 → 应用权限”展示当前构建实际包含且业务正在使用的权限。首期完整展示通知权限；页面打开或从系统设置返回时只查询，不主动弹出授权框。

用户点击开启通知时按固定顺序执行：检查隐私同意、请求 OS 权限、初始化唯一 Push Adapter、获取 Token、注册服务端设备绑定、更新通知偏好。任一步失败会保持或回滚开关状态，站内消息和登录不受影响。

## 2. 公共 Port

`uni_modules/ak-permissions` 暴露：

- `listCapabilities()`
- `getStatus(key)`
- `request(key)`
- `openSettings(key)`
- `onStatusChanged(listener)`

稳定状态为 `not_determined`、`authorized`、`limited`、`denied`、`restricted`、`unavailable`，并返回 `can_request`、`can_open_settings`、平台说明和最近检查时间。

权限定义属于编译期 Registry。相机、照片、文件选择、麦克风、定位和蓝牙已预留键值，但在业务未启用前不会展示或申请；文件访问优先使用系统文件选择器。

## 3. 平台行为

- Android 13+ 使用 `POST_NOTIFICATIONS`，优先打开应用通知设置，失败后回退应用详情页；旧版本把通知设置状态作为系统能力查询。
- iOS 区分未决定、授权、临时/受限、拒绝，优先进入通知专属设置入口。
- HarmonyOS NEXT 使用通知管理能力查询、请求和打开设置；能力不足时才受控回退。
- `ak-push` 不再维护另一份系统授权状态，而是委托 `ak-permissions`。

OS 权限状态首期不上传服务器。服务端只保存用户 Push 偏好和设备注册状态；用户从系统设置撤销权限后，客户端在 `onShow` 刷新并停用不再可用的绑定。

## 4. 验收边界

源码编译不等于物理设备行为验收。发版前仍需在 iOS、GMS Android、国内厂商 Android 和 HarmonyOS NEXT 真机验证首次授权、拒绝、永久拒绝、设置恢复、升级、重装、账号切换和 Token 更新。
