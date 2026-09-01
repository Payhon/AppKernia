package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/appkernia/appkernia/server/internal/infrastructure/db"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/appkernia/appkernia/server/internal/shared/publicurl"
	"github.com/appkernia/appkernia/server/internal/shared/richtext"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WebTranslation struct {
	Name                 string `json:"name"`
	Introduction         string `json:"introduction"`
	PromotionTitle       string `json:"promotion_title"`
	PromotionDescription string `json:"promotion_description"`
	PromotionButtonLabel string `json:"promotion_button_label"`
}
type WebTranslationInput struct {
	Name                 string  `json:"name"`
	Introduction         string  `json:"introduction"`
	PromotionTitle       *string `json:"promotion_title,omitempty"`
	PromotionDescription *string `json:"promotion_description,omitempty"`
	PromotionButtonLabel *string `json:"promotion_button_label,omitempty"`
}
type WebStore struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Enabled  bool      `json:"enabled"`
	Priority int32     `json:"priority"`
	Platform string    `json:"platform"`
	WebURL   string    `json:"web_url"`
}
type PublicWebConfig struct {
	AppID            uuid.UUID                 `json:"app_id"`
	Enabled          bool                      `json:"enabled"`
	APKEnabled       bool                      `json:"apk_enabled"`
	PromotionEnabled bool                      `json:"promotion_enabled"`
	LockVersion      int32                     `json:"lock_version"`
	Translations     map[string]WebTranslation `json:"translations"`
	Stores           []WebStore                `json:"stores"`
	DownloadPageURL  string                    `json:"download_page_url"`
}
type PublicWebInput struct {
	Enabled          bool                           `json:"enabled"`
	APKEnabled       bool                           `json:"apk_enabled"`
	PromotionEnabled *bool                          `json:"promotion_enabled,omitempty"`
	LockVersion      int32                          `json:"lock_version"`
	Translations     map[string]WebTranslationInput `json:"translations"`
	Stores           []WebStore                     `json:"stores"`
}
type WebPageLink struct {
	Slug  string
	Title string
}
type PublicWebApp struct {
	ID                                                         uuid.UUID
	Name                                                       string
	Introduction                                               string
	IconFileID                                                 *uuid.UUID
	ScreenshotIDs                                              []uuid.UUID
	Enabled                                                    bool
	APKEnabled                                                 bool
	PromotionEnabled                                           bool
	PromotionTitle, PromotionDescription, PromotionButtonLabel string
	Stores                                                     []WebStore
	Pages                                                      []WebPageLink
}

