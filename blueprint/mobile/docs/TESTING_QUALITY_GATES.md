# 测试与质量门禁

## 1. 测试层次

### Pure UTS

- 错误映射。
- Validation。
- Query Key。
- Route/Permission/Flag 判断。
- Refresh single-flight 状态机。
- Token 日志脱敏。

### Contract

- OpenAPI Schema Hash。
- DTO 编解码。
- 所有 Page API 存在。
- 稳定错误码与状态码。

### Component/Page

- AK UI 状态。
- 表单校验和服务端字段错误。
- Loading/Empty/Error/Offline/Forbidden。
- 取消请求和重复提交。

### uni-automator

- Android 关键流程。
- iOS 模拟器关键流程。
- 截图与页面跳转。
- 真机仍需补充 Smoke；自动化结果不能替代 Release 验证。

## 2. 安全测试

必须覆盖：

- 普通 Storage 中不存在 Token/密码/OTP。
- 401 并发刷新只有一个网络请求。
- 403 不刷新。
- Refresh Replay 清理整个 Session。
- 深链只允许 Registry。
- OAuth state/PKCE 失败拒绝。
- 日志/崩溃报告无敏感字段。
- 隐私同意前不初始化敏感 SDK。
- 跨租户缓存隔离。
- 短信 OTP 在交互验证码验证和单次消费前不得创建 Challenge 或进入投递队列；跨 App、手机号、场景、IP、设备、Session、用途和资源证明必须失败，邮箱 OTP 不受影响。

## 3. 构建门禁

每个平台记录：工具版本、命令/操作、时间、退出状态、构建产物 Hash、设备型号、OS/API、测试结果。无法执行必须标 `blocked`。

## 4. 性能预算

初始版本先记录可复现基线，不伪造绝对数字：

- 冷启动到可交互。
- 登录成功到 Home 稳定。
- 100/1000 条消息列表滚动。
- 页面切换卡顿。
- 前后台 20 次后的内存趋势。
- 图片上传峰值内存。

Release 包测性能，不用 Debug 结果宣布达标。

## 5. Blueprint 校验

```bash
python3 blueprint/mobile/scripts/validate_blueprint_specs.py
```

必须在 CI 中运行，并检查 Route、Tab、API、Permission、Task DAG、Component Matrix 和平台矩阵的一致性。
