# SSH 中转部署与接口契约

## 架构

```text
Codex / quicklearn-ai-wechat
  -> SSH key authentication
  -> codex-huaweicloud-1-95-190-254
  -> /usr/local/bin/quicklearn-wechat
  -> api.weixin.qq.com
  -> 快学AI草稿箱
```

中转服务器只运行按需 CLI，不开放新的 HTTP 端口。服务器出口 IP 已由用户加入公众号接口白名单。

## 服务器路径

- 程序：`/opt/quicklearn-wechat/wechat_gateway.py`
- 命令：`/usr/local/bin/quicklearn-wechat`
- 凭据：`/etc/quicklearn-wechat/credentials.json`，权限 `0600`
- 状态缓存：`/var/lib/quicklearn-wechat/`，目录权限 `0700`
- 临时上传：`/var/tmp/quicklearn-wechat-*`，每次调用后由 CLI 校验路径并清理

不得读取或输出凭据文件内容。不得把 access token 放入日志。

## 本地草稿清单

`draft-manifest.json` 示例：

```json
{
  "schema_version": "1.0",
  "title": "文章标题",
  "author": "快学AI",
  "digest": "摘要",
  "content_html": "layout.html",
  "content_source_url": "https://github.com/Payhon/AppKernia",
  "cover": "cover.jpg",
  "body_images": [],
  "need_open_comment": 1,
  "only_fans_can_comment": 0
}
```

所有路径都相对清单所在目录解析。`body_images` 项包含 `id`、`path`、`alt`，文章 HTML 中使用 `{{IMAGE:<id>}}` 占位。

## 官方接口

- 稳定 access token：`POST https://api.weixin.qq.com/cgi-bin/stable_token`
- 永久封面素材：`POST /cgi-bin/material/add_material?type=image`
- 正文图片：`POST /cgi-bin/media/uploadimg`
- 新增草稿：`POST /cgi-bin/draft/add`
- 草稿验证：`POST /cgi-bin/draft/get`
- 草稿计数：`GET /cgi-bin/draft/count`

官方文档要求这些接口从服务器端调用。草稿标题不超过 32 字、作者不超过 16 字、摘要不超过 120 字、正文少于 2 万字符且小于 1 MB；图文封面必须使用永久 MediaID。正文图片只接受微信上传接口返回的 URL。

## 失败处理

- `40164`：服务器出口 IP 未在白名单。
- `40001` / `40125`：AppSecret 或 token 不正确；不要打印凭据。
- `40005` / `40009`：图片格式或大小不符合要求。
- 其他非零错误码：停止，不重试写请求；保留本地草稿包供修正。

写请求不自动重试，避免在响应不确定时重复创建草稿。token 获取和只读校验可以重试一次。
