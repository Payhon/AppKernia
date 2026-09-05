# AKDOCS-007 — akone installation and deployment

## Page intent

- Let a developer compare every supported distribution channel without losing
  the current release-state boundary.
- Put the shortest working path first, then cover durable SQLite storage,
  administrator bootstrap, foreground start, service supervision, backup, and
  upgrade behavior.

## Installation tabs

- Use the shared `AkoneInstallTabs` component so homepage and guide commands do
  not drift.
- Source build remains the default. Shell, npm, and manual Release are live for
  `0.5.0-preview.3`; Homebrew remains visibly unavailable until a signed stable
  release.
- Tab labels must remain usable at 375 px without page-level overflow. Every tab
  is a 44 px button with visible keyboard focus and complete ARIA relationships.

## Content boundaries

- Never publish passwords, signing keys, tokens, or copy-paste production
  secrets. Point users to `config init` and `config validate` while naming the
  required production key variables.
- Package-manager deployments use an explicit `AK_SQLITE_PATH` or YAML path so
  data does not move with npm or Homebrew version directories.
- State that SQLite currently covers the standalone IAM/session/Dashboard path;
  direct readers to PostgreSQL for the complete application module set.
