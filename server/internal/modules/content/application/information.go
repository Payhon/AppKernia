package application

import (
	"context"
	"io"
	"strings"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/google/uuid"
)

func (s *Service) publicScope(ctx context.Context, token string, appID uuid.UUID) (uuid.UUID, *uuid.UUID, error) {
	if appID == uuid.Nil {
		return uuid.Nil, nil, content.ErrInvalid
	}
	if strings.TrimSpace(token) == "" {
		tenantID, err := s.repo.ResolvePublicApp(ctx, appID)
		return tenantID, nil, err
	}
	a, err := s.mobile(ctx, token)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if a.AppID == nil || *a.AppID != appID {
		return uuid.Nil, nil, content.ErrForbidden
	}
	userID := a.User.ID
	return a.Tenant.ID, &userID, nil
}

func normalizePublicFilter(f content.PublicFilter) (content.PublicFilter, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.CategorySlug = strings.TrimSpace(f.CategorySlug)
	f.TopicSlug = strings.TrimSpace(f.TopicSlug)
	f.Tag = strings.TrimSpace(strings.TrimPrefix(f.Tag, "#"))
	f.ContentType = strings.TrimSpace(f.ContentType)
	f.Cursor = strings.TrimSpace(f.Cursor)
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 50 || len([]rune(f.Query)) > 160 || !oneOf(f.ContentType, "", "article", "gallery", "video") {
		return f, content.ErrInvalid
	}
	for _, slug := range []string{f.CategorySlug, f.TopicSlug} {
		if slug != "" && !slugPattern.MatchString(slug) {
			return f, content.ErrInvalid
		}
	}
	if len([]rune(f.Tag)) > 80 {
		return f, content.ErrInvalid
	}
	return f, nil
}

func (s *Service) ListPublic(ctx context.Context, token string, appID uuid.UUID, locale string, f content.PublicFilter) (content.PublicArticlePage, error) {
	tenantID, userID, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	f, err = normalizePublicFilter(f)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	return s.repo.ListPublished(ctx, tenantID, appID, userID, locale, f)
}

func (s *Service) GetPublic(ctx context.Context, token string, appID uuid.UUID, locale, slug string) (content.PublicArticle, error) {
	tenantID, userID, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.PublicArticle{}, err
	}
	if !slugPattern.MatchString(slug) {
		return content.PublicArticle{}, content.ErrInvalid
	}
	return s.repo.GetPublished(ctx, tenantID, appID, userID, locale, slug)
}

func (s *Service) PublicCategories(ctx context.Context, token string, appID uuid.UUID, locale string) (content.PublicCategoryPage, error) {
	tenantID, _, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.PublicCategoryPage{}, err
	}
	return s.repo.ListPublishedCategories(ctx, tenantID, appID, locale)
}

func (s *Service) PublicTopics(ctx context.Context, token string, appID uuid.UUID, locale string) (content.PublicTopicPage, error) {
	tenantID, _, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.PublicTopicPage{}, err
	}
	return s.repo.ListPublishedTopics(ctx, tenantID, appID, locale)
}

func (s *Service) PublicTopic(ctx context.Context, token string, appID uuid.UUID, locale, slug string) (content.PublicTopic, error) {
	tenantID, _, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.PublicTopic{}, err
	}
	if !slugPattern.MatchString(slug) {
		return content.PublicTopic{}, content.ErrInvalid
	}
	return s.repo.GetPublishedTopic(ctx, tenantID, appID, locale, slug)
}

func (s *Service) PublicHome(ctx context.Context, token string, appID uuid.UUID, locale string, limit int32) (content.HomeFeed, error) {
	tenantID, userID, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.HomeFeed{}, err
	}
	if limit == 0 {
		limit = 10
	}
	if limit < 1 || limit > 20 {
		return content.HomeFeed{}, content.ErrInvalid
	}
	return s.repo.Home(ctx, tenantID, appID, userID, locale, limit)
}

func (s *Service) OpenPublicAsset(ctx context.Context, token string, appID, fileID uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	tenantID, _, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.ArticleAsset{}, nil, err
	}
	if fileID == uuid.Nil {
		return content.ArticleAsset{}, nil, content.ErrInvalid
	}
	return s.repo.OpenArticleAsset(ctx, tenantID, appID, fileID)
}

func (s *Service) MyBookmarks(ctx context.Context, token, locale string, f content.PublicFilter) (content.PublicArticlePage, error) {
	a, err := s.mobile(ctx, token)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	if a.AppID == nil {
		return content.PublicArticlePage{}, content.ErrForbidden
	}
	f, err = normalizePublicFilter(f)
	if err != nil {
		return content.PublicArticlePage{}, err
	}
	return s.repo.ListBookmarks(ctx, a.Tenant.ID, *a.AppID, a.User.ID, locale, f)
}

