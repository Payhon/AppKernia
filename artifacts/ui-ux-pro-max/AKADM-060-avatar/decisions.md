# AKADM-060 Avatar Decisions

- 继承现有 Profile Basic 单列设置布局和 Master token，不采用 Skill 返回的营销页结构、外部 Google Font 或橙色营销 CTA。
- 头像控件置于资料字段之前，保留原表单保存职责；上传采用独立、可取消/重试的状态，不把文件二进制或对象存储凭据写入全局状态。
- 文件选择由翻译键驱动的 Ant Design 按钮触发隐藏原生 input，避免中文界面泄漏浏览器原生英文文件文案；只接受 JPEG、PNG、WebP，客户端先校验类型和大小，服务端仍是最终边界。
- 预览使用本地 object URL 并及时释放；正式头像使用服务端提供的受控 URL和替代文本。
- 展示上传进度、成功 `role=status` 和错误 `role=alert`，不得仅用颜色传达结果；375/768/1024/1440 不横向溢出。
- 第三方对象存储缺少凭据时通过 Storage Port 与 development-only 本地 Adapter 继续；生产环境禁止启用本地 Adapter。
