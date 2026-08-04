# Request

- Task: `AKADM-060` Web device-management subtask.
- Surface: Admin `/profile/security`.
- Goal: list self-scoped registered devices, identify the current device, remove another or current device with explicit consequences, and cover loading, empty, success and error states in `zh-CN` and `en-US`.
- Required commands were run from the repository-local `.codex/skills/ui-ux-pro-max` installation on 2026-08-03.

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise account security trusted devices current device remove access responsive" --design-system -p "AppKernia Admin" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "device management current device remove confirmation empty state accessibility" --domain ux -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "react security device list destructive mutation loading error accessibility" --stack react
```
