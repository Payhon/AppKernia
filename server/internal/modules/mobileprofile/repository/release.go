package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type releaseQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const releaseColumns = `id,tenant_id,app_id,package_type,version,minimum_native_version,package_file_id,external_url,
create_env,is_silently,is_mandatory,ever_published_at,last_published_at,unpublished_at,lock_version,created_at,updated_at`

func loadRelease(ctx context.Context, query releaseQuerier, tenantID, appID, id uuid.UUID) (domain.Release, error) {
	var out domain.Release
	err := query.QueryRow(ctx, `SELECT `+releaseColumns+` FROM sys.mobile_releases
WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND deleted_at IS NULL`, id, tenantID, appID).Scan(
		&out.ID, &out.TenantID, &out.AppID, &out.PackageType, &out.Version, &out.MinimumNativeVersion,
		&out.PackageFileID, &out.ExternalURL, &out.CreateEnv, &out.IsSilently, &out.IsMandatory,
		&out.EverPublishedAt, &out.LastPublishedAt, &out.UnpublishedAt, &out.LockVersion, &out.CreatedAt, &out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if err != nil {
		return domain.Release{}, err
	}
	out.Platforms, out.PublishedPlatforms, out.StoreListingIDs = []string{}, []string{}, []uuid.UUID{}
	out.Titles, out.Contents, out.ReleaseNotes = map[string]string{}, map[string]string{}, map[string]string{}
	rows, err := query.Query(ctx, `SELECT platform FROM sys.mobile_release_targets WHERE release_id=$1 ORDER BY platform`, id)
	if err != nil {
		return domain.Release{}, err
	}
	for rows.Next() {
		var platform string
		if err = rows.Scan(&platform); err != nil {
			rows.Close()
			return domain.Release{}, err
		}
		out.Platforms = append(out.Platforms, platform)
	}
	rows.Close()
	rows, err = query.Query(ctx, `SELECT locale,title,contents FROM sys.mobile_release_translations WHERE release_id=$1 ORDER BY locale`, id)
	if err != nil {
		return domain.Release{}, err
	}
	for rows.Next() {
		var locale, title, contents string
		if err = rows.Scan(&locale, &title, &contents); err != nil {
			rows.Close()
			return domain.Release{}, err
		}
		out.Titles[locale], out.Contents[locale], out.ReleaseNotes[locale] = title, contents, contents
	}
	rows.Close()
	rows, err = query.Query(ctx, `SELECT store_listing_id FROM sys.mobile_release_store_listings WHERE release_id=$1 ORDER BY store_listing_id`, id)
	if err != nil {
		return domain.Release{}, err
	}
	for rows.Next() {
		var storeID uuid.UUID
		if err = rows.Scan(&storeID); err != nil {
			rows.Close()
			return domain.Release{}, err
		}
		out.StoreListingIDs = append(out.StoreListingIDs, storeID)
	}
	rows.Close()
	rows, err = query.Query(ctx, `SELECT platform FROM sys.mobile_release_publications WHERE release_id=$1 ORDER BY platform`, id)
	if err != nil {
		return domain.Release{}, err
	}
	for rows.Next() {
		var platform string
		if err = rows.Scan(&platform); err != nil {
			rows.Close()
			return domain.Release{}, err
		}
		out.PublishedPlatforms = append(out.PublishedPlatforms, platform)
	}
	rows.Close()
	switch {
	case out.EverPublishedAt == nil:
		out.PublishStatus = "draft"
	case len(out.PublishedPlatforms) == 0:
		out.PublishStatus = "offline"
	case len(out.PublishedPlatforms) == len(out.Platforms):
		out.PublishStatus = "online"
	default:
		out.PublishStatus = "partial"
	}
	out.Active = len(out.PublishedPlatforms) > 0
	out.CurrentVersion = out.Version
	out.MinimumVersion = "0.0.0"
	if out.MinimumNativeVersion != nil {
		out.MinimumVersion = *out.MinimumNativeVersion
	}
	if len(out.Platforms) > 0 {
		out.Platform = out.Platforms[0]
	}
	out.UpgradeURL = out.ExternalURL
	return out, nil
}

func (repository *Postgres) appTenant(ctx context.Context, appID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := repository.pool.QueryRow(ctx, `SELECT tenant_id FROM app.applications WHERE id=$1 AND deleted_at IS NULL`, appID).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.ErrReleaseNotFound
	}
	return tenantID, err
}

