# AKDOCS-003 — Architecture, story, and product evidence

## Page intent

- Help a first-time visitor understand what AppKernia is before asking them to
  install anything.
- Explain the shared contract across Admin, Server, and Mobile with rendered
  diagrams, not raw source-shaped ASCII art.
- Build confidence with traceable local acceptance screenshots while stating
  platform and deployment boundaries honestly.

## Layout decisions

- `guide/what-is-appkernia` uses a readable prose column, a four-part naming
  explanation, a decision table, a future path, and two responsive galleries.
- Admin evidence uses a two-column wide-screen grid. Mobile evidence uses four
  narrow cards so the native surfaces remain legible; both collapse to one
  column where needed.
- Diagram shells use one border, neutral surfaces, 20 px radius, and 24–32 px
  internal spacing. They can scroll internally, but the page cannot overflow.
- Architecture is split into an ecosystem overview plus dedicated Admin,
  Server, and Mobile diagrams. Supporting prose follows every diagram.

## Content and evidence boundaries

- The name is explained as the project's intended product narrative:
  `App + Kern + -ia`; it is not presented as an external linguistic fact.
- Screenshots come from isolated local Admin/API acceptance and the named iOS
  simulator run. Test accounts and fixtures are not real users or customers.
- iOS simulator evidence does not prove iOS hardware, Android, HarmonyOS,
  HBuilderX one-click resource sync, or production deployment.
- Roadmap items are directions without dates or shipped-status claims.

## Responsive and accessible behavior

- Every Mermaid block has `accTitle` and `accDescr`, a labelled group wrapper,
  and a human-readable summary below it.
- Images have localized alt text, explicit intrinsic dimensions, lazy loading,
  and captions that repeat the evidence boundary without relying on color.
- At 375 px, diagrams and tables scroll inside their own shell; galleries use a
  single column and the document itself remains free of horizontal overflow.
