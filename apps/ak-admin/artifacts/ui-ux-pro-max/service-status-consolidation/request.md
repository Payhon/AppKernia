# Service status consolidation UI request

- Product: AppKernia enterprise Admin.
- Page: System > Monitoring > Service Status.
- Goal: make this the only semantic display for eight compile-time domain modules and improve the page's visual hierarchy.
- Stack: React 19, Ant Design 6, TanStack Query, i18next.
- Constraints: retain existing diagnostic actions and safe fields, support zh-CN/en-US, use 24/16 responsive vertical spacing, keep three summary cards equal-height, and prevent page-level horizontal overflow.

Skill commands executed on 2026-08-05:

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "SaaS admin service health observability dashboard card spacing data tables professional" --design-system -p "AppKernia Admin Service Status" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "service health dashboard responsive table keyboard focus status accessibility" --domain ux -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "responsive dashboard cards table overflow accessibility" --stack react
```
