// Package application owns the App boundary that sits between a tenant and a
// mobile build. App IDs are public routing identifiers, never authorization
// credentials: every authenticated operation also verifies app membership.
package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/appkernia/appkernia/server/internal/shared/publicurl"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAppNotFound       = errors.New("app not found")
	ErrAppDisabled       = errors.New("app disabled")
	ErrMembershipMissing = errors.New("app membership missing")
	ErrInvalidInput      = errors.New("invalid app input")
	ErrConflict          = errors.New("app resource conflict")
	ErrOTPUnavailable    = errors.New("app otp notification unavailable")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var appCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)
var manifestAppIDPattern = regexp.MustCompile(`^__UNI__[A-Za-z0-9_]{2,120}$`)

type Application struct {
	ID                       uuid.UUID                 `json:"id"`
	TenantID                 uuid.UUID                 `json:"tenant_id"`
	AppID                    string                    `json:"appid"`
	AppIDPending             bool                      `json:"appid_pending"`
	AppType                  string                    `json:"app_type"`
	Code                     string                    `json:"code"`
	Name                     string                    `json:"name"`
	Description              string                    `json:"description"`
	Introduction             string                    `json:"introduction"`
	Remark                   string                    `json:"remark"`
	Status                   string                    `json:"status"`
	DefaultLocale            string                    `json:"default_locale"`
	RegistrationEnabled      bool                      `json:"registration_enabled"`
	RegistrationVerification string                    `json:"registration_verification_mode"`
	CreatorUserID            *uuid.UUID                `json:"creator_user_id,omitempty"`
	OwnerType                string                    `json:"owner_type"`
	OwnerID                  *uuid.UUID                `json:"owner_id,omitempty"`
	IconFileID               *uuid.UUID                `json:"icon_file_id,omitempty"`
	Managers                 []uuid.UUID               `json:"managers"`
	Members                  []uuid.UUID               `json:"members"`
	Screenshots              []ApplicationAsset        `json:"screenshots"`
	Channels                 []ApplicationChannel      `json:"channels"`
	StoreListings            []ApplicationStoreListing `json:"store_listings"`
	IsDefault                bool                      `json:"is_default"`
	LockVersion              int32                     `json:"lock_version"`
	CreatedAt                time.Time                 `json:"created_at"`
	UpdatedAt                time.Time                 `json:"updated_at"`
	Startup                  StartupConfiguration      `json:"startup"`
}

type ApplicationAsset struct {
	ID       uuid.UUID `json:"id"`
	FileID   uuid.UUID `json:"file_id"`
	Position int32     `json:"position"`
}

type ApplicationChannel struct {
	ID           uuid.UUID  `json:"id"`
	ChannelCode  string     `json:"channel_code"`
	Name         string     `json:"name"`
	URL          *string    `json:"url,omitempty"`
	ABMURL       *string    `json:"abm_url,omitempty"`
	QRCodeFileID *uuid.UUID `json:"qrcode_file_id,omitempty"`
	Enabled      bool       `json:"enabled"`
}

type ApplicationStoreListing struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Scheme   string    `json:"scheme"`
	Enabled  bool      `json:"enabled"`
	Priority int32     `json:"priority"`
}

type PublicPage struct {
	PublicURL    string          `json:"public_url,omitempty"`
	Slug         string          `json:"slug"`
	DocumentType string          `json:"document_type"`
	Title        string          `json:"title"`
	Body         json.RawMessage `json:"body"`
	BodyFormat   string          `json:"body_format"`
	Version      int32           `json:"version"`
	ContentHash  string          `json:"content_hash"`
	Locale       string          `json:"locale"`
	RevisionID   uuid.UUID       `json:"revision_id"`
}

type Service struct {
	publicWebBaseURL       string
	pool                   *pgxpool.Pool
	auth                   Authenticator
	otp                    OTPNotifier
	erasureQueue           ObjectErasureQueue
	objects                storagedomain.ObjectStore
	accountDeletionEnabled bool
	shareRuntime           ShareRuntimeLoader
	pushRuntime            PushRuntimeLoader
	accountDeletionStepUp  AccountDeletionStepUp
}

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}

