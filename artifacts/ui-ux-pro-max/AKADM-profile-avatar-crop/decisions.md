# AKADM Profile Avatar Crop — Decisions

- Keep the existing self-scoped avatar API. It is safer than requiring `storage.file.upload` and already routes objects through the tenant's configured local/S3/MinIO adapter without exposing credentials.
- Add reusable `AkImageCropper` and `AkAvatarUploader` components. The uploader accepts a typed upload callback so other avatar surfaces can reuse the UI without importing profile state.
- Implement crop math and Canvas export locally, without adding a second UI framework or a new crop dependency.
- Export a deterministic 512×512 PNG. The server still validates declared MIME, magic bytes, dimensions, size, tenant, owner and upload session.
- Crop supports pointer drag plus keyboard nudge buttons, zoom slider and 90-degree rotation buttons. Dragging is not the only interaction.
- Source and cropped previews use short-lived object URLs that are revoked when replaced or unmounted.
- Keep profile fields independent: avatar upload failure does not discard unsaved display name, locale or time-zone edits.
- Preserve the existing AppKernia ink/canvas tokens and system font. Skill marketing structure, external fonts and alternate palette are not adopted.
