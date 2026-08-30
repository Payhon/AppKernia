---
title: Client configuration
description: Manage share bindings, scanner allowlists, and future mobile configuration from one app-level entry.
---

# Client configuration

Open Admin → Applications and choose Client configuration from the target app's row actions. The modal groups app-specific client capabilities into tabs, currently Share configuration followed by Scanner configuration. Future capabilities register another code-owned tab instead of adding another row action.

The entry is visible when the user has read access to any client-configuration tab. Each tab still owns authorization, loading, validation, and save feedback. Switching tabs never auto-saves; closing with unsaved changes asks for confirmation. There is no ambiguous modal-wide Save button.

## Share configuration

The Share configuration tab manages the binding between a reusable platform identity and this app, including enabled state, share scenes, Share Origin, system-share fallback, preflight, save, and unbind.

- `app.share_binding.read` allows viewing and preflight.
- `app.share_binding.update` allows saving and unbinding.

The existing global Share configuration page under System remains the source for reusable platform identities; this app-level tab only owns the binding. Changes to native identities such as WeChat or Apple usually require a new client export and package before they take effect.

## Scanner configuration

The Scanner configuration tab contains Allow scanned web pages to open inside the app plus an editable hostname-rule list. Turning the switch off keeps the list for later reuse. Enabling requires at least one valid rule, with at most 100 rules.

Supported rules:

- `example.com` matches that exact hostname only.
- `*.example.com` matches any subdomain depth, such as `a.example.com` and `a.b.example.com`, but not the root `example.com`.

The server lowercases, removes trailing dots, deduplicates, and sorts the rules. Only ASCII/Punycode DNS names are accepted. Schemes, paths, query strings, credentials, IP addresses, `localhost`, non-443 ports, and public-suffix wildcards are rejected. The client performs immediate structural checks; server validation errors map back to the corresponding row.

- `app.scanner_config.read` allows viewing.
- `app.scanner_config.update` allows editing and saving.

Saving uses an independent optimistic lock. If another administrator has changed the configuration, the tab reports the conflict and reloads the latest version instead of overwriting it.

## Runtime effect

The scanner allowlist is delivered through `/api/v1/public/config` and does not require repackaging. The client must successfully refresh and parse the latest configuration before each scan can open a page; an older server, request failure, or malformed data fails closed. Plain text and URLs outside the allowlist only appear as scan results.

Scan content is never uploaded, persisted, or logged. Audit records include only before/after summaries of administrator configuration changes, never what a user scanned.

Continue with the [Scanner capability](../mobile-components/scanner) and [Security model](../concepts/security).
