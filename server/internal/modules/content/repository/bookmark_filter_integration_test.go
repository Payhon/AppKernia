//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBookmarkListFiltersContentTypeSearchAndCursor(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	suffix := uuid.NewString()
	var tenantID, appID, userID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name,status) VALUES($1,$2,'active') RETURNING id`, "bookmark-"+suffix, "Bookmark Test Tenant").Scan(&tenantID); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatalf("select default app: %v", err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,$2,'active') RETURNING id`, "bookmark-"+suffix+"@example.test", "Bookmark Test User").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenants WHERE id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.users WHERE id=$1`, userID)
	})

	type fixture struct {
		contentType string
		title       string
		createdAt   time.Time
	}
	now := time.Now().UTC()
	fixtures := []fixture{
		{contentType: "article", title: "Article bookmark", createdAt: now.Add(-3 * time.Minute)},
		{contentType: "gallery", title: "Needle gallery bookmark", createdAt: now.Add(-2 * time.Minute)},
		{contentType: "video", title: "Video bookmark", createdAt: now.Add(-time.Minute)},
	}
	for index, item := range fixtures {
		var articleID uuid.UUID
		slug := fmt.Sprintf("bookmark-%d-%s", index, suffix)
		videoSource, videoURL, duration := any(nil), any(nil), any(nil)
		if item.contentType == "video" {
			videoSource, videoURL, duration = "external", "https://example.test/video.mp4", int32(60)
		}
		if err = pool.QueryRow(ctx, `
			INSERT INTO content.articles(tenant_id,app_id,slug,status,content_type,published_at,video_source_type,video_external_url,video_duration_seconds)
			VALUES($1,$2,$3,'published',$4,$5,$6,$7,$8) RETURNING id
		`, tenantID, appID, slug, item.contentType, item.createdAt, videoSource, videoURL, duration).Scan(&articleID); err != nil {
			t.Fatalf("insert %s article: %v", item.contentType, err)
		}
		if _, err = pool.Exec(ctx, `
			INSERT INTO content.article_translations(article_id,locale,title,summary,body_format,body,search_text)
			VALUES($1,'zh-CN',$2::varchar,'','markdown',to_jsonb('body'::text),lower($2::text))
		`, articleID, item.title); err != nil {
			t.Fatalf("insert %s translation: %v", item.contentType, err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO content.article_bookmarks(tenant_id,app_id,user_id,article_id,created_at) VALUES($1,$2,$3,$4,$5)`, tenantID, appID, userID, articleID, item.createdAt); err != nil {
			t.Fatalf("insert %s bookmark: %v", item.contentType, err)
		}
	}

	repository := NewPostgres(pool, nil)
	gallery, err := repository.ListBookmarks(ctx, tenantID, appID, userID, "zh-CN", content.PublicFilter{ContentType: "gallery", Limit: 20})
	if err != nil {
		t.Fatalf("filter gallery bookmarks: %v", err)
	}
	if len(gallery.Items) != 1 || gallery.Items[0].ContentType != "gallery" {
		t.Fatalf("gallery filter returned %#v", gallery.Items)
	}
	searched, err := repository.ListBookmarks(ctx, tenantID, appID, userID, "zh-CN", content.PublicFilter{Query: "needle", Limit: 20})
	if err != nil {
		t.Fatalf("search bookmarks: %v", err)
	}
	if len(searched.Items) != 1 || searched.Items[0].ContentType != "gallery" {
		t.Fatalf("bookmark search returned %#v", searched.Items)
	}
	first, err := repository.ListBookmarks(ctx, tenantID, appID, userID, "zh-CN", content.PublicFilter{Limit: 1})
	if err != nil {
		t.Fatalf("first bookmark page: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == nil {
		t.Fatalf("first bookmark page=%#v", first)
	}
	second, err := repository.ListBookmarks(ctx, tenantID, appID, userID, "zh-CN", content.PublicFilter{Cursor: *first.NextCursor, Limit: 1})
	if err != nil {
		t.Fatalf("second bookmark page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("bookmark cursor did not advance: first=%#v second=%#v", first, second)
	}
}
