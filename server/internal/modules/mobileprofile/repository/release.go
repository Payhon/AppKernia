package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func release(notes []byte, id uuid.UUID, platform, current, minimum string, url *string, active bool, lock int32, updated time.Time) (domain.Release, error) {
	result := domain.Release{ID: id, Platform: platform, CurrentVersion: current, MinimumVersion: minimum, UpgradeURL: url, Active: active, LockVersion: lock, UpdatedAt: updated}
	if err := json.Unmarshal(notes, &result.ReleaseNotes); err != nil {
		return domain.Release{}, err
	}
	return result, nil
}
func mapActive(row db.GetActiveMobileReleaseRow) (domain.Release, error) {
	return release(row.ReleaseNotes, row.ID, row.Platform, row.CurrentVersion, row.MinimumVersion, row.UpgradeUrl, row.Active, row.LockVersion, row.UpdatedAt.Time)
}
func mapByID(row db.GetMobileReleaseByIDRow) (domain.Release, error) {
	return release(row.ReleaseNotes, row.ID, row.Platform, row.CurrentVersion, row.MinimumVersion, row.UpgradeUrl, row.Active, row.LockVersion, row.UpdatedAt.Time)
}
func mapList(row db.ListMobileReleasesRow) (domain.Release, error) {
	return release(row.ReleaseNotes, row.ID, row.Platform, row.CurrentVersion, row.MinimumVersion, row.UpgradeUrl, row.Active, row.LockVersion, row.UpdatedAt.Time)
}
func mapCreate(row db.CreateMobileReleaseRow) (domain.Release, error) {
	return release(row.ReleaseNotes, row.ID, row.Platform, row.CurrentVersion, row.MinimumVersion, row.UpgradeUrl, row.Active, row.LockVersion, row.UpdatedAt.Time)
}
func mapUpdate(row db.UpdateMobileReleaseRow) (domain.Release, error) {
	return release(row.ReleaseNotes, row.ID, row.Platform, row.CurrentVersion, row.MinimumVersion, row.UpgradeUrl, row.Active, row.LockVersion, row.UpdatedAt.Time)
}

func (repository *Postgres) ActiveRelease(ctx context.Context, platform string) (domain.Release, error) {
	row, err := db.New(repository.pool).GetActiveMobileRelease(ctx, platform)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if err != nil {
		return domain.Release{}, err
	}
	return mapActive(row)
}
func (repository *Postgres) ListReleases(ctx context.Context) ([]domain.Release, error) {
	rows, err := db.New(repository.pool).ListMobileReleases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Release, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapList(row)
		if mapErr != nil {
			return nil, mapErr
		}
		out = append(out, item)
	}
	return out, nil
}
func (repository *Postgres) CreateRelease(ctx context.Context, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	notes, err := json.Marshal(input.ReleaseNotes)
	if err != nil {
		return domain.Release{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	if input.Active {
		if err = queries.DeactivateMobileReleasesByPlatform(ctx, db.DeactivateMobileReleasesByPlatformParams{Platform: input.Platform}); err != nil {
			return domain.Release{}, err
		}
	}
	row, err := queries.CreateMobileRelease(ctx, db.CreateMobileReleaseParams{Platform: input.Platform, CurrentVersion: input.CurrentVersion, MinimumVersion: input.MinimumVersion, UpgradeUrl: input.UpgradeURL, ReleaseNotes: notes, Active: input.Active})
	if err != nil {
		return domain.Release{}, err
	}
	out, err := mapCreate(row)
	if err != nil {
		return domain.Release{}, err
	}
	after, _ := json.Marshal(out)
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,after_data,succeeded) VALUES($1,$2,'mobile','release.create','mobile.release.create','sys.mobile_release',$3,'POST','/admin-api/v1/mobile/releases',$4,true)`, actor, requestID, out.ID.String(), after); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Release{}, err
	}
	return out, nil
}
func (repository *Postgres) UpdateRelease(ctx context.Context, input domain.Release, actor uuid.UUID, requestID string) (domain.Release, error) {
	notes, err := json.Marshal(input.ReleaseNotes)
	if err != nil {
		return domain.Release{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Release{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	existingRow, err := queries.GetMobileReleaseByID(ctx, input.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if err != nil {
		return domain.Release{}, err
	}
	existing, err := mapByID(existingRow)
	if err != nil {
		return domain.Release{}, err
	}
	if existing.Platform != input.Platform {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if input.Active {
		if err = queries.DeactivateMobileReleasesByPlatform(ctx, db.DeactivateMobileReleasesByPlatformParams{Platform: input.Platform, ExceptID: &input.ID}); err != nil {
			return domain.Release{}, err
		}
	}
	row, err := queries.UpdateMobileRelease(ctx, db.UpdateMobileReleaseParams{CurrentVersion: input.CurrentVersion, MinimumVersion: input.MinimumVersion, UpgradeUrl: input.UpgradeURL, ReleaseNotes: notes, Active: input.Active, ID: input.ID, LockVersion: input.LockVersion})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrReleaseConflict
	}
	if err != nil {
		return domain.Release{}, fmt.Errorf("update mobile release: %w", err)
	}
	out, err := mapUpdate(row)
	if err != nil {
		return domain.Release{}, err
	}
	after, _ := json.Marshal(out)
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,before_data,after_data,succeeded) VALUES($1,$2,'mobile','release.update','mobile.release.update','sys.mobile_release',$3,'PATCH','/admin-api/v1/mobile/releases/{id}',$4,$5,true)`, actor, requestID, out.ID.String(), mustJSON(existing), after); err != nil {
		return domain.Release{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Release{}, err
	}
	return out, nil
}
func mustJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }
