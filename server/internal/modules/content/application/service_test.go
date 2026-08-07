package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	value iam.AuthenticatedContext
	err   error
	want  string
}

func TestContentPermissionUsesAppMatrixOnlyForNestedAppRoute(t *testing.T) {
	if got := contentPermission(uuid.New(), "content.article.create", "create"); got != "app.content.create" {
		t.Fatalf("nested App route permission = %q", got)
	}
	if got := contentPermission(uuid.Nil, "content.article.create", "create"); got != "content.article.create" {
		t.Fatalf("legacy route permission = %q", got)
	}
}

func (f fakeAuthenticator) Authenticate(_ context.Context, _ string, audience string) (iam.AuthenticatedContext, error) {
	if f.want != "" && audience != f.want {
		return iam.AuthenticatedContext{}, errors.New("wrong audience")
	}
	return f.value, f.err
}

type fakeRepository struct {
	content.Repository
	article content.Article
	called  bool
}

func (f *fakeRepository) CreateArticle(_ context.Context, p content.Principal, x content.Article) (content.Article, error) {
	f.called = true
	if p.TenantID == uuid.Nil || p.AppID == uuid.Nil || p.UserID == uuid.Nil {
		return content.Article{}, errors.New("principal was not scoped")
	}
	f.article = x
	return x, nil
}
func (f *fakeRepository) ListPublished(_ context.Context, tenant, appID, user uuid.UUID, locale string, _ content.PublicFilter) (content.PublicArticlePage, error) {
	if tenant == uuid.Nil || appID == uuid.Nil || user == uuid.Nil || locale != "en-US" {
		return content.PublicArticlePage{}, errors.New("mobile scope missing")
	}
	return content.PublicArticlePage{}, nil
}
func (f *fakeRepository) ListPublishedCategories(_ context.Context, tenant, appID uuid.UUID, locale string) (content.PublicCategoryPage, error) {
	if tenant == uuid.Nil || appID == uuid.Nil || locale != "en-US" {
		return content.PublicCategoryPage{}, errors.New("category scope missing")
	}
	return content.PublicCategoryPage{}, nil
}
func (f *fakeRepository) OpenArticleAsset(_ context.Context, tenant, appID, fileID uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	if tenant == uuid.Nil || appID == uuid.Nil || fileID == uuid.Nil {
		return content.ArticleAsset{}, nil, errors.New("asset scope missing")
	}
	return content.ArticleAsset{FileID: fileID, MediaType: "image/png", SizeBytes: 3}, io.NopCloser(strings.NewReader("png")), nil
}

func auth(perms ...string) iam.AuthenticatedContext {
	appID := uuid.New()
	return iam.AuthenticatedContext{AuthContext: iam.AuthContext{Tenant: iam.Tenant{ID: uuid.New()}, User: iam.User{ID: uuid.New()}, Permissions: perms}, SessionID: uuid.New(), AppID: &appID}
}
func article() content.Article {
	return content.Article{Slug: "hello-world", ReadingMinutes: 3, Translations: map[string]content.Translation{"zh-CN": {Title: "你好", Summary: "摘要", BodyFormat: "markdown", Body: []byte(`"正文"`)}, "en-US": {Title: "Hello", Summary: "Summary", BodyFormat: "blocks", Body: []byte(`"[{\"type\":\"paragraph\",\"text\":\"Body\"}]"`)}}}
}

func TestCreateArticleScopesPrincipalAndRequiresBothLocales(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth("app.content.create"), want: adminAudience}, repo)
	if _, err := service.CreateArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "request"}, article()); err != nil {
		t.Fatalf("CreateArticle() error=%v", err)
	}
	if !repo.called {
		t.Fatal("repository was not called")
	}
	if got := string(repo.article.Translations["en-US"].Body); got != `[{"type":"paragraph","text":"Body"}]` {
		t.Fatalf("blocks body=%s, want a normalized JSON array", got)
	}
	x := article()
	delete(x.Translations, "en-US")
	if _, err := service.CreateArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "request"}, x); !errors.Is(err, content.ErrInvalid) {
		t.Fatalf("missing locale error=%v, want ErrInvalid", err)
	}
}

func TestCreateArticleRejectsBodyFormatMismatchAndMissingPermission(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth("app.content.read"), want: adminAudience}, repo)
	if _, err := service.CreateArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "request"}, article()); !errors.Is(err, content.ErrForbidden) {
		t.Fatalf("permission error=%v, want ErrForbidden", err)
	}
	service = NewService(fakeAuthenticator{value: auth("app.content.create"), want: adminAudience}, repo)
	x := article()
	zh := x.Translations["zh-CN"]
	zh.BodyFormat = "blocks"
	zh.Body = []byte(`"not an array"`)
	x.Translations["zh-CN"] = zh
	if _, err := service.CreateArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "request"}, x); !errors.Is(err, content.ErrInvalid) {
		t.Fatalf("body mismatch error=%v, want ErrInvalid", err)
	}
}

func TestListPublishedUsesMobileAudienceScope(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth(), want: mobileAudience}, repo)
	if _, err := service.ListPublished(context.Background(), "token", "en-US", content.PublicFilter{Limit: 20}); err != nil {
		t.Fatalf("ListPublished() error=%v", err)
	}
}

func TestListPublishedCategoriesUsesMobileAudienceScope(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth(), want: mobileAudience}, repo)
	if _, err := service.ListPublishedCategories(context.Background(), "token", "en-US"); err != nil {
		t.Fatalf("ListPublishedCategories() error=%v", err)
	}
}

func TestOpenArticleAssetUsesMobileAudienceAndTenantScope(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth(), want: mobileAudience}, repo)
	asset, reader, err := service.OpenArticleAsset(context.Background(), "token", uuid.New())
	if err != nil {
		t.Fatalf("OpenArticleAsset() error=%v", err)
	}
	defer reader.Close()
	if asset.MediaType != "image/png" {
		t.Fatalf("OpenArticleAsset() media type=%q", asset.MediaType)
	}
}