type OTPNotification struct {
	AppID, TenantID              uuid.UUID
	Email, Code, Purpose, Locale string
	ExpiresMinutes               int
}
type OTPNotifier interface {
	QueueOTP(context.Context, pgx.Tx, OTPNotification) error
}
type ObjectErasureQueue interface {
	Enqueue(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
}
type ShareRuntimeProvider struct {
	ProviderCode string   `json:"provider_code"`
	Enabled      bool     `json:"enabled"`
	Scenes       []string `json:"scenes"`
	FallbackMode string   `json:"fallback_mode"`
}
type ShareRuntimeLoader func(context.Context, uuid.UUID) ([]ShareRuntimeProvider, error)
type PushRuntime struct {
	Enabled       bool     `json:"enabled"`
	Environment   string   `json:"environment"`
	Providers     []string `json:"providers"`
	BuildVariants []string `json:"build_variants"`
}
type PushRuntimeLoader func(context.Context, uuid.UUID) (PushRuntime, error)

// AccountDeletionStepUp returns the Apple OAuth account ID whose fresh
// provider reauth was verified and revoked, or uuid.Nil when no Apple account
// is currently bound.
type AccountDeletionStepUp func(context.Context, string, uuid.UUID, string) (uuid.UUID, error)
type Option func(*Service)

func WithOTPNotifier(notifier OTPNotifier) Option {
	return func(service *Service) { service.otp = notifier }
}

func WithObjectErasureQueue(queue ObjectErasureQueue) Option {
	return func(service *Service) { service.erasureQueue = queue }
}

func WithObjectStore(objects storagedomain.ObjectStore) Option {
	return func(service *Service) { service.objects = objects }
}

func WithAccountDeletionEnabled(enabled bool) Option {
	return func(service *Service) { service.accountDeletionEnabled = enabled }
}

func WithShareRuntime(loader ShareRuntimeLoader) Option {
	return func(service *Service) { service.shareRuntime = loader }
}

func WithPushRuntime(loader PushRuntimeLoader) Option {
	return func(service *Service) { service.pushRuntime = loader }
}

func WithAccountDeletionStepUp(consumer AccountDeletionStepUp) Option {
	return func(service *Service) { service.accountDeletionStepUp = consumer }
}

const (
	emailOTPLifetime = 10 * time.Minute
	emailOTPCooldown = 60 * time.Second
	emailOTPAttempts = 5
)

func NewService(pool *pgxpool.Pool, auth Authenticator, options ...Option) *Service {
	service := &Service{pool: pool, auth: auth}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) PublicShareRuntime(ctx context.Context, appID uuid.UUID) ([]ShareRuntimeProvider, error) {
	if s.shareRuntime == nil {
		return []ShareRuntimeProvider{}, nil
	}
	return s.shareRuntime(ctx, appID)
}

func (s *Service) PublicPushRuntime(ctx context.Context, appID uuid.UUID) (PushRuntime, error) {
	if s.pushRuntime == nil {
		return PushRuntime{Providers: []string{}, BuildVariants: []string{}}, nil
	}
	return s.pushRuntime(ctx, appID)
}

func (s *Service) Resolve(ctx context.Context, id uuid.UUID) (Application, error) {
	a, err := scanAdminApplication(s.pool.QueryRow(ctx, `SELECT `+adminApplicationColumns+` FROM app.applications WHERE id=$1 AND deleted_at IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrAppNotFound
	}
	if err != nil {
		return Application{}, fmt.Errorf("resolve app: %w", err)
	}
	if a.Status != "active" {
		return Application{}, ErrAppDisabled
	}
	return a, nil
}

func (s *Service) PublicPage(ctx context.Context, appID uuid.UUID, slug, locale string) (PublicPage, error) {
	if locale != "en-US" {
		locale = "zh-CN"
	}
	var out PublicPage
	var hash []byte
	err := s.pool.QueryRow(ctx, `
SELECT p.slug, p.page_type, t.title, t.body, t.body_format, r.revision_number,
       r.content_hash, t.locale, r.id
FROM content.pages p
JOIN content.page_revisions r ON r.id = p.current_revision_id AND r.app_id = p.app_id AND r.status = 'published'
JOIN content.page_revision_translations t ON t.revision_id = r.id
WHERE p.app_id = $1 AND p.slug = $2 AND p.status = 'published'
  AND t.locale = COALESCE((
       SELECT t2.locale FROM content.page_revision_translations t2
       WHERE t2.revision_id = r.id AND t2.locale = $3
  ), 'zh-CN')`, appID, strings.TrimSpace(slug), locale).Scan(
		&out.Slug, &out.DocumentType, &out.Title, &out.Body, &out.BodyFormat,
		&out.Version, &hash, &out.Locale, &out.RevisionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicPage{}, ErrAppNotFound
	}
	if err != nil {
		return PublicPage{}, fmt.Errorf("get app page: %w", err)
	}
	out.PublicURL = publicurl.Link(s.publicWebBaseURL, appID, "/pages/"+url.PathEscape(out.Slug), out.Locale)
	out.ContentHash = hex.EncodeToString(hash)
	return out, nil
}

func (s *Service) RecordLegalConsent(ctx context.Context, token string, app Application, documentType string, revisionID uuid.UUID, contentHash, locale string, ip, userAgent string) error {
	if documentType != "privacy-policy" && documentType != "terms-of-service" || revisionID == uuid.Nil || len(contentHash) != 64 || (locale != "zh-CN" && locale != "en-US") {
		return ErrInvalidInput
	}
	principal, err := s.auth.Authenticate(ctx, token, "ak-mobile")
	if err != nil || principal.Tenant.ID != app.TenantID {
		return ErrMembershipMissing
	}
	if err = s.requireMembership(ctx, app.ID, principal.User.ID); err != nil {
		return err
	}
	hash, err := hex.DecodeString(contentHash)
	if err != nil {
		return ErrInvalidInput
	}
	var authoritative []byte
	err = s.pool.QueryRow(ctx, `
SELECT r.content_hash
FROM content.page_revisions r
JOIN content.pages p ON p.current_revision_id = r.id
WHERE r.id = $1 AND r.app_id = $2 AND p.app_id = $2
  AND p.page_type = $3 AND p.status = 'published'`, revisionID, app.ID, documentType).Scan(&authoritative)
	if errors.Is(err, pgx.ErrNoRows) || len(authoritative) != len(hash) || string(authoritative) != string(hash) {
		return ErrInvalidInput
	}
	if err != nil {
		return fmt.Errorf("verify legal revision: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO app.legal_consents (app_id, tenant_id, user_id, page_type, revision_id, content_hash, locale, ip_address, user_agent)
VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::inet,NULLIF($9,''))
ON CONFLICT (app_id,user_id,page_type,revision_id) DO NOTHING`,
		app.ID, app.TenantID, principal.User.ID, documentType, revisionID, hash, locale, ip, userAgent)
	if err != nil {
		return fmt.Errorf("record legal consent: %w", err)
	}
	return nil
}

func (s *Service) requireMembership(ctx context.Context, appID, userID uuid.UUID) error {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM app.user_memberships WHERE app_id=$1 AND user_id=$2 AND status='active')`, appID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verify app membership: %w", err)
	}
	if !exists {
		return ErrMembershipMissing
	}
	return nil
}

// HashDocument is shared by the Admin writer and public reader contract: the
// canonical JSON body and locale strings are hashed server-side, never trusted
// from a client-provided content hash.
func HashDocument(locale, title, bodyFormat string, body json.RawMessage) []byte {
	canonical, _ := json.Marshal(struct {
		Locale string          `json:"locale"`
		Title  string          `json:"title"`
		Format string          `json:"body_format"`
		Body   json.RawMessage `json:"body"`
	}{locale, title, bodyFormat, body})
	digest := sha256.Sum256(canonical)
	return digest[:]
}

func (s *Service) AuthenticateMobileMembership(ctx context.Context, token string, app Application) (iam.AuthenticatedContext, error) {
	principal, err := s.auth.Authenticate(ctx, token, "ak-mobile")
	if err != nil || principal.Tenant.ID != app.TenantID {
		return iam.AuthenticatedContext{}, ErrMembershipMissing
	}
	if err = s.requireMembership(ctx, app.ID, principal.User.ID); err != nil {
		return iam.AuthenticatedContext{}, err
	}
	return principal, nil
}

// RegisterMobile creates only an App membership and never chooses a tenant
// from request data. Delivery is intentionally decoupled from this transaction:
// deployments inject their mail worker/adapter; the response is generic to
// prevent account enumeration.
func (s *Service) RegisterMobile(ctx context.Context, app Application, email, displayName, password, locale string) error {
	if !app.RegistrationEnabled {
		return ErrAppDisabled
	}
	email, ok := normalizedEmail(email)
	if !ok || len(strings.TrimSpace(displayName)) < 2 || len([]rune(strings.TrimSpace(displayName))) > 120 || (locale != "zh-CN" && locale != "en-US") {
		return ErrInvalidInput
	}
	hash, err := iamapp.HashPassword(password)
	if err != nil {
		return ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin app registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID uuid.UUID
	// Mobile identifiers are App-scoped facts. Do not reuse a global IAM user
	// solely because the same email exists in another App.
	err = tx.QueryRow(ctx, `INSERT INTO iam.users (display_name,locale,status) VALUES ($1,$2,'active') RETURNING id`, strings.TrimSpace(displayName), locale).Scan(&userID)
	if err != nil {
		return fmt.Errorf("create app user: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.user_credentials (user_id,password_hash) VALUES ($1,$2)`, userID, hash); err != nil {
		return fmt.Errorf("create app credentials: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.tenant_members (tenant_id,user_id,status) VALUES ($1,$2,'active')`, app.TenantID, userID); err != nil {
		return fmt.Errorf("create app tenant member: %w", err)
	}
	status := "active"
	if app.RegistrationVerification == "email_otp" {
		status = "pending_verification"
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.user_memberships (app_id,tenant_id,user_id,source,status) VALUES ($1,$2,$3,'self_registration',$4) ON CONFLICT (app_id,user_id) DO NOTHING`, app.ID, app.TenantID, userID, status); err != nil {
		return fmt.Errorf("create app membership: %w", err)
	}
	var verifiedAt *time.Time
	if app.RegistrationVerification != "email_otp" {
		now := time.Now().UTC()
		verifiedAt = &now
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,'email',$4,$5,$6,'active')`, app.TenantID, app.ID, userID, email, maskedEmail(email), verifiedAt); err != nil {
		return fmt.Errorf("create app login identifier: %w", err)
	}
	if app.RegistrationVerification == "email_otp" {
		if err = s.createOTP(ctx, tx, app, userID, email, "email_otp"); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit app registration: %w", err)
	}
	return nil
}

func (s *Service) VerifyRegistrationEmail(ctx context.Context, app Application, email, code string) error {
	email, ok := normalizedEmail(email)
	if !ok || !validOTP(code) {
		return ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin email verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	userID, err := s.consumeOTP(ctx, tx, app, email, code, "email_otp")
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE app.user_memberships SET status='active',verified_at=now(),disabled_at=NULL WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3 AND status='pending_verification'`, app.ID, app.TenantID, userID); err != nil {
		return fmt.Errorf("activate app membership: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE app.user_login_identifiers SET verified_at=COALESCE(verified_at,now()),status='active'
WHERE app_id=$1 AND user_id=$2 AND identifier_type='email' AND normalized_value=$3`, app.ID, userID, email); err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) ResendRegistrationEmail(ctx context.Context, app Application, email string) (int, error) {
	email, ok := normalizedEmail(email)
	if !ok {
		return 0, ErrInvalidInput
	}
	var userID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT m.user_id FROM app.user_memberships m
JOIN app.user_login_identifiers i ON i.app_id=m.app_id AND i.user_id=m.user_id AND i.tenant_id=m.tenant_id
WHERE m.app_id=$1 AND m.tenant_id=$2 AND i.identifier_type='email' AND i.normalized_value=$3
  AND i.status='active' AND m.status='pending_verification'`, app.ID, app.TenantID, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return int(emailOTPCooldown.Seconds()), nil
	}
	if err != nil {
		return 0, fmt.Errorf("find pending app membership: %w", err)
	}
	var latest time.Time
	_ = s.pool.QueryRow(ctx, `SELECT created_at FROM iam.verification_challenges WHERE user_id=$1 AND challenge_type='email_otp' AND metadata->>'app_id'=$2 ORDER BY created_at DESC LIMIT 1`, userID, app.ID.String()).Scan(&latest)
	if !latest.IsZero() && time.Since(latest) < emailOTPCooldown {
		return int((emailOTPCooldown - time.Since(latest)).Seconds()) + 1, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = s.createOTP(ctx, tx, app, userID, email, "email_otp"); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(emailOTPCooldown.Seconds()), nil
}

func (s *Service) ForgotMobilePassword(ctx context.Context, app Application, email string) (int, error) {
	email, ok := normalizedEmail(email)
	if !ok {
		return 0, ErrInvalidInput
	}
	var userID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT m.user_id FROM app.user_memberships m
JOIN app.user_login_identifiers i ON i.app_id=m.app_id AND i.user_id=m.user_id AND i.tenant_id=m.tenant_id
WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.status='active' AND i.identifier_type='email'
  AND i.normalized_value=$3 AND i.status='active' AND i.verified_at IS NOT NULL`, app.ID, app.TenantID, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return int(emailOTPCooldown.Seconds()), nil
	}
	if err != nil {
		return 0, fmt.Errorf("find app account: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = s.createOTP(ctx, tx, app, userID, email, "password_reset"); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(emailOTPCooldown.Seconds()), nil
}

func (s *Service) ResetMobilePassword(ctx context.Context, app Application, email, code, password string) error {
	email, ok := normalizedEmail(email)
	if !ok || !validOTP(code) {
		return ErrInvalidInput
	}
	newHash, err := iamapp.HashPassword(password)
	if err != nil {
		return ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	userID, err := s.consumeOTP(ctx, tx, app, email, code, "password_reset")
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.user_credentials SET password_hash=$2,password_version=password_version+1,password_changed_at=now(),force_password_change=false,failed_attempts=0,locked_until=NULL,updated_at=now() WHERE user_id=$1`, userID, newHash); err != nil {
		return fmt.Errorf("reset app password: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=now(),revoke_reason='app_password_reset',updated_at=now() WHERE user_id=$1 AND tenant_id=$2 AND audience='ak-mobile' AND (app_id=$3 OR app_id IS NULL) AND revoked_at IS NULL`, userID, app.TenantID, app.ID); err != nil {
		return fmt.Errorf("revoke app sessions: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) createOTP(ctx context.Context, tx pgx.Tx, app Application, userID uuid.UUID, email, kind string) error {
	code, err := newOTP()
	if err != nil {
		return err
	}
	metadata, _ := json.Marshal(map[string]string{"app_id": app.ID.String(), "purpose": kind})
	_, err = tx.Exec(ctx, `INSERT INTO iam.verification_challenges (user_id,challenge_type,target_hash,target_hint,secret_hash,max_attempts,expires_at,metadata)
VALUES ($1,$2,$3,$4,$5,$6,now()+interval '10 minutes',$7)`, userID, kind, sha256Bytes(email), maskedEmail(email), sha256Bytes(code), emailOTPAttempts, metadata)
	if err != nil {
		return fmt.Errorf("create app otp: %w", err)
	}
	if s.otp == nil {
		return ErrOTPUnavailable
	}
	if err = s.otp.QueueOTP(ctx, tx, OTPNotification{AppID: app.ID, TenantID: app.TenantID, Email: email, Code: code, Purpose: kind, Locale: app.DefaultLocale, ExpiresMinutes: int(emailOTPLifetime.Minutes())}); err != nil {
		return fmt.Errorf("queue app otp: %w", err)
	}
	return nil
}
func (s *Service) consumeOTP(ctx context.Context, tx pgx.Tx, app Application, email, code, kind string) (uuid.UUID, error) {
	var id, userID uuid.UUID
	var secret []byte
	var attempts, max int
	var expires time.Time
	err := tx.QueryRow(ctx, `SELECT id,user_id,secret_hash,attempts,max_attempts,expires_at FROM iam.verification_challenges WHERE challenge_type=$1 AND target_hash=$2 AND metadata->>'app_id'=$3 AND consumed_at IS NULL ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, kind, sha256Bytes(email), app.ID.String()).Scan(&id, &userID, &secret, &attempts, &max, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvalidInput
	}
	if err != nil {
		return uuid.Nil, err
	}
	valid := time.Now().UTC().Before(expires) && attempts < max && string(secret) == string(sha256Bytes(code))
	if !valid {
		_, _ = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=CASE WHEN attempts+1>=max THEN now() ELSE consumed_at END WHERE id=$1`, id)
		return uuid.Nil, ErrInvalidInput
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, id); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}
func normalizedEmail(raw string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(raw))
	parsed, err := mail.ParseAddress(email)
	return email, err == nil && parsed.Address == email && len(email) <= 254
}
func validOTP(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func newOTP() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	n := uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
	return fmt.Sprintf("%06d", n%1000000), nil
}
func sha256Bytes(value string) []byte { digest := sha256.Sum256([]byte(value)); return digest[:] }
func maskedEmail(value string) string {
	parts := strings.Split(value, "@")
	if len(parts) != 2 || len(parts[0]) < 2 {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}

type AdminAppInput struct {
	AppID                    string
	AppType                  string
	Code                     string
	Name                     string
	Description              string
	Introduction             string
	Remark                   string
	DefaultLocale            string
	RegistrationEnabled      bool
	RegistrationVerification string
	OwnerType                string
	OwnerID                  *uuid.UUID
	IconFileID               *uuid.UUID
	Managers                 []uuid.UUID
	Members                  []uuid.UUID
	ScreenshotFileIDs        []uuid.UUID
	Channels                 []ApplicationChannel
	StoreListings            []ApplicationStoreListing
	LockVersion              int32
	Startup                  *StartupInput
}

type PageTranslation struct {
	Title      string          `json:"title"`
	BodyFormat string          `json:"body_format"`
	Body       json.RawMessage `json:"body"`
}
type AdminPage struct {
	PublicURL         string                              `json:"public_url,omitempty"`
	ID                uuid.UUID                           `json:"id"`
	Slug              string                              `json:"slug"`
	PageType          string                              `json:"page_type"`
	Status            string                              `json:"status"`
	LockVersion       int32                               `json:"lock_version"`
	CurrentRevisionID *uuid.UUID                          `json:"current_revision_id"`
	UpdatedAt         time.Time                           `json:"updated_at"`
	Translations      map[string]AdminPageViewTranslation `json:"translations"`
	Revisions         []AdminPageRevision                 `json:"revisions"`
}
type AdminPageViewTranslation struct {
	Title      string          `json:"title"`
	BodyFormat string          `json:"body_format"`
	Body       json.RawMessage `json:"body"`
}
type AdminPageRevision struct {
	ID          uuid.UUID  `json:"id"`
	Version     int32      `json:"version"`
	Status      string     `json:"status"`
	ContentHash string     `json:"content_hash"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   *uuid.UUID `json:"created_by"`
}
type PageInput struct {
	Slug         string
	PageType     string
	LockVersion  int32
	Translations map[string]PageTranslation
}
type AdminAppUser struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	AvatarURL    *string    `json:"avatar_url"`
	Status       string     `json:"status"`
	Source       string     `json:"source"`
	VerifiedAt   *time.Time `json:"verified_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastSignInAt *time.Time `json:"last_sign_in_at"`
	LockVersion  int32      `json:"lock_version"`
}
type AdminAppUserInput struct {
	Email       string
	DisplayName string
	Locale      string
	Password    string
}
type AdminListFilter struct {
	Query    string
	Status   string
	AppType  string
	Page     int32
	PageSize int32
}
type AdminApplicationPage struct {
	Items []Application
	Total int
}
type AdminPagePage struct {
	Items []AdminPage
	Total int
}
type AdminAppUserPage struct {
	Items []AdminAppUser
	Total int
}

func (s *Service) authorizeAdmin(ctx context.Context, token, permission string) (iam.AuthenticatedContext, error) {
	principal, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil || !slices.Contains(principal.Permissions, permission) {
		return iam.AuthenticatedContext{}, ErrMembershipMissing
	}
	return principal, nil
}

func normalizedAdminListFilter(filter AdminListFilter, statuses ...string) (AdminListFilter, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if len([]rune(filter.Query)) > 160 || filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 ||
		(filter.Status != "" && !slices.Contains(statuses, filter.Status)) ||
		(filter.AppType != "" && filter.AppType != "uni_app" && filter.AppType != "uni_app_x") {
		return filter, ErrInvalidInput
	}
	return filter, nil
}

const adminApplicationColumns = `id, tenant_id, COALESCE(appid::text,''), appid IS NULL, app_type, code, name,
description, introduction, remark, status, default_locale, registration_enabled, registration_verification,
creator_user_id, owner_type, owner_user_id, owner_tenant_id, icon_file_id, is_default, lock_version, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanAdminApplication(row rowScanner) (Application, error) {
	var item Application
	var ownerUserID, ownerTenantID *uuid.UUID
	err := row.Scan(&item.ID, &item.TenantID, &item.AppID, &item.AppIDPending, &item.AppType, &item.Code, &item.Name,
		&item.Description, &item.Introduction, &item.Remark, &item.Status, &item.DefaultLocale,
		&item.RegistrationEnabled, &item.RegistrationVerification, &item.CreatorUserID, &item.OwnerType,
		&ownerUserID, &ownerTenantID, &item.IconFileID, &item.IsDefault, &item.LockVersion, &item.CreatedAt, &item.UpdatedAt)
	if item.OwnerType == "user" {
		item.OwnerID = ownerUserID
	} else {
		item.OwnerID = ownerTenantID
	}
	return item, err
}

func (s *Service) loadApplicationRelations(ctx context.Context, item *Application) error {
	item.Managers, item.Members = []uuid.UUID{}, []uuid.UUID{}
	item.Screenshots = []ApplicationAsset{}
	item.Channels = []ApplicationChannel{}
	item.StoreListings = []ApplicationStoreListing{}
	rows, err := s.pool.Query(ctx, `SELECT user_id,role FROM app.application_team_members WHERE app_id=$1 ORDER BY role,user_id`, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var userID uuid.UUID
		var role string
		if err = rows.Scan(&userID, &role); err != nil {
			rows.Close()
			return err
		}
		if role == "manager" {
			item.Managers = append(item.Managers, userID)
		} else {
			item.Members = append(item.Members, userID)
		}
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT id,file_id,position FROM app.application_assets WHERE app_id=$1 AND asset_type='screenshot' ORDER BY position,id`, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var asset ApplicationAsset
		if err = rows.Scan(&asset.ID, &asset.FileID, &asset.Position); err != nil {
			rows.Close()
			return err
		}
		item.Screenshots = append(item.Screenshots, asset)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT id,channel_code,name,url,abm_url,qrcode_file_id,enabled FROM app.application_channels WHERE app_id=$1 ORDER BY channel_code`, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var channel ApplicationChannel
		if err = rows.Scan(&channel.ID, &channel.ChannelCode, &channel.Name, &channel.URL, &channel.ABMURL, &channel.QRCodeFileID, &channel.Enabled); err != nil {
			rows.Close()
			return err
		}
		item.Channels = append(item.Channels, channel)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT id,name,scheme,enabled,priority FROM app.application_store_listings WHERE app_id=$1 ORDER BY priority DESC,id`, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var store ApplicationStoreListing
		if err = rows.Scan(&store.ID, &store.Name, &store.Scheme, &store.Enabled, &store.Priority); err != nil {
			rows.Close()
			return err
		}
		item.StoreListings = append(item.StoreListings, store)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	return s.loadStartupConfiguration(ctx, item)
}

func (s *Service) ListAdminApps(ctx context.Context, token string, filter AdminListFilter) (AdminApplicationPage, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.application.read")
	if err != nil {
		return AdminApplicationPage{}, err
	}
	filter, err = normalizedAdminListFilter(filter, "active", "disabled")
	if err != nil {
		return AdminApplicationPage{}, err
	}
	var total int
	where := `tenant_id=$1 AND deleted_at IS NULL AND ($2='' OR code ILIKE '%' || $2 || '%' OR COALESCE(appid::text,'') ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%') AND ($3='' OR status=$3) AND ($4='' OR app_type=$4)`
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM app.applications WHERE `+where, p.Tenant.ID, filter.Query, filter.Status, filter.AppType).Scan(&total); err != nil {
		return AdminApplicationPage{}, fmt.Errorf("count apps: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT `+adminApplicationColumns+` FROM app.applications WHERE `+where+` ORDER BY is_default DESC, created_at DESC, id LIMIT $5 OFFSET $6`, p.Tenant.ID, filter.Query, filter.Status, filter.AppType, filter.PageSize, (filter.Page-1)*filter.PageSize)
	if err != nil {
		return AdminApplicationPage{}, fmt.Errorf("list apps: %w", err)
	}
	defer rows.Close()
	items := make([]Application, 0)
	for rows.Next() {
		item, scanErr := scanAdminApplication(rows)
		if scanErr != nil {
			return AdminApplicationPage{}, fmt.Errorf("scan app: %w", scanErr)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return AdminApplicationPage{}, err
	}
	rows.Close()
	for index := range items {
		if err = s.loadApplicationRelations(ctx, &items[index]); err != nil {
			return AdminApplicationPage{}, fmt.Errorf("load app relations: %w", err)
		}
	}
	return AdminApplicationPage{Items: items, Total: total}, nil
}

func (s *Service) GetAdminApp(ctx context.Context, token string, appID uuid.UUID) (Application, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.application.read")
	if err != nil {
		return Application{}, err
	}
	if appID == uuid.Nil {
		return Application{}, ErrInvalidInput
	}
	var item Application
	item, err = scanAdminApplication(s.pool.QueryRow(ctx, `SELECT `+adminApplicationColumns+` FROM app.applications WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, appID, p.Tenant.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrAppNotFound
	}
	if err == nil {
		err = s.loadApplicationRelations(ctx, &item)
	}
	return item, err
}

func (s *Service) CreateAdminApp(ctx context.Context, token string, input AdminAppInput) (Application, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.application.create")
	if err != nil {
		return Application{}, err
	}
	if strings.TrimSpace(input.Code) == "" {
		input.Code = derivedAppCode(input.Name)
	}
	input = normalizeAdminAppInput(input)
	if input.OwnerType == "" {
		input.OwnerType, input.OwnerID = "tenant", &p.Tenant.ID
	}
	if err = validAppInput(input, false); err != nil {
		return Application{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = validateAppReferences(ctx, tx, p.Tenant.ID, input); err != nil {
		return Application{}, err
	}
	ownerUserID, ownerTenantID := ownerReferences(input.OwnerType, input.OwnerID)
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO app.applications (
tenant_id,appid,app_type,code,name,description,introduction,remark,default_locale,registration_enabled,
registration_verification,creator_user_id,owner_type,owner_user_id,owner_tenant_id,icon_file_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`,
		p.Tenant.ID, input.AppID, input.AppType, input.Code, input.Name, input.Description, input.Introduction, input.Remark,
		input.DefaultLocale, input.RegistrationEnabled, input.RegistrationVerification, p.User.ID, input.OwnerType,
		ownerUserID, ownerTenantID, input.IconFileID).Scan(&id)
	if err != nil {
		return Application{}, fmt.Errorf("create app: %w", err)
	}
	if !slices.Contains(input.Managers, p.User.ID) {
		input.Managers = append(input.Managers, p.User.ID)
	}
	if err = replaceApplicationRelations(ctx, tx, p.Tenant.ID, id, input); err != nil {
		return Application{}, err
	}
	if input.Startup != nil {
		if err = saveStartupDraft(ctx, tx, p.Tenant.ID, id, *input.Startup); err != nil {
			return Application{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return s.GetAdminApp(ctx, token, id)
}

func (s *Service) UpdateAdminApp(ctx context.Context, token string, id uuid.UUID, input AdminAppInput, status *string) (Application, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.application.update")
	if err != nil {
		return Application{}, err
	}
	if id == uuid.Nil || input.LockVersion < 1 || (status != nil && *status != "active" && *status != "disabled") {
		return Application{}, ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	existing, err := scanAdminApplication(tx.QueryRow(ctx, `SELECT `+adminApplicationColumns+` FROM app.applications WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL FOR UPDATE`, id, p.Tenant.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrAppNotFound
	}
	if err != nil {
		return Application{}, err
	}
	input = normalizeAdminAppInput(input)
	if input.AppID == "" {
		input.AppID = existing.AppID
	}
	if input.AppType == "" {
		input.AppType = existing.AppType
	}
	if input.Code == "" {
		input.Code = existing.Code
	}
	if input.OwnerType == "" {
		input.OwnerType, input.OwnerID = existing.OwnerType, existing.OwnerID
	}
	if validAppInput(input, true) != nil || input.AppType != existing.AppType || (existing.AppID != "" && !strings.EqualFold(input.AppID, existing.AppID)) {
		return Application{}, ErrInvalidInput
	}
	if err = validateAppReferences(ctx, tx, p.Tenant.ID, input); err != nil {
		return Application{}, err
	}
	ownerUserID, ownerTenantID := ownerReferences(input.OwnerType, input.OwnerID)
	command, err := tx.Exec(ctx, `UPDATE app.applications SET appid=$3,name=$4,description=$5,introduction=$6,remark=$7,
default_locale=$8,registration_enabled=$9,registration_verification=$10,owner_type=$11,owner_user_id=$12,
owner_tenant_id=$13,icon_file_id=$14,status=COALESCE($15,status),lock_version=lock_version+1
WHERE id=$1 AND tenant_id=$2 AND lock_version=$16 AND deleted_at IS NULL`, id, p.Tenant.ID, input.AppID, input.Name,
		input.Description, input.Introduction, input.Remark, input.DefaultLocale, input.RegistrationEnabled,
		input.RegistrationVerification, input.OwnerType, ownerUserID, ownerTenantID, input.IconFileID, status, input.LockVersion)
	if err != nil {
		return Application{}, fmt.Errorf("update app: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Application{}, ErrConflict
	}
	if err = replaceApplicationRelations(ctx, tx, p.Tenant.ID, id, input); err != nil {
		return Application{}, err
	}
	if input.Startup != nil {
		if err = saveStartupDraft(ctx, tx, p.Tenant.ID, id, *input.Startup); err != nil {
			return Application{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return s.GetAdminApp(ctx, token, id)
}

// SetAdminAppStatus is intentionally separate from UpdateAdminApp: lifecycle
// access is granted by app.application.disable, whereas editing an App record
// requires app.application.update.  The same lifecycle permission covers both
// disable and re-enable because the seed has no separate enable permission.
func (s *Service) SetAdminAppStatus(ctx context.Context, token string, id uuid.UUID, status string, lockVersion int32) (Application, error) {
	p, err := s.authorizeAdmin(ctx, token, appStatusPermission(status))
	if err != nil {
		return Application{}, err
	}
	if id == uuid.Nil || lockVersion < 1 || (status != "active" && status != "disabled") {
		return Application{}, ErrInvalidInput
	}
	item, err := scanAdminApplication(s.pool.QueryRow(ctx, `UPDATE app.applications SET status=$3,lock_version=lock_version+1
WHERE id=$1 AND tenant_id=$2 AND lock_version=$4 AND deleted_at IS NULL
RETURNING `+adminApplicationColumns, id, p.Tenant.ID, status, lockVersion))
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, id, p.Tenant.ID).Scan(&exists); checkErr != nil {
			return Application{}, checkErr
		}
		if exists {
			return Application{}, ErrConflict
		}
		return Application{}, ErrAppNotFound
	}
	if err != nil {
		return Application{}, fmt.Errorf("set app status: %w", err)
	}
	if err = s.loadApplicationRelations(ctx, &item); err != nil {
		return Application{}, err
	}
	return item, nil
}

func (s *Service) DeleteAdminApps(ctx context.Context, token string, ids []uuid.UUID, requestIDs ...string) error {
	p, err := s.authorizeAdmin(ctx, token, "app.application.delete")
	if err != nil {
		return err
	}
	requestedCount := len(ids)
	ids = uniqueUUIDs(ids)
	if len(ids) < 1 || len(ids) > 100 || len(ids) != requestedCount {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var eligible int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.applications
WHERE tenant_id=$1 AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND status='disabled' AND NOT is_default`, p.Tenant.ID, ids).Scan(&eligible); err != nil {
		return err
	}
	if eligible != len(ids) {
		return ErrConflict
	}
	command, err := tx.Exec(ctx, `UPDATE app.applications SET deleted_at=now(),lock_version=lock_version+1
WHERE tenant_id=$1 AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND status='disabled' AND NOT is_default`, p.Tenant.ID, ids)
	if err != nil {
		return err
	}
	if command.RowsAffected() != int64(len(ids)) {
		return ErrConflict
	}
	requestID := ""
	if len(requestIDs) > 0 {
		requestID = strings.TrimSpace(requestIDs[0])
	}
	for _, id := range ids {
		if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(
tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,
http_method,request_path,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'app','app.application.delete','app.application.delete','app.application',$4::text,
'DELETE','/admin-api/v1/apps/'||$4::text,$5,true)`, p.Tenant.ID, p.User.ID, requestID, id.String(), []byte(`{"deleted":true}`)); err != nil {
			return fmt.Errorf("audit app deletion: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func appStatusPermission(status string) string {
	// The permission seed intentionally defines a single lifecycle grant.  Do
	// not fall back to app.application.update for enable: that would make a UI
	// restriction bypassable by an update-only role.
	return "app.application.disable"
}

func validAppInput(input AdminAppInput, update bool) error {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	if (!update && !appCodePattern.MatchString(input.Code)) || !manifestAppIDPattern.MatchString(input.AppID) ||
		(input.AppType != "uni_app" && input.AppType != "uni_app_x") || len(input.Name) < 1 || len(input.Name) > 120 ||
		len([]rune(input.Description)) > 4000 || len([]rune(input.Introduction)) > 1000 || len([]rune(input.Remark)) > 4000 ||
		(input.DefaultLocale != "zh-CN" && input.DefaultLocale != "en-US") || (input.RegistrationVerification != "none" && input.RegistrationVerification != "email_otp") {
		return ErrInvalidInput
	}
	if input.OwnerID == nil || (input.OwnerType != "user" && input.OwnerType != "tenant") || len(input.ScreenshotFileIDs) > 20 || len(input.Channels) > 20 || len(input.StoreListings) > 100 || len(input.Managers) > 100 || len(input.Members) > 100 {
		return ErrInvalidInput
	}
	return nil
}

func normalizeAdminAppInput(input AdminAppInput) AdminAppInput {
	input.AppID = strings.TrimSpace(input.AppID)
	input.AppType = strings.TrimSpace(input.AppType)
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Introduction = strings.TrimSpace(input.Introduction)
	input.Remark = strings.TrimSpace(input.Remark)
	input.OwnerType = strings.TrimSpace(input.OwnerType)
	input.Managers = uniqueUUIDs(input.Managers)
	input.Members = uniqueUUIDs(input.Members)
	input.ScreenshotFileIDs = uniqueUUIDs(input.ScreenshotFileIDs)
	for index := range input.Channels {
		input.Channels[index].ChannelCode = strings.TrimSpace(input.Channels[index].ChannelCode)
		input.Channels[index].Name = strings.TrimSpace(input.Channels[index].Name)
		input.Channels[index].URL = trimStringPointer(input.Channels[index].URL)
		input.Channels[index].ABMURL = trimStringPointer(input.Channels[index].ABMURL)
	}
	for index := range input.StoreListings {
		input.StoreListings[index].Name = strings.TrimSpace(input.StoreListings[index].Name)
		input.StoreListings[index].Scheme = strings.TrimSpace(input.StoreListings[index].Scheme)
	}
	return input
}

func trimStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func ownerReferences(ownerType string, ownerID *uuid.UUID) (*uuid.UUID, *uuid.UUID) {
	if ownerType == "user" {
		return ownerID, nil
	}
	return nil, ownerID
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateAppReferences(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, input AdminAppInput) error {
	if input.OwnerType == "tenant" && (input.OwnerID == nil || *input.OwnerID != tenantID) {
		return ErrInvalidInput
	}
	if input.OwnerType == "user" {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.tenant_members WHERE tenant_id=$1 AND user_id=$2)`, tenantID, input.OwnerID).Scan(&exists); err != nil || !exists {
			return ErrInvalidInput
		}
	}
	validChannels := []string{"android", "ios", "harmony", "h5", "quickapp", "mp_weixin", "mp_alipay", "mp_baidu", "mp_toutiao", "mp_qq", "mp_kuaishou", "mp_lark", "mp_jd", "mp_dingtalk"}
	channelCodes := make(map[string]struct{}, len(input.Channels))
	fileIDs := append([]uuid.UUID{}, input.ScreenshotFileIDs...)
	if input.IconFileID != nil {
		fileIDs = append(fileIDs, *input.IconFileID)
	}
	for _, channel := range input.Channels {
		if !slices.Contains(validChannels, channel.ChannelCode) || len(channel.Name) > 160 {
			return ErrInvalidInput
		}
		if _, duplicate := channelCodes[channel.ChannelCode]; duplicate {
			return ErrInvalidInput
		}
		channelCodes[channel.ChannelCode] = struct{}{}
		for _, value := range []*string{channel.URL, channel.ABMURL} {
			if value != nil && !strings.HasPrefix(strings.ToLower(*value), "https://") {
				return ErrInvalidInput
			}
		}
		if channel.QRCodeFileID != nil {
			fileIDs = append(fileIDs, *channel.QRCodeFileID)
		}
	}
	fileIDs = uniqueUUIDs(fileIDs)
	if len(fileIDs) > 0 {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM storage.files WHERE tenant_id=$1 AND id=ANY($2::uuid[])
AND deleted_at IS NULL AND status='ready' AND scan_status IN ('clean','skipped') AND media_type LIKE 'image/%'`, tenantID, fileIDs).Scan(&count); err != nil || count != len(fileIDs) {
			return ErrInvalidInput
		}
	}
	storeNames := make(map[string]struct{}, len(input.StoreListings))
	for _, store := range input.StoreListings {
		key := strings.ToLower(store.Name)
		if store.Name == "" || len([]rune(store.Name)) > 160 || store.Priority < -100000 || store.Priority > 100000 {
			return ErrInvalidInput
		}
		if _, duplicate := storeNames[key]; duplicate {
			return ErrInvalidInput
		}
		storeNames[key] = struct{}{}
	}
	return nil
}

func replaceApplicationRelations(ctx context.Context, tx pgx.Tx, tenantID, appID uuid.UUID, input AdminAppInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='app' AND entity_type='application' AND entity_id=$2`, tenantID, appID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app.application_team_members WHERE app_id=$1`, appID); err != nil {
		return err
	}
	managerSet := make(map[uuid.UUID]struct{}, len(input.Managers))
	for _, userID := range input.Managers {
		managerSet[userID] = struct{}{}
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_team_members(tenant_id,app_id,user_id,role) VALUES($1,$2,$3,'manager')`, tenantID, appID, userID); err != nil {
			return ErrInvalidInput
		}
	}
	for _, userID := range input.Members {
		if _, manager := managerSet[userID]; manager {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_team_members(tenant_id,app_id,user_id,role) VALUES($1,$2,$3,'member')`, tenantID, appID, userID); err != nil {
			return ErrInvalidInput
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app.application_assets WHERE app_id=$1`, appID); err != nil {
		return err
	}
	for position, fileID := range input.ScreenshotFileIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_assets(tenant_id,app_id,file_id,asset_type,position) VALUES($1,$2,$3,'screenshot',$4)`, tenantID, appID, fileID, position); err != nil {
			return err
		}
		if err := addApplicationFileUsage(ctx, tx, tenantID, appID, fileID, "screenshots"); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app.application_channels WHERE app_id=$1`, appID); err != nil {
		return err
	}
	for _, channel := range input.Channels {
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_channels(tenant_id,app_id,channel_code,name,url,abm_url,qrcode_file_id,enabled) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, tenantID, appID, channel.ChannelCode, channel.Name, channel.URL, channel.ABMURL, channel.QRCodeFileID, channel.Enabled); err != nil {
			return ErrInvalidInput
		}
		if channel.QRCodeFileID != nil {
			if err := addApplicationFileUsage(ctx, tx, tenantID, appID, *channel.QRCodeFileID, "channel_qrcode"); err != nil {
				return err
			}
		}
	}
	if input.IconFileID != nil {
		if err := addApplicationFileUsage(ctx, tx, tenantID, appID, *input.IconFileID, "icon"); err != nil {
			return err
		}
	}
	keptStoreIDs := make([]uuid.UUID, 0, len(input.StoreListings))
	for _, store := range input.StoreListings {
		storeID := store.ID
		if storeID == uuid.Nil {
			storeID = uuid.New()
		}
		command, err := tx.Exec(ctx, `UPDATE app.application_store_listings SET name=$4,scheme=$5,enabled=$6,priority=$7 WHERE id=$1 AND tenant_id=$2 AND app_id=$3`, storeID, tenantID, appID, store.Name, store.Scheme, store.Enabled, store.Priority)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			if _, err = tx.Exec(ctx, `INSERT INTO app.application_store_listings(id,tenant_id,app_id,name,scheme,enabled,priority) VALUES($1,$2,$3,$4,$5,$6,$7)`, storeID, tenantID, appID, store.Name, store.Scheme, store.Enabled, store.Priority); err != nil {
				return ErrInvalidInput
			}
		}
		keptStoreIDs = append(keptStoreIDs, storeID)
	}
	if _, err := tx.Exec(ctx, `UPDATE app.application_store_listings s SET enabled=false
WHERE s.app_id=$1 AND NOT (s.id=ANY($2::uuid[])) AND EXISTS(SELECT 1 FROM sys.mobile_release_store_listings r WHERE r.store_listing_id=s.id)`, appID, keptStoreIDs); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM app.application_store_listings s WHERE s.app_id=$1 AND NOT (s.id=ANY($2::uuid[]))
AND NOT EXISTS(SELECT 1 FROM sys.mobile_release_store_listings r WHERE r.store_listing_id=s.id)`, appID, keptStoreIDs)
	return err
}

func addApplicationFileUsage(ctx context.Context, tx pgx.Tx, tenantID, appID, fileID uuid.UUID, field string) error {
	_, err := tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name)
VALUES($1,$2,'app','application',$3,$4) ON CONFLICT DO NOTHING`, fileID, tenantID, appID, field)
	return err
}

func derivedAppCode(name string) string {
	value := strings.ToLower(strings.TrimSpace(name))
	var builder strings.Builder
	lastDash := false
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			lastDash = false
		} else if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" || result[0] < 'a' || result[0] > 'z' {
		result = "app-" + result
	}
	if len(result) < 2 {
		result += "app"
	}
	if len(result) > 63 {
		result = result[:63]
		result = strings.TrimRight(result, "-")
	}
	return result
}

func (s *Service) AdminPages(ctx context.Context, token string, appID uuid.UUID, filter AdminListFilter) (AdminPagePage, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.content.read")
	if err != nil {
		return AdminPagePage{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminPagePage{}, err
	}
	filter, err = normalizedAdminListFilter(filter, "draft", "published", "archived")
	if err != nil {
		return AdminPagePage{}, err
	}
	where := `app_id=$1 AND tenant_id=$2 AND ($3='' OR status=$3) AND ($4='' OR slug ILIKE '%' || $4 || '%' OR EXISTS(SELECT 1 FROM content.page_revision_translations t WHERE t.revision_id=current_revision_id AND t.title ILIKE '%' || $4 || '%'))`
	var total int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM content.pages WHERE `+where, appID, p.Tenant.ID, filter.Status, filter.Query).Scan(&total); err != nil {
		return AdminPagePage{}, fmt.Errorf("count app pages: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id,slug,page_type,status,lock_version,current_revision_id,updated_at FROM content.pages WHERE `+where+` ORDER BY page_type,slug LIMIT $5 OFFSET $6`, appID, p.Tenant.ID, filter.Status, filter.Query, filter.PageSize, (filter.Page-1)*filter.PageSize)
	if err != nil {
		return AdminPagePage{}, err
	}
	defer rows.Close()
	out := make([]AdminPage, 0)
	for rows.Next() {
		var x AdminPage
		if err = rows.Scan(&x.ID, &x.Slug, &x.PageType, &x.Status, &x.LockVersion, &x.CurrentRevisionID, &x.UpdatedAt); err != nil {
			return AdminPagePage{}, err
		}
		if x.Status == "published" {
			x.PublicURL = publicurl.Link(s.publicWebBaseURL, appID, "/pages/"+url.PathEscape(x.Slug), "")
		}
		if err = s.hydrateAdminPage(ctx, appID, &x); err != nil {
			return AdminPagePage{}, err
		}
		out = append(out, x)
	}
	if err = rows.Err(); err != nil {
		return AdminPagePage{}, err
	}
	return AdminPagePage{Items: out, Total: total}, nil
}

func (s *Service) SaveAdminPage(ctx context.Context, token string, appID uuid.UUID, input PageInput, publish bool) (AdminPage, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.content.update")
	if err != nil {
		return AdminPage{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminPage{}, err
	}
	if err = validPageInput(input); err != nil {
		return AdminPage{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AdminPage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var page AdminPage
	err = tx.QueryRow(ctx, `SELECT id,slug,page_type,status,lock_version,current_revision_id,updated_at FROM content.pages WHERE app_id=$1 AND slug=$2 FOR UPDATE`, appID, input.Slug).Scan(&page.ID, &page.Slug, &page.PageType, &page.Status, &page.LockVersion, &page.CurrentRevisionID, &page.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if input.PageType != "custom" {
			return AdminPage{}, ErrInvalidInput
		}
		err = tx.QueryRow(ctx, `INSERT INTO content.pages (app_id,tenant_id,slug,page_type,created_by,updated_by) SELECT id,tenant_id,$2,'custom',$3,$3 FROM app.applications WHERE id=$1 RETURNING id,slug,page_type,status,lock_version,current_revision_id,updated_at`, appID, input.Slug, p.User.ID).Scan(&page.ID, &page.Slug, &page.PageType, &page.Status, &page.LockVersion, &page.CurrentRevisionID, &page.UpdatedAt)
	}
	if err != nil {
		return AdminPage{}, err
	}
	if input.LockVersion > 0 && page.LockVersion != input.LockVersion {
		return AdminPage{}, ErrInvalidInput
	}
	var version int32
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(revision_number),0)+1 FROM content.page_revisions WHERE page_id=$1`, page.ID).Scan(&version); err != nil {
		return AdminPage{}, err
	}
	zh := input.Translations["zh-CN"]
	digest := HashDocument("zh-CN", zh.Title, zh.BodyFormat, zh.Body)
	status := "draft"
	if publish {
		status = "published"
	}
	var revisionID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO content.page_revisions (page_id,app_id,tenant_id,revision_number,content_hash,status,published_at,created_by) VALUES ($1,$2,$3,$4,$5,$6::varchar,CASE WHEN $6::varchar='published' THEN now() ELSE NULL END,$7) RETURNING id`, page.ID, appID, p.Tenant.ID, version, digest, status, p.User.ID).Scan(&revisionID)
	if err != nil {
		return AdminPage{}, err
	}
	for locale, value := range input.Translations {
		if _, err = tx.Exec(ctx, `INSERT INTO content.page_revision_translations (revision_id,locale,title,body_format,body) VALUES ($1,$2,$3,$4,$5)`, revisionID, locale, value.Title, value.BodyFormat, value.Body); err != nil {
			return AdminPage{}, err
		}
	}
	err = tx.QueryRow(ctx, `UPDATE content.pages
SET current_revision_id=CASE WHEN $4='published' OR current_revision_id IS NULL THEN $3 ELSE current_revision_id END,
    status=CASE WHEN $4='published' THEN 'published' ELSE status END,
    updated_by=$5,lock_version=lock_version+1
WHERE id=$1 AND app_id=$2
RETURNING id,slug,page_type,status,lock_version,current_revision_id,updated_at`, page.ID, appID, revisionID, status, p.User.ID).Scan(&page.ID, &page.Slug, &page.PageType, &page.Status, &page.LockVersion, &page.CurrentRevisionID, &page.UpdatedAt)
	if err != nil {
		return AdminPage{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminPage{}, err
	}
	if err = s.hydrateAdminPage(ctx, appID, &page); err != nil {
		return AdminPage{}, err
	}
	return page, nil
}

func (s *Service) PublishAdminPage(ctx context.Context, token string, appID uuid.UUID, slug string, lockVersion int32) (AdminPage, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.content.publish")
	if err != nil {
		return AdminPage{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminPage{}, err
	}
	if !slugPattern.MatchString(slug) || lockVersion < 1 {
		return AdminPage{}, ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AdminPage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var page AdminPage
	err = tx.QueryRow(ctx, `SELECT id,slug,page_type,status,lock_version,current_revision_id,updated_at FROM content.pages WHERE app_id=$1 AND slug=$2 FOR UPDATE`, appID, slug).Scan(&page.ID, &page.Slug, &page.PageType, &page.Status, &page.LockVersion, &page.CurrentRevisionID, &page.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminPage{}, ErrAppNotFound
	}
	if err != nil {
		return AdminPage{}, err
	}
	if page.LockVersion != lockVersion {
		return AdminPage{}, ErrInvalidInput
	}
	var revisionID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM content.page_revisions WHERE page_id=$1 AND app_id=$2 AND status='draft' ORDER BY revision_number DESC LIMIT 1 FOR UPDATE`, page.ID, appID).Scan(&revisionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminPage{}, ErrInvalidInput
	}
	if err != nil {
		return AdminPage{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE content.page_revisions SET status='archived' WHERE page_id=$1 AND app_id=$2 AND status='published'`, page.ID, appID); err != nil {
		return AdminPage{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE content.page_revisions SET status='published',published_at=now() WHERE id=$1 AND app_id=$2`, revisionID, appID); err != nil {
		return AdminPage{}, err
	}
	err = tx.QueryRow(ctx, `UPDATE content.pages SET current_revision_id=$3,status='published',updated_by=$4,lock_version=lock_version+1 WHERE id=$1 AND app_id=$2 RETURNING id,slug,page_type,status,lock_version,current_revision_id,updated_at`, page.ID, appID, revisionID, p.User.ID).Scan(&page.ID, &page.Slug, &page.PageType, &page.Status, &page.LockVersion, &page.CurrentRevisionID, &page.UpdatedAt)
	if err != nil {
		return AdminPage{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminPage{}, err
	}
	if err = s.hydrateAdminPage(ctx, appID, &page); err != nil {
		return AdminPage{}, err
	}
	return page, nil
}

func (s *Service) hydrateAdminPage(ctx context.Context, appID uuid.UUID, page *AdminPage) error {
	page.Translations = map[string]AdminPageViewTranslation{}
	if page.CurrentRevisionID != nil {
		rows, err := s.pool.Query(ctx, `SELECT locale,title,body_format,body FROM content.page_revision_translations WHERE revision_id=$1`, *page.CurrentRevisionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var locale, title, bodyFormat string
			var body json.RawMessage
			if err = rows.Scan(&locale, &title, &bodyFormat, &body); err != nil {
				rows.Close()
				return err
			}
			page.Translations[locale] = AdminPageViewTranslation{Title: title, BodyFormat: bodyFormat, Body: append(json.RawMessage(nil), body...)}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	rows, err := s.pool.Query(ctx, `SELECT id,revision_number,status,content_hash,created_at,created_by FROM content.page_revisions WHERE page_id=$1 AND app_id=$2 ORDER BY revision_number DESC`, page.ID, appID)
	if err != nil {
		return err
	}
	defer rows.Close()
	page.Revisions = []AdminPageRevision{}
	for rows.Next() {
		var x AdminPageRevision
		var hash []byte
		if err = rows.Scan(&x.ID, &x.Version, &x.Status, &hash, &x.CreatedAt, &x.CreatedBy); err != nil {
			return err
		}
		x.ContentHash = hex.EncodeToString(hash)
		page.Revisions = append(page.Revisions, x)
	}
	return rows.Err()
}

func (s *Service) DeleteAdminPage(ctx context.Context, token string, appID uuid.UUID, slug string, lockVersion int32) error {
	p, err := s.authorizeAdmin(ctx, token, "app.content.delete")
	if err != nil {
		return err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return err
	}
	if !slugPattern.MatchString(slug) || lockVersion < 1 {
		return ErrInvalidInput
	}
	var pageType string
	err = s.pool.QueryRow(ctx, `SELECT page_type FROM content.pages WHERE app_id=$1 AND slug=$2`, appID, slug).Scan(&pageType)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAppNotFound
	}
	if err != nil {
		return err
	}
	if pageType != "custom" {
		return ErrInvalidInput
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM content.pages WHERE app_id=$1 AND slug=$2 AND lock_version=$3`, appID, slug, lockVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidInput
	}
	return nil
}
func (s *Service) requireAdminApp(ctx context.Context, appID, tenantID uuid.UUID) error {
	var value bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE id=$1 AND tenant_id=$2)`, appID, tenantID).Scan(&value)
	if err != nil {
		return err
	}
	if !value {
		return ErrAppNotFound
	}
	return nil
}
func validPageInput(input PageInput) error {
	if len(input.Slug) < 1 || len(input.Slug) > 120 || !slugPattern.MatchString(input.Slug) || (input.PageType != "custom" && input.PageType != "privacy-policy" && input.PageType != "terms-of-service" && input.PageType != "about-us" && input.PageType != "faq" && input.PageType != "contact-support") || len(input.Translations) != 2 {
		return ErrInvalidInput
	}
	for _, locale := range []string{"zh-CN", "en-US"} {
		x, ok := input.Translations[locale]
		if !ok || len(strings.TrimSpace(x.Title)) < 1 || !json.Valid(x.Body) || (x.BodyFormat != "markdown" && x.BodyFormat != "blocks") {
			return ErrInvalidInput
		}
	}
	return nil
}

func (s *Service) AdminUsers(ctx context.Context, token string, appID uuid.UUID, filter AdminListFilter) (AdminAppUserPage, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.read")
	if err != nil {
		return AdminAppUserPage{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUserPage{}, err
	}
	filter, err = normalizedAdminListFilter(filter, "pending_verification", "active", "disabled")
	if err != nil {
		return AdminAppUserPage{}, err
	}
	where := `m.app_id=$1 AND m.tenant_id=$2 AND ($3='' OR m.status=$3) AND ($4='' OR COALESCE(li.normalized_value::text,'') ILIKE '%' || $4 || '%' OR u.display_name ILIKE '%' || $4 || '%')`
	from := ` FROM app.user_memberships m
JOIN iam.users u ON u.id=m.user_id
LEFT JOIN app.user_login_identifiers li
  ON li.app_id=m.app_id AND li.user_id=m.user_id
 AND li.identifier_type='email' AND li.status='active'`
	var total int
	if err = s.pool.QueryRow(ctx, `SELECT count(*)`+from+` WHERE `+where, appID, p.Tenant.ID, filter.Status, filter.Query).Scan(&total); err != nil {
		return AdminAppUserPage{}, fmt.Errorf("count app users: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT m.user_id,m.user_id,COALESCE(li.normalized_value::text,''),u.display_name,u.avatar_file_id,m.status,m.source,m.verified_at,m.created_at,(SELECT max(last_seen_at) FROM iam.sessions s WHERE s.app_id=m.app_id AND s.user_id=m.user_id AND s.audience='ak-mobile'),m.lock_version`+from+` WHERE `+where+` ORDER BY m.created_at DESC,m.user_id LIMIT $5 OFFSET $6`, appID, p.Tenant.ID, filter.Status, filter.Query, filter.PageSize, (filter.Page-1)*filter.PageSize)
	if err != nil {
		return AdminAppUserPage{}, err
	}
	defer rows.Close()
	out := make([]AdminAppUser, 0)
	for rows.Next() {
		var x AdminAppUser
		var avatarFileID *uuid.UUID
		if err = rows.Scan(&x.ID, &x.UserID, &x.Email, &x.DisplayName, &avatarFileID, &x.Status, &x.Source, &x.VerifiedAt, &x.CreatedAt, &x.LastSignInAt, &x.LockVersion); err != nil {
			return AdminAppUserPage{}, err
		}
		x.AvatarURL = adminAppUserAvatarURL(appID, x.UserID, avatarFileID)
		out = append(out, x)
	}
	if err = rows.Err(); err != nil {
		return AdminAppUserPage{}, err
	}
	return AdminAppUserPage{Items: out, Total: total}, nil
}

func (s *Service) GetAdminUser(ctx context.Context, token string, appID, userID uuid.UUID) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.read")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	out, err := s.readAdminAppUser(ctx, appID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminAppUser{}, ErrAppNotFound
	}
	return out, err
}
func (s *Service) readAdminAppUser(ctx context.Context, appID, userID uuid.UUID) (out AdminAppUser, err error) {
	var avatarFileID *uuid.UUID
	err = s.pool.QueryRow(ctx, `SELECT m.user_id,m.user_id,COALESCE(li.normalized_value::text,''),u.display_name,u.avatar_file_id,m.status,m.source,m.verified_at,m.created_at,(SELECT max(last_seen_at) FROM iam.sessions s WHERE s.app_id=m.app_id AND s.user_id=m.user_id AND s.audience='ak-mobile'),m.lock_version
FROM app.user_memberships m
JOIN iam.users u ON u.id=m.user_id
LEFT JOIN app.user_login_identifiers li
  ON li.app_id=m.app_id AND li.user_id=m.user_id
 AND li.identifier_type='email' AND li.status='active'
WHERE m.app_id=$1 AND m.user_id=$2`, appID, userID).Scan(&out.ID, &out.UserID, &out.Email, &out.DisplayName, &avatarFileID, &out.Status, &out.Source, &out.VerifiedAt, &out.CreatedAt, &out.LastSignInAt, &out.LockVersion)
	out.AvatarURL = adminAppUserAvatarURL(appID, out.UserID, avatarFileID)
	return out, err
}

func adminAppUserAvatarURL(appID, userID uuid.UUID, fileID *uuid.UUID) *string {
	if fileID == nil || *fileID == uuid.Nil {
		return nil
	}
	value := "/apps/" + appID.String() + "/users/" + userID.String() + "/avatar/content?v=" + fileID.String()
	return &value
}
func (s *Service) SetAdminUserStatus(ctx context.Context, token string, appID, userID uuid.UUID, status string, lockVersion int32) (AdminAppUser, error) {
	permission := "app.user.disable"
	if status == "active" {
		permission = "app.user.enable"
	}
	p, err := s.authorizeAdmin(ctx, token, permission)
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	if userID == uuid.Nil || lockVersion < 1 || (status != "active" && status != "disabled") {
		return AdminAppUser{}, ErrInvalidInput
	}
	tag, err := s.pool.Exec(ctx, `UPDATE app.user_memberships SET status=$4,disabled_at=CASE WHEN $4='disabled' THEN now() ELSE NULL END,lock_version=lock_version+1 WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3 AND lock_version=$5`, appID, p.Tenant.ID, userID, status, lockVersion)
	if tag.RowsAffected() == 0 {
		return AdminAppUser{}, s.adminUserMutationMiss(ctx, appID, p.Tenant.ID, userID)
	}
	if err != nil {
		return AdminAppUser{}, err
	}
	if status == "disabled" {
		_, err = s.pool.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=now(),revoke_reason='app_membership_disabled',updated_at=now() WHERE user_id=$1 AND app_id=$2 AND audience='ak-mobile' AND revoked_at IS NULL`, userID, appID)
		if err != nil {
			return AdminAppUser{}, err
		}
	}
	return s.readAdminAppUser(ctx, appID, userID)
}

func (s *Service) CreateAdminUser(ctx context.Context, token string, appID uuid.UUID, input AdminAppUserInput) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.create")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	email, ok := normalizedEmail(input.Email)
	if !ok || len(strings.TrimSpace(input.DisplayName)) < 2 || (input.Locale != "zh-CN" && input.Locale != "en-US") {
		return AdminAppUser{}, ErrInvalidInput
	}
	hash, err := iamapp.HashPassword(input.Password)
	if err != nil {
		return AdminAppUser{}, ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AdminAppUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID uuid.UUID
	// This creates an App user, not an ak-admin identity. Email ownership is
	// scoped to the App identifier table so the same address may belong to a
	// different user in another App without colliding on iam.users.email.
	err = tx.QueryRow(ctx, `INSERT INTO iam.users (display_name,locale,status) VALUES ($1,$2,'active') RETURNING id`, strings.TrimSpace(input.DisplayName), input.Locale).Scan(&userID)
	if err != nil {
		return AdminAppUser{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.user_credentials (user_id,password_hash,force_password_change) VALUES ($1,$2,true)`, userID, hash); err != nil {
		return AdminAppUser{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.tenant_members (tenant_id,user_id,status) VALUES ($1,$2,'active')`, p.Tenant.ID, userID); err != nil {
		return AdminAppUser{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO app.user_memberships (app_id,tenant_id,user_id,source,status,verified_at,invited_by) VALUES ($1,$2,$3,'admin_created','active',now(),$4)`, appID, p.Tenant.ID, userID, p.User.ID)
	if err != nil {
		return AdminAppUser{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,'email',$4,$5,now(),'active')`, p.Tenant.ID, appID, userID, email, maskedEmail(email)); err != nil {
		return AdminAppUser{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminAppUser{}, err
	}
	return s.readAdminAppUser(ctx, appID, userID)
}
func (s *Service) UpdateAdminUser(ctx context.Context, token string, appID, userID uuid.UUID, displayName *string, lockVersion int32) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.update")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	if userID == uuid.Nil || lockVersion < 1 || displayName == nil || len(strings.TrimSpace(*displayName)) < 2 {
		return AdminAppUser{}, ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdminAppUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE app.user_memberships SET lock_version=lock_version+1 WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3 AND lock_version=$4`, appID, p.Tenant.ID, userID, lockVersion)
	if err != nil {
		return AdminAppUser{}, err
	}
	if tag.RowsAffected() == 0 {
		return AdminAppUser{}, s.adminUserMutationMiss(ctx, appID, p.Tenant.ID, userID)
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.users SET display_name=$2,updated_at=now() WHERE id=$1`, userID, strings.TrimSpace(*displayName)); err != nil {
		return AdminAppUser{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminAppUser{}, err
	}
	return s.readAdminAppUser(ctx, appID, userID)
}
func (s *Service) UnlockAdminUser(ctx context.Context, token string, appID, userID uuid.UUID, lockVersion int32) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.unlock")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	if err = s.bumpAdminUserLock(ctx, appID, p.Tenant.ID, userID, lockVersion); err != nil {
		return AdminAppUser{}, err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE iam.user_credentials c SET failed_attempts=0,locked_until=NULL,updated_at=now() FROM app.user_memberships m WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND c.user_id=m.user_id`, appID, p.Tenant.ID, userID)
	if err != nil {
		return AdminAppUser{}, err
	}
	if tag.RowsAffected() == 0 {
		return AdminAppUser{}, ErrAppNotFound
	}
	return s.readAdminAppUser(ctx, appID, userID)
}
func (s *Service) ResetAdminUserPassword(ctx context.Context, token string, appID, userID uuid.UUID, password string, lockVersion int32) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.reset_password")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	if lockVersion < 1 {
		return AdminAppUser{}, ErrInvalidInput
	}
	hash, err := iamapp.HashPassword(password)
	if err != nil {
		return AdminAppUser{}, ErrInvalidInput
	}
	if err = s.bumpAdminUserLock(ctx, appID, p.Tenant.ID, userID, lockVersion); err != nil {
		return AdminAppUser{}, err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE iam.user_credentials c SET password_hash=$4,password_version=password_version+1,password_changed_at=now(),force_password_change=true,failed_attempts=0,locked_until=NULL,updated_at=now() FROM app.user_memberships m WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND c.user_id=m.user_id`, appID, p.Tenant.ID, userID, hash)
	if err != nil {
		return AdminAppUser{}, err
	}
	if tag.RowsAffected() == 0 {
		return AdminAppUser{}, ErrAppNotFound
	}
	if err = s.revokeAppMobileSessions(ctx, appID, userID, "app_admin_password_reset"); err != nil {
		return AdminAppUser{}, err
	}
	return s.readAdminAppUser(ctx, appID, userID)
}
func (s *Service) RevokeAdminUserSessions(ctx context.Context, token string, appID, userID uuid.UUID, lockVersion int32) (AdminAppUser, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.user.revoke_session")
	if err != nil {
		return AdminAppUser{}, err
	}
	if err = s.requireAdminApp(ctx, appID, p.Tenant.ID); err != nil {
		return AdminAppUser{}, err
	}
	if err = s.bumpAdminUserLock(ctx, appID, p.Tenant.ID, userID, lockVersion); err != nil {
		return AdminAppUser{}, err
	}
	if err = s.revokeAppMobileSessions(ctx, appID, userID, "app_admin_session_revoke"); err != nil {
		return AdminAppUser{}, err
	}
	return s.readAdminAppUser(ctx, appID, userID)
}

func (s *Service) bumpAdminUserLock(ctx context.Context, appID, tenantID, userID uuid.UUID, lockVersion int32) error {
	if lockVersion < 1 {
		return ErrInvalidInput
	}
	tag, err := s.pool.Exec(ctx, `UPDATE app.user_memberships SET lock_version=lock_version+1 WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3 AND lock_version=$4`, appID, tenantID, userID, lockVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.adminUserMutationMiss(ctx, appID, tenantID, userID)
	}
	return nil
}

func (s *Service) adminUserMutationMiss(ctx context.Context, appID, tenantID, userID uuid.UUID) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_memberships WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3)`, appID, tenantID, userID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrConflict
	}
	return ErrAppNotFound
}
func (s *Service) revokeAppMobileSessions(ctx context.Context, appID, userID uuid.UUID, reason string) error {
	_, err := s.pool.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=now(),revoke_reason=$3,access_token_version=access_token_version+1,updated_at=now() WHERE app_id=$1 AND user_id=$2 AND audience='ak-mobile' AND revoked_at IS NULL`, appID, userID, reason)
	return err
}
