package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPublicArticleJSONPreservesMobileContractFields(t *testing.T) {
	article := PublicArticle{
		ID:             uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Slug:           "welcome",
		Featured:       true,
		PublishedAt:    pointer(time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)),
		Title:          "Welcome",
		Summary:        "Summary",
		BodyFormat:     "markdown",
		Body:           json.RawMessage(`"Body"`),
		Category:       &PublicCategory{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"), Slug: "guides", Name: "Guides"},
		ReadingMinutes: 5,
		Bookmarked:     true,
	}
	raw, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("marshal public article: %v", err)
	}
	var value map[string]any
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal public article: %v", err)
	}
	for _, field := range []string{"featured", "published_at", "category", "reading_minutes", "bookmarked"} {
		if _, ok := value[field]; !ok {
			t.Fatalf("missing %q in mobile article response: %s", field, raw)
		}
	}
	for _, internalField := range []string{"category_id", "sort_order"} {
		if _, ok := value[internalField]; ok {
			t.Fatalf("public article must not expose internal %q: %s", internalField, raw)
		}
	}
	category, ok := value["category"].(map[string]any)
	if !ok || category["name"] != "Guides" {
		t.Fatalf("category is not the localized object: %#v", value["category"])
	}
}

func TestPublicArticleJSONIncludesNullCategory(t *testing.T) {
	raw, err := json.Marshal(PublicArticle{ID: uuid.New(), Slug: "uncategorized", Title: "Title", Summary: "", BodyFormat: "markdown", Body: json.RawMessage(`"Body"`)})
	if err != nil {
		t.Fatalf("marshal public article without category: %v", err)
	}
	var value map[string]any
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal public article without category: %v", err)
	}
	if category, ok := value["category"]; !ok || category != nil {
		t.Fatalf("category must be explicit null, got %#v", value["category"])
	}
}

func pointer(value time.Time) *time.Time { return &value }