func validateTopic(x content.Topic, update bool) error {
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) || !slugPattern.MatchString(x.Slug) || len(x.Slug) > 160 || !oneOf(x.Status, "active", "disabled") || x.SortOrder < 0 || len(x.Translations) != 2 {
		return content.ErrInvalid
	}
	for _, locale := range []string{"zh-CN", "en-US"} {
		value, ok := x.Translations[locale]
		if !ok || len([]rune(strings.TrimSpace(value.Name))) < 1 || len([]rune(value.Name)) > 160 || len([]rune(value.Description)) > 2000 {
			return content.ErrInvalid
		}
	}
	return nil
}

func (s *Service) ListTopics(ctx context.Context, token string, appID uuid.UUID, f content.PageFilter) (content.TopicPage, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.read", "read")
	if err != nil {
		return content.TopicPage{}, err
	}
	f, err = page(f)
	if err != nil || !oneOf(f.Status, "", "active", "disabled") {
		return content.TopicPage{}, content.ErrInvalid
	}
	return s.repo.ListTopics(ctx, a.Tenant.ID, appID, f)
}
func (s *Service) CreateTopic(ctx context.Context, token string, appID uuid.UUID, p content.Principal, x content.Topic) (content.Topic, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.create", "create")
	if err != nil {
		return content.Topic{}, err
	}
	if validateTopic(x, false) != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Topic{}, content.ErrInvalid
	}
	return s.repo.CreateTopic(ctx, principal(a, appID, p), x)
}
func (s *Service) UpdateTopic(ctx context.Context, token string, appID uuid.UUID, p content.Principal, x content.Topic) (content.Topic, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.update", "update")
	if err != nil {
		return content.Topic{}, err
	}
	if validateTopic(x, true) != nil || strings.TrimSpace(p.RequestID) == "" {
		return content.Topic{}, content.ErrInvalid
	}
	return s.repo.UpdateTopic(ctx, principal(a, appID, p), x)
}
func (s *Service) DeleteTopic(ctx context.Context, token string, appID uuid.UUID, p content.Principal, id uuid.UUID, version int32) error {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.delete", "delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil || version < 1 {
		return content.ErrInvalid
	}
	return s.repo.DeleteTopic(ctx, principal(a, appID, p), id, version)
}

func (s *Service) ListTags(ctx context.Context, token string, appID uuid.UUID, f content.PageFilter) (content.TagPage, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.read", "read")
	if err != nil {
		return content.TagPage{}, err
	}
	f, err = page(f)
	if err != nil || !oneOf(f.Status, "", "active", "disabled") {
		return content.TagPage{}, content.ErrInvalid
	}
	return s.repo.ListTags(ctx, a.Tenant.ID, appID, f)
}
func validTagName(value string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	return len([]rune(value)) >= 1 && len([]rune(value)) <= 80 && !strings.ContainsAny(value, "\r\n\t")
}
func (s *Service) UpsertTag(ctx context.Context, token string, appID uuid.UUID, p content.Principal, name string) (content.Tag, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.create", "create")
	if err != nil {
		return content.Tag{}, err
	}
	if !validTagName(name) {
		return content.Tag{}, content.ErrInvalid
	}
	return s.repo.UpsertTag(ctx, principal(a, appID, p), name)
}
func (s *Service) UpdateTag(ctx context.Context, token string, appID uuid.UUID, p content.Principal, x content.Tag) (content.Tag, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.update", "update")
	if err != nil {
		return content.Tag{}, err
	}
	if x.ID == uuid.Nil || x.LockVersion < 1 || !validTagName(x.Name) || !oneOf(x.Status, "active", "disabled") {
		return content.Tag{}, content.ErrInvalid
	}
	return s.repo.UpdateTag(ctx, principal(a, appID, p), x)
}
func (s *Service) MergeTag(ctx context.Context, token string, appID uuid.UUID, p content.Principal, source, target uuid.UUID, version int32) error {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.update", "update")
	if err != nil {
		return err
	}
	if source == uuid.Nil || target == uuid.Nil || source == target || version < 1 {
		return content.ErrInvalid
	}
	return s.repo.MergeTag(ctx, principal(a, appID, p), source, target, version)
}
func (s *Service) DeleteTag(ctx context.Context, token string, appID uuid.UUID, p content.Principal, id uuid.UUID, version int32) error {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.delete", "delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil || version < 1 {
		return content.ErrInvalid
	}
	return s.repo.DeleteTag(ctx, principal(a, appID, p), id, version)
}

