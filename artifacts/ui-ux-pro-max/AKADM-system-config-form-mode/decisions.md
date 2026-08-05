# Decisions

- Default to form mode and omit that default from the URL; persist only `mode=table`.
- Put the labelled form/table segmented control in the page heading so the two jobs are explicit.
- Preserve the existing table, filters, create/edit drawer, and secret rotation flow without changing their semantics.
- Generate direct controls from `value_type` and `validation_schema`, including enums and numeric bounds.
- Use one save action, but retain backend optimistic versions and separate secret rotation. Save operations are sequential and partial failure is explicit; failed drafts remain available for retry.
- Do not fetch or render secrets. Empty secret controls mean unchanged.
- Confirm before discarding dirty fields during category or mode changes and warn on browser unload.
- Keep category navigation sticky on wide layouts and collapse it to a select on narrower layouts.
- Keep the unchecked Boolean switch and public-scope table tag within WCAG AA contrast after real-browser axe exposed Ant Design's default low-contrast colors.
