# Decisions

## Canonical contract

- Add one stable tag to every path operation and to the three reusable component path-item operations that Scalar also renders.
- Declare 31 ordered top-level tags and three ordered `x-tagGroups`. Each tag has an English `x-displayName` and semantic `x-appkernia-i18n-key`.
- Keep path-family classification executable and prioritize App subresources before the `/apps` base family.

## Localization runtime

- Generate a separate `api_reference` namespace from the unified Admin i18n sources, but import it only from the OpenAPI MPA entry.
- Parse the fetched canonical YAML with exact `yaml@2.9.0`, clone it in memory, and replace only group names, tag display names, and operation summaries.
- Fail closed on missing keys, duplicate operation IDs, unknown tags, malformed documents, or unsupported OpenAPI versions.
- Keep direct downloads raw and byte-identical. Remove stale entity headers only from the in-memory JSON response consumed by Scalar.

## Navigation and search

- Use Scalar's native `x-tagGroups` and `x-displayName` support; do not fork or replace its sidebar.
- Configure `defaultOpenAllTags=false`, `defaultOpenFirstTag=false`, and `operationTitleSource=summary`.
- Preserve canonical tag codes in anchors, so changing locale does not change deep-link identity.

## Responsive and security

- Retain the existing AK neutral page treatment and warning banner; do not add external fonts or visual dependencies.
- Add a localized `×` close button to the warning banner. Store only a `sessionStorage` dismissal marker, scoped to the current tab, so the warning returns in a new browsing context.
- Continue to force interactive requests to omit credentials and send the selected `Accept-Language`.
- Verify 1440 × 900 and 375 × 812 as Chromium browser viewports only, not physical-device evidence.
