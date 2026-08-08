---
title: Mobile 用户与公共资源
description: 当前用户、会话、通知、内容、地区与公开配置 API。
---

# Mobile 用户与公共资源

## 公开资源

| 方法  | 路径                               | 说明                                   |
| ----- | ---------------------------------- | -------------------------------------- |
| `GET` | `/public/config`                   | App 名称、默认语言、注册开关与验证模式 |
| `GET` | `/public/legal/{document_type}`    | 当前发布的隐私政策或用户协议           |
| `GET` | `/public/pages/{slug}`             | 当前 App 发布的单页                    |
| `GET` | `/public/app-version?platform=ios` | 三平台升级策略与本地化说明             |
| `GET` | `/public/dictionaries/{code}`      | 公开字典                               |
| `GET` | `/regions`                         | 按 parent / level / q 懒加载地区       |

## 当前用户

| 方法            | 路径                  | 说明                                             |
| --------------- | --------------------- | ------------------------------------------------ |
| `GET` / `PATCH` | `/me`                 | 读取或更新 `display_name`、`locale`、`time_zone` |
| `GET` / `PATCH` | `/me/preferences`     | 语言、外观与通知偏好                             |
| `GET`           | `/me/sessions`        | 自己的 Mobile Sessions                           |
| `DELETE`        | `/me/sessions/{id}`   | 撤销一个自己的 Session                           |
| `GET`           | `/me/devices`         | 自己的设备                                       |
| `DELETE`        | `/me/devices/{id}`    | 删除设备并撤销关联会话                           |
| `GET`           | `/me/login-events`    | 脱敏登录记录                                     |
| `GET`           | `/me/security-events` | 与本人相关的安全事件                             |

更新资料示例：

```bash
curl -X PATCH 'http://localhost:8080/api/v1/me' \
  -H 'Authorization: Bearer YOUR_ACCESS_TOKEN' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: zh-CN' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -d '{"display_name":"AK Developer","locale":"zh-CN","time_zone":"Asia/Shanghai"}'
```

## 消息与内容

- `GET /me/notifications?cursor=…&limit=20` 使用 opaque cursor。
- `PATCH /me/notifications/{id}/read` 由服务端确认最终已读状态。
- `GET /me/notifications/unread-count` 返回未读数量。
- `GET /articles`、`GET /articles/{slug}` 与 `GET /article-categories` 只返回当前 App/租户已发布内容。
- 收藏使用 `PUT` / `DELETE /me/article-bookmarks/{article_id}`。
