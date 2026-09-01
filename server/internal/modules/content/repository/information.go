package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type informationQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Postgres) ResolvePublicApp(ctx context.Context, appID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM app.applications WHERE id=$1 AND status='active'`, appID).Scan(&tenantID)
	return tenantID, mapNotFound(err)
}

func validCategoryReferences(ctx context.Context, tx pgx.Tx, tenantID, appID, id uuid.UUID, parentID, imageID *uuid.UUID) error {
	if parentID != nil {
		var grandparent *uuid.UUID
		err := tx.QueryRow(ctx, `SELECT parent_id FROM content.categories WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, tenantID, appID, *parentID).Scan(&grandparent)
		if err != nil || grandparent != nil || *parentID == id {
			return content.ErrInvalid
		}
	}
	if imageID != nil && !validStoredFile(ctx, tx, tenantID, *imageID, false) {
		return content.ErrInvalid
	}
	return nil
}

func validInformationReferences(ctx context.Context, tx pgx.Tx, tenantID, appID uuid.UUID, x content.Article) error {
	if len(x.CategoryIDs) > 10 {
		return content.ErrInvalid
	}
	var count int
	if len(x.CategoryIDs) > 0 {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM content.categories WHERE tenant_id=$1 AND app_id=$2 AND id=ANY($3::uuid[]) AND status='active'`, tenantID, appID, x.CategoryIDs).Scan(&count); err != nil || count != len(x.CategoryIDs) {
			return content.ErrInvalid
		}
	}
	if x.TopicID != nil {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM content.topics WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='active')`, tenantID, appID, *x.TopicID).Scan(&ok); err != nil || !ok {
			return content.ErrInvalid
		}
	}
	if len(x.TagIDs) > 0 {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM content.tags WHERE tenant_id=$1 AND app_id=$2 AND id=ANY($3::uuid[]) AND status='active'`, tenantID, appID, x.TagIDs).Scan(&count); err != nil || count != len(x.TagIDs) {
			return content.ErrInvalid
		}
	}
	if x.CoverFileID != nil && !validStoredFile(ctx, tx, tenantID, *x.CoverFileID, false) {
		return content.ErrInvalid
	}
	if x.VideoFileID != nil && !validStoredFile(ctx, tx, tenantID, *x.VideoFileID, true) {
		return content.ErrInvalid
	}
	if x.VideoExternalURL != nil {
		parsed, err := url.Parse(*x.VideoExternalURL)
		if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" {
			return content.ErrInvalid
		}
		path := strings.ToLower(parsed.Path)
		if !(strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".m3u8")) {
			return content.ErrInvalid
		}
		var allowed bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM content.video_external_hosts WHERE tenant_id=$1 AND app_id=$2 AND hostname=$3 AND active)`, tenantID, appID, strings.ToLower(parsed.Hostname())).Scan(&allowed); err != nil || !allowed {
			return content.ErrInvalid
		}
	}
	for _, media := range x.Media {
		if !validStoredFile(ctx, tx, tenantID, media.FileID, false) {
			return content.ErrInvalid
		}
	}
	return nil
}

func validStoredFile(ctx context.Context, tx pgx.Tx, tenantID, fileID uuid.UUID, video bool) bool {
	var mediaType string
	var size int64
	err := tx.QueryRow(ctx, `SELECT lower(COALESCE(media_type,'')),size_bytes FROM storage.files WHERE tenant_id=$1 AND id=$2 AND COALESCE(metadata->>'purpose','')<>'feedback' AND status='ready' AND scan_status IN ('clean','skipped') AND deleted_at IS NULL`, tenantID, fileID).Scan(&mediaType, &size)
	if err != nil {
		return false
	}
	if video {
		return mediaType == "video/mp4" && size <= 500*1024*1024
	}
	return mediaType == "image/jpeg" || mediaType == "image/png" || mediaType == "image/webp"
}

