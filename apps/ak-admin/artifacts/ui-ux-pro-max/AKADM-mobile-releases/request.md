# Request

Build the AppKernia Admin mobile release-policy page from the final `/admin-api/v1/mobile/releases` contract.

Required capabilities: platform filtering, list, create/edit Drawer, Android/iOS/Harmony policies, current/minimum SemVer, HTTPS upgrade URL, active state, bilingual release notes, optimistic `lock_version` conflict handling, permission gates, responsive layouts, bilingual runtime switching, screenshots, and axe validation.

Constraints: React + TanStack Query + RHF/Zod + Ant Design; preserve the existing AppKernia Admin Master design system; do not modify backend code.
