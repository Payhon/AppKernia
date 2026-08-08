# Decisions

1. Keep AppKernia blue as the product tint; use Apple-inspired semantic layering instead of copying the iOS system palette or SF Symbols.
2. Fix systemic causes first: shared safe-area shell, explicit AK button label colors, flexible scroll regions, native TabBar assets and consistent icons.
3. Use `#F2F2F7` grouped background, white elevated surfaces, `#1C1C1E` primary label, `#6C6C70` secondary label and `#246BFE` tint for the light baseline.
4. Use original geometric SVG/PNG icons with outline inactive and filled selected states. Apple icon files are not copied or redistributed.
5. Preserve the three-tab information architecture and native TabBar behavior. Do not introduce hamburger menus, floating action buttons or horizontal gesture conflicts.
6. Keep all business pages on `ak-*`; shared visual behavior belongs in AK UI and semantic tokens.
7. Treat dark mode, Dynamic Type/VoiceOver and Android/Harmony physical-device smoke as explicit acceptance boundaries, not assumptions derived from the iOS simulator.
