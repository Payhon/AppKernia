// Package repository is the PostgreSQL adapter for content.Repository.  SQL
// belongs here; transports never receive a pool or a generated query object.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	objects storagedomain.ObjectStore
}

func NewPostgres(pool *pgxpool.Pool, objects storagedomain.ObjectStore) *Postgres {
	return &Postgres{pool: pool, queries: db.New(pool), objects: objects}
}

const categorySelect = `SELECT c.id,c.parent_id,c.image_file_id,c.slug,c.status,c.sort_order,c.lock_version,c.created_at,c.updated_at,
	COALESCE((SELECT jsonb_object_agg(locale,jsonb_build_object('name',name,'description',description)) FROM content.category_translations WHERE category_id=c.id),'{}'::jsonb)
	FROM content.categories c`
const articleSelect = `SELECT a.id,a.category_id,a.slug,a.status,a.content_type,a.allow_comments,a.pinned,a.featured,a.is_latest,a.sort_order,a.cover_file_id,a.reading_minutes,a.topic_id,a.video_source_type,a.video_file_id,a.video_external_url,a.video_duration_seconds,a.lock_version,a.published_at,a.created_at,a.updated_at,
	COALESCE((SELECT jsonb_object_agg(locale,jsonb_build_object('title',title,'summary',summary,'body_format',body_format,'body',body)) FROM content.article_translations WHERE article_id=a.id),'{}'::jsonb)
	FROM content.articles a`

func scanCategory(row pgx.Row) (content.Category, error) {
	var x content.Category
	var raw []byte
	err := row.Scan(&x.ID, &x.ParentID, &x.ImageFileID, &x.Slug, &x.Status, &x.SortOrder, &x.LockVersion, &x.CreatedAt, &x.UpdatedAt, &raw)
	if err != nil {
		return x, err
	}
	err = json.Unmarshal(raw, &x.Translations)
	return x, err
}
func scanArticle(row pgx.Row) (content.Article, error) {
	var x content.Article
	var raw []byte
	err := row.Scan(&x.ID, &x.CategoryID, &x.Slug, &x.Status, &x.ContentType, &x.AllowComments, &x.Pinned, &x.Featured, &x.Latest, &x.SortOrder, &x.CoverFileID, &x.ReadingMinutes, &x.TopicID, &x.VideoSourceType, &x.VideoFileID, &x.VideoExternalURL, &x.VideoDurationSeconds, &x.LockVersion, &x.PublishedAt, &x.CreatedAt, &x.UpdatedAt, &raw)
	if err != nil {
		return x, err
	}
	if err = json.Unmarshal(raw, &x.Translations); err != nil {
		return x, err
	}
	x.CoverURL = adminCoverURL(x.CoverFileID)
	return x, nil
}
func adminCoverURL(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := "/admin-api/v1/files/" + id.String() + "/content"
	return &value
}
func mobileCoverURL(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := "/api/v1/public/content/assets/" + id.String()
	return &value
}

