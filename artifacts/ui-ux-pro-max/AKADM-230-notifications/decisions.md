# AKADM-230 UI Decisions

- Keep the established Admin shell, Ant Design tokens, typography, and compact data-table rhythm.
- Split the domain into four static routes: announcements, in-app messages, templates, and deliveries.
- Use a structured editor with plain, Markdown, or sanitized HTML preview. Never render unsanitized server or user HTML.
- Show the resolved recipient count and a bounded recipient preview before publish; publish confirmation repeats the exact count and message title.
- Expose delivery error code and safe summary only. Never display encrypted target data, raw provider payloads, or credentials.
- Retry is enabled only for failed deliveries and requires a separate confirmation; the row reports loading and terminal feedback.
- Validate both locales at 1440 and 375 pixels; capture detail, confirmation, error, and empty states where applicable.