func WithPublicWebBaseURL(base string) Option {
	return func(s *Service) { s.publicWebBaseURL = strings.TrimRight(base, "/") }
}
func (s *Service) DownloadPageURL(id uuid.UUID, locale string) string {
	return publicurl.Link(s.publicWebBaseURL, id, "/download", locale)
}
func (s *Service) PublicDownloadPageURL(ctx context.Context, app Application, locale string) (string, error) {
	cfg, err := db.New(s.pool).PublicWebGetConfig(ctx, db.PublicWebGetConfigParams{TenantID: app.TenantID, ID: app.ID})
	if err != nil {
		return "", err
	}
	if !cfg.Enabled {
		return "", nil
	}
	return s.DownloadPageURL(app.ID, locale), nil
}
func validWebInput(in PublicWebInput) error {
	if in.LockVersion < 0 || len(in.Translations) != 2 || len(in.Stores) > 100 {
		return ErrInvalidInput
	}
	for _, loc := range []string{"zh-CN", "en-US"} {
		t, ok := in.Translations[loc]
		if !ok || utf8.RuneCountInString(t.Name) > 160 || utf8.RuneCountInString(t.Introduction) > 20000 || (t.PromotionTitle != nil && utf8.RuneCountInString(*t.PromotionTitle) > 160) || (t.PromotionDescription != nil && utf8.RuneCountInString(*t.PromotionDescription) > 500) || (t.PromotionButtonLabel != nil && utf8.RuneCountInString(*t.PromotionButtonLabel) > 80) || (in.Enabled && (strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Introduction) == "")) {
			return ErrInvalidInput
		}
	}
	seen := map[uuid.UUID]bool{}
	for _, x := range in.Stores {
		if x.ID == uuid.Nil || seen[x.ID] || (x.Platform != "" && x.Platform != "ios" && x.Platform != "android" && x.Platform != "harmony") {
			return ErrInvalidInput
		}
		seen[x.ID] = true
		if x.WebURL != "" {
			u, e := url.Parse(x.WebURL)
			if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(x.WebURL) > 2048 || strings.ContainsAny(x.WebURL, "\r\n\t ") || x.Platform == "" {
				return ErrInvalidInput
			}
		}
	}
	return nil
}
func readWebConfig(ctx context.Context, q *db.Queries, tenant, id uuid.UUID) (PublicWebConfig, error) {
	out := PublicWebConfig{AppID: id, Translations: map[string]WebTranslation{"zh-CN": {}, "en-US": {}}, Stores: []WebStore{}}
	c, e := q.PublicWebGetConfig(ctx, db.PublicWebGetConfigParams{TenantID: tenant, ID: id})
	if e != nil {
		return out, e
	}
	out.Enabled, out.APKEnabled, out.PromotionEnabled, out.LockVersion = c.Enabled, c.ApkEnabled, c.PromotionEnabled, c.LockVersion
	ts, e := q.PublicWebTranslations(ctx, db.PublicWebTranslationsParams{TenantID: tenant, AppID: id})
	if e != nil {
		return out, e
	}
	for _, t := range ts {
		out.Translations[t.Locale] = WebTranslation{Name: t.Name, Introduction: t.Introduction, PromotionTitle: t.PromotionTitle, PromotionDescription: t.PromotionDescription, PromotionButtonLabel: t.PromotionButtonLabel}
	}
	ss, e := q.PublicWebStores(ctx, db.PublicWebStoresParams{TenantID: tenant, AppID: id})
	if e != nil {
		return out, e
	}
	for _, x := range ss {
		out.Stores = append(out.Stores, WebStore{ID: x.ID, Name: x.Name, Enabled: x.Enabled, Priority: x.Priority, Platform: x.Platform, WebURL: x.WebUrl})
	}
	return out, nil
}
func (s *Service) GetAdminPublicWebConfig(ctx context.Context, token string, id uuid.UUID) (PublicWebConfig, error) {
	p, e := s.authorizeAdmin(ctx, token, "app.public_web.read")
	if e != nil {
		return PublicWebConfig{}, e
	}
	out, e := readWebConfig(ctx, db.New(s.pool), p.Tenant.ID, id)
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrAppNotFound
	}
	out.DownloadPageURL = s.DownloadPageURL(id, "")
	return out, e
}
func (s *Service) UpdateAdminPublicWebConfig(ctx context.Context, token string, id uuid.UUID, in PublicWebInput, requestID string) (PublicWebConfig, error) {
	p, e := s.authorizeAdmin(ctx, token, "app.public_web.update")
	if e != nil {
		return PublicWebConfig{}, e
	}
	if e = validWebInput(in); e != nil {
		return PublicWebConfig{}, e
	}
	if in.Enabled && s.publicWebBaseURL == "" {
		return PublicWebConfig{}, ErrInvalidInput
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return PublicWebConfig{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	if _, e = q.PublicWebLockApp(ctx, db.PublicWebLockAppParams{TenantID: p.Tenant.ID, ID: id}); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			e = ErrAppNotFound
		}
		return PublicWebConfig{}, e
	}
	before, e := readWebConfig(ctx, q, p.Tenant.ID, id)
	if e != nil {
		return PublicWebConfig{}, e
	}
	if before.LockVersion != in.LockVersion {
		return PublicWebConfig{}, ErrConflict
	}
	promotionEnabled := before.PromotionEnabled
	if in.PromotionEnabled != nil {
		promotionEnabled = *in.PromotionEnabled
	}
	if e = q.PublicWebWriteConfig(ctx, db.PublicWebWriteConfigParams{TenantID: p.Tenant.ID, AppID: id, Enabled: in.Enabled, ApkEnabled: in.APKEnabled, PromotionEnabled: promotionEnabled}); e != nil {
		return PublicWebConfig{}, e
	}
	for _, loc := range []string{"zh-CN", "en-US"} {
		t := in.Translations[loc]
		beforeTranslation := before.Translations[loc]
		promotionTitle, promotionDescription, promotionButtonLabel := beforeTranslation.PromotionTitle, beforeTranslation.PromotionDescription, beforeTranslation.PromotionButtonLabel
		if t.PromotionTitle != nil {
			promotionTitle = strings.TrimSpace(*t.PromotionTitle)
		}
		if t.PromotionDescription != nil {
			promotionDescription = strings.TrimSpace(*t.PromotionDescription)
		}
		if t.PromotionButtonLabel != nil {
			promotionButtonLabel = strings.TrimSpace(*t.PromotionButtonLabel)
		}
		e = q.PublicWebWriteTranslation(ctx, db.PublicWebWriteTranslationParams{TenantID: p.Tenant.ID, AppID: id, Locale: loc, Name: strings.TrimSpace(t.Name), Introduction: strings.TrimSpace(t.Introduction), PromotionTitle: promotionTitle, PromotionDescription: promotionDescription, PromotionButtonLabel: promotionButtonLabel})
		if e != nil {
			return PublicWebConfig{}, e
		}
	}
	for _, x := range in.Stores {
		n, err := q.PublicWebWriteStore(ctx, db.PublicWebWriteStoreParams{TenantID: p.Tenant.ID, AppID: id, ID: x.ID, Platform: x.Platform, WebUrl: x.WebURL})
		if err != nil {
			return PublicWebConfig{}, err
		}
		if n != 1 {
			return PublicWebConfig{}, ErrInvalidInput
		}
	}
	after, e := readWebConfig(ctx, q, p.Tenant.ID, id)
	if e != nil {
		return PublicWebConfig{}, e
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_, e = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,before_data,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'app','app.public_web.update','app.public_web.update','application_public_web_config',$4::text,'PUT','/admin-api/v1/apps/'||$4::text||'/public-web-config',$5,$6,true)`, p.Tenant.ID, p.User.ID, requestID, id.String(), b, a)
	if e != nil {
		return PublicWebConfig{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return PublicWebConfig{}, e
	}
	after.DownloadPageURL = s.DownloadPageURL(id, "")
	return after, nil
}

// PublicWeb reads only the public presentation of a server-resolved active App.
func (s *Service) PublicWeb(ctx context.Context, id uuid.UUID, locale string) (PublicWebApp, error) {
	a, e := s.Resolve(ctx, id)
	if e != nil {
		return PublicWebApp{}, e
	}
	q := db.New(s.pool)
	cfg, e := readWebConfig(ctx, q, a.TenantID, id)
	if e != nil {
		return PublicWebApp{}, e
	}
	t := WebTranslation{}
	if cfg.Enabled {
		t = cfg.Translations[locale]
	}
	if cfg.Enabled && t.Name == "" {
		t = cfg.Translations["zh-CN"]
	}
	if t.Name == "" {
		t.Name = a.Name
	}
	out := PublicWebApp{ID: id, Name: t.Name, Introduction: t.Introduction, IconFileID: a.IconFileID, Enabled: cfg.Enabled, APKEnabled: cfg.APKEnabled, PromotionEnabled: cfg.Enabled && cfg.PromotionEnabled, PromotionTitle: t.PromotionTitle, PromotionDescription: t.PromotionDescription, PromotionButtonLabel: t.PromotionButtonLabel, Stores: cfg.Stores}
	if cfg.Enabled {
		out.ScreenshotIDs, e = q.PublicWebScreenshots(ctx, db.PublicWebScreenshotsParams{TenantID: a.TenantID, AppID: id})
		if e != nil {
			return out, e
		}
	}
	links, e := q.PublicWebPageLinks(ctx, db.PublicWebPageLinksParams{TenantID: a.TenantID, AppID: id, Locale: locale})
	if e != nil {
		return out, e
	}
	for _, x := range links {
		out.Pages = append(out.Pages, WebPageLink{x.Slug, x.Title})
	}
	return out, nil
}
func (s *Service) OpenPublicWebAsset(ctx context.Context, id, fileID uuid.UUID) (string, int64, io.ReadCloser, error) {
	a, e := s.Resolve(ctx, id)
	if e != nil {
		return "", 0, nil, e
	}
	if s.objects == nil {
		return "", 0, nil, ErrAppNotFound
	}
	x, e := db.New(s.pool).PublicWebAsset(ctx, db.PublicWebAssetParams{TenantID: a.TenantID, ID: id, ID_2: fileID})

	if errors.Is(e, pgx.ErrNoRows) {
		// Older page revisions have no file_usages rows. Check the current published body itself.
		q := db.New(s.pool)
		candidates, err := q.PublicWebPageImageCandidates(ctx, db.PublicWebPageImageCandidatesParams{TenantID: a.TenantID, AppID: id, FileIDText: fileID.String()})
		if err != nil {
			return "", 0, nil, err
		}
		referenced := false
		for _, page := range candidates {
			if richtext.ReferencesImage(page.Body, page.BodyFormat, id, fileID) {
				referenced = true
				break
			}
		}
		if !referenced {
			return "", 0, nil, ErrAppNotFound
		}
		f, err := q.PublicWebPageImageFile(ctx, db.PublicWebPageImageFileParams{TenantID: a.TenantID, ID: fileID})
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, nil, ErrAppNotFound
		}
		if err != nil {
			return "", 0, nil, err
		}
		x.ID, x.Provider, x.BucketName, x.ObjectKey, x.MediaType, x.SizeBytes = f.ID, f.Provider, f.BucketName, f.ObjectKey, f.MediaType, f.SizeBytes
		e = nil
	}
	if e != nil {
		return "", 0, nil, e
	}

	reader, e := s.objects.Open(ctx, storage.ObjectRef{TenantID: a.TenantID, Provider: x.Provider, Bucket: x.BucketName, Key: x.ObjectKey})
	return x.MediaType, x.SizeBytes, reader, e
}
