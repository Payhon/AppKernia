package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotificationNotFound = errors.New("notification recipient not found")

type Notification struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	BodyFormat  string     `json:"body_format"`
	MessageType string     `json:"message_type"`
	CreatedAt   time.Time  `json:"created_at"`
	ReadAt      *time.Time `json:"read_at"`
}
type NotificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

type Preferences struct {
	Locale                  string          `json:"locale"`
	Appearance              string          `json:"appearance"`
	NotificationPreferences map[string]bool `json:"notification_preferences"`
}

type LoginEvent struct {
	ID         uuid.UUID `json:"id"`
	AuthMethod string    `json:"auth_method"`
	Result     string    `json:"result"`
	OccurredAt time.Time `json:"occurred_at"`
	IPAddress  *string   `json:"ip_address"`
}
type SecurityEvent struct {
	ID         uuid.UUID `json:"id"`
	EventType  string    `json:"event_type"`
	Severity   string    `json:"severity"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PreferenceUpdate struct {
	UserID                  uuid.UUID
	AppID                   uuid.UUID
	TenantID                uuid.UUID
	SessionID               uuid.UUID
	RequestID               string
	Locale                  *string
	Appearance              *string
	NotificationPreferences map[string]bool
}

type Repository interface {
	GetPreferences(context.Context, uuid.UUID, uuid.UUID) (Preferences, error)
	UpdatePreferences(context.Context, PreferenceUpdate) (Preferences, error)
	UnreadCount(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (int64, error)
	LoginEvents(context.Context, uuid.UUID, uuid.UUID) ([]LoginEvent, error)
	SecurityEvents(context.Context, uuid.UUID, uuid.UUID) ([]SecurityEvent, error)
	Notifications(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, int) (NotificationPage, error)
	Notification(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (Notification, error)
	MarkNotificationRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string) error
}
