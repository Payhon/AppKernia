---
name: quicklearn-ai-wechat
description: 为“快学AI”微信公众号执行端到端内容工作流：联网检索和核验资料，按项目作者口吻撰写中文技术文章，生成微信安全的内联 HTML、摘要与封面，并通过已配置的 SSH 中转服务器调用微信公众号官方 API 创建并验证草稿。用户提到给快学AI或微信公众号写文章、推广 AppKernia、生成微信推文、同步或推送到草稿箱时使用。默认只创建草稿，绝不自动群发或发布。
---

# 快学AI公众号工作流

## 核心约定

- 将“给快学AI写公众号文章”“写一篇公众号文章”等请求视为：研究、写作、排版并创建草稿箱草稿。用户明确说“只写不推送”“先看初稿”时停止在对应阶段。
- 使用同项目的 `$wechat-article-writer` 完成文章写作和质量检查；本 Skill 负责品牌约束、可重复构建与安全中转。
- 只调用草稿箱接口。不得调用群发、发布、删除草稿或删除素材接口。
- 不读取、打印、复制或回传 AppSecret。凭据只存在中转服务器的 root-only 配置中。
- 不把计划、未验证信息、未执行命令或提示词描述成真实结果。

## 默认值

- 公众号：快学AI
- 作者署名：快学AI
- 风格：项目作者第一人称，具体、克制、有技术细节，不写通用 AI 营销腔
- 正文：1500–2500 字；以信息完整为准，不为凑字数扩写
- 评论：开启；不限制为仅粉丝评论
- 封面：未提供时使用 `assets/default-cover.jpg`
- 中转主机：SSH 别名 `codex-huaweicloud-1-95-190-254`

## 工作流

### 1. 明确交付

从用户请求确定主题、目标读者、希望读者采取的行动和是否有必须引用的资料。信息足够时直接继续；只有缺失会实质改变文章方向的信息时才询问，最多两个问题。

### 2. 搜集和核验资料

1. AppKernia 主题先读取仓库 `README.md`、`docs/IMPLEMENTATION_STATUS.md`、相关蓝图和真实代码/测试证据。
2. 对会变化的外部事实进行联网检索，技术事实优先官方文档、官方仓库和原始公告。
3. 区分已实现、已通过自动化校验、浏览器验证、模拟器验证和真机/生产验证；不得混写。
4. 建立简短事实清单，删除无法核验或只有单一低可信来源的关键主张。

详细编辑规则见 `references/editorial-policy.md`。

### 3. 写作和复核

调用 `$wechat-article-writer` 的 `article` 模式并覆盖其默认风格：

- 使用自然的项目作者口吻，说明为什么做、解决什么问题、实际边界是什么。
- 标题不使用“震惊、神器、暴涨、完胜、行业第一”等无法证明的词。
- 不编造“我亲测”“已有大量用户”“效率提升 N 倍”等经历或数据。
- 文中涉及外部事实时在文末列出来源；正文避免堆砌裸链接。
- 输出一个主标题、两个备选标题、摘要、正文和配图建议。

按 `$wechat-article-writer/references/quality-checklist.md` 完成事实、语言、结构、平台合规和反翻译腔检查。未通过关键项时先修改，不得推送草稿。

### 4. 保存文章并构建草稿包

在 `tmp/wechat/YYYY-MM-DD-<slug>/` 保存 `article.md`。文章第一行必须是唯一的 `# 标题`。

运行：

```bash
python3 .agents/skills/quicklearn-ai-wechat/scripts/build_draft.py \
  --article tmp/wechat/YYYY-MM-DD-<slug>/article.md \
  --output-dir tmp/wechat/YYYY-MM-DD-<slug>/draft \
  --source-url https://github.com/Payhon/AppKernia
```

需要自定义封面时加 `--cover /absolute/path/cover.jpg`。需要正文图片时，每张使用一次 `--body-image img-id=/absolute/path/image.png`，并在文章中放置单独一行 `{{IMAGE:img-id}}`。

构建必须生成：

- `layout.html`
- `cover.jpg` 或 `cover.png`
- `draft-manifest.json`

### 5. 创建并验证草稿

除非用户明确要求停在初稿/预览阶段，否则运行：

```bash
python3 .agents/skills/quicklearn-ai-wechat/scripts/publish_via_ssh.py publish \
  tmp/wechat/YYYY-MM-DD-<slug>/draft/draft-manifest.json
```

脚本经 SSH 传输草稿包，在服务器端上传封面/正文图片并调用微信官方 `/cgi-bin/draft/add`，随后调用 `/cgi-bin/draft/get` 验证草稿。只有返回 `ok: true`、`verified: true` 和 `media_id` 才能报告草稿创建成功。

出现权限、IP 白名单、凭据、素材或正文限制错误时，保留本地草稿包并准确报告微信错误码；不得改走浏览器发布或尝试群发。

### 6. 报告

最终报告包括：标题、草稿创建状态、验证状态、草稿 `media_id`、使用的封面、资料来源、文件路径和任何未验证边界。不得包含 access token、AppSecret、服务器凭据文件内容或原始 API 请求。

## 运维

- 连接测试：`python3 .agents/skills/quicklearn-ai-wechat/scripts/publish_via_ssh.py doctor`
- 只验证草稿包：在 `publish` 后追加 `--dry-run`
- 中转部署和官方接口说明：读取 `references/deployment.md`
- 更新凭据时使用服务器交互式安全输入；不得把新凭据放入命令行、Git 或对话输出。
