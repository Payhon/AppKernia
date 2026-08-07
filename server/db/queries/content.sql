-- name: ContentPublishedArticleExists :one
SELECT EXISTS (
  SELECT 1 FROM content.articles
  WHERE tenant_id = sqlc.arg('tenant_id')
    AND app_id = sqlc.arg('app_id')
    AND id = sqlc.arg('article_id')
    AND status = 'published'
);

-- name: ContentUpsertBookmark :execrows
INSERT INTO content.article_bookmarks(tenant_id, app_id, user_id, article_id)
VALUES (sqlc.arg('tenant_id'), sqlc.arg('app_id'), sqlc.arg('user_id'), sqlc.arg('article_id'))
ON CONFLICT (tenant_id, user_id, article_id) DO NOTHING;

-- name: ContentListPublishedCategories :many
SELECT c.id, c.slug, c.sort_order, t.name, t.description
FROM content.categories c
JOIN LATERAL (
  SELECT name, description
  FROM content.category_translations
  WHERE category_id = c.id AND locale IN (sqlc.arg('locale'), 'zh-CN')
  ORDER BY (locale = sqlc.arg('locale')) DESC
  LIMIT 1
) t ON TRUE
WHERE c.tenant_id = sqlc.arg('tenant_id')
  AND c.app_id = sqlc.arg('app_id')
  AND c.status = 'active'
ORDER BY c.sort_order ASC, c.slug ASC, c.id ASC;

-- name: ContentDeleteBookmark :execrows
DELETE FROM content.article_bookmarks
WHERE tenant_id = sqlc.arg('tenant_id')
  AND app_id = sqlc.arg('app_id')
  AND user_id = sqlc.arg('user_id')
  AND article_id = sqlc.arg('article_id');
