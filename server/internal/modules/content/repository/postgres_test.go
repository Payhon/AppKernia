package repository

import (
	"strings"
	"testing"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/google/uuid"
)

func TestArticleAssetPolicyRejectsUnsafeFiles(t *testing.T) {
	for _, test := range []struct {
		status, scan, media string
		allowed             bool
	}{
		{"ready", "clean", "image/png", true},
		{"ready", "skipped", "image/webp", true},
		{"pending", "clean", "image/png", false},
		{"ready", "infected", "image/png", false},
		{"ready", "clean", "application/pdf", false},
	} {
		if got := articleAssetAllowed(test.status, test.scan, test.media); got != test.allowed {
			t.Fatalf("articleAssetAllowed(%q,%q,%q)=%v, want %v", test.status, test.scan, test.media, got, test.allowed)
		}
	}
}

func TestArticleAssetQueryRequiresTenantPublishedCoverAndSafeImage(t *testing.T) {
	for _, clause := range []string{
		"f.tenant_id=$1", "f.status='ready'", "f.scan_status IN ('clean','skipped')",
		"image/jpeg", "a.tenant_id=$1", "a.cover_file_id=f.id", "a.status='published'",
	} {
		if !strings.Contains(articleAssetSelect, clause) {
			t.Fatalf("article asset query missing guard %q", clause)
		}
	}
	if got := mobileCoverURL(ptrUUID(uuid.MustParse("00000000-0000-0000-0000-000000000001"))); got == nil || *got != "/api/v1/public/content/assets/00000000-0000-0000-0000-000000000001" {
		t.Fatalf("mobile cover URL=%v", got)
	}
}

func TestPublicProjectionHidesDisabledCategories(t *testing.T) {
	if !strings.Contains(publicArticleSelect, "c.status='active'") {
		t.Fatal("public article projection must join active categories only")
	}
	for _, clause := range []string{"cover.status='ready'", "cover.scan_status IN ('clean','skipped')", "image/png"} {
		if !strings.Contains(publicArticleSelect, clause) {
			t.Fatalf("public article projection missing safe cover guard %q", clause)
		}
	}
}

func TestTransitionArticleStatusQueryPinsTheStateParameterType(t *testing.T) {
	for _, clause := range []string{"status=$1::varchar", "$1::varchar='published'", "$1::varchar='draft'"} {
		if !strings.Contains(transitionArticleStatusSQL, clause) {
			t.Fatalf("article transition query must pin the shared state parameter type with %q", clause)
		}
	}
}

func TestCommentRateLimitQueryUsesEveryBoundParameter(t *testing.T) {
	if strings.Contains(commentRateLimitSQL, "$5") {
		t.Fatal("comment rate-limit query must not skip a parameter number")
	}
	for _, clause := range []string{"tenant_id=$1", "app_id=$2", "author_id=$3", "body_fingerprint=$4"} {
		if !strings.Contains(commentRateLimitSQL, clause) {
			t.Fatalf("comment rate-limit query missing bound parameter clause %q", clause)
		}
	}
}

func TestBookmarkListQueryAppliesTypeSearchCursorAndLimit(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	appID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	cursor := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	query, args, err := bookmarkListQuery(tenantID, appID, userID, "zh-CN", content.PublicFilter{
		Query:       "相册",
		ContentType: "gallery",
		Cursor:      cursor.String(),
		Limit:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, clause := range []string{
		"bm.tenant_id=a.tenant_id",
		"a.tenant_id=$3",
		"a.app_id=$4",
		"t.search_text ILIKE '%'||$5||'%'",
		"a.content_type=$6",
		"cursor_bm.tenant_id=$3",
		"cursor_bm.app_id=$4",
		"cursor_bm.user_id=$1",
		"cursor_bm.article_id=$7",
		"ORDER BY bm.created_at DESC,a.id DESC LIMIT $8",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("bookmark list query missing %q\n%s", clause, query)
		}
	}
	if len(args) != 8 || args[4] != "相册" || args[5] != "gallery" || args[6] != cursor || args[7] != int32(21) {
		t.Fatalf("unexpected bookmark list args: %#v", args)
	}
}

func TestBookmarkListQueryRejectsInvalidCursor(t *testing.T) {
	_, _, err := bookmarkListQuery(uuid.New(), uuid.New(), uuid.New(), "en-US", content.PublicFilter{Cursor: "not-a-uuid", Limit: 20})
	if err != content.ErrInvalid {
		t.Fatalf("bookmarkListQuery invalid cursor error=%v", err)
	}
}

func ptrUUID(value uuid.UUID) *uuid.UUID { return &value }
