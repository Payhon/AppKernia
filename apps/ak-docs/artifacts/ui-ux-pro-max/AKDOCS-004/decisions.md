# AKDOCS-004 decisions

## Root cause and rendering

- The content was already authored, but Rspress 2.0.19's stock `HomeLayout`
  renders frontmatter Hero/features and omits the MDX body. A local HomeLayout
  now renders `Content` after the existing custom Hero and preserves extension
  points, footer, SSG Markdown output, and a single main landmark.
- Text inside MDX JSX uses explicit expressions so the build does not emit
  invalid nested paragraphs during server rendering.

## Features and technology

- Add six semantic feature links for sessions, authorization/tenancy, i18n,
  OpenAPI, AK UI, and verification evidence.
- Show nine stack cards. React, TypeScript, Vite, Go, PostgreSQL, OpenAPI,
  Docker, and Ant Design use `simple-icons@16.27.0`; the unchanged official
  DCloud uni-app image is self-hosted in an SVG wrapper.
- Logos identify dependencies only and do not imply vendor endorsement.

## Product sliders

- Use one Admin/Web slider and one Mobile slider, each with four existing
  repository evidence images and bilingual alt/caption text.
- Do not auto-play. Provide previous/next, dot navigation, Arrow Left/Right,
  visible focus, a counter, and a polite screen-reader announcement.
- Admin evidence remains local Docker/API browser evidence; Mobile remains an
  iPhone 16 Pro / iOS 18.6 simulator boundary.
