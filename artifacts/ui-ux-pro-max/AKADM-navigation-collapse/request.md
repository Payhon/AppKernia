# AKADM Navigation Contrast and Collapse Request

## Source

- User screenshot: `/var/folders/52/mkm9shln40x3hjzr_swl3qfc0000gp/T/codex-clipboard-bacf6e4f-5a64-46f9-8721-e64b6898637e.png`
- Affected route: `/system/settings/dictionaries`

## Request

1. Keep the first-level `System` and second-level `System Settings` labels readable while a third-level leaf is selected.
2. Remove the bottom-right hamburger collapse control.
3. Reveal a directional collapse handle at the vertical midpoint of the sidebar/content boundary while the desktop sidebar is hovered.
4. Reverse the arrow after collapse and allow the same control to expand the sidebar again.

## Constraints

- Preserve Ant Design Menu semantics and nested popup behavior.
- Preserve i18next labels and both `zh-CN` / `en-US` locales.
- Keep the control keyboard reachable and visible on non-hover pointer devices.
- Preserve the existing visual-refresh worktree changes.
