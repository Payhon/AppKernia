# Decisions

1. Use a reusable `AkCreatableSelect` backed by Ant Design tags mode, controlled as a single string through React Hook Form.
2. Represent the empty persisted value with an internal-only sentinel so the user sees an explicit “Default (not set)” selection. The sentinel is never submitted to the API.
3. Provide seven named color presets and six semantic style presets in both locales. Presets are presentation helpers, not business dictionaries.
4. Keep arbitrary color and class strings supported; typing and pressing Enter creates the value.
5. Only hexadecimal custom colors receive a live swatch. Only compiled style preset classes receive a live style preview. Arbitrary custom classes are displayed as text and are not attached to the management UI DOM.
6. Show both color and CSS class in the table when both are configured instead of hiding one behind the other.
