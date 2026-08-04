# AKADM Header Utilities Request

## Source

- User reference screenshot: `/var/folders/52/mkm9shln40x3hjzr_swl3qfc0000gp/T/codex-clipboard-f97fbf60-004e-415b-9c34-1f42037f3101.png`

## Request

1. Replace visible username and sign-out text in the Admin header with a circular Avatar account trigger.
2. Use the stored avatar when present and an uppercase initial fallback otherwise.
3. Account dropdown shows current user, roles, Personal Center, and Sign Out.
4. Replace the header language select with a language icon and selected-state locale dropdown.
5. Add a fullscreen icon before language; toggle browser fullscreen and restore mode on second click.

## Constraints

- Preserve the text language select on anonymous authentication pages.
- Use the protected avatar Blob flow rather than unauthenticated image URLs.
- Preserve `zh-CN` / `en-US`, keyboard focus, responsive layout, and secure sign-out behavior.
