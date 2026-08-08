---
title: Internationalization
description: The language contract shared by AppKernia Backend, Admin, and Mobile.
---

# Internationalization

The first release accepts canonical `zh-CN` and `en-US`. The default and final fallback are `zh-CN`. Aliases such as `zh`, `zh-Hans`, `zh_CN`, and `en` are normalized first.

For signed-in users:

```text
server user locale
> explicit client choice
> Accept-Language / device locale
> zh-CN
```

Every request sends `Accept-Language`; the server returns `Content-Language` and adds `Vary: Accept-Language` to language-dependent cacheable responses.

- Business logic uses stable error codes, never translated display messages.
- Time stays RFC 3339 UTC and money stays an integer in the smallest currency unit.
- Admin synchronizes i18next, Ant Design, Day.js, HTML `lang`, and titles.
- Mobile synchronizes AkI18n, navigation titles, TabBar, AK UI, and later requests.
- Both catalogs must have identical keys and placeholder sets.
