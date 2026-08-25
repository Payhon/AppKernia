# Decisions

- Privacy is a pre-bootstrap offline route using only the generated packaged snapshot and static legal pages.
- Use a top-rounded full-height neutral container, centered App identity and safe-area-bottom legal/actions to match the reference structure without copying its product branding.
- Android/Harmony cancel uses the platform exit port; iOS remains blocked on the consent page.
- Onboarding preloads the entire published locale image set before navigation; any ordinary failure continues bootstrap without recording completion.
- No skip is provided. Completion requires the last page and a visited marker for every position.
- Persist the highest completed version per public App UUID so upgrades reappear and rollbacks do not.
