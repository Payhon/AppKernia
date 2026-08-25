# Decisions

- Extend the existing sectioned App Drawer instead of introducing a second settings route.
- Reuse `AkLocalizedFormTabs` for display name and subtitle so both required locales share existing validation behavior.
- Represent each onboarding position as one ordered bilingual asset pair with two required non-visual descriptions.
- Keep draft save and immutable publish as separate permissioned actions; surface version, time, drift and conflict state together.
- Use labelled Up/Down buttons instead of drag-only ordering, with disabled first/last boundaries.
- Keep AppKernia semantic tokens and Ant Design components; no external typeface, new palette or decorative motion is introduced.
