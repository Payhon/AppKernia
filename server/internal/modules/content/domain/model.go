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
	ParentID     *uuid.UUID                     `json:"parent_id"`
	ImageFileID  *uuid.UUID                     `json:"image_file_id"`
	ImageURL     *string                        `json:"image_url"`
	Slug         string                         `json:"slug"`
	Status       string                         `json:"status"`
	SortOrder    int32                          `json:"sort_order"`
	LockVersion  int32                          `json:"lock_version"`
	Translations map[string]CategoryTranslation `json:"translations"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
}
type Article struct {
	ShareURL             string                 `json:"share_url,omitempty"`
	ID                   uuid.UUID              `json:"id"`
	CategoryID           *uuid.UUID             `json:"category_id"`
	CategoryIDs          []uuid.UUID            `json:"category_ids"`
	TopicID              *uuid.UUID             `json:"topic_id"`
	TagIDs               []uuid.UUID            `json:"tag_ids"`
	Tags                 []Tag                  `json:"tags"`
	Media                []Media                `json:"media"`
	Slug                 string                 `json:"slug"`
	Status               string                 `json:"status"`
	ContentType          string                 `json:"content_type"`
	AllowComments        bool                   `json:"allow_comments"`
	Pinned               bool                   `json:"pinned"`
	Featured             bool                   `json:"featured"`
	Latest               bool                   `json:"latest"`
	SortOrder            int32                  `json:"sort_order"`
	CoverFileID          *uuid.UUID             `json:"cover_file_id"`
	CoverURL             *string                `json:"cover_url"`
	ReadingMinutes       int16                  `json:"reading_minutes"`
	VideoSourceType      *string                `json:"video_source_type"`
	VideoFileID          *uuid.UUID             `json:"video_file_id"`
	VideoExternalURL     *string                `json:"video_external_url"`
	VideoDurationSeconds *int32                 `json:"video_duration_seconds"`
	LockVersion          int32                  `json:"lock_version"`
	PublishedAt          *time.Time             `json:"published_at"`
	Translations         map[string]Translation `json:"translations"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}
