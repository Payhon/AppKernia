# System Configurations Override

- Layout: page heading, module/group URL filters, responsive configuration table, create/edit drawer, isolated secret replacement modal.
- Secret rows show status and key version only; never render a masked value that could be mistaken for the actual secret.
- Public and secret are mutually exclusive. Secret writes require explicit confirmation and success feedback.
- Typed values use the matching control: boolean switch, numeric input, datetime input, JSON editor textarea, string input.
- The version supplied by the server is submitted on updates; conflict presents a localized reload action.
- At 375 px retain key/name/status/action columns with internal table scroll and no page-level overflow.