func (r *Postgres) ListCategories(ctx context.Context, tenant, appID uuid.UUID, f content.PageFilter) (content.CategoryPage, error) {
	appID, err := r.scopedApp(ctx, tenant, appID)
	if err != nil {
		return content.CategoryPage{}, err
	}
	where, args := []string{"c.tenant_id=$1", "c.app_id=$2"}, []any{tenant, appID}
	add := func(q string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(q, len(args))) }
	if f.Query != "" {
		add("(c.slug ILIKE '%%'||$%[1]d||'%%' OR EXISTS(SELECT 1 FROM content.category_translations t WHERE t.category_id=c.id AND t.name ILIKE '%%'||$%[1]d||'%%'))", f.Query)
	}
	if f.Status != "" {
		add("c.status=$%d", f.Status)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM content.categories c WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return content.CategoryPage{}, err
	}
	order := "c.sort_order,c.slug,c.id"
	if f.Sort == "updated_desc" {
		order = "c.updated_at DESC,c.id"
	} else if f.Sort == "slug" {
		order = "c.slug,c.id"
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, categorySelect+" WHERE "+whereSQL+fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args)), args...)
	if err != nil {
		return content.CategoryPage{}, err
	}
	defer rows.Close()
	out := content.CategoryPage{Items: []content.Category{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		x, e := scanCategory(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) GetCategory(ctx context.Context, tenant, appID, id uuid.UUID) (content.Category, error) {
	appID, e := r.scopedApp(ctx, tenant, appID)
	if e != nil {
		return content.Category{}, e
	}
	x, e := scanCategory(r.pool.QueryRow(ctx, categorySelect+" WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.id=$3", tenant, appID, id))
	if e == nil {
		x.ImageURL = adminCoverURL(x.ImageFileID)
	}
	return x, mapNotFound(e)
}
func (r *Postgres) CreateCategory(ctx context.Context, p content.Principal, x content.Category) (content.Category, error) {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return x, e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return content.Category{}, e
	}
	defer tx.Rollback(ctx)
	if e = validCategoryReferences(ctx, tx, p.TenantID, p.AppID, uuid.Nil, x.ParentID, x.ImageFileID); e != nil {
		return x, e
	}
	e = tx.QueryRow(ctx, `INSERT INTO content.categories(tenant_id,app_id,parent_id,image_file_id,slug,status,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, p.TenantID, p.AppID, x.ParentID, x.ImageFileID, x.Slug, x.Status, x.SortOrder).Scan(&x.ID)
	if e != nil {
		return x, mapWrite(e)
	}
	if e = upsertCategoryTranslations(ctx, tx, p.TenantID, x.ID, x.Translations); e != nil {
		return x, e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='content' AND entity_type='content.category' AND entity_id=$2`, p.TenantID, x.ID); e != nil {
		return x, e
	}
	if x.ImageFileID != nil {
		if _, e = tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'content','content.category',$3,'image') ON CONFLICT DO NOTHING`, *x.ImageFileID, p.TenantID, x.ID); e != nil {
			return x, e
		}
	}
	out, e := getCategory(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if e = audit(ctx, tx, p, "content.category.create", "content.category", x.ID, "POST", nil, safeCategory(out)); e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	return out, nil
}
func (r *Postgres) UpdateCategory(ctx context.Context, p content.Principal, x content.Category) (content.Category, error) {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return x, e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return content.Category{}, e
	}
	defer tx.Rollback(ctx)
	before, e := getCategory(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if e = validCategoryReferences(ctx, tx, p.TenantID, p.AppID, x.ID, x.ParentID, x.ImageFileID); e != nil {
		return x, e
	}
	tag, e := tx.Exec(ctx, `UPDATE content.categories SET parent_id=$1,image_file_id=$2,slug=$3,status=$4,sort_order=$5,lock_version=lock_version+1 WHERE tenant_id=$6 AND app_id=$7 AND id=$8 AND lock_version=$9`, x.ParentID, x.ImageFileID, x.Slug, x.Status, x.SortOrder, p.TenantID, p.AppID, x.ID, x.LockVersion)
	if e != nil {
		return x, mapWrite(e)
	}
	if tag.RowsAffected() == 0 {
		return x, content.ErrConflict
	}
	if e = upsertCategoryTranslations(ctx, tx, p.TenantID, x.ID, x.Translations); e != nil {
		return x, e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='content' AND entity_type='content.category' AND entity_id=$2`, p.TenantID, x.ID); e != nil {
		return x, e
	}
	if x.ImageFileID != nil {
		if _, e = tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'content','content.category',$3,'image') ON CONFLICT DO NOTHING`, *x.ImageFileID, p.TenantID, x.ID); e != nil {
			return x, e
		}
	}
	out, e := getCategory(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if e = audit(ctx, tx, p, "content.category.update", "content.category", x.ID, "PATCH", safeCategory(before), safeCategory(out)); e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	return out, nil
}
func (r *Postgres) DeleteCategory(ctx context.Context, p content.Principal, id uuid.UUID, v int32) error {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	before, e := getCategory(ctx, tx, p.TenantID, p.AppID, id)
	if e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, `DELETE FROM content.categories c WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.id=$3 AND c.lock_version=$4 AND NOT EXISTS(SELECT 1 FROM content.categories child WHERE child.parent_id=c.id) AND NOT EXISTS(SELECT 1 FROM content.article_categories ac WHERE ac.article_id IN (SELECT id FROM content.articles WHERE tenant_id=c.tenant_id AND app_id=c.app_id) AND ac.category_id=c.id)`, p.TenantID, p.AppID, id, v)
	if e != nil {
		return mapWrite(e)
	}
	if tag.RowsAffected() == 0 {
		return content.ErrConflict
	}
	if e = audit(ctx, tx, p, "content.category.delete", "content.category", id, "DELETE", safeCategory(before), map[string]bool{"deleted": true}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ListArticles(ctx context.Context, tenant, appID uuid.UUID, f content.PageFilter) (content.ArticlePage, error) {
	appID, err := r.scopedApp(ctx, tenant, appID)
	if err != nil {
		return content.ArticlePage{}, err
	}
	where, args := []string{"a.tenant_id=$1", "a.app_id=$2"}, []any{tenant, appID}
	add := func(q string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(q, len(args))) }
	if f.Query != "" {
		add("(a.slug ILIKE '%%'||$%[1]d||'%%' OR EXISTS(SELECT 1 FROM content.article_translations t WHERE t.article_id=a.id AND (t.title ILIKE '%%'||$%[1]d||'%%' OR t.summary ILIKE '%%'||$%[1]d||'%%')))", f.Query)
	}
	if f.Status != "" {
		add("a.status=$%d", f.Status)
	}
	if f.CategoryID != nil {
		add("EXISTS(SELECT 1 FROM content.article_categories ac WHERE ac.article_id=a.id AND ac.category_id=$%d)", *f.CategoryID)
	}
	if f.TopicID != nil {
		add("a.topic_id=$%d", *f.TopicID)
	}
	if f.ContentType != "" {
		add("a.content_type=$%d", f.ContentType)
	}
	if f.Featured != nil {
		add("a.featured=$%d", *f.Featured)
	}
	ws := strings.Join(where, " AND ")
	var total int64
	if e := r.pool.QueryRow(ctx, "SELECT count(*) FROM content.articles a WHERE "+ws, args...).Scan(&total); e != nil {
		return content.ArticlePage{}, e
	}
	order := "a.updated_at DESC,a.id"
	switch f.Sort {
	case "published_desc":
		order = "a.published_at DESC NULLS LAST,a.id"
	case "sort_order":
		order = "a.sort_order DESC,a.id"
	case "slug":
		order = "a.slug,a.id"
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, e := r.pool.Query(ctx, articleSelect+" WHERE "+ws+fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args)), args...)
	if e != nil {
		return content.ArticlePage{}, e
	}
	defer rows.Close()
	out := content.ArticlePage{Items: []content.Article{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		x, e := scanArticle(rows)
		if e != nil {
			return out, e
		}
		if e = r.hydrateArticle(ctx, r.pool, tenant, appID, &x); e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) GetArticle(ctx context.Context, tenant, appID, id uuid.UUID) (content.Article, error) {
	appID, e := r.scopedApp(ctx, tenant, appID)
	if e != nil {
		return content.Article{}, e
	}
	x, e := scanArticle(r.pool.QueryRow(ctx, articleSelect+" WHERE a.tenant_id=$1 AND a.app_id=$2 AND a.id=$3", tenant, appID, id))
	if e == nil {
		e = r.hydrateArticle(ctx, r.pool, tenant, appID, &x)
	}
	return x, mapNotFound(e)
}
func (r *Postgres) CreateArticle(ctx context.Context, p content.Principal, x content.Article) (content.Article, error) {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return x, e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return content.Article{}, e
	}
	defer tx.Rollback(ctx)
	if e = validInformationReferences(ctx, tx, p.TenantID, p.AppID, x); e != nil {
		return x, e
	}
	e = tx.QueryRow(ctx, `INSERT INTO content.articles(tenant_id,app_id,category_id,slug,status,content_type,allow_comments,pinned,featured,is_latest,sort_order,cover_file_id,reading_minutes,topic_id,video_source_type,video_file_id,video_external_url,video_duration_seconds,created_by,updated_by) VALUES($1,$2,$3,$4,'draft',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$18) RETURNING id`, p.TenantID, p.AppID, x.CategoryID, x.Slug, x.ContentType, x.AllowComments, x.Pinned, x.Featured, x.Latest, x.SortOrder, x.CoverFileID, x.ReadingMinutes, x.TopicID, x.VideoSourceType, x.VideoFileID, x.VideoExternalURL, x.VideoDurationSeconds, p.UserID).Scan(&x.ID)
	if e != nil {
		return x, mapWrite(e)
	}
	if e = upsertArticleTranslations(ctx, tx, x.ID, x.Translations); e != nil {
		return x, e
	}
	if e = syncArticleRelations(ctx, tx, p, x); e != nil {
		return x, e
	}
	out, e := getArticle(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if e = r.hydrateArticle(ctx, tx, p.TenantID, p.AppID, &out); e != nil {
		return x, e
	}
	if e = audit(ctx, tx, p, "content.article.create", "content.article", x.ID, "POST", nil, safeArticle(out)); e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	return out, nil
}
func (r *Postgres) UpdateArticle(ctx context.Context, p content.Principal, x content.Article) (content.Article, error) {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return x, e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return content.Article{}, e
	}
	defer tx.Rollback(ctx)
	before, e := getArticle(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if before.Status == "archived" {
		return x, content.ErrConflict
	}
	if before.Status == "published" && before.ContentType != x.ContentType {
		return x, content.ErrConflict
	}
	if e = validInformationReferences(ctx, tx, p.TenantID, p.AppID, x); e != nil {
		return x, e
	}
	tag, e := tx.Exec(ctx, `UPDATE content.articles SET category_id=$1,slug=$2,content_type=$3,allow_comments=$4,pinned=$5,featured=$6,is_latest=$7,sort_order=$8,cover_file_id=$9,reading_minutes=$10,topic_id=$11,video_source_type=$12,video_file_id=$13,video_external_url=$14,video_duration_seconds=$15,updated_by=$16,lock_version=lock_version+1 WHERE tenant_id=$17 AND app_id=$18 AND id=$19 AND lock_version=$20`, x.CategoryID, x.Slug, x.ContentType, x.AllowComments, x.Pinned, x.Featured, x.Latest, x.SortOrder, x.CoverFileID, x.ReadingMinutes, x.TopicID, x.VideoSourceType, x.VideoFileID, x.VideoExternalURL, x.VideoDurationSeconds, p.UserID, p.TenantID, p.AppID, x.ID, x.LockVersion)
	if e != nil {
		return x, mapWrite(e)
	}
	if tag.RowsAffected() == 0 {
		return x, content.ErrConflict
	}
	if e = upsertArticleTranslations(ctx, tx, x.ID, x.Translations); e != nil {
		return x, e
	}
	if e = syncArticleRelations(ctx, tx, p, x); e != nil {
		return x, e
	}
	out, e := getArticle(ctx, tx, p.TenantID, p.AppID, x.ID)
	if e != nil {
		return x, e
	}
	if e = r.hydrateArticle(ctx, tx, p.TenantID, p.AppID, &out); e != nil {
		return x, e
	}
	if e = audit(ctx, tx, p, "content.article.update", "content.article", x.ID, "PATCH", safeArticle(before), safeArticle(out)); e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	return out, nil
}
func (r *Postgres) DeleteArticle(ctx context.Context, p content.Principal, id uuid.UUID, v int32) error {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	before, e := getArticle(ctx, tx, p.TenantID, p.AppID, id)
	if e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, `DELETE FROM content.articles WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='draft' AND lock_version=$4`, p.TenantID, p.AppID, id, v)
	if e != nil {
		return mapWrite(e)
	}
	if tag.RowsAffected() == 0 {
		return content.ErrConflict
	}
	if e = audit(ctx, tx, p, "content.article.delete", "content.article", id, "DELETE", safeArticle(before), map[string]bool{"deleted": true}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (r *Postgres) TransitionArticle(ctx context.Context, p content.Principal, id uuid.UUID, v int32, state string) (content.Article, error) {
	var e error
	p.AppID, e = r.scopedApp(ctx, p.TenantID, p.AppID)
	if e != nil {
		return content.Article{}, e
	}
	tx, e := r.begin(ctx)
	if e != nil {
		return content.Article{}, e
	}
	defer tx.Rollback(ctx)
	before, e := getArticle(ctx, tx, p.TenantID, p.AppID, id)
	if e != nil {
		return content.Article{}, e
	}
	if !transitionAllowed(before.Status, state) {
		return content.Article{}, content.ErrConflict
	}
	if state == "published" {
		if e = r.hydrateArticle(ctx, tx, p.TenantID, p.AppID, &before); e != nil {
			return content.Article{}, e
		}
		if e = validInformationReferences(ctx, tx, p.TenantID, p.AppID, before); e != nil {
			return content.Article{}, e
		}
	}
	tag, e := tx.Exec(ctx, transitionArticleStatusSQL, state, p.UserID, p.TenantID, p.AppID, id, v)
	if e != nil {
		return content.Article{}, mapWrite(e)
	}
	if tag.RowsAffected() == 0 {
		return content.Article{}, content.ErrConflict
	}
	out, e := getArticle(ctx, tx, p.TenantID, p.AppID, id)
	if e != nil {
		return content.Article{}, e
	}
	if e = r.hydrateArticle(ctx, tx, p.TenantID, p.AppID, &out); e != nil {
		return content.Article{}, e
	}
	action := "content.article.publish"
	if state == "draft" {
		action = "content.article.unpublish"
	}
	if state == "archived" {
		action = "content.article.archive"
	}
	if e = audit(ctx, tx, p, action, "content.article", id, "POST", safeArticle(before), safeArticle(out)); e != nil {
		return content.Article{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return content.Article{}, e
	}
	return out, nil
}
func transitionAllowed(from, to string) bool {
	return (from == "draft" && (to == "published" || to == "archived")) || (from == "published" && (to == "draft" || to == "archived"))
}

const publicArticleSelect = `SELECT a.id,a.category_id,a.slug,a.content_type,a.allow_comments,a.pinned,a.featured,a.is_latest,a.sort_order,cover.id,a.reading_minutes,a.published_at,t.title,t.summary,t.body_format,t.body,c.id,c.parent_id,c.slug,c.sort_order,ct.name,ct.description,CASE WHEN $1::uuid IS NULL THEN NULL ELSE EXISTS(SELECT 1 FROM content.article_bookmarks b WHERE b.tenant_id=a.tenant_id AND b.app_id=a.app_id AND b.article_id=a.id AND b.user_id=$1) END,a.topic_id,a.video_source_type,a.video_external_url,a.video_duration_seconds,a.video_file_id FROM content.articles a JOIN LATERAL(SELECT * FROM content.article_translations WHERE article_id=a.id AND locale IN ($2,'zh-CN') ORDER BY (locale=$2) DESC LIMIT 1)t ON true LEFT JOIN storage.files cover ON cover.tenant_id=a.tenant_id AND cover.id=a.cover_file_id AND COALESCE(cover.metadata->>'purpose','')<>'feedback' AND cover.status='ready' AND cover.scan_status IN ('clean','skipped') AND lower(COALESCE(cover.media_type,'')) IN ('image/jpeg','image/png','image/webp') AND cover.deleted_at IS NULL LEFT JOIN content.categories c ON c.id=a.category_id AND c.tenant_id=a.tenant_id AND c.app_id=a.app_id AND c.status='active' LEFT JOIN LATERAL(SELECT * FROM content.category_translations WHERE category_id=c.id AND locale IN ($2,'zh-CN') ORDER BY (locale=$2) DESC LIMIT 1)ct ON true`

const transitionArticleStatusSQL = `UPDATE content.articles SET status=$1::varchar,published_at=CASE WHEN $1::varchar='published' THEN COALESCE(published_at,now()) WHEN $1::varchar='draft' THEN NULL ELSE published_at END,updated_by=$2,lock_version=lock_version+1 WHERE tenant_id=$3 AND app_id=$4 AND id=$5 AND lock_version=$6`

func scanPublic(row pgx.Row) (content.PublicArticle, error) {
	var x content.PublicArticle
	var cover *uuid.UUID
	var cid *uuid.UUID
	var parentID *uuid.UUID
	var cs, cn, cd *string
	var categorySort *int32
	var raw []byte
	var topicID, videoFileID *uuid.UUID
	e := row.Scan(&x.ID, &x.CategoryID, &x.Slug, &x.ContentType, &x.AllowComments, &x.Pinned, &x.Featured, &x.Latest, &x.SortOrder, &cover, &x.ReadingMinutes, &x.PublishedAt, &x.Title, &x.Summary, &x.BodyFormat, &raw, &cid, &parentID, &cs, &categorySort, &cn, &cd, &x.Bookmarked, &topicID, &x.VideoSourceType, &x.VideoURL, &x.VideoDurationSeconds, &videoFileID)
	if e != nil {
		return x, e
	}
	x.Body = json.RawMessage(raw)
	x.CoverURL = mobileCoverURL(cover)
	if cid != nil {
		x.Category = &content.PublicCategory{ID: *cid, ParentID: parentID}
		if categorySort != nil {
			x.Category.SortOrder = *categorySort
		}
		if cs != nil {
			x.Category.Slug = *cs
		}
		if cn != nil {
			x.Category.Name = *cn
		}
		if cd != nil {
			x.Category.Description = *cd
		}
	} else {
		x.CategoryID = nil
	}
	x.Topic = nil
	x.Categories = []content.PublicCategory{}
	x.Tags = []content.Tag{}
	x.Media = []content.PublicMedia{}
	if videoFileID != nil {
		x.VideoURL = mobileCoverURL(videoFileID)
	}
	_ = topicID
	return x, nil
}
func (r *Postgres) ListPublished(ctx context.Context, tenant, appID uuid.UUID, user *uuid.UUID, locale string, f content.PublicFilter) (content.PublicArticlePage, error) {
	appID, err := r.scopedApp(ctx, tenant, appID)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	args := []any{user, locale, tenant, appID}
	where := []string{"a.tenant_id=$3", "a.app_id=$4", "a.status='published'"}
	add := func(q string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(q, len(args))) }
	if f.Query != "" {
		add("(t.search_text ILIKE '%%'||$%[1]d||'%%' OR EXISTS(SELECT 1 FROM content.article_tags sat JOIN content.tags st ON st.id=sat.tag_id WHERE sat.article_id=a.id AND st.status='active' AND st.normalized_name ILIKE '%%'||$%[1]d||'%%') OR EXISTS(SELECT 1 FROM content.article_categories sac JOIN content.category_translations sct ON sct.category_id=sac.category_id WHERE sac.article_id=a.id AND sct.locale IN ($2,'zh-CN') AND (lower(sct.name) ILIKE '%%'||$%[1]d||'%%' OR lower(sct.description) ILIKE '%%'||$%[1]d||'%%')) OR EXISTS(SELECT 1 FROM content.topic_translations stt WHERE stt.topic_id=a.topic_id AND stt.locale IN ($2,'zh-CN') AND (lower(stt.name) ILIKE '%%'||$%[1]d||'%%' OR lower(stt.description) ILIKE '%%'||$%[1]d||'%%')))", strings.ToLower(f.Query))
	}
	if f.CategorySlug != "" {
		add("EXISTS(SELECT 1 FROM content.article_categories ac JOIN content.categories selected ON selected.id=ac.category_id AND selected.status='active' LEFT JOIN content.categories parent ON parent.id=selected.parent_id WHERE ac.article_id=a.id AND (selected.slug=$%[1]d OR parent.slug=$%[1]d))", f.CategorySlug)
	}
	if f.Featured != nil {
		add("a.featured=$%d", *f.Featured)
	}
	if f.ContentType != "" {
		add("a.content_type=$%d", f.ContentType)
	}
	if f.TopicSlug != "" {
		add("EXISTS(SELECT 1 FROM content.topics tp WHERE tp.id=a.topic_id AND tp.app_id=a.app_id AND tp.slug=$%d AND tp.status='active')", f.TopicSlug)
	}
	if f.Tag != "" {
		add("EXISTS(SELECT 1 FROM content.article_tags at JOIN content.tags tg ON tg.id=at.tag_id WHERE at.article_id=a.id AND tg.app_id=a.app_id AND tg.normalized_name=$%d AND tg.status='active')", strings.ToLower(strings.TrimPrefix(f.Tag, "#")))
	}
	if f.Cursor != "" {
		id, e := uuid.Parse(f.Cursor)
		if e != nil {
			return content.PublicArticlePage{}, content.ErrInvalid
		}
		add("(a.featured,a.sort_order,a.published_at,a.id) < (SELECT featured,sort_order,published_at,id FROM content.articles WHERE tenant_id=$3 AND app_id=$4 AND id=$%d AND status='published')", id)
	}
	args = append(args, f.Limit+1)
	rows, e := r.pool.Query(ctx, publicArticleSelect+" WHERE "+strings.Join(where, " AND ")+fmt.Sprintf(" ORDER BY a.featured DESC,a.sort_order DESC,a.published_at DESC,a.id DESC LIMIT $%d", len(args)), args...)
	if e != nil {
		return content.PublicArticlePage{}, e
	}
	defer rows.Close()
	out := content.PublicArticlePage{Items: []content.PublicArticle{}}
	for rows.Next() {
		x, e := scanPublic(rows)
		if e != nil {
			return out, e
		}
		if e = r.hydratePublicArticle(ctx, tenant, appID, locale, &x); e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	if len(out.Items) > int(f.Limit) {
		next := out.Items[f.Limit-1].ID.String()
		out.NextCursor = &next
		out.Items = out.Items[:f.Limit]
	}
	return out, nil
}
func (r *Postgres) ListPublishedCategories(ctx context.Context, tenant, appID uuid.UUID, locale string) (content.PublicCategoryPage, error) {
	appID, err := r.scopedApp(ctx, tenant, appID)
	if err != nil {
		return content.PublicCategoryPage{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT c.id,c.parent_id,c.slug,c.sort_order,t.name,t.description,c.image_file_id FROM content.categories c JOIN LATERAL(SELECT * FROM content.category_translations WHERE category_id=c.id AND locale IN ($3,'zh-CN') ORDER BY (locale=$3) DESC LIMIT 1)t ON true WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.status='active' ORDER BY c.parent_id NULLS FIRST,c.sort_order,c.slug,c.id`, tenant, appID, locale)
	if err != nil {
		return content.PublicCategoryPage{}, err
	}
	defer rows.Close()
	out := content.PublicCategoryPage{Items: []content.PublicCategory{}}
	for rows.Next() {
		var x content.PublicCategory
		var imageID *uuid.UUID
		if err = rows.Scan(&x.ID, &x.ParentID, &x.Slug, &x.SortOrder, &x.Name, &x.Description, &imageID); err != nil {
			return out, err
		}
		x.ImageURL = mobileCoverURL(imageID)
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) GetPublished(ctx context.Context, tenant, appID uuid.UUID, user *uuid.UUID, locale, slug string) (content.PublicArticle, error) {
	appID, e := r.scopedApp(ctx, tenant, appID)
	if e != nil {
		return content.PublicArticle{}, e
	}
	x, e := scanPublic(r.pool.QueryRow(ctx, publicArticleSelect+" WHERE a.tenant_id=$3 AND a.app_id=$4 AND a.status='published' AND a.slug=$5", user, locale, tenant, appID, slug))
	if e == nil {
		e = r.hydratePublicArticle(ctx, tenant, appID, locale, &x)
	}
	return x, mapNotFound(e)
}