func syncArticleRelations(ctx context.Context, tx pgx.Tx, p content.Principal, x content.Article) error {
	if _, err := tx.Exec(ctx, `DELETE FROM content.article_categories WHERE article_id=$1`, x.ID); err != nil {
		return err
	}
	for index, categoryID := range x.CategoryIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO content.article_categories(tenant_id,app_id,article_id,category_id,sort_order) VALUES($1,$2,$3,$4,$5)`, p.TenantID, p.AppID, x.ID, categoryID, index); err != nil {
			return mapWrite(err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM content.article_tags WHERE article_id=$1`, x.ID); err != nil {
		return err
	}
	for index, tagID := range x.TagIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO content.article_tags(tenant_id,app_id,article_id,tag_id,sort_order) VALUES($1,$2,$3,$4,$5)`, p.TenantID, p.AppID, x.ID, tagID, index); err != nil {
			return mapWrite(err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM content.article_media WHERE article_id=$1`, x.ID); err != nil {
		return err
	}
	for _, media := range x.Media {
		var mediaID uuid.UUID
		if media.ID == uuid.Nil {
			media.ID = uuid.New()
		}
		if err := tx.QueryRow(ctx, `INSERT INTO content.article_media(id,tenant_id,app_id,article_id,file_id,role,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, media.ID, p.TenantID, p.AppID, x.ID, media.FileID, media.Role, media.SortOrder).Scan(&mediaID); err != nil {
			return mapWrite(err)
		}
		for _, locale := range []string{"zh-CN", "en-US"} {
			if _, err := tx.Exec(ctx, `INSERT INTO content.article_media_translations(media_id,locale,alt_text) VALUES($1,$2,$3)`, mediaID, locale, media.Translations[locale].AltText); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='content' AND entity_type='content.article' AND entity_id=$2`, p.TenantID, x.ID); err != nil {
		return err
	}
	register := func(fileID *uuid.UUID, field string) error {
		if fileID == nil {
			return nil
		}
		_, err := tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'content','content.article',$3,$4) ON CONFLICT DO NOTHING`, *fileID, p.TenantID, x.ID, field)
		return err
	}
	if err := register(x.CoverFileID, "cover"); err != nil {
		return err
	}
	if err := register(x.VideoFileID, "video"); err != nil {
		return err
	}
	for _, media := range x.Media {
		fileID := media.FileID
		if err := register(&fileID, fmt.Sprintf("%s:%d", media.Role, media.SortOrder)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Postgres) hydrateArticle(ctx context.Context, q informationQuerier, tenantID, appID uuid.UUID, x *content.Article) error {
	rows, err := q.Query(ctx, `SELECT category_id FROM content.article_categories WHERE tenant_id=$1 AND app_id=$2 AND article_id=$3 ORDER BY sort_order,category_id`, tenantID, appID, x.ID)
	if err != nil {
		return err
	}
	x.CategoryIDs = []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		x.CategoryIDs = append(x.CategoryIDs, id)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT t.id,t.name,t.status,t.lock_version,(SELECT count(*) FROM content.article_tags at2 WHERE at2.tag_id=t.id) FROM content.article_tags at JOIN content.tags t ON t.id=at.tag_id WHERE at.tenant_id=$1 AND at.app_id=$2 AND at.article_id=$3 ORDER BY at.sort_order,t.id`, tenantID, appID, x.ID)
	if err != nil {
		return err
	}
	x.TagIDs, x.Tags = []uuid.UUID{}, []content.Tag{}
	for rows.Next() {
		var tag content.Tag
		if err = rows.Scan(&tag.ID, &tag.Name, &tag.Status, &tag.LockVersion, &tag.UsageCount); err != nil {
			rows.Close()
			return err
		}
		x.TagIDs = append(x.TagIDs, tag.ID)
		x.Tags = append(x.Tags, tag)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT m.id,m.file_id,m.role,m.sort_order,COALESCE((SELECT jsonb_object_agg(locale,jsonb_build_object('alt_text',alt_text)) FROM content.article_media_translations WHERE media_id=m.id),'{}'::jsonb) FROM content.article_media m WHERE m.tenant_id=$1 AND m.app_id=$2 AND m.article_id=$3 ORDER BY m.role,m.sort_order,m.id`, tenantID, appID, x.ID)
	if err != nil {
		return err
	}
	x.Media = []content.Media{}
	for rows.Next() {
		var media content.Media
		var raw []byte
		if err = rows.Scan(&media.ID, &media.FileID, &media.Role, &media.SortOrder, &raw); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal(raw, &media.Translations); err != nil {
			rows.Close()
			return err
		}
		x.Media = append(x.Media, media)
	}
	rows.Close()
	return nil
}

func (r *Postgres) hydratePublicArticle(ctx context.Context, tenantID, appID uuid.UUID, locale string, x *content.PublicArticle) error {
	rows, err := r.pool.Query(ctx, `SELECT c.id,c.parent_id,c.slug,c.sort_order,ct.name,ct.description,c.image_file_id FROM content.article_categories ac JOIN content.categories c ON c.id=ac.category_id AND c.status='active' JOIN LATERAL(SELECT * FROM content.category_translations WHERE category_id=c.id AND locale IN ($4,'zh-CN') ORDER BY (locale=$4) DESC LIMIT 1) ct ON true WHERE ac.tenant_id=$1 AND ac.app_id=$2 AND ac.article_id=$3 ORDER BY ac.sort_order,c.id`, tenantID, appID, x.ID, locale)
	if err != nil {
		return err
	}
	x.Categories = []content.PublicCategory{}
	for rows.Next() {
		var category content.PublicCategory
		var imageID *uuid.UUID
		if err = rows.Scan(&category.ID, &category.ParentID, &category.Slug, &category.SortOrder, &category.Name, &category.Description, &imageID); err != nil {
			rows.Close()
			return err
		}
		category.ImageURL = mobileCoverURL(imageID)
		x.Categories = append(x.Categories, category)
	}
	rows.Close()
	if len(x.Categories) > 0 {
		category := x.Categories[0]
		x.Category = &category
	}
	var topic content.PublicTopic
	var topicCover *uuid.UUID
	if err = r.pool.QueryRow(ctx, `SELECT tp.id,tp.slug,tp.sort_order,tt.name,tt.description,tp.cover_file_id FROM content.articles a JOIN content.topics tp ON tp.id=a.topic_id AND tp.status='active' JOIN LATERAL(SELECT * FROM content.topic_translations WHERE topic_id=tp.id AND locale IN ($4,'zh-CN') ORDER BY (locale=$4) DESC LIMIT 1) tt ON true WHERE a.tenant_id=$1 AND a.app_id=$2 AND a.id=$3`, tenantID, appID, x.ID, locale).Scan(&topic.ID, &topic.Slug, &topic.SortOrder, &topic.Name, &topic.Description, &topicCover); err == nil {
		topic.CoverURL = mobileCoverURL(topicCover)
		x.Topic = &topic
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	rows, err = r.pool.Query(ctx, `SELECT t.id,t.name FROM content.article_tags at JOIN content.tags t ON t.id=at.tag_id AND t.status='active' WHERE at.tenant_id=$1 AND at.app_id=$2 AND at.article_id=$3 ORDER BY at.sort_order,t.id`, tenantID, appID, x.ID)
	if err != nil {
		return err
	}
	x.Tags = []content.Tag{}
	for rows.Next() {
		var tag content.Tag
		if err = rows.Scan(&tag.ID, &tag.Name); err != nil {
			rows.Close()
			return err
		}
		x.Tags = append(x.Tags, tag)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT m.id,m.role,m.sort_order,m.file_id,mt.alt_text FROM content.article_media m JOIN content.article_media_translations mt ON mt.media_id=m.id AND mt.locale=$4 JOIN storage.files f ON f.tenant_id=m.tenant_id AND f.id=m.file_id AND COALESCE(f.metadata->>'purpose','')<>'feedback' AND f.status='ready' AND f.scan_status IN ('clean','skipped') AND f.deleted_at IS NULL WHERE m.tenant_id=$1 AND m.app_id=$2 AND m.article_id=$3 ORDER BY m.role,m.sort_order,m.id`, tenantID, appID, x.ID, locale)
	if err != nil {
		return err
	}
	x.Media = []content.PublicMedia{}
	for rows.Next() {
		var media content.PublicMedia
		if err = rows.Scan(&media.ID, &media.Role, &media.SortOrder, &media.FileID, &media.AltText); err != nil {
			rows.Close()
			return err
		}
		media.URL = *mobileCoverURL(&media.FileID)
		x.Media = append(x.Media, media)
	}
	rows.Close()
	return nil
}

