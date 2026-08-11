# Review checklist

## Contract and localization

- [x] Three ordered interface surfaces and 31 ordered modules are declared.
- [x] All 278 direct path operations and three reusable path-item operations have one registered module.
- [x] Both locales contain every rendered operation title; English exactly matches canonical summaries.
- [x] No stale or missing `api_reference` keys are accepted.
- [x] Paths, methods, operation IDs, tag codes, schemas, security, descriptions, and examples remain unchanged.

## Navigation and search

- [x] Module operation lists start collapsed.
- [x] Surface and module hierarchy is rendered by native Scalar metadata.
- [x] Navigation, search result, and body operation titles share the same localized summary.
- [x] Canonical anchors remain stable across locales.

## Packaging and security

- [x] Canonical and emitted YAML are byte-identical, with no locale-specific spec files.
- [x] YAML, Scalar, and `api_reference` catalogs are isolated from the Admin application graph.
- [x] Authorization is not persisted and browser credentials are omitted from interactive requests.
- [x] No Agent, telemetry, remote proxy, external font, or plugin URL is enabled.
- [x] Interactive-testing notice has a localized accessible close button and session-scoped dismissal.

## Browser evidence

- [x] `zh-CN` and `en-US` verified at 1440 × 900 and 375 × 812.
- [x] Module expansion, localized module/title search, body-title consistency, health request, language header, and stable anchor verified.
- [x] No external requests, unexpected console errors, horizontal overflow, or axe serious/critical findings.

## Verification boundary

- 375 × 812 is a Chromium viewport, not physical-device evidence.
- Scalar's transient interactive client is exercised for the public health endpoint; accessibility audit targets the stable reference surface after the client is closed.
