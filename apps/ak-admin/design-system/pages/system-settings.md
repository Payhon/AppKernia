# System configuration page override

- Use a category-first master/detail layout: categories at left, selected category configuration at right.
- Category rows show localized name, localized description, item count, and configured-secret status where relevant.
- Persist category and filters in the URL. A refresh or back navigation must retain context.
- Keep custom configuration CRUD available, but initial system items should read as a coherent form rather than an undifferentiated table.
- Secret fields never reveal existing plaintext. Show configured/unconfigured state and use the existing rotate-secret flow.
- Cloud storage shows driver summary, upload constraints, and a reusable test uploader only to users with the matching permissions.
- Use skeleton/spinner feedback for loading and an explicit retry affordance for failures.
