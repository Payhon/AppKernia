# Service status page override

- This page is the only Admin surface for the compile-time module catalog. Do not recreate a module page under System settings.
- Use one vertical content stack: safety notice, equal-height status cards, dependency card, and runtime summary card. Use 24 px gaps on desktop and 16 px below 768 px.
- Keep 16 px between runtime descriptions, the compiled-module heading, and the module table.
- Module identity is a three-line semantic unit: localized name, stable code, and localized short description. Unknown translation keys fall back to the raw code.
- Render only enabled compile-time capabilities as localized text tags. Status must always include text and must not rely on color alone.
- Dependency and module tables use labeled, keyboard-focusable horizontal scroll regions. The page itself must not overflow horizontally.
- Preserve the existing diagnostic safety notice, refresh action, copy action, loading/error/empty states, and restrained Ant Design visual language.
