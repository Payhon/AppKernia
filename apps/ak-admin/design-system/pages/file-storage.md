# File storage page override

- Reuse one uploader across the file list, file picker, and cloud-storage configuration view.
- Show policy-derived accepted file types, maximum size, active provider, progress, pause/resume/cancel, completion, and actionable errors.
- Keep upload selection keyboard accessible and do not start network work before the user selects a file.
- File rows expose provider, media type, size, scan state, usage count, and lifecycle state; downloads and selection remain scan-gated.
- Provider is a filter and a non-color-only tag. Never expose bucket names, object keys, credentials, or presigned URLs in list responses.
