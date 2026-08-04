# System Files Page Override

This page inherits `../MASTER.md`.

- Layout: compact filter row, permission-aware upload action, internally scrollable file table, and detail/usage drawer. At 375px filters stack and secondary columns collapse without hiding scan state or primary actions.
- Upload queue: filename, byte progress, localized state text, cancel and retry/resume controls; announce changes through a polite live region and respect reduced motion.
- Scan gate: pending/failed/infected files cannot be selected, referenced or downloaded. Status must use text and icon in addition to color.
- Delete: fetch current usages before confirmation. If usages exist, show module/entity/field references and no destructive action. Never expose a force-delete control.
- Picker: modal title, filter label, selected count and confirm action all use translation keys. Restore focus to the opener on close and avoid keyboard traps.
- Security: do not accept arbitrary remote URLs, do not expose object keys or provider credentials, and do not infer authorization from hidden buttons.
