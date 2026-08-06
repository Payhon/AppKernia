# Region management UI review checklist

- [x] Existing Ant Design tokens and system font retained.
- [x] All visible strings use zh-CN/en-US translation keys.
- [x] Action visibility follows create/update/delete permissions.
- [x] Code, parent, and level are visibly immutable during edit.
- [x] Delete requires confirmation and never cascades.
- [x] Form controls have explicit labels and mutation feedback.
- [x] Scrollable table remains keyboard focusable.
- [x] 1440 px zh-CN/en-US screenshots captured.
- [x] Narrow viewport screenshot captured.
- [x] Region page `main` content reports zero axe violations in all five captured states.
- [ ] Alternate application theme review is not applicable because Admin currently exposes no theme switch; the existing light content/dark navigation shell was reviewed.
