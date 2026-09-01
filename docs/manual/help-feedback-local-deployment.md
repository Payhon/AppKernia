# 帮助与反馈：本地管理端部署记录

部署日期：2026-08-31。目标为已有 Docker Compose 项目 `appkernia-news-demo`，保留现有数据库、对象存储卷、运行环境和账号。未部署生产环境、未提交或推送 Git。

## 访问入口

- 管理端：http://localhost:4174
- API：http://localhost:8080
- 就绪检查：http://localhost:8080/internal/v1/health/ready
- 管理端同源代理就绪检查：http://localhost:4174/internal/v1/health/ready

继续使用原管理员账号密码。进入“应用管理 → 问题反馈”，先在顶部“当前应用”选择 App；帮助文案位于“应用管理 → 内容管理 → 单页内容”。`faq`、`contact-support` 已补齐为草稿，需运营填写中英文并发布；未覆盖原页面或填入虚构联系方式。

## 执行步骤与结果

1. 检查当前容器 Compose 标签、端口、挂载与环境。API/Worker 的当前配置与仓库 Compose 一致；服务环境保持不变。
2. `docker compose -p appkernia-news-demo build api worker migrate seed admin`，退出 0；使用当前工作区构建。旧 API/Worker/Admin 镜像已额外打 tag 保留。
3. `docker compose -p appkernia-news-demo stop api worker`，退出 0；停止旧写入端后备份数据库，避免旧上传 SQL 跨迁移运行。
4. `docker exec appkernia-news-demo-postgres-1 pg_dump -U appkernia -d appkernia --format=custom`，退出 0。备份 `/Users/payhon/project/AppKernia/.secrets/local-deploy-20260831-140502/database-before-27.dump`，672807 bytes，权限 0600；`pg_restore --list` 退出 0，987 行 TOC。此处验证备份可解析，未声称执行恢复演练。
5. `docker compose -p appkernia-news-demo run --rm --no-deps migrate`，退出 0；`migration version=27 dirty=false`，River 无新增迁移。
6. `docker compose -p appkernia-news-demo run --rm --no-deps seed`，退出 0；180 权限、49 菜单，`development_admin=false`，未创建或重置管理员。
7. `docker compose -p appkernia-news-demo up -d --no-deps --force-recreate api worker admin`，退出 0。API/Admin healthy，Worker running；三者 restart_count=0。

数据库备份和旧容器配置位于 `/Users/payhon/project/AppKernia/.secrets/local-deploy-20260831-140502`（目录 0700，包含敏感信息，禁止提交）。旧镜像 tag：`appkernia-news-demo-{api,worker,admin}:before-help-feedback-20260831-140502`。Down 会删除反馈业务数据，不作为无损回滚；需要回退时先保存部署后新增数据，协调恢复数据库与对应镜像。

## 镜像

| 服务 | 已部署 Image ID |
|---|---|
| API | `sha256:5b7066c002af98496fd4e62546d21196f855bdff17893cd181ca4ccd838d21eb` |
| Worker | `sha256:91ccf145c091e5f67b840eb944150c05195a77a4ca5871f4101e65be9dd7c2e6` |
| Admin | `sha256:2babec23ef7b5a9382d292f16b7e977ce9fd87979b7cfe74179188fb7ff319de` |

构建元信息沿用 Compose 默认 `dev/unknown`；上表 Image ID 是本次部署的精确标识，不把未提交代码描述为 Git 发布版本。

## 核验与限制

- API 就绪、Admin healthz、Admin 同源 API 就绪均 HTTP 200。
- 现有 1 App、2 账号及密码状态、6 文件和原 3 个页面指针/状态校验不变；原发布内容集合不变。本地原本无已发布单页修订。
- 当前 5 个保留页存在，新增 FAQ/联系支持未发布；3 项反馈权限已授予当前超级管理员角色。
- `make check-blueprints` 退出 0，覆盖 Backend、Admin、Mobile 与 i18n 四项。
- 最终浏览器/API 冒烟退出 0，11 项检查通过，3 张截图。结果及截图索引：`output/playwright/local-deploy-help-feedback/evidence.json`。截图是本机 Chromium 管理端，不是移动真机证据。
- 部署过程日志：`output/local-deploy-help-feedback/`。早期浏览器脚本存在未选择 App、未展开子菜单/标题断言不符和选择器动画等待不足等问题，修正脚本后重新执行；不将这些早期失败日志删除或称为通过。直接完整跳转还验证出当前 Admin 依赖内存登录状态、刷新后要求重新登录的既有行为，本次没有修改认证代码。
- 没有启用 ClamAV：`AK_FEEDBACK_CLAMD_SOCKET` 未配置，截图上传按设计拒绝；文字反馈、后台查询/回复不受影响。真实扫描引擎和病毒库未验证。
- 本次仅部署本地 Admin/API/Worker，未重新打包或安装 Mobile，也未重新运行此前已记录的全仓单元/集成和设备矩阵。上一轮已有 IAM 并发集成失败仍见交付报告，不声称本次部署修复。
