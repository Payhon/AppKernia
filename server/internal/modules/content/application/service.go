package application

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"slices"
	"strings"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"
const mobileAudience = "ak-mobile"

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}
type Service struct {
	auth Authenticator
	repo content.Repository
}

func NewService(auth Authenticator, repo content.Repository) *Service {
	return &Service{auth: auth, repo: repo}
}
func (s *Service) authorize(ctx context.Context, token, permission string) (iam.AuthenticatedContext, error) {
	a, e := s.auth.Authenticate(ctx, token, adminAudience)
	if e != nil {
		return iam.AuthenticatedContext{}, e
	}
	if !slices.Contains(a.Permissions, permission) {
		return iam.AuthenticatedContext{}, content.ErrForbidden
	}
	return a, nil
}
func (s *Service) mobile(ctx context.Context, token string) (iam.AuthenticatedContext, error) {
	return s.auth.Authenticate(ctx, token, mobileAudience)
}
func principal(a iam.AuthenticatedContext, p content.Principal) content.Principal {
	p.TenantID, p.UserID, p.SessionID = a.Tenant.ID, a.User.ID, a.SessionID
	return p
}

func page(f content.PageFilter) (content.PageFilter, error) {
	f.Query, f.Status, f.Sort = strings.TrimSpace(f.Query), strings.TrimSpace(f.Status), strings.TrimSpace(f.Sort)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len([]rune(f.Query)) > 160 {
		return f, content.ErrInvalid
	}
	return f, nil
}
func validateCategory(x content.Category, update bool) error {
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) || !slugPattern.MatchString(x.Slug) || len(x.Slug) > 120 || !oneOf(x.Status, "active", "disabled") || x.SortOrder < 0 || !categoryLocales(x.Translations) {
		return content.ErrInvalid
	}
	return nil
}
func validateArticle(x content.Article, update bool) error {
	for locale, translation := range x.Translations {
		if translation.BodyFormat == "blocks" {
			var encoded string
			if json.Unmarshal(translation.Body, &encoded) == nil && json.Valid([]byte(encoded)) {
				translation.Body = json.RawMessage(encoded)
				x.Translations[locale] = translation
			}
		}
	}
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) || !slugPattern.MatchString(x.Slug) || len(x.Slug) > 160 || x.SortOrder < 0 || x.ReadingMinutes < 1 || x.ReadingMinutes > 120 || !articleLocales(x.Translations) {
		return content.ErrInvalid
	}
	return nil
}
func categoryLocales(values map[string]content.CategoryTranslation) bool {
	if len(values) != 2 {
		return false
	}
	for _, key := range []string{"zh-CN", "en-US"} {
		x, ok := values[key]
		if !ok || len([]rune(strings.TrimSpace(x.Name))) < 1 || len([]rune(x.Name)) > 160 || len([]rune(x.Description)) > 500 {
			return false
		}
	}
	return true
}
func articleLocales(values map[string]content.Translation) bool {
	if len(values) != 2 {
		return false
	}
	for _, key := range []string{"zh-CN", "en-US"} {
		x, ok := values[key]
		if !ok || len([]rune(strings.TrimSpace(x.Title))) < 1 || len([]rune(x.Title)) > 300 || len([]rune(x.Summary)) > 1000 || !oneOf(x.BodyFormat, "markdown", "blocks") || !validBody(x) {
			return false
		}
	}
	return true
}
func validBody(x content.Translation) bool {
	if len(x.Body) == 0 || !json.Valid(x.Body) {
		return false
	}
	var value any
	if json.Unmarshal(x.Body, &value) != nil {
		return false
	}
	if x.BodyFormat == "markdown" {
		_, ok := value.(string)
		return ok
	}
	_, ok := value.([]any)
	return ok
}
func oneOf(value string, valid ...string) bool { return slices.Contains(valid, value) }

