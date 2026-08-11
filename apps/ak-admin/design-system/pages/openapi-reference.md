# OpenAPI Reference Override

## Scope

This override governs the independent `/openapi/` Scalar page. It does not change the Admin shell, API wire contract, generated clients, or canonical descriptions and schemas.

## Information architecture

- Navigation order is interface surface → business module → operation.
- The fixed interface surfaces are Platform and Public APIs, Mobile APIs, and Admin APIs. Modules retain canonical order; do not alphabetize them.
- All module operation lists start collapsed. Users expand only the module they need, avoiding a flat list of hundreds of operations.
- A canonical operation belongs to exactly one registered module. New paths must extend the path-family validator before delivery.

## Localization

- The URL `?lang=zh-CN|en-US` selects both Scalar controls and the in-memory reference labels; an explicit query value takes precedence over browser language.
- Localize interface surfaces, module display names, and operation summaries from the documentation-only `api_reference` namespace.
- Navigation, search results, and body headings use the same localized operation title. Missing keys are fatal; never mix fallback English into a Chinese reference.
- Keep parameter, response, Schema, example, and detailed-description text in canonical English for this scope.

## Interaction and accessibility

- Search indexes the localized in-memory document, including module names and operation titles.
- The interactive-testing notice has a visible, keyboard-operable close button with a localized accessible name. Dismissal is kept in `sessionStorage` for the current tab only, so a new browsing session receives the write-risk reminder again.
- Keep stable anchors based on canonical tag code, HTTP method, and path. Do not localize `operationId`, tag codes, paths, methods, schemas, or security definitions.
- Preserve Scalar keyboard navigation, visible focus, skip link, reduced-motion behavior, and no horizontal document overflow at 375 px.

## Packaging and security

- Fetch only `/openapi/openapi.yaml`, parse it in the independent page, and expose the localized object to Scalar without writing locale-specific specs.
- Direct download remains the byte-identical canonical YAML. YAML parsing, Scalar, and `api_reference` catalogs must not enter the Admin application graph.
- Interactive requests continue to use `credentials: omit`, the canonical `Accept-Language`, no persisted authorization, no external fonts, telemetry, Agent, remote proxy, or plugin URLs.

## Verification matrix

- Chromium 1440 × 900 and 375 × 812 for `zh-CN` and `en-US`.
- Initial collapsed navigation, module expansion, operation navigation, localized module search, localized operation search, body-title consistency, and stable anchor.
- Public health request, Cookie omission, language header, canonical YAML bytes, no external requests, console errors, serious/critical axe findings, or horizontal overflow.
