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
func (f *fakeRepository) ListPublished(_ context.Context, tenant, appID uuid.UUID, user *uuid.UUID, locale string, _ content.PublicFilter) (content.PublicArticlePage, error) {
	if tenant == uuid.Nil || appID == uuid.Nil || user == nil || *user == uuid.Nil || locale != "en-US" {
		return content.PublicArticlePage{}, errors.New("mobile scope missing")
	}
	return content.PublicArticlePage{}, nil
}
func (f *fakeRepository) GetArticle(_ context.Context, _, _, _ uuid.UUID) (content.Article, error) {
	return f.article, nil
}
func (f *fakeRepository) TransitionArticle(_ context.Context, _ content.Principal, _ uuid.UUID, _ int32, state string) (content.Article, error) {
	f.called = true
	f.article.Status = state
	return f.article, nil
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
	return content.Article{Slug: "hello-world", ContentType: "article", CategoryIDs: []uuid.UUID{uuid.New()}, AllowComments: true, ReadingMinutes: 3, Translations: map[string]content.Translation{"zh-CN": {Title: "你好", Summary: "摘要", BodyFormat: "markdown", Body: []byte(`"正文"`)}, "en-US": {Title: "Hello", Summary: "Summary", BodyFormat: "blocks", Body: []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Body"}]}]}`)}}}
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
	if got := string(repo.article.Translations["en-US"].Body); got != `"Body"` || repo.article.Translations["en-US"].BodyFormat != "markdown" {
		t.Fatalf("legacy body=%s format=%s, want Markdown string", got, repo.article.Translations["en-US"].BodyFormat)
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

func TestDraftAllowsIncompleteLocaleButPublishRequiresCompleteLocales(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(fakeAuthenticator{value: auth("app.content.create"), want: adminAudience}, repo)
	x := article()
	en := x.Translations["en-US"]
	en.Title = ""
	en.Summary = ""
	en.Body = []byte(`{"type":"doc","content":[{"type":"paragraph"}]}`)
	x.Translations["en-US"] = en
	if _, err := service.CreateArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "request"}, x); err != nil {
		t.Fatalf("incomplete draft error=%v", err)
	}

	x.ID = uuid.New()
	x.LockVersion = 1
	repo.article = x
	repo.called = false
	service = NewService(fakeAuthenticator{value: auth("app.content.publish"), want: adminAudience}, repo)
	if _, err := service.TransitionArticle(context.Background(), "token", uuid.New(), content.Principal{RequestID: "publish"}, x.ID, 1, "published"); !errors.Is(err, content.ErrInvalid) {
		t.Fatalf("incomplete publish error=%v, want ErrInvalid", err)
	}
	if repo.called {
		t.Fatal("repository transition was called for an invalid publish")
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
