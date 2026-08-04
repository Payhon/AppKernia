# Regions Override

- Use a read-only tree table with stable region code as row key. Expansion calls the child endpoint once per parent and shows row-level loading feedback.
- In browse mode show only roots initially; search mode is a flat server-filtered result list with full-name context and no misleading tree expansion.
- Filters: code/name query, level, status. Store all values in URL search params.
- Show code, name/full name, level, postal code, coordinates, status and updated time; at 375px preserve name/code/status and allow an accessible internal table scroll.
- No create/edit/delete action is exposed; versioned region imports belong to the CLI pipeline.
