package repository

import (
	"strings"
	"testing"

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
	if got := mobileCoverURL(ptrUUID(uuid.MustParse("00000000-0000-0000-0000-000000000001"))); got == nil || *got != "/api/v1/article-assets/00000000-0000-0000-0000-000000000001" {
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

func ptrUUID(value uuid.UUID) *uuid.UUID { return &value }
