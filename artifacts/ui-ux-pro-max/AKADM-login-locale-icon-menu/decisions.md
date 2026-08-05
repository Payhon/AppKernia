# Decisions

1. Reuse `LocaleSwitcher` with its existing `icon` variant instead of creating a login-only dropdown.
2. Opt the shared `AuthFrame` into the icon variant so login, registration, password recovery, and reset pages remain visually consistent.
3. Keep the default `select` variant available for existing component compatibility and authenticated persistence-failure testing.
4. Keep Ant Design `Dropdown` selectable semantics and `selectedKeys`, so current language is not communicated by color alone.
5. Add the persistence error relationship to the icon trigger through `aria-describedby`.
6. Do not add new translations: the existing `profile.language.title` and `common.language.*` keys already cover the accessible label and menu text in both locales.
7. This request supersedes the earlier header-utilities decision that anonymous authentication pages should retain a text select.
