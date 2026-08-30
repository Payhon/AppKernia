# Request

Design a production Admin flow for reusable tenant share provider configurations under System Settings, plus an App-level binding Drawer. One tenant may own multiple configurations. One App may bind one active configuration per provider. Phase one registers WeChat and must preserve extension seams for future providers. The UI must cover lifecycle, optimistic locking, native Android/iOS/HarmonyOS identity, secret non-disclosure, preflight, system-share fallback, responsive table/card layouts, and zh-CN/en-US.

## 2026-08-29 navigation visibility follow-up

Diagnose why the seeded and authorized Share Configurations menu was absent from the local Admin navigation, then restore it without changing the approved visual hierarchy.

## 2026-08-29 application action menu follow-up

Replace the wide row of text actions in the Application Management table with one compact action Dropdown. Every available action must retain its permission and lifecycle behavior and include a suitable icon.

## 2026-08-29 page gutter follow-up

Align the Share Configurations page container with the rest of Admin. The page and its rightmost content must retain the shared responsive outer gutter instead of touching the shell content boundary.

## 2026-08-29 WeChat application guide follow-up

Add a question-mark help entry beside the share configuration Drawer title. It must explain the WeChat Open Platform application process step by step, expose useful official links in a new browsing context, use friendly independent bilingual copy, and remain usable on desktop and 375px layouts.