func (s *Service) PublicComments(ctx context.Context, token string, appID, articleID uuid.UUID, f content.PageFilter) (content.CommentPage, error) {
	tenantID, userID, err := s.publicScope(ctx, token, appID)
	if err != nil {
		return content.CommentPage{}, err
	}
	f, err = page(f)
	if err != nil {
		return content.CommentPage{}, err
	}
	return s.repo.ListComments(ctx, tenantID, appID, userID, articleID, "approved", f)
}
func (s *Service) AdminComments(ctx context.Context, token string, appID uuid.UUID, f content.PageFilter, articleID *uuid.UUID, status string) (content.CommentPage, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.read", "read")
	if err != nil {
		return content.CommentPage{}, err
	}
	f, err = page(f)
	if err != nil || !oneOf(status, "", "pending", "approved", "rejected", "hidden", "deleted") {
		return content.CommentPage{}, content.ErrInvalid
	}
	id := uuid.Nil
	if articleID != nil {
		id = *articleID
	}
	return s.repo.ListComments(ctx, a.Tenant.ID, appID, nil, id, status, f)
}
func (s *Service) CreateComment(ctx context.Context, token string, p content.Principal, articleID uuid.UUID, parentID *uuid.UUID, body string) (content.Comment, error) {
	a, err := s.mobile(ctx, token)
	if err != nil {
		return content.Comment{}, err
	}
	if a.AppID == nil || articleID == uuid.Nil || len([]rune(strings.TrimSpace(body))) < 1 || len([]rune(strings.TrimSpace(body))) > 500 {
		return content.Comment{}, content.ErrInvalid
	}
	return s.repo.CreateComment(ctx, principal(a, *a.AppID, p), articleID, parentID, strings.TrimSpace(body))
}
func (s *Service) DeleteOwnComment(ctx context.Context, token string, p content.Principal, id uuid.UUID) error {
	a, err := s.mobile(ctx, token)
	if err != nil {
		return err
	}
	if a.AppID == nil || id == uuid.Nil {
		return content.ErrInvalid
	}
	return s.repo.DeleteOwnComment(ctx, principal(a, *a.AppID, p), id)
}
func (s *Service) ModerateComment(ctx context.Context, token string, appID uuid.UUID, p content.Principal, id uuid.UUID, status, reason string) (content.Comment, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.update", "update")
	if err != nil {
		return content.Comment{}, err
	}
	if id == uuid.Nil || !oneOf(status, "approved", "rejected", "hidden") || len([]rune(reason)) > 500 {
		return content.Comment{}, content.ErrInvalid
	}
	return s.repo.ModerateComment(ctx, principal(a, appID, p), id, status, strings.TrimSpace(reason))
}
func (s *Service) BatchModerateComments(ctx context.Context, token string, appID uuid.UUID, p content.Principal, ids []uuid.UUID, status, reason string) ([]content.Comment, error) {
	if len(ids) < 1 || len(ids) > 100 {
		return nil, content.ErrInvalid
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]content.Comment, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, content.ErrInvalid
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		item, err := s.ModerateComment(ctx, token, appID, p, id, status, reason)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
func (s *Service) AdminCommentReports(ctx context.Context, token string, appID uuid.UUID, status string, f content.PageFilter) (content.CommentReportPage, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.read", "read")
	if err != nil {
		return content.CommentReportPage{}, err
	}
	f, err = page(f)
	if err != nil || !oneOf(status, "", "open", "resolved", "dismissed") {
		return content.CommentReportPage{}, content.ErrInvalid
	}
	return s.repo.ListCommentReports(ctx, a.Tenant.ID, appID, status, f)
}
func (s *Service) ResolveCommentReport(ctx context.Context, token string, appID uuid.UUID, p content.Principal, id uuid.UUID, status, resolution string) (content.CommentReport, error) {
	a, err := s.authorizeContent(ctx, token, appID, "content.article.update", "update")
	if err != nil {
		return content.CommentReport{}, err
	}
	if id == uuid.Nil || !oneOf(status, "resolved", "dismissed") || len([]rune(strings.TrimSpace(resolution))) < 1 || len([]rune(resolution)) > 500 {
		return content.CommentReport{}, content.ErrInvalid
	}
	return s.repo.ResolveCommentReport(ctx, principal(a, appID, p), id, status, strings.TrimSpace(resolution))
}
func (s *Service) ReportComment(ctx context.Context, token string, p content.Principal, id uuid.UUID, reason, details string) (content.CommentReport, error) {
	a, err := s.mobile(ctx, token)
	if err != nil {
		return content.CommentReport{}, err
	}
	if a.AppID == nil || id == uuid.Nil || !oneOf(reason, "spam", "abuse", "illegal", "privacy", "other") || len([]rune(details)) > 500 {
		return content.CommentReport{}, content.ErrInvalid
	}
	return s.repo.ReportComment(ctx, principal(a, *a.AppID, p), id, reason, strings.TrimSpace(details))
}
func (s *Service) BlockUser(ctx context.Context, token string, p content.Principal, id uuid.UUID, blocked bool) error {
	a, err := s.mobile(ctx, token)
	if err != nil {
		return err
	}
	if a.AppID == nil || id == uuid.Nil || id == a.User.ID {
		return content.ErrInvalid
	}
	scope := principal(a, *a.AppID, p)
	if blocked {
		return s.repo.BlockUser(ctx, scope, id)
	}
	return s.repo.UnblockUser(ctx, scope, id)
}
