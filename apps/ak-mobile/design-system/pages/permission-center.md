# Permission Center Override

- Use the existing grouped system-settings surface with one section per registered capability.
- Each row contains an AK icon well, capability name, short purpose, textual state and one explicit action.
- Keep OS authorization, Push preference and server registration as separate labelled values.
- A denied or restricted state offers a system-settings action; a not-determined state offers a request action; authorized state does not show a redundant prompt action.
- Refresh status on `onShow` without flashing the whole page. Announce recoverable errors locally.
- Future capabilities remain absent until their feature and native adapter are compiled into the current build.
