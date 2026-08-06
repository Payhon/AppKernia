# System configuration page override

- Use a category-first master/detail layout: categories at left, selected category configuration at right.
- Category rows show localized name, localized description, item count, and configured-secret status where relevant.
- Persist category and filters in the URL. A refresh or back navigation must retain context.
- Keep custom configuration CRUD available, but initial system items should read as a coherent form rather than an undifferentiated table.
- Secret fields never reveal existing plaintext. Show configured/unconfigured state and use the existing rotate-secret flow.
- Cloud storage shows driver summary, upload constraints, and a reusable test uploader only to users with the matching permissions.
- Dictionary management follows `dictionary-management.md`: a namespace-category/type hierarchy at left and an editable item table at right. System types and built-in rows expose text lock/override state, while extension policy and tenant overrides remain visible without relying on color.
- Dictionary-backed configuration fields show loading, retry, empty, and disabled-current-value states; provider-specific fields appear conditionally without clearing secrets owned by another provider.
- Notification template test delivery uses a labeled drawer. SMS displays a billable-action warning and requires explicit confirmation before enqueueing.
- Region management retains the lazy tree table. Writable rows use explicit text actions; add/edit opens a labeled drawer with immutable hierarchy fields, while delete requires confirmation and never cascades. Narrow screens keep the table scroll region keyboard-focusable.
- Use skeleton/spinner feedback for loading and an explicit retry affordance for failures.
