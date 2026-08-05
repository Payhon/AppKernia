# System Configurations Override

- Layout: page heading with a labelled form/table segmented control, category navigation, and one responsive content card. Form mode is the default; table mode remains the definition-management workspace.
- Form mode renders the selected category directly from the server-provided definitions. It uses a two-column grid on desktop and one column at 767 px and below, with a single save action for all changed values.
- The form/table mode is URL-addressable. The default form mode is omitted from the query string; table mode uses `mode=table` so reload and sharing preserve the selected workspace.
- Changing category or mode with dirty form values requires an explicit discard confirmation. Browser unload also warns while unsaved values exist.
- Dynamic form controls are controlled and type-matched: boolean switch, enum select, numeric input, datetime input, JSON textarea, secret password input, and string input/textarea.
- Each control has a persistent label, definition key, permission/lock state, validation feedback, and keyboard-visible focus. Async save feedback is announced through status/alert regions.
- Secret values always start empty and use “leave blank to keep unchanged”; only non-empty dirty secret inputs call the isolated secret replacement API.
- Unified save is a single UI action over existing versioned update/secret endpoints. Partial success must be reported; failed draft values remain dirty for correction or retry.
- Table mode keeps the existing server-side filters, responsive configuration table, create/edit drawer, and isolated secret replacement modal.
- Secret rows show status and key version only; never render a masked value that could be mistaken for the actual secret.
- Public and secret are mutually exclusive. Secret writes require explicit confirmation and success feedback.
- Typed values use the matching control: boolean switch, numeric input, datetime input, JSON editor textarea, string input.
- The version supplied by the server is submitted on updates; conflict presents a localized reload action.
- At 375 px retain key/name/status/action columns with internal table scroll and no page-level overflow.