const topicSelect = `SELECT tp.id,tp.slug,tp.status,tp.sort_order,tp.cover_file_id,tp.lock_version,tp.created_at,tp.updated_at,COALESCE((SELECT jsonb_object_agg(locale,jsonb_build_object('name',name,'description',description)) FROM content.topic_translations WHERE topic_id=tp.id),'{}'::jsonb) FROM content.topics tp`

func scanTopic(row pgx.Row) (content.Topic, error) {
	var x content.Topic
	var raw []byte
	err := row.Scan(&x.ID, &x.Slug, &x.Status, &x.SortOrder, &x.CoverFileID, &x.LockVersion, &x.CreatedAt, &x.UpdatedAt, &raw)
	if err == nil {
		err = json.Unmarshal(raw, &x.Translations)
		x.CoverURL = adminCoverURL(x.CoverFileID)
	}
	return x, err
}
func (r *Postgres) ListTopics(ctx context.Context, tenantID, appID uuid.UUID, f content.PageFilter) (content.TopicPage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.TopicPage{}, err
	}
	where := `tp.tenant_id=$1 AND tp.app_id=$2`
	args := []any{tenantID, appID}
	if f.Query != "" {
		args = append(args, f.Query)
		where += fmt.Sprintf(` AND (tp.slug ILIKE '%%'||$%d||'%%' OR EXISTS(SELECT 1 FROM content.topic_translations tt WHERE tt.topic_id=tp.id AND tt.name ILIKE '%%'||$%d||'%%'))`, len(args), len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND tp.status=$%d`, len(args))
	}
	var total int64
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM content.topics tp WHERE `+where, args...).Scan(&total); err != nil {
		return content.TopicPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, topicSelect+` WHERE `+where+fmt.Sprintf(` ORDER BY tp.sort_order,tp.updated_at DESC,tp.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return content.TopicPage{}, err
	}
	defer rows.Close()
	out := content.TopicPage{Items: []content.Topic{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		x, e := scanTopic(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) GetTopic(ctx context.Context, tenantID, appID, id uuid.UUID) (content.Topic, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.Topic{}, err
	}
	x, err := scanTopic(r.pool.QueryRow(ctx, topicSelect+` WHERE tp.tenant_id=$1 AND tp.app_id=$2 AND tp.id=$3`, tenantID, appID, id))
	return x, mapNotFound(err)
}
func upsertTopicTranslations(ctx context.Context, tx pgx.Tx, id uuid.UUID, values map[string]content.TopicTranslation) error {
	for _, locale := range []string{"zh-CN", "en-US"} {
		x := values[locale]
		if _, err := tx.Exec(ctx, `INSERT INTO content.topic_translations(topic_id,locale,name,description) VALUES($1,$2,$3,$4) ON CONFLICT(topic_id,locale) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description`, id, locale, x.Name, x.Description); err != nil {
			return err
		}
	}
	return nil
}
func syncTopicCoverUsage(ctx context.Context, tx pgx.Tx, p content.Principal, topicID uuid.UUID, fileID *uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='content' AND entity_type='content.topic' AND entity_id=$2`, p.TenantID, topicID); err != nil {
		return err
	}
	if fileID == nil {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'content','content.topic',$3,'cover') ON CONFLICT DO NOTHING`, *fileID, p.TenantID, topicID)
	return err
}
func (r *Postgres) CreateTopic(ctx context.Context, p content.Principal, x content.Topic) (content.Topic, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return x, err
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return x, err
	}
	defer tx.Rollback(ctx)
	if x.CoverFileID != nil && !validStoredFile(ctx, tx, p.TenantID, *x.CoverFileID, false) {
		return x, content.ErrInvalid
	}
	if err = tx.QueryRow(ctx, `INSERT INTO content.topics(tenant_id,app_id,slug,status,sort_order,cover_file_id,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7) RETURNING id`, p.TenantID, p.AppID, x.Slug, x.Status, x.SortOrder, x.CoverFileID, p.UserID).Scan(&x.ID); err != nil {
		return x, mapWrite(err)
	}
	if err = upsertTopicTranslations(ctx, tx, x.ID, x.Translations); err != nil {
		return x, err
	}
	if err = syncTopicCoverUsage(ctx, tx, p, x.ID, x.CoverFileID); err != nil {
		return x, err
	}
	out, err := scanTopic(tx.QueryRow(ctx, topicSelect+` WHERE tp.tenant_id=$1 AND tp.app_id=$2 AND tp.id=$3`, p.TenantID, p.AppID, x.ID))
	if err != nil {
		return x, err
	}
	if err = audit(ctx, tx, p, "content.topic.create", "content.topic", x.ID, "POST", nil, out); err != nil {
		return x, err
	}
	if err = tx.Commit(ctx); err != nil {
		return x, err
	}
	return out, nil
}
func (r *Postgres) UpdateTopic(ctx context.Context, p content.Principal, x content.Topic) (content.Topic, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return x, err
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return x, err
	}
	defer tx.Rollback(ctx)
	before, err := scanTopic(tx.QueryRow(ctx, topicSelect+` WHERE tp.tenant_id=$1 AND tp.app_id=$2 AND tp.id=$3`, p.TenantID, p.AppID, x.ID))
	if err != nil {
		return x, mapNotFound(err)
	}
	if x.CoverFileID != nil && !validStoredFile(ctx, tx, p.TenantID, *x.CoverFileID, false) {
		return x, content.ErrInvalid
	}
	tag, err := tx.Exec(ctx, `UPDATE content.topics SET slug=$1,status=$2,sort_order=$3,cover_file_id=$4,updated_by=$5,lock_version=lock_version+1 WHERE tenant_id=$6 AND app_id=$7 AND id=$8 AND lock_version=$9`, x.Slug, x.Status, x.SortOrder, x.CoverFileID, p.UserID, p.TenantID, p.AppID, x.ID, x.LockVersion)
	if err != nil {
		return x, mapWrite(err)
	}
	if tag.RowsAffected() == 0 {
		return x, content.ErrConflict
	}
	if err = upsertTopicTranslations(ctx, tx, x.ID, x.Translations); err != nil {
		return x, err
	}
	if err = syncTopicCoverUsage(ctx, tx, p, x.ID, x.CoverFileID); err != nil {
		return x, err
	}
	out, err := scanTopic(tx.QueryRow(ctx, topicSelect+` WHERE tp.tenant_id=$1 AND tp.app_id=$2 AND tp.id=$3`, p.TenantID, p.AppID, x.ID))
	if err != nil {
		return x, err
	}
	if err = audit(ctx, tx, p, "content.topic.update", "content.topic", x.ID, "PATCH", before, out); err != nil {
		return x, err
	}
	if err = tx.Commit(ctx); err != nil {
		return x, err
	}
	return out, nil
}
func (r *Postgres) DeleteTopic(ctx context.Context, p content.Principal, id uuid.UUID, version int32) error {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM content.topics tp WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND lock_version=$4 AND NOT EXISTS(SELECT 1 FROM content.articles a WHERE a.topic_id=tp.id)`, p.TenantID, p.AppID, id, version)
	if err != nil {
		return mapWrite(err)
	}
	if tag.RowsAffected() == 0 {
		return content.ErrConflict
	}
	if err = audit(ctx, tx, p, "content.topic.delete", "content.topic", id, "DELETE", nil, map[string]bool{"deleted": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ListTags(ctx context.Context, tenantID, appID uuid.UUID, f content.PageFilter) (content.TagPage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.TagPage{}, err
	}
	where := `t.tenant_id=$1 AND t.app_id=$2`
	args := []any{tenantID, appID}
	if f.Query != "" {
		args = append(args, f.Query)
		where += fmt.Sprintf(` AND t.name ILIKE '%%'||$%d||'%%'`, len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND t.status=$%d`, len(args))
	}
	var total int64
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM content.tags t WHERE `+where, args...).Scan(&total); err != nil {
		return content.TagPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, `SELECT t.id,t.name,t.status,t.lock_version,(SELECT count(*) FROM content.article_tags at WHERE at.tag_id=t.id) FROM content.tags t WHERE `+where+fmt.Sprintf(` ORDER BY t.name,t.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return content.TagPage{}, err
	}
	defer rows.Close()
	out := content.TagPage{Items: []content.Tag{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		var x content.Tag
		if err = rows.Scan(&x.ID, &x.Name, &x.Status, &x.LockVersion, &x.UsageCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) UpsertTag(ctx context.Context, p content.Principal, name string) (content.Tag, error) {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return content.Tag{}, scopeErr
	}
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "#")))
	var x content.Tag
	err := r.pool.QueryRow(ctx, `INSERT INTO content.tags(tenant_id,app_id,name,normalized_name) VALUES($1,$2,$3,$4) ON CONFLICT(app_id,normalized_name) DO UPDATE SET name=EXCLUDED.name,lock_version=content.tags.lock_version+1 RETURNING id,name,status,lock_version`, p.TenantID, p.AppID, strings.TrimSpace(strings.TrimPrefix(name, "#")), normalized).Scan(&x.ID, &x.Name, &x.Status, &x.LockVersion)
	return x, mapWrite(err)
}
func (r *Postgres) UpdateTag(ctx context.Context, p content.Principal, x content.Tag) (content.Tag, error) {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return x, scopeErr
	}
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(x.Name, "#")))
	tag, err := r.pool.Exec(ctx, `UPDATE content.tags SET name=$1,normalized_name=$2,status=$3,lock_version=lock_version+1 WHERE tenant_id=$4 AND app_id=$5 AND id=$6 AND lock_version=$7`, strings.TrimSpace(strings.TrimPrefix(x.Name, "#")), normalized, x.Status, p.TenantID, p.AppID, x.ID, x.LockVersion)
	if err != nil {
		return x, mapWrite(err)
	}
	if tag.RowsAffected() == 0 {
		return x, content.ErrConflict
	}
	return r.tagByID(ctx, p.TenantID, p.AppID, x.ID)
}
func (r *Postgres) tagByID(ctx context.Context, tenantID, appID, id uuid.UUID) (content.Tag, error) {
	var x content.Tag
	err := r.pool.QueryRow(ctx, `SELECT t.id,t.name,t.status,t.lock_version,(SELECT count(*) FROM content.article_tags at WHERE at.tag_id=t.id) FROM content.tags t WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, tenantID, appID, id).Scan(&x.ID, &x.Name, &x.Status, &x.LockVersion, &x.UsageCount)
	return x, mapNotFound(err)
}
func (r *Postgres) MergeTag(ctx context.Context, p content.Principal, source, target uuid.UUID, version int32) error {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return scopeErr
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO content.article_tags(tenant_id,app_id,article_id,tag_id,sort_order) SELECT tenant_id,app_id,article_id,$1,sort_order FROM content.article_tags WHERE tag_id=$2 ON CONFLICT DO NOTHING`, target, source); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM content.article_tags WHERE tag_id=$1`, source); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM content.tags WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND lock_version=$4 AND EXISTS(SELECT 1 FROM content.tags WHERE tenant_id=$1 AND app_id=$2 AND id=$5)`, p.TenantID, p.AppID, source, version, target)
	if err != nil {
		return mapWrite(err)
	}
	if tag.RowsAffected() == 0 {
		return content.ErrConflict
	}
	return tx.Commit(ctx)
}
func (r *Postgres) DeleteTag(ctx context.Context, p content.Principal, id uuid.UUID, version int32) error {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return scopeErr
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM content.tags t WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND lock_version=$4 AND NOT EXISTS(SELECT 1 FROM content.article_tags at WHERE at.tag_id=t.id)`, p.TenantID, p.AppID, id, version)
	if err != nil {
		return mapWrite(err)
	}
	if tag.RowsAffected() == 0 {
		return content.ErrConflict
	}
	return nil
}

func (r *Postgres) ListPublishedTopics(ctx context.Context, tenantID, appID uuid.UUID, locale string) (content.PublicTopicPage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.PublicTopicPage{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT tp.id,tp.slug,tp.sort_order,tt.name,tt.description,tp.cover_file_id FROM content.topics tp JOIN LATERAL(SELECT * FROM content.topic_translations WHERE topic_id=tp.id AND locale IN ($3,'zh-CN') ORDER BY (locale=$3) DESC LIMIT 1)tt ON true WHERE tp.tenant_id=$1 AND tp.app_id=$2 AND tp.status='active' ORDER BY tp.sort_order,tp.updated_at DESC,tp.id`, tenantID, appID, locale)
	if err != nil {
		return content.PublicTopicPage{}, err
	}
	defer rows.Close()
	out := content.PublicTopicPage{Items: []content.PublicTopic{}}
	for rows.Next() {
		var x content.PublicTopic
		var cover *uuid.UUID
		if err = rows.Scan(&x.ID, &x.Slug, &x.SortOrder, &x.Name, &x.Description, &cover); err != nil {
			return out, err
		}
		x.CoverURL = mobileCoverURL(cover)
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) GetPublishedTopic(ctx context.Context, tenantID, appID uuid.UUID, locale, slug string) (content.PublicTopic, error) {
	items, err := r.ListPublishedTopics(ctx, tenantID, appID, locale)
	if err != nil {
		return content.PublicTopic{}, err
	}
	for _, item := range items.Items {
		if item.Slug == slug {
			return item, nil
		}
	}
	return content.PublicTopic{}, content.ErrNotFound
}
func (r *Postgres) Home(ctx context.Context, tenantID, appID uuid.UUID, user *uuid.UUID, locale string, limit int32) (content.HomeFeed, error) {
	base := content.PublicFilter{Limit: limit}
	pinned := true
	base.Featured = nil
	items, err := r.listPublishedWithFlag(ctx, tenantID, appID, user, locale, base, "pinned", &pinned)
	if err != nil {
		return content.HomeFeed{}, err
	}
	featuredItems, err := r.listPublishedWithFlag(ctx, tenantID, appID, user, locale, base, "featured", &pinned)
	if err != nil {
		return content.HomeFeed{}, err
	}
	latestItems, err := r.listPublishedWithFlag(ctx, tenantID, appID, user, locale, base, "is_latest", &pinned)
	if err != nil {
		return content.HomeFeed{}, err
	}
	seen := map[uuid.UUID]bool{}
	out := content.HomeFeed{Pinned: dedupe(items.Items, seen), Featured: dedupe(featuredItems.Items, seen), Latest: dedupe(latestItems.Items, seen)}
	return out, nil
}
func dedupe(items []content.PublicArticle, seen map[uuid.UUID]bool) []content.PublicArticle {
	out := []content.PublicArticle{}
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			out = append(out, item)
		}
	}
	return out
}
func (r *Postgres) listPublishedWithFlag(ctx context.Context, tenantID, appID uuid.UUID, user *uuid.UUID, locale string, f content.PublicFilter, column string, value *bool) (content.PublicArticlePage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	rows, err := r.pool.Query(ctx, publicArticleSelect+` WHERE a.tenant_id=$3 AND a.app_id=$4 AND a.status='published' AND a.`+column+`=$5 ORDER BY a.sort_order DESC,a.published_at DESC,a.id DESC LIMIT $6`, user, locale, tenantID, appID, *value, f.Limit)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	defer rows.Close()
	out := content.PublicArticlePage{Items: []content.PublicArticle{}}
	for rows.Next() {
		x, e := scanPublic(rows)
		if e != nil {
			return out, e
		}
		if e = r.hydratePublicArticle(ctx, tenantID, appID, locale, &x); e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) ListBookmarks(ctx context.Context, tenantID, appID, userID uuid.UUID, locale string, f content.PublicFilter) (content.PublicArticlePage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	query, args, err := bookmarkListQuery(tenantID, appID, userID, locale, f)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	defer rows.Close()
	out := content.PublicArticlePage{Items: []content.PublicArticle{}}
	for rows.Next() {
		x, e := scanPublic(rows)
		if e != nil {
			return out, e
		}
		if e = r.hydratePublicArticle(ctx, tenantID, appID, locale, &x); e != nil {
			return out, e
		}
		out.Items = append(out.Items, x)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > int(f.Limit) {
		next := out.Items[f.Limit-1].ID.String()
		out.NextCursor = &next
		out.Items = out.Items[:f.Limit]
	}
	return out, nil
}

func bookmarkListQuery(tenantID, appID, userID uuid.UUID, locale string, f content.PublicFilter) (string, []any, error) {
	args := []any{&userID, locale, tenantID, appID}
	where := []string{"a.tenant_id=$3", "a.app_id=$4", "a.status='published'"}
	add := func(query string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(query, len(args)))
	}
	if f.Query != "" {
		add("(t.search_text ILIKE '%%'||$%[1]d||'%%' OR EXISTS(SELECT 1 FROM content.article_tags sat JOIN content.tags st ON st.id=sat.tag_id WHERE sat.article_id=a.id AND st.status='active' AND st.normalized_name ILIKE '%%'||$%[1]d||'%%') OR EXISTS(SELECT 1 FROM content.article_categories sac JOIN content.category_translations sct ON sct.category_id=sac.category_id WHERE sac.article_id=a.id AND sct.locale IN ($2,'zh-CN') AND (lower(sct.name) ILIKE '%%'||$%[1]d||'%%' OR lower(sct.description) ILIKE '%%'||$%[1]d||'%%')))", strings.ToLower(f.Query))
	}
	if f.ContentType != "" {
		add("a.content_type=$%d", f.ContentType)
	}
	if f.Cursor != "" {
		id, err := uuid.Parse(f.Cursor)
		if err != nil {
			return "", nil, content.ErrInvalid
		}
		add("(bm.created_at,a.id) < (SELECT cursor_bm.created_at,cursor_bm.article_id FROM content.article_bookmarks cursor_bm WHERE cursor_bm.tenant_id=$3 AND cursor_bm.app_id=$4 AND cursor_bm.user_id=$1 AND cursor_bm.article_id=$%d)", id)
	}
	args = append(args, f.Limit+1)
	query := publicArticleSelect + ` JOIN content.article_bookmarks bm ON bm.tenant_id=a.tenant_id AND bm.app_id=a.app_id AND bm.article_id=a.id AND bm.user_id=$1 WHERE ` + strings.Join(where, " AND ") + fmt.Sprintf(" ORDER BY bm.created_at DESC,a.id DESC LIMIT $%d", len(args))
	return query, args, nil
}

func (r *Postgres) ListComments(ctx context.Context, tenantID, appID uuid.UUID, viewer *uuid.UUID, articleID uuid.UUID, status string, f content.PageFilter) (content.CommentPage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.CommentPage{}, err
	}
	where := `c.tenant_id=$1 AND c.app_id=$2`
	args := []any{tenantID, appID}
	if articleID != uuid.Nil {
		args = append(args, articleID)
		where += fmt.Sprintf(` AND c.article_id=$%d`, len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(` AND c.status=$%d`, len(args))
	}
	if viewer != nil {
		args = append(args, *viewer)
		where += fmt.Sprintf(` AND NOT EXISTS(SELECT 1 FROM content.blocked_users b WHERE b.app_id=c.app_id AND b.blocker_id=$%d AND b.blocked_id=c.author_id)`, len(args))
	}
	var total int64
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM content.comments c WHERE `+where, args...).Scan(&total); err != nil {
		return content.CommentPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, `SELECT c.id,c.article_id,c.author_id,u.display_name,u.avatar_file_id,c.parent_id,c.root_id,c.status,c.body,c.moderation_reason,c.created_at,c.updated_at FROM content.comments c JOIN iam.users u ON u.id=c.author_id WHERE `+where+fmt.Sprintf(` ORDER BY c.created_at DESC,c.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return content.CommentPage{}, err
	}
	defer rows.Close()
	out := content.CommentPage{Items: []content.Comment{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		var x content.Comment
		var avatarFileID *uuid.UUID
		if err = rows.Scan(&x.ID, &x.ArticleID, &x.AuthorID, &x.AuthorName, &avatarFileID, &x.ParentID, &x.RootID, &x.Status, &x.Body, &x.ModerationReason, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return out, err
		}
		x.AuthorAvatarURL = mobileAvatarURL(avatarFileID)
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}

const commentRateLimitSQL = `SELECT count(*),count(*) FILTER(WHERE body_fingerprint=$4 AND created_at>now()-interval '24 hours') FROM content.comments WHERE tenant_id=$1 AND app_id=$2 AND author_id=$3 AND created_at>now()-interval '1 minute'`

func (r *Postgres) CreateComment(ctx context.Context, p content.Principal, articleID uuid.UUID, parentID *uuid.UUID, body string) (content.Comment, error) {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return content.Comment{}, scopeErr
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return content.Comment{}, err
	}
	defer tx.Rollback(ctx)
	var allow bool
	if err = tx.QueryRow(ctx, `SELECT allow_comments FROM content.articles WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='published'`, p.TenantID, p.AppID, articleID).Scan(&allow); err != nil || !allow {
		return content.Comment{}, content.ErrForbidden
	}
	var rootID *uuid.UUID
	if parentID != nil {
		var parentRoot *uuid.UUID
		if err = tx.QueryRow(ctx, `SELECT root_id FROM content.comments WHERE tenant_id=$1 AND app_id=$2 AND article_id=$3 AND id=$4 AND status='approved'`, p.TenantID, p.AppID, articleID, *parentID).Scan(&parentRoot); err != nil || parentRoot != nil {
			return content.Comment{}, content.ErrInvalid
		}
		root := *parentID
		rootID = &root
	}
	var recent, duplicate int
	var sensitive bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM content.sensitive_words WHERE tenant_id=$1 AND app_id=$2 AND active AND lower($3) LIKE '%'||normalized_word||'%')`, p.TenantID, p.AppID, body).Scan(&sensitive); err != nil {
		return content.Comment{}, err
	}
	if sensitive {
		return content.Comment{}, content.ErrInvalid
	}
	fingerprint := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(body))))
	if err = tx.QueryRow(ctx, commentRateLimitSQL, p.TenantID, p.AppID, p.UserID, fingerprint[:]).Scan(&recent, &duplicate); err != nil {
		return content.Comment{}, err
	}
	if recent >= 3 || duplicate > 0 {
		return content.Comment{}, content.ErrConflict
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO content.comments(tenant_id,app_id,article_id,author_id,parent_id,root_id,body,body_fingerprint) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, p.TenantID, p.AppID, articleID, p.UserID, parentID, rootID, body, fingerprint[:]).Scan(&id); err != nil {
		return content.Comment{}, mapWrite(err)
	}
	page, err := r.listCommentsTx(ctx, tx, p.TenantID, p.AppID, id)
	if err != nil {
		return content.Comment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return content.Comment{}, err
	}
	return page, nil
}
func (r *Postgres) listCommentsTx(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID, appID, id uuid.UUID) (content.Comment, error) {
	var x content.Comment
	var avatarFileID *uuid.UUID
	err := q.QueryRow(ctx, `SELECT c.id,c.article_id,c.author_id,u.display_name,u.avatar_file_id,c.parent_id,c.root_id,c.status,c.body,c.moderation_reason,c.created_at,c.updated_at FROM content.comments c JOIN iam.users u ON u.id=c.author_id WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.id=$3`, tenantID, appID, id).Scan(&x.ID, &x.ArticleID, &x.AuthorID, &x.AuthorName, &avatarFileID, &x.ParentID, &x.RootID, &x.Status, &x.Body, &x.ModerationReason, &x.CreatedAt, &x.UpdatedAt)
	x.AuthorAvatarURL = mobileAvatarURL(avatarFileID)
	return x, mapNotFound(err)
}

