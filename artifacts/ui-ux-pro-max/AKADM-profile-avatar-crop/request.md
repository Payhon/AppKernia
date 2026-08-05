# AKADM Profile Avatar Crop — Request

## User request

在个人中心新增可修改头像的完整能力：通过已有云存储 API 上传，上传前在浏览器裁剪，并将裁剪与头像上传整理为公共、可复用组件。

## Product and stack context

- Enterprise SaaS administration console.
- React 19 + TypeScript strict + Ant Design 6 + TanStack Query.
- Existing `/admin-api/v1/me/avatar/*` self-scoped API writes through the configured local/S3/MinIO ObjectStore adapter.
- Existing Profile Basic page and AppKernia Admin Master design system must be preserved.

## Required states

- Existing avatar and initials fallback.
- File validation, image decoding, crop dialog, zoom, rotate, keyboard nudge and preview.
- Upload progress, retryable error and success feedback.
- `zh-CN` and `en-US`, 375/768/1024/1440 responsive behavior.
- Object URLs released; raw source image and cropped bytes never persisted in browser storage.
