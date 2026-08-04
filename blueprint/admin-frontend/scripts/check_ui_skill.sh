#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
candidates=(
  ".codex/skills/ui-ux-pro-max/scripts/search.py"
  ".agents/skills/ui-ux-pro-max/scripts/search.py"
  ".claude/skills/ui-ux-pro-max/scripts/search.py"
  ".continue/skills/ui-ux-pro-max/scripts/search.py"
  ".factory/skills/ui-ux-pro-max/scripts/search.py"
)
for p in "${candidates[@]}"; do
  if [[ -f "$repo_root/$p" ]]; then echo "FOUND:$p"; exit 0; fi
done
echo "MISSING: ui-ux-pro-max skill is not installed in a supported project path." >&2
exit 1