func mobileAvatarURL(fileID *uuid.UUID) *string {
	if fileID == nil || *fileID == uuid.Nil {
		return nil
	}
	value := "/api/v1/public/content/assets/" + fileID.String()
	return &value
}
func (r *Postgres) DeleteOwnComment(ctx context.Context, p content.Principal, id uuid.UUID) error {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return scopeErr
	}
	tag, err := r.pool.Exec(ctx, `UPDATE content.comments SET status='deleted',body='[deleted]',deleted_at=now(),moderated_at=COALESCE(moderated_at,now()) WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND author_id=$4 AND status IN ('pending','approved')`, p.TenantID, p.AppID, id, p.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return content.ErrNotFound
	}
	return nil
}
func (r *Postgres) ModerateComment(ctx context.Context, p content.Principal, id uuid.UUID, status, reason string) (content.Comment, error) {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return content.Comment{}, scopeErr
	}
	tag, err := r.pool.Exec(ctx, `UPDATE content.comments SET status=$1,moderation_reason=NULLIF($2,''),moderated_by=$3,moderated_at=now() WHERE tenant_id=$4 AND app_id=$5 AND id=$6`, status, reason, p.UserID, p.TenantID, p.AppID, id)
	if err != nil {
		return content.Comment{}, err
	}
	if tag.RowsAffected() == 0 {
		return content.Comment{}, content.ErrNotFound
	}
	return r.listCommentsTx(ctx, r.pool, p.TenantID, p.AppID, id)
}
func (r *Postgres) ReportComment(ctx context.Context, p content.Principal, id uuid.UUID, reason, details string) (content.CommentReport, error) {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return content.CommentReport{}, scopeErr
	}
	var x content.CommentReport
	err := r.pool.QueryRow(ctx, `INSERT INTO content.comment_reports(tenant_id,app_id,comment_id,reporter_id,reason,details) SELECT $1,$2,c.id,$3,$4,$5 FROM content.comments c WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.id=$6 AND c.status='approved' RETURNING id,comment_id,reporter_id,reason,details,status,resolution,created_at,resolved_at`, p.TenantID, p.AppID, p.UserID, reason, details, id).Scan(&x.ID, &x.CommentID, &x.ReporterID, &x.Reason, &x.Details, &x.Status, &x.Resolution, &x.CreatedAt, &x.ResolvedAt)
	return x, mapWrite(mapNotFound(err))
}
func (r *Postgres) ListCommentReports(ctx context.Context, tenantID, appID uuid.UUID, status string, f content.PageFilter) (content.CommentReportPage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return content.CommentReportPage{}, err
	}
	where := `tenant_id=$1 AND app_id=$2`
	args := []any{tenantID, appID}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(` AND status=$%d`, len(args))
	}
	var total int64
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM content.comment_reports WHERE `+where, args...).Scan(&total); err != nil {
		return content.CommentReportPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, `SELECT id,comment_id,reporter_id,reason,details,status,resolution,created_at,resolved_at FROM content.comment_reports WHERE `+where+fmt.Sprintf(` ORDER BY created_at,id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return content.CommentReportPage{}, err
	}
	defer rows.Close()
	out := content.CommentReportPage{Items: []content.CommentReport{}, Page: f.Page, PageSize: f.PageSize, Total: total}
	for rows.Next() {
		var x content.CommentReport
		if err = rows.Scan(&x.ID, &x.CommentID, &x.ReporterID, &x.Reason, &x.Details, &x.Status, &x.Resolution, &x.CreatedAt, &x.ResolvedAt); err != nil {
			return out, err
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
func (r *Postgres) ResolveCommentReport(ctx context.Context, p content.Principal, id uuid.UUID, status, resolution string) (content.CommentReport, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return content.CommentReport{}, err
	}
	tx, err := r.begin(ctx)
	if err != nil {
		return content.CommentReport{}, err
	}
	defer tx.Rollback(ctx)
	var x content.CommentReport
	err = tx.QueryRow(ctx, `UPDATE content.comment_reports SET status=$1,resolution=$2,resolved_by=$3,resolved_at=now() WHERE tenant_id=$4 AND app_id=$5 AND id=$6 AND status='open' RETURNING id,comment_id,reporter_id,reason,details,status,resolution,created_at,resolved_at`, status, resolution, p.UserID, p.TenantID, p.AppID, id).Scan(&x.ID, &x.CommentID, &x.ReporterID, &x.Reason, &x.Details, &x.Status, &x.Resolution, &x.CreatedAt, &x.ResolvedAt)
	if err != nil {
		return content.CommentReport{}, mapNotFound(err)
	}
	if err = audit(ctx, tx, p, "app.content.update", "comment_report", id, "POST", nil, x); err != nil {
		return content.CommentReport{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return content.CommentReport{}, err
	}
	return x, nil
}
func (r *Postgres) BlockUser(ctx context.Context, p content.Principal, id uuid.UUID) error {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return scopeErr
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO content.blocked_users(tenant_id,app_id,blocker_id,blocked_id) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, p.TenantID, p.AppID, p.UserID, id)
	return mapWrite(err)
}
func (r *Postgres) UnblockUser(ctx context.Context, p content.Principal, id uuid.UUID) error {
	var scopeErr error
	p.AppID, scopeErr = r.scopedApp(ctx, p.TenantID, p.AppID)
	if scopeErr != nil {
		return scopeErr
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM content.blocked_users WHERE tenant_id=$1 AND app_id=$2 AND blocker_id=$3 AND blocked_id=$4`, p.TenantID, p.AppID, p.UserID, id)
	return err
}

var _ = time.Now
