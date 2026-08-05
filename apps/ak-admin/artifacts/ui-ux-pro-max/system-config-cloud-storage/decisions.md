# Design decisions

1. Preserve `src/app/theme.ts` and the existing Ant Design component language instead of replacing the application shell.
2. Use category-first navigation inspired by the reference page, with localized AppKernia category metadata and URL-persisted selection.
3. Keep the generic CRUD/secret rotation capabilities for extensibility; provide richer rendering for catalog-backed initial items.
4. Build one policy-aware uploader and reuse it in the file page, file picker, and storage settings context.
5. Never render configured secrets, bucket names, object keys, or credentials. Secret changes stay explicit and separately audited.
6. Treat local storage as development-only and expose S3-compatible/MinIO as the production cloud drivers.
