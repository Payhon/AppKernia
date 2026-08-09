# AKDOCS-003 decisions

## Rendering

- Use `rspress-plugin-mermaid@1.0.1`. The current Rspress community-plugin
  catalog lists it for Mermaid rendering, and this release declares support for
  `@rspress/core ^2.0.0`.
- Configure Mermaid with strict security, responsive flow/sequence layouts, and
  repository-controlled source only.
- Wrap every diagram in an accessible labelled shell and provide a prose
  summary so SVG is not the sole representation of meaning.

## Content

- Add `guide/what-is-appkernia` as a first-class Guide route in both locales.
- Describe the 2026 origin as the maintainer's motivation, not as a market-share
  or uniqueness claim that cannot be proven by the repository.
- Explain `App + Kern + -ia` as the intended brand story.
- Separate current engineering contracts from roadmap direction and link readers
  to live implementation status rather than inventing maturity metrics.

## Product evidence

- Reuse eight repository acceptance screenshots: four Admin local-browser
  surfaces and four iPhone 16 Pro / iOS 18.6 simulator surfaces.
- Publish stripped copies under the docs public directory; retain original files
  in their acceptance artifact locations as provenance.
- Do not use AI-generated mobile surfaces, customer logos, production data, or
  claims for Android/HarmonyOS runtime acceptance.
