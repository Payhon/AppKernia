# AKADM-150 Review Checklist

- [x] Backend tenant scope and exact read/revoke permissions verified.
- [x] Refresh-token family and immutable audit behavior verified.
- [x] Current-session warning and authentication cleanup verified.
- [ ] URL filters/pagination/sort restore after navigation and reload.
- [x] All visible copy uses matching zh-CN/en-US keys.
- [x] Loading, empty, error, retry, success and destructive-confirm states verified.
- [x] Keyboard focus, reduced motion and axe serious/critical checks verified.
- [ ] 375/768/1024/1440 screenshots reviewed.
- [x] Latest API/Admin production images and PostgreSQL integration verified.

Notes: URL-backed filters and server pagination were verified, but this page has no sort contract and reload restoration was not separately exercised. Screenshots were reviewed at 375 and 1440; 768/1024 were not captured for this page, so both broader items remain unchecked.
