package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrStartupAssetRejected = errors.New("startup asset rejected")

var startupLocales = []string{"zh-CN", "en-US"}

type StartupTranslation struct {
	DisplayName string `json:"display_name"`
	Subtitle    string `json:"subtitle"`
}

type StartupSlideAsset struct {
	FileID             uuid.UUID `json:"file_id"`
	AccessibilityLabel string    `json:"accessibility_label"`
}

type StartupSlide struct {
	ID       uuid.UUID                    `json:"id"`
	Position int32                        `json:"position"`
	Assets   map[string]StartupSlideAsset `json:"assets"`
}

type StartupConfiguration struct {
	Translations      map[string]StartupTranslation `json:"translations"`
	OnboardingEnabled bool                          `json:"onboarding_enabled"`
	DraftSlides       []StartupSlide                `json:"draft_slides"`
	PublishedVersion  int32                         `json:"published_version"`
	PublishedAt       *time.Time                    `json:"published_at"`
	DraftChanged      bool                          `json:"draft_changed"`
}

type StartupInput struct {
	Translations      map[string]StartupTranslation `json:"translations"`
	OnboardingEnabled bool                          `json:"onboarding_enabled"`
	DraftSlides       []StartupSlide                `json:"draft_slides"`
}

type PublicStartupSlide struct {
	Position           int32  `json:"position"`
	ImageURL           string `json:"image_url"`
	AccessibilityLabel string `json:"accessibility_label"`
}

type PublicStartupConfiguration struct {
	DisplayName       string               `json:"display_name"`
	Subtitle          string               `json:"subtitle"`
	IconURL           *string              `json:"icon_url"`
	OnboardingEnabled bool                 `json:"onboarding_enabled"`
	PublishedVersion  int32                `json:"published_version"`
	Slides            []PublicStartupSlide `json:"slides"`
}

type StartupAsset struct {
	FileID, TenantID            uuid.UUID
	Provider, Bucket, ObjectKey string
	MediaType                   string
	SizeBytes                   int64
	SHA256                      []byte
	PublishedVersion            int32
}

func normalizeStartupInput(input StartupInput) StartupInput {
	if input.Translations == nil {
		input.Translations = map[string]StartupTranslation{}
	}
	for locale, translation := range input.Translations {
		translation.DisplayName = strings.TrimSpace(translation.DisplayName)
		translation.Subtitle = strings.TrimSpace(translation.Subtitle)
		input.Translations[locale] = translation
	}
	for slideIndex := range input.DraftSlides {
		input.DraftSlides[slideIndex].Position = int32(slideIndex)
		if input.DraftSlides[slideIndex].Assets == nil {
			input.DraftSlides[slideIndex].Assets = map[string]StartupSlideAsset{}
		}
		for locale, asset := range input.DraftSlides[slideIndex].Assets {
			asset.AccessibilityLabel = strings.TrimSpace(asset.AccessibilityLabel)
			input.DraftSlides[slideIndex].Assets[locale] = asset
		}
	}
	return input
}

func validStartupInput(input StartupInput) error {
	if len(input.Translations) != len(startupLocales) || len(input.DraftSlides) > 10 {
		return ErrInvalidInput
	}
	for _, locale := range startupLocales {
		translation, ok := input.Translations[locale]
		if !ok || len([]rune(translation.DisplayName)) < 1 || len([]rune(translation.DisplayName)) > 120 || len([]rune(translation.Subtitle)) > 240 {
			return ErrInvalidInput
		}
	}
	for _, slide := range input.DraftSlides {
		if len(slide.Assets) != len(startupLocales) {
			return ErrInvalidInput
		}
		for _, locale := range startupLocales {
			asset, ok := slide.Assets[locale]
			if !ok || asset.FileID == uuid.Nil || len([]rune(asset.AccessibilityLabel)) < 1 || len([]rune(asset.AccessibilityLabel)) > 500 {
				return ErrInvalidInput
			}
		}
	}
	return nil
}

