# 帮助内容与问题反馈

## 后台使用

1. 应用管理 → 单页内容，选择 App，分别编辑 FAQ (`faq`)、联系支持 (`contact-support`) 和关于我们 (`about-us`) 的中文、英文正文。FAQ 是整篇文档；联系方式由运营填写，不预置虚构地址。
2. 保存草稿不会改变公开内容；发布后移动端下次进入/重试加载新版本。未发布页面显示“内容暂未配置”。保留页面不能删除。
3. 应用管理 → 问题反馈，选择 App 后按描述关键词、状态及提交日期查询。详情展示用户、联系方式、版本、平台、私有截图、回复及状态历史。
4. `app.feedback.read` 控制查看与附件读取；`app.feedback.update` 控制单独改状态；`app.feedback.reply` 控制追加回复及回复后的状态。并发修改冲突时先刷新详情，再重新处理。回复历史不覆盖。

## 移动端

“我的 → 帮助与关于”沿用登录规则，包含 FAQ、联系支持、关于、问题反馈和我的反馈。页底版本来自当前安装包，不依赖联网与后台最新版本。

描述必填，联系方式可选；最多 3 张 JPEG/PNG/WebP 截图，单张不超过 5 MiB，后台更严格策略优先。截图上传完成后才能提交；失败可重试、取消和移除。提交失败保留表单内存，离开页面丢弃草稿，不写普通持久化缓存。提交成功进入详情；“我的反馈”返回时重新加载。

## 发布与运维

- 2026-08-31 已部署至本地 `appkernia-news-demo` 的 API、Worker、Admin；数据库已迁移至 27，并同步权限/菜单。见 [本地部署记录](help-feedback-local-deployment.md)。生产发布仍需先备份数据库、执行迁移和种子、协调部署同版本服务，并构建移动安装包。
- Migration 调整通用文件去重索引的谓词，旧 API 的上传 SQL 不兼容该新索引：采用维护窗口或协调切换 API/Worker；不要让旧上传服务跨迁移继续写入。普通历史文件及现有已发布 CMS 修订不被覆盖。
- Down 会删除反馈、回复、历史和附件关联，仅保留 CMS 正文为自定义页，并保留物理截图/校验摘要以便恢复。执行 Down 前必须先导出反馈；不是无损业务回滚。
- 私有对象存储桶不得允许匿名读取。截图响应为 `private, no-store`，不返回公开对象 URL；普通素材库、头像完成接口、公开文章资产均拒绝反馈用途文件。
- 截图校验包含 MIME/真实格式、尺寸、完整解码及 ClamAV 扫描。只有 `ready + clean` 可提交/读取，`skipped/pending/infected/failed` 均拒绝。`AK_FEEDBACK_CLAMD_SOCKET` 未配置时关闭截图上传；扫描超时、异常或检出时不创建文件，文字反馈不受影响。扫描通过后才写对象存储并标记 clean。
- Worker 每分钟批量清理过期未关联上传；默认上传有效期 24 小时，取消会提前到期。账号注销复用对象清理队列并删除当前 App 的反馈和关联。不要只停 Worker 而长期保留过期上传。

## 验证

- Backend：`make -C server check build`、`make -C server test-race`；为专用测试库设置 `AK_TEST_DATABASE_URL`，执行 `go test -tags=integration -race ./internal/modules/feedback/...`（在 server 目录）。测试覆盖发布/草稿、三种页面、多语言、幂等、并发、权限范围、图片解码、扫描状态、私有文件隔离和过期清理。
- Admin：`pnpm --dir apps/ak-admin check`。`scripts/e2e_feedback.mjs` 仅对已配置受信任 ClamD 的隔离环境执行，通过 `AK_FEEDBACK_E2E_CREDENTIALS` 提供权限为 0600 的测试账户 JSON（email/password/mobile_email/app_id/tenant_id），`AK_PLAYWRIGHT_MODULE` 可指定已安装 Playwright 模块。用 API 创建反馈，再在真实后台回复并验证 Mobile API 结果；明确标识故障注入步骤。
- Mobile：安装 `apps/ak-mobile/scripts/requirements.txt` 后执行 `check-project.sh`，使用 `build-platform.sh android|ios|harmony`。编译结果、模拟器与物理设备验收分别记录，不互相替代。
- 实际结果及截图索引见 `docs/CODEX_DELIVERY_REPORT.md`；不把未签名 HAP 或模拟器视为发布包或真机结果。

## ClamAV 配置与上线门槛

部署维护的 ClamD 服务与 freshclam 病毒库更新，将受信任的 Unix socket 共享给 API 容器，设置 `AK_FEEDBACK_CLAMD_SOCKET` 为容器内绝对路径。适配器只使用本地 Unix INSTREAM，不开启无认证 TCP；超时为 10 秒，最多发送 5 MiB。设置 ClamD `StreamMaxLength`、`MaxFileSize`、`MaxScanSize` 至少覆盖 5 MiB，并启用 `AlertExceedsMax`，避免引擎跳过文件仍报告成功。没有运行时 mock 或绕过开关；协议假服务和允许/拒绝 Scanner 仅存在测试中。

实现依据：[ClamD INSTREAM 协议](https://docs.clamav.net/manual/Usage/ClamdProtocol.html)。本轮未启动真实病毒库服务，协议测试不等同杀毒有效性验收。上线前必须用实际引擎、病毒库与测试样本验收。