func (s *Service) ListCategories(c context.Context, t string, f content.PageFilter) (content.CategoryPage, error) {
	a, e := s.authorize(c, t, "content.category.read")
	if e != nil {
		return content.CategoryPage{}, e
	}
	f, e = page(f)
	if e != nil || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.Sort, "", "sort_order", "updated_desc", "slug") {
		return content.CategoryPage{}, content.ErrInvalid
	}
	return s.repo.ListCategories(c, a.Tenant.ID, f)
}
func (s *Service) GetCategory(c context.Context, t string, id uuid.UUID) (content.Category, error) {
	a, e := s.authorize(c, t, "content.category.read")
	if e != nil {
		return content.Category{}, e
	}
	if id == uuid.Nil {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.GetCategory(c, a.Tenant.ID, id)
}
func (s *Service) CreateCategory(c context.Context, t string, p content.Principal, x content.Category) (content.Category, error) {
	a, e := s.authorize(c, t, "content.category.create")
	if e != nil {
		return content.Category{}, e
	}
	if e = validateCategory(x, false); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.CreateCategory(c, principal(a, p), x)
}
func (s *Service) UpdateCategory(c context.Context, t string, p content.Principal, x content.Category) (content.Category, error) {
	a, e := s.authorize(c, t, "content.category.update")
	if e != nil {
		return content.Category{}, e
	}
	if e = validateCategory(x, true); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.UpdateCategory(c, principal(a, p), x)
}
func (s *Service) DeleteCategory(c context.Context, t string, p content.Principal, id uuid.UUID, v int32) error {
	a, e := s.authorize(c, t, "content.category.delete")
	if e != nil {
		return e
	}
	if id == uuid.Nil || v < 1 || strings.TrimSpace(p.RequestID) == "" {
		return content.ErrInvalid
	}
	return s.repo.DeleteCategory(c, principal(a, p), id, v)
}
func (s *Service) ListArticles(c context.Context, t string, f content.PageFilter) (content.ArticlePage, error) {
	a, e := s.authorize(c, t, "content.article.read")
	if e != nil {
		return content.ArticlePage{}, e
	}
	f, e = page(f)
	if e != nil || !oneOf(f.Status, "", "draft", "published", "archived") || !oneOf(f.Sort, "", "updated_desc", "published_desc", "sort_order", "slug") {
		return content.ArticlePage{}, content.ErrInvalid
	}
	return s.repo.ListArticles(c, a.Tenant.ID, f)
}
func (s *Service) GetArticle(c context.Context, t string, id uuid.UUID) (content.Article, error) {
	a, e := s.authorize(c, t, "content.article.read")
	if e != nil {
		return content.Article{}, e
	}
	if id == uuid.Nil {
		return content.Article{}, content.ErrInvalid
	}
	return s.repo.GetArticle(c, a.Tenant.ID, id)
}
func (s *Service) CreateArticle(c context.Context, t string, p content.Principal, x content.Article) (content.Article, error) {
	a, e := s.authorize(c, t, "content.article.create")
	if e != nil {
		return content.Article{}, e
	}
	if e = validateArticle(x, false); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	return s.repo.CreateArticle(c, principal(a, p), x)
}
func (s *Service) UpdateArticle(c context.Context, t string, p content.Principal, x content.Article) (content.Article, error) {
	a, e := s.authorize(c, t, "content.article.update")
	if e != nil {
		return content.Article{}, e
	}
	if e = validateArticle(x, true); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	return s.repo.UpdateArticle(c, principal(a, p), x)
}
func (s *Service) DeleteArticle(c context.Context, t string, p content.Principal, id uuid.UUID, v int32) error {
	a, e := s.authorize(c, t, "content.article.delete")
	if e != nil {
		return e
	}
	if id == uuid.Nil || v < 1 || strings.TrimSpace(p.RequestID) == "" {
		return content.ErrInvalid
	}
	return s.repo.DeleteArticle(c, principal(a, p), id, v)
}
func (s *Service) TransitionArticle(c context.Context, t string, p content.Principal, id uuid.UUID, v int32, state string) (content.Article, error) {
	perm := "content.article.publish"
	if state == "archived" {
		perm = "content.article.archive"
	}
	a, e := s.authorize(c, t, perm)
	if e != nil {
		return content.Article{}, e
	}
	if id == uuid.Nil || v < 1 || !oneOf(state, "published", "draft", "archived") || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	return s.repo.TransitionArticle(c, principal(a, p), id, v, state)
}
func (s *Service) ListPublished(c context.Context, t, locale string, f content.PublicFilter) (content.PublicArticlePage, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.PublicArticlePage{}, e
	}
	f.Query, f.CategorySlug, f.Cursor = strings.TrimSpace(f.Query), strings.TrimSpace(f.CategorySlug), strings.TrimSpace(f.Cursor)
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 50 || len([]rune(f.Query)) > 160 || (f.CategorySlug != "" && !slugPattern.MatchString(f.CategorySlug)) {
		return content.PublicArticlePage{}, content.ErrInvalid
	}
	return s.repo.ListPublished(c, a.Tenant.ID, a.User.ID, locale, f)
}
func (s *Service) ListPublishedCategories(c context.Context, t, locale string) (content.PublicCategoryPage, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.PublicCategoryPage{}, e
	}
	return s.repo.ListPublishedCategories(c, a.Tenant.ID, locale)
}
func (s *Service) GetPublished(c context.Context, t, locale, slug string) (content.PublicArticle, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.PublicArticle{}, e
	}
	if !slugPattern.MatchString(slug) {
		return content.PublicArticle{}, content.ErrInvalid
	}
	return s.repo.GetPublished(c, a.Tenant.ID, a.User.ID, locale, slug)
}
func (s *Service) OpenArticleAsset(c context.Context, t string, id uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.ArticleAsset{}, nil, e
	}
	if id == uuid.Nil {
		return content.ArticleAsset{}, nil, content.ErrInvalid
	}
	return s.repo.OpenArticleAsset(c, a.Tenant.ID, id)
}
func (s *Service) Bookmark(c context.Context, t string, id uuid.UUID) error {
	a, e := s.mobile(c, t)
	if e != nil {
		return e
	}
	if id == uuid.Nil {
		return content.ErrInvalid
	}
	return s.repo.Bookmark(c, a.Tenant.ID, a.User.ID, id)
}
func (s *Service) RemoveBookmark(c context.Context, t string, id uuid.UUID) error {
	a, e := s.mobile(c, t)
	if e != nil {
		return e
	}
	if id == uuid.Nil {
		return content.ErrInvalid
	}
	return s.repo.RemoveBookmark(c, a.Tenant.ID, a.User.ID, id)
}
