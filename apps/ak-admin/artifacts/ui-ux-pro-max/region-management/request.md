# Region management UI request

- Product: AppKernia enterprise Admin.
- Page: System > System settings > Regions.
- Goal: super administrators can edit rows, add direct children below level 0/1 nodes, and soft-delete leaf nodes.
- Stack: React 19, Ant Design 6, React Hook Form, Zod, TanStack Query.
- Constraints: retain lazy tree loading, preserve zh-CN/en-US, permission-gate every action, keep immutable hierarchy fields visible, and avoid cascade deletion.

Skill commands executed on 2026-08-05:

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise admin geographic region hierarchy tree table editable child drawer" --design-system -p "AppKernia Region Management" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "tree table drawer form delete confirmation keyboard focus accessibility" --domain ux -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "tree table mutation drawer form accessibility" --stack react
```