const articleAssetSelect = `SELECT f.id,f.provider,f.bucket_name,f.object_key,f.media_type,f.size_bytes,f.sha256
	FROM storage.files f
	WHERE f.tenant_id=$1 AND f.id=$3 AND COALESCE(f.metadata->>'purpose','')<>'feedback' AND f.status='ready' AND f.scan_status IN ('clean','skipped')
	AND lower(COALESCE(f.media_type,'')) IN ('image/jpeg','image/png','image/webp','video/mp4') AND f.deleted_at IS NULL
	AND (
		EXISTS(SELECT 1 FROM content.articles a WHERE a.tenant_id=$1 AND a.app_id=$2 AND a.status='published' AND (a.cover_file_id=f.id OR a.video_file_id=f.id OR EXISTS(SELECT 1 FROM content.article_media m WHERE m.article_id=a.id AND m.file_id=f.id)))
		OR EXISTS(
			SELECT 1 FROM iam.users u
			JOIN content.comments c ON c.author_id=u.id AND c.tenant_id=$1 AND c.app_id=$2 AND c.status='approved'
			JOIN content.articles a ON a.id=c.article_id AND a.tenant_id=c.tenant_id AND a.app_id=c.app_id AND a.status='published'
			WHERE u.avatar_file_id=f.id
		)
	)`

