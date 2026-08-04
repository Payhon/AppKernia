# AKADM-220 UI Decisions

- The existing `design-system/appkernia-admin/MASTER.md` remains authoritative; the Skill-persisted file-storage master is retained as raw evidence only.
- The file list will use URL-restored filters and expose scan state with text plus color. A file is selectable/downloadable only when backend status is `ready` and scan status is `clean` or explicitly `skipped`.
- Upload items will expose progress, cancel, retry/resume capability and an accessible live status. Client cancellation never implies the server session was deleted unless the cancel API succeeds.
- Delete confirmation must show server-returned usages. In-use files remain undeletable; the UI does not offer a force-delete bypass.
- AkFilePicker will be a reusable permission-aware dialog over the same server-filtered ready/clean file contract.
