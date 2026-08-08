# Skill output

## Sources used before implementation

1. Repository `ui-ux-pro-max` Skill:

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "cross-platform mobile application Apple HIG native iOS calm premium productivity grouped lists safe area tab bar"
  --design-system -p "AppKernia Mobile Apple Refresh" -f markdown

python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "mobile safe area tab bar navigation grouped cards touch target accessibility contrast"
  --domain ux -n 15

python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "native iOS hierarchy tab bar cards forms lists accessibility"
  --stack swiftui
```

Useful output retained: 44 × 44 px minimum targets, 8 px target separation, 4.5:1 text contrast, predictable back navigation, non-overlapping fixed chrome, grouped forms and explicit accessibility labels.

The generated teal/orange palette, web landing-page pattern, external Lora/Raleway fonts and AI chat animation suggestions were rejected because they conflict with the AppKernia product, the packaged system-font rule and the Mobile blueprint.

2. skills.sh search and selected Skill:

- Selected `ios-hig-design` from `wondelai/skills` because it is framework independent and directly covers safe areas, system hierarchy, grouped lists, TabBar, semantic colors, typography and accessibility.
- Installed with `npx --yes skills add https://github.com/wondelai/skills --skill ios-hig-design --yes`.
- Installation security summary: Gen Safe, Socket 0 alerts, Snyk Low Risk.
- The SwiftUI-specific `mobile-ios-design` and traditional `uni-app` documentation Skill were not used as implementation authorities because this codebase is uni-app x / UTS.

Relevant HIG rules retained: clarity/deference/depth, 16–20 pt margins, 8/16/24 spacing, 44 pt controls, semantic grouped surfaces, system typography, two-to-five primary tabs, outline/filled tab states and content below the safe area.