func (r *Postgres) OpenArticleAsset(ctx context.Context, tenant, appID, fileID uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	if r.objects == nil {
		return content.ArticleAsset{}, nil, content.ErrNotFound
	}
	appID, err := r.scopedApp(ctx, tenant, appID)
	if err != nil {
		return content.ArticleAsset{}, nil, err
	}
	var asset content.ArticleAsset
	var mediaType *string
	err = r.pool.QueryRow(ctx, articleAssetSelect, tenant, appID, fileID).Scan(&asset.FileID, &asset.Provider, &asset.Bucket, &asset.ObjectKey, &mediaType, &asset.SizeBytes, &asset.SHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return content.ArticleAsset{}, nil, content.ErrNotFound
	}
	if err != nil {
		return content.ArticleAsset{}, nil, err
	}
	if mediaType == nil || (!articleAssetAllowed("ready", "clean", *mediaType) && !articleAssetAllowed("ready", "skipped", *mediaType)) {
		return content.ArticleAsset{}, nil, content.ErrNotFound
	}
	asset.MediaType = strings.ToLower(strings.TrimSpace(*mediaType))
	reader, err := r.objects.Open(ctx, storagedomain.ObjectRef{TenantID: tenant, Provider: asset.Provider, Bucket: asset.Bucket, Key: asset.ObjectKey})
	if err != nil {
		if errors.Is(err, storagedomain.ErrObjectNotFound) {
			return content.ArticleAsset{}, nil, content.ErrNotFound
		}
		return content.ArticleAsset{}, nil, err
	}
	return asset, reader, nil
}