type PublicCategory struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Slug        string     `json:"slug"`
	SortOrder   int32      `json:"sort_order"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ImageURL    *string    `json:"image_url"`
}
type PublicArticle struct {
	ShareURL             string           `json:"share_url,omitempty"`
	ID                   uuid.UUID        `json:"id"`
	CategoryID           *uuid.UUID       `json:"-"`
	Slug                 string           `json:"slug"`
	ContentType          string           `json:"content_type"`
	AllowComments        bool             `json:"allow_comments"`
	Pinned               bool             `json:"pinned"`
	Featured             bool             `json:"featured"`
	Latest               bool             `json:"latest"`
	SortOrder            int32            `json:"-"`
	CoverURL             *string          `json:"cover_url"`
	ReadingMinutes       int16            `json:"reading_minutes"`
	PublishedAt          *time.Time       `json:"published_at"`
	Title                string           `json:"title"`
	Summary              string           `json:"summary"`
	BodyFormat           string           `json:"body_format"`
	Body                 json.RawMessage  `json:"body"`
	Category             *PublicCategory  `json:"category"`
	Categories           []PublicCategory `json:"categories"`
	Topic                *PublicTopic     `json:"topic"`
	Tags                 []Tag            `json:"tags"`
	Media                []PublicMedia    `json:"media"`
	VideoSourceType      *string          `json:"video_source_type"`
	VideoURL             *string          `json:"video_url"`
	VideoDurationSeconds *int32           `json:"video_duration_seconds"`
	Bookmarked           *bool            `json:"bookmarked"`
}

type TopicTranslation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Topic struct {
	ID           uuid.UUID                   `json:"id"`
	Slug         string                      `json:"slug"`
	Status       string                      `json:"status"`
	SortOrder    int32                       `json:"sort_order"`
	CoverFileID  *uuid.UUID                  `json:"cover_file_id"`
	CoverURL     *string                     `json:"cover_url"`
	LockVersion  int32                       `json:"lock_version"`
	Translations map[string]TopicTranslation `json:"translations"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
}
type PublicTopic struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	SortOrder   int32     `json:"sort_order"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CoverURL    *string   `json:"cover_url"`
}
type Tag struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status,omitempty"`
	LockVersion int32     `json:"lock_version,omitempty"`
	UsageCount  int64     `json:"usage_count,omitempty"`
}
type MediaTranslation struct {
	AltText string `json:"alt_text"`
}
type Media struct {
	ID           uuid.UUID                   `json:"id"`
	FileID       uuid.UUID                   `json:"file_id"`
	Role         string                      `json:"role"`
	SortOrder    int16                       `json:"sort_order"`
	Translations map[string]MediaTranslation `json:"translations"`
}
type PublicMedia struct {
	ID        uuid.UUID `json:"id"`
	FileID    uuid.UUID `json:"file_id"`
	URL       string    `json:"url"`
	Role      string    `json:"role"`
	SortOrder int16     `json:"sort_order"`
	AltText   string    `json:"alt_text"`
}
type Comment struct {
	ID               uuid.UUID  `json:"id"`
	ArticleID        uuid.UUID  `json:"article_id"`
	AuthorID         uuid.UUID  `json:"author_id"`
	AuthorName       string     `json:"author_name"`
	AuthorAvatarURL  *string    `json:"author_avatar_url"`
	ParentID         *uuid.UUID `json:"parent_id"`
	RootID           *uuid.UUID `json:"root_id"`
	Status           string     `json:"status"`
	Body             string     `json:"body"`
	ModerationReason *string    `json:"moderation_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
type CommentReport struct {
	ID         uuid.UUID  `json:"id"`
	CommentID  uuid.UUID  `json:"comment_id"`
	ReporterID uuid.UUID  `json:"reporter_id"`
	Reason     string     `json:"reason"`
	Details    string     `json:"details"`
	Status     string     `json:"status"`
	Resolution string     `json:"resolution,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}
type CommentReportPage struct {
	Items    []CommentReport `json:"items"`
	Page     int32           `json:"page"`
	PageSize int32           `json:"page_size"`
	Total    int64           `json:"total"`
}
type CommentPage struct {
	Items      []Comment `json:"items"`
	Page       int32     `json:"page"`
	PageSize   int32     `json:"page_size"`
	Total      int64     `json:"total"`
	NextCursor *string   `json:"next_cursor,omitempty"`
}
type TopicPage struct {
	Items    []Topic `json:"items"`
	Page     int32   `json:"page"`
	PageSize int32   `json:"page_size"`
	Total    int64   `json:"total"`
}
type TagPage struct {
	Items    []Tag `json:"items"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
	Total    int64 `json:"total"`
}
type PublicTopicPage struct {
	Items []PublicTopic `json:"items"`
}
type HomeFeed struct {
	Pinned   []PublicArticle `json:"pinned"`
	Featured []PublicArticle `json:"featured"`
	Latest   []PublicArticle `json:"latest"`
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
	TopicID             *uuid.UUID
	ContentType         string
	Featured            *bool
	Page, PageSize      int32
}
type PublicFilter struct {
	Query, CategorySlug, TopicSlug, Tag, ContentType, Cursor string
	Featured                                                 *bool
	Limit                                                    int32
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
	ResolvePublicApp(context.Context, uuid.UUID) (uuid.UUID, error)
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
	ListPublished(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, string, PublicFilter) (PublicArticlePage, error)
	ListPublishedCategories(context.Context, uuid.UUID, uuid.UUID, string) (PublicCategoryPage, error)
	GetPublished(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, string, string) (PublicArticle, error)
	OpenArticleAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ArticleAsset, io.ReadCloser, error)
	Bookmark(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	RemoveBookmark(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	ListBookmarks(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, PublicFilter) (PublicArticlePage, error)
	ListTopics(context.Context, uuid.UUID, uuid.UUID, PageFilter) (TopicPage, error)
	GetTopic(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Topic, error)
	CreateTopic(context.Context, Principal, Topic) (Topic, error)
	UpdateTopic(context.Context, Principal, Topic) (Topic, error)
	DeleteTopic(context.Context, Principal, uuid.UUID, int32) error
	ListTags(context.Context, uuid.UUID, uuid.UUID, PageFilter) (TagPage, error)
	UpsertTag(context.Context, Principal, string) (Tag, error)
	UpdateTag(context.Context, Principal, Tag) (Tag, error)
	MergeTag(context.Context, Principal, uuid.UUID, uuid.UUID, int32) error
	DeleteTag(context.Context, Principal, uuid.UUID, int32) error
	ListPublishedTopics(context.Context, uuid.UUID, uuid.UUID, string) (PublicTopicPage, error)
	GetPublishedTopic(context.Context, uuid.UUID, uuid.UUID, string, string) (PublicTopic, error)
	Home(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, string, int32) (HomeFeed, error)
	ListComments(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, uuid.UUID, string, PageFilter) (CommentPage, error)
	CreateComment(context.Context, Principal, uuid.UUID, *uuid.UUID, string) (Comment, error)
	DeleteOwnComment(context.Context, Principal, uuid.UUID) error
	ModerateComment(context.Context, Principal, uuid.UUID, string, string) (Comment, error)
	ListCommentReports(context.Context, uuid.UUID, uuid.UUID, string, PageFilter) (CommentReportPage, error)
	ResolveCommentReport(context.Context, Principal, uuid.UUID, string, string) (CommentReport, error)
	ReportComment(context.Context, Principal, uuid.UUID, string, string) (CommentReport, error)
	BlockUser(context.Context, Principal, uuid.UUID) error
	UnblockUser(context.Context, Principal, uuid.UUID) error
}
