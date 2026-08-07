// Package domain contains editorial content state and persistence ports. It
// deliberately has no HTTP, pgx, or sqlc dependency.
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("content permission denied")
	ErrInvalid   = errors.New("invalid content input")
	ErrNotFound  = errors.New("content resource not found")
	ErrConflict  = errors.New("content optimistic lock conflict")
)

type Principal struct {
	TenantID, AppID, UserID, SessionID uuid.UUID
	RequestID, UserAgent               string
}

type Translation struct {
	Title      string          `json:"title"`
	Summary    string          `json:"summary"`
	BodyFormat string          `json:"body_format"`
	Body       json.RawMessage `json:"body"`
}
type CategoryTranslation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Category struct {
	ID           uuid.UUID                      `json:"id"`
	Slug         string                         `json:"slug"`
	Status       string                         `json:"status"`
	SortOrder    int32                          `json:"sort_order"`
	LockVersion  int32                          `json:"lock_version"`
	Translations map[string]CategoryTranslation `json:"translations"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
}
type Article struct {
	ID             uuid.UUID              `json:"id"`
	CategoryID     *uuid.UUID             `json:"category_id"`
	Slug           string                 `json:"slug"`
	Status         string                 `json:"status"`
	Featured       bool                   `json:"featured"`
	SortOrder      int32                  `json:"sort_order"`
	CoverFileID    *uuid.UUID             `json:"cover_file_id"`
	CoverURL       *string                `json:"cover_url"`
	ReadingMinutes int16                  `json:"reading_minutes"`
	LockVersion    int32                  `json:"lock_version"`
	PublishedAt    *time.Time             `json:"published_at"`
	Translations   map[string]Translation `json:"translations"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
type PublicCategory struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	SortOrder   int32     `json:"sort_order"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}
type PublicArticle struct {
	ID             uuid.UUID       `json:"id"`
	CategoryID     *uuid.UUID      `json:"-"`
	Slug           string          `json:"slug"`
	Featured       bool            `json:"featured"`
	SortOrder      int32           `json:"-"`
	CoverURL       *string         `json:"cover_url"`
	ReadingMinutes int16           `json:"reading_minutes"`
	PublishedAt    *time.Time      `json:"published_at"`
	Title          string          `json:"title"`
	Summary        string          `json:"summary"`
	BodyFormat     string          `json:"body_format"`
	Body           json.RawMessage `json:"body"`
	Category       *PublicCategory `json:"category"`
	Bookmarked     bool            `json:"bookmarked"`
}
type ArticleAsset struct {
	FileID    uuid.UUID
	Provider  string
	Bucket    string
	ObjectKey string
	MediaType string
	SizeBytes int64
	SHA256    []byte
}

type PageFilter struct {
	Query, Status, Sort string
	CategoryID          *uuid.UUID
	Featured            *bool
	Page, PageSize      int32
}
type PublicFilter struct {
	Query, CategorySlug, Cursor string
	Featured                    *bool
	Limit                       int32
}
type CategoryPage struct {
	Items    []Category `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}
type ArticlePage struct {
	Items    []Article `json:"items"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
	Total    int64     `json:"total"`
}
type PublicArticlePage struct {
	Items      []PublicArticle `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}
type PublicCategoryPage struct {
	Items []PublicCategory `json:"items"`
}

type Repository interface {
	ListCategories(context.Context, uuid.UUID, uuid.UUID, PageFilter) (CategoryPage, error)
	GetCategory(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Category, error)
	CreateCategory(context.Context, Principal, Category) (Category, error)
	UpdateCategory(context.Context, Principal, Category) (Category, error)
	DeleteCategory(context.Context, Principal, uuid.UUID, int32) error
	ListArticles(context.Context, uuid.UUID, uuid.UUID, PageFilter) (ArticlePage, error)
	GetArticle(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Article, error)
	CreateArticle(context.Context, Principal, Article) (Article, error)
	UpdateArticle(context.Context, Principal, Article) (Article, error)
	DeleteArticle(context.Context, Principal, uuid.UUID, int32) error
	TransitionArticle(context.Context, Principal, uuid.UUID, int32, string) (Article, error)
	ListPublished(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, PublicFilter) (PublicArticlePage, error)
	ListPublishedCategories(context.Context, uuid.UUID, uuid.UUID, string) (PublicCategoryPage, error)
	GetPublished(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (PublicArticle, error)
	OpenArticleAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ArticleAsset, io.ReadCloser, error)
	Bookmark(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	RemoveBookmark(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
}
