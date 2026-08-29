package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
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

func contentPermission(appID uuid.UUID, legacyPermission, appAction string) string {
	if appID != uuid.Nil {
		return "app.content." + appAction
	}
	return legacyPermission
}

// authorizeContent preserves the legacy system-content permission matrix while
// enforcing the App matrix for nested /apps/{app_id}/content routes.
func (s *Service) authorizeContent(ctx context.Context, token string, appID uuid.UUID, legacyPermission, appAction string) (iam.AuthenticatedContext, error) {
	return s.authorize(ctx, token, contentPermission(appID, legacyPermission, appAction))
}
func (s *Service) mobile(ctx context.Context, token string) (iam.AuthenticatedContext, error) {
	return s.auth.Authenticate(ctx, token, mobileAudience)
}
func principal(a iam.AuthenticatedContext, appID uuid.UUID, p content.Principal) content.Principal {
	p.TenantID, p.AppID, p.UserID, p.SessionID = a.Tenant.ID, appID, a.User.ID, a.SessionID
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
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) || (x.ParentID != nil && *x.ParentID == x.ID) || !slugPattern.MatchString(x.Slug) || len(x.Slug) > 120 || !oneOf(x.Status, "active", "disabled") || x.SortOrder < 0 || !categoryLocales(x.Translations) {
		return content.ErrInvalid
	}
	return nil
}
func validateArticle(x content.Article, update, publish bool) error {
	for locale, translation := range x.Translations {
		if translation.BodyFormat == "blocks" {
			if markdown, ok := legacyBodyToMarkdown(translation.Body); ok {
				translation.BodyFormat = "markdown"
				translation.Body = json.RawMessage(fmt.Sprintf("%q", markdown))
				x.Translations[locale] = translation
			}
		}
	}
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) || !slugPattern.MatchString(x.Slug) || len(x.Slug) > 160 || x.SortOrder < 0 || x.ReadingMinutes < 1 || x.ReadingMinutes > 120 || !oneOf(x.ContentType, "article", "gallery", "video") || len(x.CategoryIDs) > 10 || (len(x.CategoryIDs) > 0 && !uniqueUUIDs(x.CategoryIDs)) || len(x.TagIDs) > 10 || (len(x.TagIDs) > 0 && !uniqueUUIDs(x.TagIDs)) || !articleLocalesForType(x.ContentType, x.Translations, publish) || !validMedia(x, publish) || !validVideo(x) {
		return content.ErrInvalid
	}
	if publish && len(x.CategoryIDs) < 1 {
		return content.ErrInvalid
	}
	return nil
}
func normalizeArticle(x *content.Article) {
	if x.ContentType == "" {
		x.ContentType = "article"
	}
	if len(x.CategoryIDs) == 0 && x.CategoryID != nil {
		x.CategoryIDs = []uuid.UUID{*x.CategoryID}
	}
	if x.CategoryID == nil && len(x.CategoryIDs) > 0 {
		first := x.CategoryIDs[0]
		x.CategoryID = &first
	}
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
	return articleLocalesForType("article", values, true)
}
func articleLocalesForType(contentType string, values map[string]content.Translation, publish bool) bool {
	if len(values) != 2 {
		return false
	}
	for _, key := range []string{"zh-CN", "en-US"} {
		x, ok := values[key]
		limit := 30000
		if contentType == "gallery" {
			limit = 3000
		}
		if contentType == "video" {
			limit = 1000
		}
		summaryLimit := 1000
		if contentType == "gallery" {
			summaryLimit = 3000
		}
		titleLength := len([]rune(strings.TrimSpace(x.Title)))
		summaryLength := len([]rune(strings.TrimSpace(x.Summary)))
		if !ok || titleLength > 300 || len([]rune(x.Summary)) > summaryLimit || x.BodyFormat != "markdown" || !validBody(x) || visibleLength(x.Body) > limit {
			return false
		}
		if publish && (titleLength < 1 || summaryLength < 1 || (contentType == "article" && visibleLength(x.Body) < 1)) {
			return false
		}
	}
	return true
}
func visibleLength(raw json.RawMessage) int {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return 1 << 30
	}
	return len([]rune(strings.TrimSpace(visibleText(value))))
}
func visibleText(value any) string {
	switch x := value.(type) {
	case string:
		return x
	case []any:
		parts := make([]string, 0, len(x))
		for _, child := range x {
			parts = append(parts, visibleText(child))
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		parts := []string{}
		if text, ok := x["text"].(string); ok {
			parts = append(parts, text)
		}
		if children, ok := x["content"].([]any); ok {
			parts = append(parts, visibleText(children))
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}
func uniqueUUIDs(values []uuid.UUID) bool {
	seen := map[uuid.UUID]bool{}
	for _, value := range values {
		if value == uuid.Nil || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func validMedia(x content.Article, publish bool) bool {
	if x.ContentType == "gallery" && (len(x.Media) > 9 || (publish && len(x.Media) < 1)) {
		return false
	}
	if x.ContentType != "gallery" && len(x.Media) > 100 {
		return false
	}
	seen := map[uuid.UUID]bool{}
	for _, media := range x.Media {
		if media.FileID == uuid.Nil || seen[media.FileID] || !oneOf(media.Role, "gallery", "inline") || len(media.Translations) != 2 || (x.ContentType == "gallery" && media.Role != "gallery") || (x.ContentType == "article" && media.Role != "inline") {
			return false
		}
		seen[media.FileID] = true
		for _, locale := range []string{"zh-CN", "en-US"} {
			alt, ok := media.Translations[locale]
			altLength := len([]rune(strings.TrimSpace(alt.AltText)))
			if !ok || len([]rune(alt.AltText)) > 500 || (publish && altLength < 1) {
				return false
			}
		}
	}
	return true
}
func validVideo(x content.Article) bool {
	if x.ContentType != "video" {
		return x.VideoSourceType == nil && x.VideoFileID == nil && x.VideoExternalURL == nil && x.VideoDurationSeconds == nil
	}
	if x.VideoSourceType == nil || !oneOf(*x.VideoSourceType, "upload", "external") {
		return false
	}
	if x.VideoDurationSeconds != nil && (*x.VideoDurationSeconds < 1 || *x.VideoDurationSeconds > 86400) {
		return false
	}
	if *x.VideoSourceType == "upload" {
		return x.VideoFileID != nil && x.VideoExternalURL == nil
	}
	if x.VideoFileID != nil || x.VideoExternalURL == nil {
		return false
	}
	u, err := url.Parse(*x.VideoExternalURL)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && regexp.MustCompile(`(?i)\.(mp4|m3u8)$`).MatchString(u.Path)
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
		text, ok := value.(string)
		return ok && validMarkdown(text)
	}
	if root, ok := value.(map[string]any); ok {
		return root["type"] == "doc" && validDocumentChildren(root["content"], 0)
	}
	_, legacy := value.([]any)
	return legacy
}
func validDocumentChildren(value any, depth int) bool {
	if depth > 20 {
		return false
	}
	children, ok := value.([]any)
	if !ok {
		return false
	}
	for _, child := range children {
		node, ok := child.(map[string]any)
		if !ok || !validDocumentNode(node, depth+1) {
			return false
		}
	}
	return true
}
func validDocumentNode(node map[string]any, depth int) bool {
	typeName, ok := node["type"].(string)
	if !ok || !oneOf(typeName, "paragraph", "heading", "text", "bulletList", "orderedList", "listItem", "blockquote", "codeBlock", "horizontalRule", "image", "hardBreak") {
		return false
	}
	if typeName == "text" {
		if _, ok = node["text"].(string); !ok {
			return false
		}
		if marks, exists := node["marks"]; exists {
			list, ok := marks.([]any)
			if !ok {
				return false
			}
			for _, item := range list {
				mark, ok := item.(map[string]any)
				if !ok {
					return false
				}
				kind, _ := mark["type"].(string)
				if !oneOf(kind, "bold", "italic", "code", "link") {
					return false
				}
				if kind == "link" {
					attrs, ok := mark["attrs"].(map[string]any)
					if !ok {
						return false
					}
					href, ok := attrs["href"].(string)
					if !ok || !safeDocumentLink(href) {
						return false
					}
				}
			}
		}
		return true
	}
	if typeName == "heading" {
		attrs, ok := node["attrs"].(map[string]any)
		if !ok {
			return false
		}
		level, ok := attrs["level"].(float64)
		if !ok || (level != 2 && level != 3) {
			return false
		}
	}
	if typeName == "image" {
		attrs, ok := node["attrs"].(map[string]any)
		if !ok {
			return false
		}
		fileID, ok := attrs["file_id"].(string)
		if !ok {
			return false
		}
		if _, err := uuid.Parse(fileID); err != nil {
			return false
		}
		return true
	}
	if typeName == "horizontalRule" || typeName == "hardBreak" {
		return true
	}
	if (typeName == "paragraph" || typeName == "codeBlock") && node["content"] == nil {
		return true
	}
	return validDocumentChildren(node["content"], depth)
}
func safeDocumentLink(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && u.User == nil && ((u.Scheme == "https" && u.Host != "") || (u.Scheme == "mailto" && u.Opaque != ""))
}
func oneOf(value string, valid ...string) bool { return slices.Contains(valid, value) }

func (s *Service) ListCategories(c context.Context, t string, appID uuid.UUID, f content.PageFilter) (content.CategoryPage, error) {
	a, e := s.authorizeContent(c, t, appID, "content.category.read", "read")
	if e != nil {
		return content.CategoryPage{}, e
	}
	f, e = page(f)
	if e != nil || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.Sort, "", "sort_order", "updated_desc", "slug") {
		return content.CategoryPage{}, content.ErrInvalid
	}
	return s.repo.ListCategories(c, a.Tenant.ID, appID, f)
}
func (s *Service) GetCategory(c context.Context, t string, appID, id uuid.UUID) (content.Category, error) {
	a, e := s.authorizeContent(c, t, appID, "content.category.read", "read")
	if e != nil {
		return content.Category{}, e
	}
	if id == uuid.Nil {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.GetCategory(c, a.Tenant.ID, appID, id)
}
func (s *Service) CreateCategory(c context.Context, t string, appID uuid.UUID, p content.Principal, x content.Category) (content.Category, error) {
	a, e := s.authorizeContent(c, t, appID, "content.category.create", "create")
	if e != nil {
		return content.Category{}, e
	}
	if e = validateCategory(x, false); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.CreateCategory(c, principal(a, appID, p), x)
}
func (s *Service) UpdateCategory(c context.Context, t string, appID uuid.UUID, p content.Principal, x content.Category) (content.Category, error) {
	a, e := s.authorizeContent(c, t, appID, "content.category.update", "update")
	if e != nil {
		return content.Category{}, e
	}
	if e = validateCategory(x, true); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Category{}, content.ErrInvalid
	}
	return s.repo.UpdateCategory(c, principal(a, appID, p), x)
}
func (s *Service) DeleteCategory(c context.Context, t string, appID uuid.UUID, p content.Principal, id uuid.UUID, v int32) error {
	a, e := s.authorizeContent(c, t, appID, "content.category.delete", "delete")
	if e != nil {
		return e
	}
	if id == uuid.Nil || v < 1 || strings.TrimSpace(p.RequestID) == "" {
		return content.ErrInvalid
	}
	return s.repo.DeleteCategory(c, principal(a, appID, p), id, v)
}
func (s *Service) ListArticles(c context.Context, t string, appID uuid.UUID, f content.PageFilter) (content.ArticlePage, error) {
	a, e := s.authorizeContent(c, t, appID, "content.article.read", "read")
	if e != nil {
		return content.ArticlePage{}, e
	}
	f, e = page(f)
	if e != nil || !oneOf(f.Status, "", "draft", "published", "archived") || !oneOf(f.ContentType, "", "article", "gallery", "video") || !oneOf(f.Sort, "", "updated_desc", "published_desc", "sort_order", "slug") {
		return content.ArticlePage{}, content.ErrInvalid
	}
	result, err := s.repo.ListArticles(c, a.Tenant.ID, appID, f)
	for index := range result.Items {
		normalizeArticleBodies(&result.Items[index])
	}
	return result, err
}
func (s *Service) GetArticle(c context.Context, t string, appID, id uuid.UUID) (content.Article, error) {
	a, e := s.authorizeContent(c, t, appID, "content.article.read", "read")
	if e != nil {
		return content.Article{}, e
	}
	if id == uuid.Nil {
		return content.Article{}, content.ErrInvalid
	}
	result, err := s.repo.GetArticle(c, a.Tenant.ID, appID, id)
	normalizeArticleBodies(&result)
	return result, err
}
func (s *Service) CreateArticle(c context.Context, t string, appID uuid.UUID, p content.Principal, x content.Article) (content.Article, error) {
	a, e := s.authorizeContent(c, t, appID, "content.article.create", "create")
	if e != nil {
		return content.Article{}, e
	}
	normalizeArticle(&x)
	if e = validateArticle(x, false, false); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	normalizeArticleBodies(&x)
	return s.repo.CreateArticle(c, principal(a, appID, p), x)
}
func (s *Service) UpdateArticle(c context.Context, t string, appID uuid.UUID, p content.Principal, x content.Article) (content.Article, error) {
	a, e := s.authorizeContent(c, t, appID, "content.article.update", "update")
	if e != nil {
		return content.Article{}, e
	}
	normalizeArticle(&x)
	if e = validateArticle(x, true, false); e != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	normalizeArticleBodies(&x)
	return s.repo.UpdateArticle(c, principal(a, appID, p), x)
}
func (s *Service) DeleteArticle(c context.Context, t string, appID uuid.UUID, p content.Principal, id uuid.UUID, v int32) error {
	a, e := s.authorizeContent(c, t, appID, "content.article.delete", "delete")
	if e != nil {
		return e
	}
	if id == uuid.Nil || v < 1 || strings.TrimSpace(p.RequestID) == "" {
		return content.ErrInvalid
	}
	return s.repo.DeleteArticle(c, principal(a, appID, p), id, v)
}
func (s *Service) TransitionArticle(c context.Context, t string, appID uuid.UUID, p content.Principal, id uuid.UUID, v int32, state string) (content.Article, error) {
	perm := "content.article.publish"
	appAction := "publish"
	if state == "archived" {
		perm = "content.article.archive"
		appAction = "update"
	}
	a, e := s.authorizeContent(c, t, appID, perm, appAction)
	if e != nil {
		return content.Article{}, e
	}
	if id == uuid.Nil || v < 1 || !oneOf(state, "published", "draft", "archived") || strings.TrimSpace(p.RequestID) == "" {
		return content.Article{}, content.ErrInvalid
	}
	if state == "published" {
		article, getErr := s.repo.GetArticle(c, a.Tenant.ID, appID, id)
		if getErr != nil {
			return content.Article{}, getErr
		}
		if article.LockVersion != v {
			return content.Article{}, content.ErrConflict
		}
		normalizeArticle(&article)
		if validateArticle(article, true, true) != nil {
			return content.Article{}, content.ErrInvalid
		}
	}
	result, err := s.repo.TransitionArticle(c, principal(a, appID, p), id, v, state)
	normalizeArticleBodies(&result)
	return result, err
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
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.PublicArticlePage{}, content.ErrForbidden
	}
	userID := a.User.ID
	result, err := s.repo.ListPublished(c, a.Tenant.ID, *a.AppID, &userID, locale, f)
	for index := range result.Items {
		normalizePublicArticleBody(&result.Items[index])
	}
	return result, err
}
func (s *Service) ListPublishedCategories(c context.Context, t, locale string) (content.PublicCategoryPage, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.PublicCategoryPage{}, e
	}
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.PublicCategoryPage{}, content.ErrForbidden
	}
	return s.repo.ListPublishedCategories(c, a.Tenant.ID, *a.AppID, locale)
}
func (s *Service) GetPublished(c context.Context, t, locale, slug string) (content.PublicArticle, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.PublicArticle{}, e
	}
	if !slugPattern.MatchString(slug) {
		return content.PublicArticle{}, content.ErrInvalid
	}
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.PublicArticle{}, content.ErrForbidden
	}
	userID := a.User.ID
	result, err := s.repo.GetPublished(c, a.Tenant.ID, *a.AppID, &userID, locale, slug)
	normalizePublicArticleBody(&result)
	return result, err
}
func (s *Service) OpenArticleAsset(c context.Context, t string, id uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	a, e := s.mobile(c, t)
	if e != nil {
		return content.ArticleAsset{}, nil, e
	}
	if id == uuid.Nil {
		return content.ArticleAsset{}, nil, content.ErrInvalid
	}
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.ArticleAsset{}, nil, content.ErrForbidden
	}
	return s.repo.OpenArticleAsset(c, a.Tenant.ID, *a.AppID, id)
}
func (s *Service) Bookmark(c context.Context, t string, id uuid.UUID) error {
	a, e := s.mobile(c, t)
	if e != nil {
		return e
	}
	if id == uuid.Nil {
		return content.ErrInvalid
	}
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.ErrForbidden
	}
	return s.repo.Bookmark(c, a.Tenant.ID, *a.AppID, a.User.ID, id)
}
func (s *Service) RemoveBookmark(c context.Context, t string, id uuid.UUID) error {
	a, e := s.mobile(c, t)
	if e != nil {
		return e
	}
	if id == uuid.Nil {
		return content.ErrInvalid
	}
	if a.AppID == nil || *a.AppID == uuid.Nil {
		return content.ErrForbidden
	}
	return s.repo.RemoveBookmark(c, a.Tenant.ID, *a.AppID, a.User.ID, id)
}
