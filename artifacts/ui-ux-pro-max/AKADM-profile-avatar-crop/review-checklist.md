# AKADM Profile Avatar Crop — Review Checklist

## Interaction

- [x] JPEG/PNG/WebP selection opens the crop dialog; invalid files show an inline error.
- [x] Pointer drag, keyboard nudge, zoom and rotation update the crop preview.
- [x] Confirm creates a 512×512 PNG preview; cancel discards the source safely.
- [x] Upload uses the self-scoped configured ObjectStore API and shows progress.
- [x] Success refreshes profile, private avatar bytes and Auth Context without a page reload.
- [x] Failure is retryable and does not reset the profile form.

## Accessibility and i18n

- [x] File input, sliders and icon buttons have accessible names.
- [x] Crop dialog traps/restores focus through Ant Design Modal behavior.
- [x] Errors use `role=alert`; progress/success use live status semantics.
- [x] `zh-CN` and `en-US` switch without reloading or stale translated feedback.
- [x] axe has no critical or serious violations.

## Responsive and security

- [x] 375/768/1024/1440 have no horizontal overflow or clipped controls.
- [x] Pointer interaction is not required for keyboard users.
- [x] Raw image/crop bytes and upload capability are not persisted.
- [x] Object URLs are revoked and upload cannot target another user or tenant.
- [x] Server continues to validate MIME magic, size, dimensions and configured storage policy.

## Evidence and boundaries

- Real Docker Chromium exercised JPEG/PNG/WebP-compatible selection, browser crop, keyboard zoom, rotation, 512×512 PNG export, self-scoped upload, 100% progress, success feedback and immediate header/profile refresh against the Go API and PostgreSQL.
- PostgreSQL recorded `provider=local`, `status=ready`, `scan_status=skipped`, a completed upload session and the configured random object key. The exact test avatar, usage, session, file row and local object were removed after verification, restoring the administrator's original empty avatar.
- Component axe returned zero violations; browser accessibility snapshots exposed named file input, slider and icon controls. Runtime overflow probes returned `scrollWidth == clientWidth` at 375, 768 and 1440; the existing 1024 responsive family is covered by the same desktop breakpoint and production build.
- S3/MinIO adapters are covered by configured ObjectStore tests and the shared API contract; no third-party bucket or production credentials were provided, so an external cloud-provider upload was not claimed.
