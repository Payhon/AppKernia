# Decisions

- Provider schemas come from the server catalog; the UI does not expose arbitrary JSON.
- Secrets have a separate rotation form and never repopulate an input.
- Activation requires a ready preflight; test targets come from registered devices rather than raw tokens.
- Environment is explicit in filters, forms and confirmations.
- Acceptance and open statistics use precise, non-interchangeable labels.

## 2026-08-29 application guidance

- The question-mark button is the rightmost page-heading action and does not require a selected App because application guidance is provider-wide.
- The provider order follows the existing compiled capability registry: APNs, FCM, Huawei Android, HONOR, Xiaomi, OPPO, vivo, Meizu, HarmonyOS NEXT.
- Application steps and field explanations live in the bilingual catalog; stable provider, field, and official-link structure lives in a typed registry.
- Every external resource is HTTPS, opens in a new browsing context with `noopener noreferrer`, and is restricted by tests to official vendor hostnames.
- Documentation was checked on 2026-08-29. HONOR IAM was corrected to `iam.developer.honor.com`; Xiaomi regional API hosts were aligned with its current global-domain documentation.
- Category values for vivo and HarmonyOS are never invented by the UI; users are instructed to enter the uppercase value approved in their own provider console.