func startupFileIDs(input StartupInput) []uuid.UUID {
	values := make([]uuid.UUID, 0, len(input.DraftSlides)*2)
	for _, slide := range input.DraftSlides {
		for _, locale := range startupLocales {
			values = append(values, slide.Assets[locale].FileID)
		}
	}
	return uniqueUUIDs(values)
}

func validateStartupFiles(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, fileIDs []uuid.UUID) error {
	if len(fileIDs) == 0 {
		return nil
	}
	var count int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM storage.files
WHERE tenant_id=$1 AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND status='ready'
  AND scan_status='clean' AND lower(COALESCE(media_type,'')) IN ('image/jpeg','image/png','image/webp')`, tenantID, fileIDs).Scan(&count)
	if err != nil {
		return err
	}
	if count != len(fileIDs) {
		return ErrStartupAssetRejected
	}
	return nil
}

func saveStartupDraft(ctx context.Context, tx pgx.Tx, tenantID, appID uuid.UUID, raw StartupInput) error {
	input := normalizeStartupInput(raw)
	if err := validStartupInput(input); err != nil {
		return err
	}
	if err := validateStartupFiles(ctx, tx, tenantID, startupFileIDs(input)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.application_startup_configs(tenant_id,app_id,onboarding_enabled)
VALUES($1,$2,$3) ON CONFLICT(app_id) DO UPDATE SET onboarding_enabled=EXCLUDED.onboarding_enabled,
draft_generation=app.application_startup_configs.draft_generation+1`, tenantID, appID, input.OnboardingEnabled); err != nil {
		return err
	}
	for _, locale := range startupLocales {
		translation := input.Translations[locale]
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_startup_translations(tenant_id,app_id,locale,display_name,subtitle)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(app_id,locale) DO UPDATE SET display_name=EXCLUDED.display_name,subtitle=EXCLUDED.subtitle`,
			tenantID, appID, locale, translation.DisplayName, translation.Subtitle); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='app' AND entity_type='application_onboarding_draft' AND entity_id=$2`, tenantID, appID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app.application_onboarding_draft_slides WHERE tenant_id=$1 AND app_id=$2`, tenantID, appID); err != nil {
		return err
	}
	for position, slide := range input.DraftSlides {
		slideID := uuid.New()
		if _, err := tx.Exec(ctx, `INSERT INTO app.application_onboarding_draft_slides(id,tenant_id,app_id,position) VALUES($1,$2,$3,$4)`, slideID, tenantID, appID, position); err != nil {
			return err
		}
		for _, locale := range startupLocales {
			asset := slide.Assets[locale]
			if _, err := tx.Exec(ctx, `INSERT INTO app.application_onboarding_draft_assets(tenant_id,app_id,slide_id,locale,file_id,accessibility_label) VALUES($1,$2,$3,$4,$5,$6)`, tenantID, appID, slideID, locale, asset.FileID, asset.AccessibilityLabel); err != nil {
				return err
			}
			field := fmt.Sprintf("slide_%02d_%s", position, locale)
			if _, err := tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'app','application_onboarding_draft',$3,$4) ON CONFLICT DO NOTHING`, asset.FileID, tenantID, appID, field); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) loadStartupConfiguration(ctx context.Context, item *Application) error {
	item.Startup = StartupConfiguration{Translations: map[string]StartupTranslation{}, DraftSlides: []StartupSlide{}}
	var generation, sourceGeneration int32
	err := s.pool.QueryRow(ctx, `SELECT onboarding_enabled,published_version,published_at,draft_generation,
COALESCE((SELECT source_generation FROM app.application_onboarding_revisions r WHERE r.id=c.published_revision_id),0)
FROM app.application_startup_configs c WHERE tenant_id=$1 AND app_id=$2`, item.TenantID, item.ID).Scan(
		&item.Startup.OnboardingEnabled, &item.Startup.PublishedVersion, &item.Startup.PublishedAt, &generation, &sourceGeneration)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, `SELECT locale,display_name,subtitle FROM app.application_startup_translations WHERE tenant_id=$1 AND app_id=$2 ORDER BY locale`, item.TenantID, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var locale string
		var translation StartupTranslation
		if err = rows.Scan(&locale, &translation.DisplayName, &translation.Subtitle); err != nil {
			rows.Close()
			return err
		}
		item.Startup.Translations[locale] = translation
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT s.id,s.position,a.locale,a.file_id,a.accessibility_label
FROM app.application_onboarding_draft_slides s
JOIN app.application_onboarding_draft_assets a ON a.slide_id=s.id AND a.tenant_id=s.tenant_id AND a.app_id=s.app_id
WHERE s.tenant_id=$1 AND s.app_id=$2 ORDER BY s.position,a.locale`, item.TenantID, item.ID)
	if err != nil {
		return err
	}
	byID := map[uuid.UUID]int{}
	for rows.Next() {
		var id, fileID uuid.UUID
		var position int32
		var locale, label string
		if err = rows.Scan(&id, &position, &locale, &fileID, &label); err != nil {
			rows.Close()
			return err
		}
		index, ok := byID[id]
		if !ok {
			index = len(item.Startup.DraftSlides)
			byID[id] = index
			item.Startup.DraftSlides = append(item.Startup.DraftSlides, StartupSlide{ID: id, Position: position, Assets: map[string]StartupSlideAsset{}})
		}
		item.Startup.DraftSlides[index].Assets[locale] = StartupSlideAsset{FileID: fileID, AccessibilityLabel: label}
	}
	rows.Close()
	item.Startup.DraftChanged = item.Startup.PublishedVersion == 0 && len(item.Startup.DraftSlides) > 0 || item.Startup.PublishedVersion > 0 && generation != sourceGeneration
	return rows.Err()
}

func (s *Service) PublicStartup(ctx context.Context, app Application, locale string) (PublicStartupConfiguration, error) {
	if locale != "en-US" {
		locale = "zh-CN"
	}
	out := PublicStartupConfiguration{Slides: []PublicStartupSlide{}}
	err := s.pool.QueryRow(ctx, `SELECT t.display_name,t.subtitle,c.onboarding_enabled,c.published_version
FROM app.application_startup_configs c
JOIN app.application_startup_translations t ON t.app_id=c.app_id AND t.tenant_id=c.tenant_id AND t.locale=$3
WHERE c.tenant_id=$1 AND c.app_id=$2`, app.TenantID, app.ID, locale).Scan(&out.DisplayName, &out.Subtitle, &out.OnboardingEnabled, &out.PublishedVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		out.DisplayName = app.Name
	} else if err != nil {
		return out, err
	}
	if app.IconFileID != nil {
		value := "/api/v1/public/startup-assets/" + app.IconFileID.String()
		out.IconURL = &value
	}
	if !out.OnboardingEnabled || out.PublishedVersion < 1 {
		out.OnboardingEnabled = false
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT s.position,a.file_id,a.accessibility_label
FROM app.application_startup_configs c
JOIN app.application_onboarding_revision_slides s ON s.revision_id=c.published_revision_id AND s.tenant_id=c.tenant_id AND s.app_id=c.app_id
JOIN app.application_onboarding_revision_assets a ON a.slide_id=s.id AND a.revision_id=s.revision_id AND a.locale=$3
WHERE c.tenant_id=$1 AND c.app_id=$2 ORDER BY s.position`, app.TenantID, app.ID, locale)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var slide PublicStartupSlide
		var fileID uuid.UUID
		if err = rows.Scan(&slide.Position, &fileID, &slide.AccessibilityLabel); err != nil {
			return out, err
		}
		slide.ImageURL = "/api/v1/public/startup-assets/" + fileID.String()
		out.Slides = append(out.Slides, slide)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Slides) == 0 {
		out.OnboardingEnabled = false
	}
	return out, nil
}