func articleAssetAllowed(status, scanStatus, mediaType string) bool {
	if status != "ready" || (scanStatus != "clean" && scanStatus != "skipped") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg", "image/png", "image/webp", "video/mp4":
		return true
	default:
		return false
	}
}
func (r *Postgres) Bookmark(ctx context.Context, tenant, appID, user, article uuid.UUID) error {
	appID, e := r.scopedApp(ctx, tenant, appID)
	if e != nil {
		return e
	}
	exists, e := r.queries.ContentPublishedArticleExists(ctx, db.ContentPublishedArticleExistsParams{TenantID: tenant, AppID: appID, ArticleID: article})
	if e != nil {
		return e
	}
	if !exists {
		return content.ErrNotFound
	}
	_, e = r.queries.ContentUpsertBookmark(ctx, db.ContentUpsertBookmarkParams{TenantID: tenant, AppID: appID, UserID: user, ArticleID: article})
	return e
}
func (r *Postgres) RemoveBookmark(ctx context.Context, tenant, appID, user, article uuid.UUID) error {
	appID, e := r.scopedApp(ctx, tenant, appID)
	if e != nil {
		return e
	}
	_, e = r.queries.ContentDeleteBookmark(ctx, db.ContentDeleteBookmarkParams{TenantID: tenant, AppID: appID, UserID: user, ArticleID: article})
	return e
}

func getCategory(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenant, appID, id uuid.UUID) (content.Category, error) {
	x, e := scanCategory(q.QueryRow(ctx, categorySelect+" WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.id=$3", tenant, appID, id))
	if e == nil {
		x.ImageURL = adminCoverURL(x.ImageFileID)
	}
	return x, mapNotFound(e)
}
func getArticle(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenant, appID, id uuid.UUID) (content.Article, error) {
	x, e := scanArticle(q.QueryRow(ctx, articleSelect+" WHERE a.tenant_id=$1 AND a.app_id=$2 AND a.id=$3", tenant, appID, id))
	return x, mapNotFound(e)
}
func upsertCategoryTranslations(ctx context.Context, tx pgx.Tx, tenant, id uuid.UUID, values map[string]content.CategoryTranslation) error {
	for _, locale := range []string{"zh-CN", "en-US"} {
		x := values[locale]
		if _, e := tx.Exec(ctx, `INSERT INTO content.category_translations(category_id,tenant_id,locale,name,description) VALUES($1,$2,$3,$4,$5) ON CONFLICT(category_id,locale) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description`, id, tenant, locale, x.Name, x.Description); e != nil {
			return e
		}
	}
	return nil
}
func upsertArticleTranslations(ctx context.Context, tx pgx.Tx, id uuid.UUID, values map[string]content.Translation) error {
	for _, locale := range []string{"zh-CN", "en-US"} {
		x := values[locale]
		searchText := strings.ToLower(strings.Join([]string{x.Title, x.Summary, string(x.Body)}, " "))
		if _, e := tx.Exec(ctx, `INSERT INTO content.article_translations(article_id,locale,title,summary,body_format,body,search_text) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(article_id,locale) DO UPDATE SET title=EXCLUDED.title,summary=EXCLUDED.summary,body_format=EXCLUDED.body_format,body=EXCLUDED.body,search_text=EXCLUDED.search_text`, id, locale, x.Title, x.Summary, x.BodyFormat, x.Body, searchText); e != nil {
			return e
		}
	}
	return nil
}
func validReferences(ctx context.Context, tx pgx.Tx, tenant, appID uuid.UUID, category, cover *uuid.UUID) error {
	if category != nil {
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM content.categories WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='active')`, tenant, appID, *category).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return content.ErrInvalid
		}
	}
	if cover != nil {
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM storage.files WHERE tenant_id=$1 AND id=$2 AND COALESCE(metadata->>'purpose','')<>'feedback' AND status='ready' AND scan_status IN ('clean','skipped') AND lower(COALESCE(media_type,'')) IN ('image/jpeg','image/png','image/webp') AND deleted_at IS NULL)`, tenant, *cover).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return content.ErrInvalid
		}
	}
	return nil
}
func (r *Postgres) begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
}
func (r *Postgres) scopedApp(ctx context.Context, tenant, appID uuid.UUID) (uuid.UUID, error) {
	if appID != uuid.Nil {
		var found uuid.UUID
		err := r.pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND id=$2 AND status='active'`, tenant, appID).Scan(&found)
		return found, mapNotFound(err)
	}
	var found uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default AND status='active'`, tenant).Scan(&found)
	return found, mapNotFound(err)
}
func audit(ctx context.Context, tx pgx.Tx, p content.Principal, action, resource string, id uuid.UUID, method string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_, e := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,api_client_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,NULLIF($3,'00000000-0000-0000-0000-000000000000'::uuid),NULLIF($4,'00000000-0000-0000-0000-000000000000'::uuid),$5,'content',$6,$6,$7,$8,$9,$10,200,NULLIF($11,'')::inet,NULLIF($12,''),$13,$14,true)`, p.TenantID, p.UserID, p.SessionID, p.APIClientID, p.RequestID, action, resource, id.String(), method, "/admin-api/v1/content", p.IPAddress, p.UserAgent, b, a)
	return e
}
func safeCategory(x content.Category) any {
	return map[string]any{"id": x.ID, "parent_id": x.ParentID, "image_file_id": x.ImageFileID, "slug": x.Slug, "status": x.Status, "sort_order": x.SortOrder, "lock_version": x.LockVersion, "translations": x.Translations}
}
func safeArticle(x content.Article) any {
	return map[string]any{"id": x.ID, "slug": x.Slug, "status": x.Status, "content_type": x.ContentType, "category_ids": x.CategoryIDs, "topic_id": x.TopicID, "tag_ids": x.TagIDs, "allow_comments": x.AllowComments, "pinned": x.Pinned, "featured": x.Featured, "latest": x.Latest, "sort_order": x.SortOrder, "cover_file_id": x.CoverFileID, "reading_minutes": x.ReadingMinutes, "lock_version": x.LockVersion, "translations": x.Translations}
}
func mapNotFound(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return content.ErrNotFound
	}
	return e
}
func mapWrite(e error) error {
	var pg *pgconn.PgError
	if errors.As(e, &pg) && pg.Code == "23505" {
		return content.ErrConflict
	}
	return e
}

var _ = time.Now
