# AKADM-110 ui-ux-pro-max Actual Output

实际执行 4 条仓库本地命令，均退出 0。

## Accounts design-system query

- Pattern: Enterprise Gateway；建议 Navy/Grey corporate strategy。
- Style: Dark Mode (OLED)；Primary `#1E40AF`、Secondary `#3B82F6`、CTA `#F59E0B`、Background `#F8FAFC`、Text `#1E3A8A`。
- Typography: Fira Code + Fira Sans，并给出 Google Fonts import。
- Checklist: SVG icons、pointer、150–300ms、4.5:1、visible focus、reduced motion、375/768/1024/1440。
- 原始 anti-pattern：Slow updates、No automation。

## Account-detail design-system query

- Pattern: Enterprise Gateway；Style: Minimalism & Swiss。
- Primary `#0369A1`、Secondary `#0EA5E9`、CTA `#22C55E`、Background `#F0F9FF`、Text `#0C4A6E`。
- Typography: Inter；建议外部 Google Fonts。
- 建议避免 playful design、poor security UX、AI purple/pink gradients。

## UX query (8 results)

1. Inline validation: validate on blur, not submit-only.
2. Confirmation messages: brief success feedback, not silent success.
3. Error recovery: retry and clear next steps.
4. Bulk actions: checkbox selection + action bar.
5. Confirmation dialogs: confirm irreversible actions.
6. Accessible errors: `role=alert`.
7. Keyboard navigation: logical tab order and no traps.
8. Multi-step progress: progress/step indication.

## React query (8 results)

1. Prefer accessible Testing Library queries.
2. Controlled form components.
3. Separate data containers from presentation.
4. Avoid deep prop drilling.
5. Context only for true app-global state.
6. Avoid effects for derived data/event handling.
7. Debounce or defer rapid search input.
8. Strongly type event handlers.

原始生成器还建议营销 Hero、Contact Sales、logo carousel、WebGL/3D/视差、琥珀 CTA 和外部字体；这些内容作为真实输出保留，但因与仓库 Master、性能、隐私和后台语义冲突而未采用。