func (repository *Postgres) ActiveRelease(ctx context.Context, appID uuid.UUID, platform string) (domain.Release, error) {
	return repository.ActivePackageRelease(ctx, appID, "native_app", platform)
}

func (repository *Postgres) ActivePackageRelease(ctx context.Context, appID uuid.UUID, packageType, platform string) (domain.Release, error) {
	tenantID, err := repository.appTenant(ctx, appID)
	if err != nil {
		return domain.Release{}, err
	}
	var releaseID uuid.UUID
	err = repository.pool.QueryRow(ctx, `SELECT release_id FROM sys.mobile_release_publications
WHERE app_id=$1 AND package_type=$2 AND platform=$3`, appID, packageType, platform).Scan(&releaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if err != nil {
		return domain.Release{}, err
	}
	return loadRelease(ctx, repository.pool, tenantID, appID, releaseID)
}

func (repository *Postgres) ListReleases(ctx context.Context, appID uuid.UUID) ([]domain.Release, error) {
	tenantID, err := repository.appTenant(ctx, appID)
	if err != nil {
		return nil, err
	}
	page, err := repository.ListReleasePage(ctx, tenantID, appID, domain.ReleaseFilter{Page: 1, PageSize: 100})
	return page.Items, err
}

func (repository *Postgres) ListReleasePage(ctx context.Context, tenantID, appID uuid.UUID, filter domain.ReleaseFilter) (domain.ReleasePage, error) {
	where := `r.tenant_id=$1 AND r.app_id=$2 AND r.deleted_at IS NULL
AND ($3='' OR r.package_type=$3)
AND ($4='' OR EXISTS(SELECT 1 FROM sys.mobile_release_targets t WHERE t.release_id=r.id AND t.platform=$4))
AND ($5='' OR
    ($5='draft' AND r.ever_published_at IS NULL) OR
    ($5='offline' AND r.ever_published_at IS NOT NULL AND NOT EXISTS(SELECT 1 FROM sys.mobile_release_publications p WHERE p.release_id=r.id)) OR
    ($5='online' AND (SELECT count(*) FROM sys.mobile_release_publications p WHERE p.release_id=r.id)=(SELECT count(*) FROM sys.mobile_release_targets t WHERE t.release_id=r.id)) OR
    ($5='partial' AND (SELECT count(*) FROM sys.mobile_release_publications p WHERE p.release_id=r.id)>0 AND (SELECT count(*) FROM sys.mobile_release_publications p WHERE p.release_id=r.id)<(SELECT count(*) FROM sys.mobile_release_targets t WHERE t.release_id=r.id)))
AND ($6='' OR r.version ILIKE '%'||$6||'%' OR EXISTS(SELECT 1 FROM sys.mobile_release_translations x WHERE x.release_id=r.id AND (x.title ILIKE '%'||$6||'%' OR x.contents ILIKE '%'||$6||'%')))`
	var total int64
	if err := repository.pool.QueryRow(ctx, `SELECT count(*) FROM sys.mobile_releases r WHERE `+where, tenantID, appID, filter.PackageType, filter.Platform, filter.PublishStatus, filter.Query).Scan(&total); err != nil {
		return domain.ReleasePage{}, err
	}
	rows, err := repository.pool.Query(ctx, `SELECT r.id FROM sys.mobile_releases r WHERE `+where+` ORDER BY r.created_at DESC,r.id DESC LIMIT $7 OFFSET $8`, tenantID, appID, filter.PackageType, filter.Platform, filter.PublishStatus, filter.Query, filter.PageSize, (filter.Page-1)*filter.PageSize)
	if err != nil {
		return domain.ReleasePage{}, err
	}
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return domain.ReleasePage{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	items := make([]domain.Release, 0, len(ids))
	for _, id := range ids {
		item, loadErr := loadRelease(ctx, repository.pool, tenantID, appID, id)
		if loadErr != nil {
			return domain.ReleasePage{}, loadErr
		}
		items = append(items, item)
	}
	return domain.ReleasePage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

func (repository *Postgres) GetRelease(ctx context.Context, tenantID, appID, id uuid.UUID) (domain.Release, error) {
	return loadRelease(ctx, repository.pool, tenantID, appID, id)
}

func compatibilityInput(input domain.Release) domain.Release {
	if input.PackageType == "" {
		input.PackageType = "native_app"
	}
	if len(input.Platforms) == 0 && input.Platform != "" {
		input.Platforms = []string{input.Platform}
	}
	if input.Version == "" {
		input.Version = input.CurrentVersion
	}
	if input.MinimumNativeVersion == nil && input.MinimumVersion != "" {
		value := input.MinimumVersion
		input.MinimumNativeVersion = &value
	}
	if input.ExternalURL == nil {
		input.ExternalURL = input.UpgradeURL
	}
	if len(input.Contents) == 0 {
		input.Contents = input.ReleaseNotes
	}
	if len(input.Titles) == 0 {
		input.Titles = map[string]string{"zh-CN": input.Version, "en-US": input.Version}
	}
	if input.CreateEnv == "" {
		input.CreateEnv = "upgrade_center"
	}
	return input
}

func (repository *Postgres) CreateRelease(ctx context.Context, appID uuid.UUID, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	tenantID, err := repository.appTenant(ctx, appID)
	if err != nil {
		return domain.Release{}, err
	}
	input = compatibilityInput(input)
	created, err := repository.CreateDraft(ctx, tenantID, appID, input, actor, requestID)
	if err != nil || !input.Active {
		return created, err
	}
	return repository.Publish(ctx, tenantID, appID, created.ID, created.LockVersion, actor, requestID)
}

func (repository *Postgres) UpdateRelease(ctx context.Context, appID uuid.UUID, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	tenantID, err := repository.appTenant(ctx, appID)
	if err != nil {
		return domain.Release{}, err
	}
	input = compatibilityInput(input)
	updated, err := repository.UpdateDraft(ctx, tenantID, appID, input, actor, requestID)
	if errors.Is(err, domain.ErrReleaseFrozen) {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if err != nil || !input.Active {
		return updated, err
	}
	return repository.Publish(ctx, tenantID, appID, updated.ID, updated.LockVersion, actor, requestID)
}

func (repository *Postgres) CreateDraft(ctx context.Context, tenantID, appID uuid.UUID, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	input = compatibilityInput(input)
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = ensureReleaseApp(ctx, tx, tenantID, appID, false); err != nil {
		return domain.Release{}, err
	}
	if input.PackageFileID != nil {
		if err = validateReleaseFile(ctx, tx, tenantID, input); err != nil {
			return domain.Release{}, err
		}
	}
	notes, _ := json.Marshal(input.Contents)
	minimum := "0.0.0"
	if input.MinimumNativeVersion != nil {
		minimum = *input.MinimumNativeVersion
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO sys.mobile_releases(
tenant_id,app_id,platform,current_version,minimum_version,upgrade_url,release_notes,active,package_type,version,
minimum_native_version,package_file_id,external_url,create_env,is_silently,is_mandatory,created_by)
VALUES($1,$2,$3,$4,$5,$6,$7,false,$8,$4,$9,$10,$6,$11,$12,$13,$14) RETURNING id`, tenantID, appID,
		input.Platforms[0], input.Version, minimum, input.ExternalURL, notes, input.PackageType, input.MinimumNativeVersion,
		input.PackageFileID, input.CreateEnv, input.IsSilently, input.IsMandatory, actor).Scan(&id)
	if err != nil {
		return domain.Release{}, err
	}
	if err = replaceReleaseRelations(ctx, tx, tenantID, appID, id, input); err != nil {
		return domain.Release{}, err
	}
	if err = auditRelease(ctx, tx, tenantID, appID, actor, requestID, "mobile.release.create", "mobile.release.create", id, nil, input); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Release{}, err
	}
	return loadRelease(ctx, repository.pool, tenantID, appID, id)
}

func (repository *Postgres) UpdateDraft(ctx context.Context, tenantID, appID uuid.UUID, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	input = compatibilityInput(input)
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	existing, err := loadRelease(ctx, tx, tenantID, appID, input.ID)
	if err != nil {
		return domain.Release{}, err
	}
	if existing.EverPublishedAt != nil {
		return domain.Release{}, domain.ErrReleaseFrozen
	}
	if existing.LockVersion != input.LockVersion {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if input.PackageFileID != nil {
		if err = validateReleaseFile(ctx, tx, tenantID, input); err != nil {
			return domain.Release{}, err
		}
	}
	notes, _ := json.Marshal(input.Contents)
	minimum := "0.0.0"
	if input.MinimumNativeVersion != nil {
		minimum = *input.MinimumNativeVersion
	}
	command, err := tx.Exec(ctx, `UPDATE sys.mobile_releases SET platform=$4,current_version=$5,minimum_version=$6,upgrade_url=$7,
release_notes=$8,package_type=$9,version=$5,minimum_native_version=$10,package_file_id=$11,external_url=$7,
create_env=$12,is_silently=$13,is_mandatory=$14,lock_version=lock_version+1
WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND lock_version=$15 AND ever_published_at IS NULL AND deleted_at IS NULL`,
		input.ID, tenantID, appID, input.Platforms[0], input.Version, minimum, input.ExternalURL, notes, input.PackageType,
		input.MinimumNativeVersion, input.PackageFileID, input.CreateEnv, input.IsSilently, input.IsMandatory, input.LockVersion)
	if err != nil {
		return domain.Release{}, err
	}
	if command.RowsAffected() != 1 {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if err = replaceReleaseRelations(ctx, tx, tenantID, appID, input.ID, input); err != nil {
		return domain.Release{}, err
	}
	if err = auditRelease(ctx, tx, tenantID, appID, actor, requestID, "mobile.release.update", "mobile.release.update", input.ID, existing, input); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Release{}, err
	}
	return loadRelease(ctx, repository.pool, tenantID, appID, input.ID)
}

func (repository *Postgres) Publish(ctx context.Context, tenantID, appID, id uuid.UUID, lockVersion int32, actor uuid.UUID, requestID string) (domain.Release, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = ensureReleaseApp(ctx, tx, tenantID, appID, true); err != nil {
		return domain.Release{}, err
	}
	var currentLock int32
	if err = tx.QueryRow(ctx, `SELECT lock_version FROM sys.mobile_releases WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND deleted_at IS NULL FOR UPDATE`, id, tenantID, appID).Scan(&currentLock); errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if err != nil {
		return domain.Release{}, err
	}
	if currentLock != lockVersion {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	release, err := loadRelease(ctx, tx, tenantID, appID, id)
	if err != nil {
		return domain.Release{}, err
	}
	if (release.PackageFileID == nil) == (release.ExternalURL == nil) {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if release.PackageFileID != nil {
		if err = validateReleaseFile(ctx, tx, tenantID, release); err != nil {
			return domain.Release{}, err
		}
	}
	if strings.TrimSpace(release.Titles["zh-CN"]) == "" || strings.TrimSpace(release.Titles["en-US"]) == "" || strings.TrimSpace(release.Contents["zh-CN"]) == "" || strings.TrimSpace(release.Contents["en-US"]) == "" {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	newVersion, ok := parseVersion(release.Version)
	if !ok {
		return domain.Release{}, domain.ErrReleaseVersionNotIncreasing
	}
	oldReleaseIDs := make(map[uuid.UUID]struct{})
	for _, platform := range release.Platforms {
		var currentID uuid.UUID
		var currentVersion string
		err = tx.QueryRow(ctx, `SELECT p.release_id,r.version FROM sys.mobile_release_publications p JOIN sys.mobile_releases r ON r.id=p.release_id
WHERE p.app_id=$1 AND p.package_type=$2 AND p.platform=$3 FOR UPDATE`, appID, release.PackageType, platform).Scan(&currentID, &currentVersion)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return domain.Release{}, err
		}
		if err == nil && currentID != id {
			oldReleaseIDs[currentID] = struct{}{}
			current, valid := parseVersion(currentVersion)
			if !valid || compareVersion(newVersion, current) <= 0 {
				return domain.Release{}, domain.ErrReleaseVersionNotIncreasing
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO sys.mobile_release_publications(tenant_id,app_id,package_type,platform,release_id,published_by,published_at)
VALUES($1,$2,$3,$4,$5,$6,now()) ON CONFLICT(app_id,package_type,platform) DO UPDATE SET release_id=EXCLUDED.release_id,published_by=EXCLUDED.published_by,published_at=now()`, tenantID, appID, release.PackageType, platform, id, actor); err != nil {
			return domain.Release{}, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE sys.mobile_releases SET active=true,ever_published_at=COALESCE(ever_published_at,now()),last_published_at=now(),unpublished_at=NULL,lock_version=lock_version+1 WHERE id=$1`, id); err != nil {
		return domain.Release{}, err
	}
	for oldID := range oldReleaseIDs {
		if _, err = tx.Exec(ctx, `UPDATE sys.mobile_releases r SET active=EXISTS(SELECT 1 FROM sys.mobile_release_publications p WHERE p.release_id=r.id) WHERE r.id=$1`, oldID); err != nil {
			return domain.Release{}, err
		}
	}
	if err = auditRelease(ctx, tx, tenantID, appID, actor, requestID, "mobile.release.publish", "mobile.release.publish", id, release, map[string]any{"platforms": release.Platforms}); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		if isSerializationFailure(err) {
			return domain.Release{}, domain.ErrReleaseConflict
		}
		return domain.Release{}, err
	}
	return loadRelease(ctx, repository.pool, tenantID, appID, id)
}

func (repository *Postgres) Unpublish(ctx context.Context, tenantID, appID, id uuid.UUID, lockVersion int32, actor uuid.UUID, requestID string) (domain.Release, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	existing, err := loadRelease(ctx, tx, tenantID, appID, id)
	if err != nil {
		return domain.Release{}, err
	}
	if existing.EverPublishedAt == nil || existing.LockVersion != lockVersion {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM sys.mobile_release_publications WHERE release_id=$1`, id); err != nil {
		return domain.Release{}, err
	}
	command, err := tx.Exec(ctx, `UPDATE sys.mobile_releases SET active=false,unpublished_at=now(),lock_version=lock_version+1 WHERE id=$1 AND lock_version=$2`, id, lockVersion)
	if err != nil || command.RowsAffected() != 1 {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if err = auditRelease(ctx, tx, tenantID, appID, actor, requestID, "mobile.release.unpublish", "mobile.release.publish", id, existing, map[string]any{"active": false}); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Release{}, err
	}
	return loadRelease(ctx, repository.pool, tenantID, appID, id)
}

func (repository *Postgres) Delete(ctx context.Context, tenantID, appID uuid.UUID, ids []uuid.UUID, actor uuid.UUID, requestID string) error {
	requestedCount := len(ids)
	ids = uniqueIDs(ids)
	if len(ids) < 1 || len(ids) > 100 || len(ids) != requestedCount {
		return domain.ErrReleaseConflict
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM sys.mobile_releases WHERE tenant_id=$1 AND app_id=$2 AND id=ANY($3::uuid[]) AND deleted_at IS NULL AND ever_published_at IS NULL`, tenantID, appID, ids).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return domain.ErrReleaseDeleteForbidden
	}
	if _, err = tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='mobile' AND entity_type='mobile_release' AND entity_id=ANY($2::uuid[])`, tenantID, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM sys.mobile_releases WHERE tenant_id=$1 AND app_id=$2 AND id=ANY($3::uuid[]) AND ever_published_at IS NULL`, tenantID, appID, ids); err != nil {
		return err
	}
	for _, id := range ids {
		if err = auditRelease(ctx, tx, tenantID, appID, actor, requestID, "mobile.release.delete", "mobile.release.delete", id, nil, map[string]bool{"deleted": true}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (repository *Postgres) PublishedPackageFile(ctx context.Context, appID, releaseID, fileID uuid.UUID) (domain.PackageFile, error) {
	var file domain.PackageFile
	err := repository.pool.QueryRow(ctx, `SELECT f.id,f.tenant_id,f.original_name,COALESCE(f.media_type,'application/octet-stream'),f.size_bytes,f.provider,f.bucket_name,f.object_key
FROM sys.mobile_releases r
JOIN storage.files f ON f.tenant_id=r.tenant_id AND f.id=r.package_file_id
WHERE r.app_id=$1 AND r.id=$2 AND f.id=$3 AND r.deleted_at IS NULL AND f.deleted_at IS NULL
  AND f.status='ready' AND f.scan_status IN ('clean','skipped')
  AND EXISTS(SELECT 1 FROM sys.mobile_release_publications p WHERE p.release_id=r.id)`, appID, releaseID, fileID).Scan(
		&file.ID, &file.TenantID, &file.OriginalName, &file.MediaType, &file.SizeBytes, &file.Provider, &file.Bucket, &file.ObjectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PackageFile{}, domain.ErrReleaseNotFound
	}
	return file, err
}

// PackageFile returns a tenant-scoped, scanned internal package reference for
// pre-publication archive inspection. Object keys remain inside the backend.
func (repository *Postgres) PackageFile(ctx context.Context, tenantID, fileID uuid.UUID) (domain.PackageFile, error) {
	var file domain.PackageFile
	err := repository.pool.QueryRow(ctx, `SELECT id,tenant_id,original_name,COALESCE(media_type,'application/octet-stream'),size_bytes,provider,bucket_name,object_key
FROM storage.files
WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL AND status='ready' AND scan_status IN ('clean','skipped')`, tenantID, fileID).Scan(
		&file.ID, &file.TenantID, &file.OriginalName, &file.MediaType, &file.SizeBytes, &file.Provider, &file.Bucket, &file.ObjectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PackageFile{}, domain.ErrReleaseFileInvalid
	}
	return file, err
}

func ensureReleaseApp(ctx context.Context, query releaseQuerier, tenantID, appID uuid.UUID, requireManifest bool) error {
	var appid *string
	var status string
	err := query.QueryRow(ctx, `SELECT appid::text,status FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, appID).Scan(&appid, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrReleaseNotFound
	}
	if err != nil {
		return err
	}
	if requireManifest && (appid == nil || status != "active") {
		return domain.ErrReleaseConflict
	}
	return nil
}

func replaceReleaseRelations(ctx context.Context, tx pgx.Tx, tenantID, appID, releaseID uuid.UUID, input domain.Release) error {
	if _, err := tx.Exec(ctx, `DELETE FROM sys.mobile_release_targets WHERE release_id=$1`, releaseID); err != nil {
		return err
	}
	for _, platform := range input.Platforms {
		if _, err := tx.Exec(ctx, `INSERT INTO sys.mobile_release_targets(tenant_id,app_id,release_id,package_type,platform) VALUES($1,$2,$3,$4,$5)`, tenantID, appID, releaseID, input.PackageType, platform); err != nil {
			return domain.ErrReleaseConflict
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sys.mobile_release_translations WHERE release_id=$1`, releaseID); err != nil {
		return err
	}
	for _, locale := range []string{"zh-CN", "en-US"} {
		if _, err := tx.Exec(ctx, `INSERT INTO sys.mobile_release_translations(release_id,locale,title,contents) VALUES($1,$2,$3,$4)`, releaseID, locale, input.Titles[locale], input.Contents[locale]); err != nil {
			return domain.ErrReleaseConflict
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sys.mobile_release_store_listings WHERE release_id=$1`, releaseID); err != nil {
		return err
	}
	for _, storeID := range uniqueIDs(input.StoreListingIDs) {
		command, err := tx.Exec(ctx, `INSERT INTO sys.mobile_release_store_listings(tenant_id,app_id,release_id,store_listing_id)
SELECT $3,$4,$1,id FROM app.application_store_listings WHERE id=$2 AND tenant_id=$3 AND app_id=$4 AND enabled`, releaseID, storeID, tenantID, appID)
		if err != nil || command.RowsAffected() != 1 {
			return domain.ErrReleaseConflict
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND module_code='mobile' AND entity_type='mobile_release' AND entity_id=$2`, tenantID, releaseID); err != nil {
		return err
	}
	if input.PackageFileID != nil {
		_, err := tx.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name)
VALUES($1,$2,'mobile','mobile_release',$3,'package_file')`, input.PackageFileID, tenantID, releaseID)
		return err
	}
	return nil
}

func validateReleaseFile(ctx context.Context, query releaseQuerier, tenantID uuid.UUID, release domain.Release) error {
	if release.PackageFileID == nil {
		return nil
	}
	var extension, mediaType, status, scanStatus string
	var archiveType *string
	err := query.QueryRow(ctx, `SELECT lower(extension),lower(COALESCE(media_type,'')),status,scan_status,metadata->>'archive_type'
FROM storage.files WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, release.PackageFileID).Scan(&extension, &mediaType, &status, &scanStatus, &archiveType)
	if err != nil || status != "ready" || (scanStatus != "clean" && scanStatus != "skipped") {
		return domain.ErrReleaseFileInvalid
	}
	expected := ""
	if release.PackageType == "wgt" {
		expected = "wgt"
	} else if len(release.Platforms) == 1 {
		expected = map[string]string{"android": "apk", "ios": "ipa", "harmony": "hap"}[release.Platforms[0]]
	}
	if extension != expected || archiveType != nil && *archiveType != "" && !strings.EqualFold(*archiveType, expected) {
		return domain.ErrReleaseFileInvalid
	}
	allowedMedia := map[string]bool{"application/zip": true, "application/octet-stream": true, "application/vnd.android.package-archive": true, "application/x-zip-compressed": true}
	if !allowedMedia[mediaType] {
		return domain.ErrReleaseFileInvalid
	}
	return nil
}

func auditRelease(ctx context.Context, query releaseQuerier, tenantID, appID, actor uuid.UUID, requestID, action, permission string, id uuid.UUID, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	_, err := query.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,before_data,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'mobile',$4,$5,'sys.mobile_release',$6,'POST','/admin-api/v1/apps/'||$7||'/mobile/releases',$8,$9,true)`, tenantID, actor, requestID, action, permission, id.String(), appID.String(), beforeJSON, afterJSON)
	return err
}

type version [3]uint64

func parseVersion(raw string) (version, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var out version
	for index, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return version{}, false
		}
		value, err := strconv.ParseUint(part, 10, 63)
		if err != nil {
			return version{}, false
		}
		out[index] = value
	}
	return out, true
}

func compareVersion(left, right version) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func uniqueIDs(values []uuid.UUID) []uuid.UUID {
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
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func isSerializationFailure(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "40001"
}
