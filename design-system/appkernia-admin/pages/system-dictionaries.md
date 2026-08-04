# System Dictionaries Override

- Desktop: dictionary type list on the left, selected type and localized items on the right.
- Narrow screens: type selector above the item table; no gesture-only navigation.
- Selected type, type filters, item filters, locale, sort and pagination live in URL Search Params.
- System types and their items display a lock label; all mutation buttons are absent or disabled with a reason.
- Labels show stored locale and fallback source. Supported writable locales are only `zh-CN` and `en-US`; locale-neutral is allowed only when the backend contract permits it.
- Item create/edit uses controlled forms, explicit labels, stable values and color as supplemental metadata only.