func (s *Service) PublishOnboarding(ctx context.Context, token string, appID uuid.UUID, expectedVersion int32, requestID string) (Application, error) {
	p, err := s.authorizeAdmin(ctx, token, "app.onboarding.publish")
	if err != nil {
		return Application{}, err
	}
	if appID == uuid.Nil || expectedVersion < 0 {
		return Application{}, ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Application{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion, generation int32
	err = tx.QueryRow(ctx, `SELECT c.published_version,c.draft_generation FROM app.application_startup_configs c
JOIN app.applications a ON a.id=c.app_id AND a.tenant_id=c.tenant_id
WHERE c.tenant_id=$1 AND c.app_id=$2 AND a.deleted_at IS NULL FOR UPDATE`, p.Tenant.ID, appID).Scan(&currentVersion, &generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrAppNotFound
	}
	if err != nil {
		return Application{}, err
	}
	if currentVersion != expectedVersion {
		return Application{}, ErrConflict
	}
	var slideCount, assetCount int
	if err = tx.QueryRow(ctx, `SELECT count(DISTINCT s.id),count(a.*) FROM app.application_onboarding_draft_slides s
LEFT JOIN app.application_onboarding_draft_assets a ON a.slide_id=s.id
WHERE s.tenant_id=$1 AND s.app_id=$2`, p.Tenant.ID, appID).Scan(&slideCount, &assetCount); err != nil {
		return Application{}, err
	}
	if slideCount < 1 || slideCount > 10 || assetCount != slideCount*2 {
		return Application{}, ErrInvalidInput
	}
	rows, err := tx.Query(ctx, `SELECT s.id,s.position,a.locale,a.file_id,a.accessibility_label
FROM app.application_onboarding_draft_slides s JOIN app.application_onboarding_draft_assets a ON a.slide_id=s.id
WHERE s.tenant_id=$1 AND s.app_id=$2 ORDER BY s.position,a.locale`, p.Tenant.ID, appID)
	if err != nil {
		return Application{}, err
	}
	type draftRow struct {
		slideID, fileID uuid.UUID
		position        int32
		locale, label   string
	}
	drafts := make([]draftRow, 0, assetCount)
	fileIDs := make([]uuid.UUID, 0, assetCount)
	for rows.Next() {
		var row draftRow
		if err = rows.Scan(&row.slideID, &row.position, &row.locale, &row.fileID, &row.label); err != nil {
			rows.Close()
			return Application{}, err
		}
		drafts = append(drafts, row)
		fileIDs = append(fileIDs, row.fileID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return Application{}, err
	}
	if err = validateStartupFiles(ctx, tx, p.Tenant.ID, uniqueUUIDs(fileIDs)); err != nil {
		return Application{}, err
	}
	newVersion := currentVersion + 1
	revisionID := uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO app.application_onboarding_revisions(id,tenant_id,app_id,version,source_generation,published_by) VALUES($1,$2,$3,$4,$5,$6)`, revisionID, p.Tenant.ID, appID, newVersion, generation, p.User.ID); err != nil {
		return Application{}, err
	}
	revisionSlides := map[uuid.UUID]uuid.UUID{}
	for _, row := range drafts {
		revisionSlideID, exists := revisionSlides[row.slideID]
		if !exists {
			revisionSlideID = uuid.New()
			revisionSlides[row.slideID] = revisionSlideID
			if _, err = tx.Exec(ctx, `INSERT INTO app.application_onboarding_revision_slides(id,tenant_id,app_id,revision_id,position) VALUES($1,$2,$3,$4,$5)`, revisionSlideID, p.Tenant.ID, appID, revisionID, row.position); err != nil {
				return Application{}, err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.application_onboarding_revision_assets(tenant_id,app_id,revision_id,slide_id,locale,file_id,accessibility_label) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.Tenant.ID, appID, revisionID, revisionSlideID, row.locale, row.fileID, row.label); err != nil {
			return Application{}, err
		}
		field := fmt.Sprintf("slide_%02d_%s", row.position, row.locale)
		if _, err = tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'app','application_onboarding_revision',$3,$4) ON CONFLICT DO NOTHING`, row.fileID, p.Tenant.ID, revisionID, field); err != nil {
			return Application{}, err
		}
	}
	command, err := tx.Exec(ctx, `UPDATE app.application_startup_configs SET published_version=$3,published_revision_id=$4,published_at=now(),published_by=$5 WHERE tenant_id=$1 AND app_id=$2 AND published_version=$6`, p.Tenant.ID, appID, newVersion, revisionID, p.User.ID, currentVersion)
	if err != nil {
		return Application{}, err
	}
	if command.RowsAffected() != 1 {
		return Application{}, ErrConflict
	}
	after := []byte(fmt.Sprintf(`{"published_version":%d,"revision_id":%q}`, newVersion, revisionID.String()))
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'app','app.onboarding.publish','app.onboarding.publish','app.onboarding_revision',$4,'POST','/admin-api/v1/apps/'||$5||'/startup/onboarding/publish',200,$6,true)`, p.Tenant.ID, p.User.ID, strings.TrimSpace(requestID), revisionID.String(), appID.String(), after); err != nil {
		return Application{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "serialization") {
			return Application{}, ErrConflict
		}
		return Application{}, err
	}
	return s.GetAdminApp(ctx, token, appID)
}

func (s *Service) OpenStartupAsset(ctx context.Context, app Application, fileID uuid.UUID) (StartupAsset, io.ReadCloser, error) {
	if fileID == uuid.Nil || s.objects == nil {
		return StartupAsset{}, nil, ErrAppNotFound
	}
	var asset StartupAsset
	asset.FileID, asset.TenantID = fileID, app.TenantID
	var mediaType *string
	err := s.pool.QueryRow(ctx, `SELECT f.provider,f.bucket_name,f.object_key,f.media_type,f.size_bytes,f.sha256,c.published_version
FROM storage.files f JOIN app.application_startup_configs c ON c.tenant_id=f.tenant_id AND c.app_id=$2
WHERE f.tenant_id=$1 AND f.id=$3 AND f.deleted_at IS NULL AND f.status='ready' AND f.scan_status='clean'
AND lower(COALESCE(f.media_type,'')) IN ('image/jpeg','image/png','image/webp')
AND (EXISTS(SELECT 1 FROM app.applications a WHERE a.tenant_id=$1 AND a.id=$2 AND a.icon_file_id=f.id AND a.deleted_at IS NULL AND a.status='active')
 OR (c.onboarding_enabled AND EXISTS(SELECT 1 FROM app.application_onboarding_revision_assets x WHERE x.tenant_id=$1 AND x.app_id=$2 AND x.revision_id=c.published_revision_id AND x.file_id=f.id)))`, app.TenantID, app.ID, fileID).Scan(
		&asset.Provider, &asset.Bucket, &asset.ObjectKey, &mediaType, &asset.SizeBytes, &asset.SHA256, &asset.PublishedVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return StartupAsset{}, nil, ErrAppNotFound
	}
	if err != nil || mediaType == nil {
		return StartupAsset{}, nil, err
	}
	asset.MediaType = strings.ToLower(strings.TrimSpace(*mediaType))
	reader, err := s.objects.Open(ctx, storagedomain.ObjectRef{TenantID: app.TenantID, Provider: asset.Provider, Bucket: asset.Bucket, Key: asset.ObjectKey})
	if errors.Is(err, storagedomain.ErrObjectNotFound) {
		return StartupAsset{}, nil, ErrAppNotFound
	}
	return asset, reader, err
}

func StartupAssetDigest(asset StartupAsset) []byte {
	if len(asset.SHA256) > 0 {
		return asset.SHA256
	}
	digest := sha256.Sum256([]byte(asset.FileID.String()))
	return digest[:]
}
