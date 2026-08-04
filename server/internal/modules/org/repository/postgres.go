package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	orgdomain "github.com/appkernia/appkernia/server/internal/modules/org/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (repository *Postgres) ListUnits(ctx context.Context, tenantID uuid.UUID) ([]orgdomain.Unit, error) {
	rows, err := db.New(repository.pool).ListOrgUnits(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list organization units: %w", err)
	}
	result := make([]orgdomain.Unit, 0, len(rows))
	for _, row := range rows {
		result = append(result, orgdomain.Unit{ID: row.ID, ParentID: row.ParentID, Code: row.Code, Name: row.Name, UnitType: row.UnitType, Phone: stringValue(row.Phone), Email: anyString(row.Email), SortOrder: row.SortOrder, Status: row.Status, DirectMemberCount: row.DirectMemberCount, ChildCount: row.ChildCount, UpdatedAt: row.UpdatedAt.Time, Children: []orgdomain.Unit{}})
	}
	return result, nil
}

func (repository *Postgres) CreateUnit(ctx context.Context, principal orgdomain.Principal, input orgdomain.UnitInput) (orgdomain.Unit, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.Unit{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if input.ParentID != nil {
		exists, checkErr := queries.OrgUnitParentExists(ctx, db.OrgUnitParentExistsParams{TenantID: principal.TenantID, ParentID: *input.ParentID})
		if checkErr != nil {
			return orgdomain.Unit{}, fmt.Errorf("check parent unit: %w", checkErr)
		}
		if !exists {
			return orgdomain.Unit{}, orgdomain.ErrUnitNotFound
		}
	}
	row, err := queries.CreateOrgUnit(ctx, db.CreateOrgUnitParams{TenantID: principal.TenantID, ParentID: input.ParentID, Code: input.Code, Name: input.Name, UnitType: input.UnitType, Phone: pointer(input.Phone), Email: pointer(input.Email), SortOrder: input.SortOrder, Status: input.Status})
	if err != nil {
		return orgdomain.Unit{}, mapUnitWriteError(err)
	}
	unit := orgdomain.Unit{ID: row.ID, ParentID: row.ParentID, Code: row.Code, Name: row.Name, UnitType: row.UnitType, Phone: stringValue(row.Phone), Email: anyString(row.Email), SortOrder: row.SortOrder, Status: row.Status, UpdatedAt: row.UpdatedAt.Time, Children: []orgdomain.Unit{}}
	if err = audit(ctx, queries, principal, "org.unit.create", "org.unit", unit.ID, "POST", "/admin-api/v1/org/units", nil, unit); err != nil {
		return orgdomain.Unit{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return orgdomain.Unit{}, fmt.Errorf("commit create unit: %w", err)
	}
	return unit, nil
}

func (repository *Postgres) UpdateUnit(ctx context.Context, principal orgdomain.Principal, id uuid.UUID, input orgdomain.UnitInput) (orgdomain.Unit, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.Unit{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := queries.GetOrgUnitForUpdate(ctx, db.GetOrgUnitForUpdateParams{TenantID: principal.TenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return orgdomain.Unit{}, orgdomain.ErrUnitNotFound
	}
	if err != nil {
		return orgdomain.Unit{}, fmt.Errorf("lock unit: %w", err)
	}
	row, err := queries.UpdateOrgUnit(ctx, db.UpdateOrgUnitParams{Code: input.Code, Name: input.Name, UnitType: input.UnitType, Phone: pointer(input.Phone), Email: pointer(input.Email), SortOrder: input.SortOrder, Status: input.Status, TenantID: principal.TenantID, ID: id})
	if err != nil {
		return orgdomain.Unit{}, mapUnitWriteError(err)
	}
	unit := orgdomain.Unit{ID: row.ID, ParentID: row.ParentID, Code: row.Code, Name: row.Name, UnitType: row.UnitType, Phone: stringValue(row.Phone), Email: anyString(row.Email), SortOrder: row.SortOrder, Status: row.Status, UpdatedAt: row.UpdatedAt.Time, Children: []orgdomain.Unit{}}
	if err = audit(ctx, queries, principal, "org.unit.update", "org.unit", id, "PATCH", "/admin-api/v1/org/units/{id}", before, unit); err != nil {
		return orgdomain.Unit{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return orgdomain.Unit{}, fmt.Errorf("commit update unit: %w", err)
	}
	return unit, nil
}

func (repository *Postgres) MoveUnit(ctx context.Context, principal orgdomain.Principal, id uuid.UUID, input orgdomain.UnitMove) (orgdomain.Unit, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.Unit{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := queries.GetOrgUnitForUpdate(ctx, db.GetOrgUnitForUpdateParams{TenantID: principal.TenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return orgdomain.Unit{}, orgdomain.ErrUnitNotFound
	}
	if err != nil {
		return orgdomain.Unit{}, fmt.Errorf("lock unit for move: %w", err)
	}
	if input.ParentID != nil {
		var cycle bool
		err = tx.QueryRow(ctx, `WITH RECURSIVE d AS (SELECT id FROM org.units WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL UNION ALL SELECT c.id FROM org.units c JOIN d ON c.parent_id=d.id WHERE c.tenant_id=$1 AND c.deleted_at IS NULL) SELECT EXISTS(SELECT 1 FROM d WHERE id=$3)`, principal.TenantID, id, *input.ParentID).Scan(&cycle)
		if err != nil {
			return orgdomain.Unit{}, fmt.Errorf("check unit cycle: %w", err)
		}
		if cycle {
			return orgdomain.Unit{}, orgdomain.ErrUnitCycle
		}
		exists, checkErr := queries.OrgUnitParentExists(ctx, db.OrgUnitParentExistsParams{TenantID: principal.TenantID, ParentID: *input.ParentID})
		if checkErr != nil {
			return orgdomain.Unit{}, fmt.Errorf("check move parent: %w", checkErr)
		}
		if !exists {
			return orgdomain.Unit{}, orgdomain.ErrUnitNotFound
		}
	}
	row, err := queries.MoveOrgUnit(ctx, db.MoveOrgUnitParams{ParentID: input.ParentID, SortOrder: input.SortOrder, TenantID: principal.TenantID, ID: id})
	if err != nil {
		return orgdomain.Unit{}, mapUnitWriteError(err)
	}
	unit := orgdomain.Unit{ID: row.ID, ParentID: row.ParentID, Code: row.Code, Name: row.Name, UnitType: row.UnitType, Phone: stringValue(row.Phone), Email: anyString(row.Email), SortOrder: row.SortOrder, Status: row.Status, UpdatedAt: row.UpdatedAt.Time, Children: []orgdomain.Unit{}}
	if err = audit(ctx, queries, principal, "org.unit.move", "org.unit", id, "POST", "/admin-api/v1/org/units/{id}/move", before, unit); err != nil {
		return orgdomain.Unit{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return orgdomain.Unit{}, fmt.Errorf("commit move unit: %w", err)
	}
	return unit, nil
}

func (repository *Postgres) DeleteUnit(ctx context.Context, principal orgdomain.Principal, id uuid.UUID) (orgdomain.UnitOccupancy, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.UnitOccupancy{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := queries.GetOrgUnitForUpdate(ctx, db.GetOrgUnitForUpdateParams{TenantID: principal.TenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return orgdomain.UnitOccupancy{}, orgdomain.ErrUnitNotFound
	}
	if err != nil {
		return orgdomain.UnitOccupancy{}, fmt.Errorf("lock unit for delete: %w", err)
	}
	occupancy, err := queries.GetOrgUnitOccupancy(ctx, db.GetOrgUnitOccupancyParams{TenantID: principal.TenantID, ID: &id})
	if err != nil {
		return orgdomain.UnitOccupancy{}, fmt.Errorf("count unit occupancy: %w", err)
	}
	result := orgdomain.UnitOccupancy{ChildCount: occupancy.ChildCount, MemberCount: occupancy.MemberCount}
	if result.ChildCount > 0 || result.MemberCount > 0 {
		return result, orgdomain.ErrUnitOccupied
	}
	affected, err := queries.SoftDeleteOrgUnit(ctx, db.SoftDeleteOrgUnitParams{TenantID: principal.TenantID, ID: id})
	if err != nil {
		return result, fmt.Errorf("delete unit: %w", err)
	}
	if affected != 1 {
		return result, orgdomain.ErrUnitNotFound
	}
	if err = audit(ctx, queries, principal, "org.unit.delete", "org.unit", id, "DELETE", "/admin-api/v1/org/units/{id}", before, nil); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit delete unit: %w", err)
	}
	return result, nil
}

func (repository *Postgres) ListPositions(ctx context.Context, tenantID uuid.UUID, query, status string, unitID *uuid.UUID) ([]orgdomain.Position, error) {
	rows, err := db.New(repository.pool).ListOrgPositions(ctx, db.ListOrgPositionsParams{TenantID: tenantID, Query: query, Status: status, UnitID: unitID})
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	result := make([]orgdomain.Position, 0, len(rows))
	for _, row := range rows {
		result = append(result, orgdomain.Position{ID: row.ID, Code: row.Code, Name: row.Name, Description: stringValue(row.Description), SortOrder: row.SortOrder, Status: row.Status, MemberCount: row.MemberCount, UpdatedAt: row.UpdatedAt.Time})
	}
	return result, nil
}

func (repository *Postgres) CreatePosition(ctx context.Context, principal orgdomain.Principal, input orgdomain.PositionInput) (orgdomain.Position, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.Position{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := queries.CreateOrgPosition(ctx, db.CreateOrgPositionParams{TenantID: principal.TenantID, Code: input.Code, Name: input.Name, Description: pointer(input.Description), SortOrder: input.SortOrder, Status: input.Status})
	if err != nil {
		return orgdomain.Position{}, mapPositionWriteError(err)
	}
	position := orgdomain.Position{ID: row.ID, Code: row.Code, Name: row.Name, Description: stringValue(row.Description), SortOrder: row.SortOrder, Status: row.Status, UpdatedAt: row.UpdatedAt.Time}
	if err = audit(ctx, queries, principal, "org.position.create", "org.position", position.ID, "POST", "/admin-api/v1/org/positions", nil, position); err != nil {
		return orgdomain.Position{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return orgdomain.Position{}, fmt.Errorf("commit create position: %w", err)
	}
	return position, nil
}

func (repository *Postgres) UpdatePosition(ctx context.Context, principal orgdomain.Principal, id uuid.UUID, input orgdomain.PositionInput) (orgdomain.Position, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return orgdomain.Position{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := queries.GetOrgPositionForUpdate(ctx, db.GetOrgPositionForUpdateParams{TenantID: principal.TenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return orgdomain.Position{}, orgdomain.ErrPositionNotFound
	}
	if err != nil {
		return orgdomain.Position{}, fmt.Errorf("lock position: %w", err)
	}
	row, err := queries.UpdateOrgPosition(ctx, db.UpdateOrgPositionParams{Code: input.Code, Name: input.Name, Description: pointer(input.Description), SortOrder: input.SortOrder, Status: input.Status, TenantID: principal.TenantID, ID: id})
	if err != nil {
		return orgdomain.Position{}, mapPositionWriteError(err)
	}
	position := orgdomain.Position{ID: row.ID, Code: row.Code, Name: row.Name, Description: stringValue(row.Description), SortOrder: row.SortOrder, Status: row.Status, UpdatedAt: row.UpdatedAt.Time}
	if err = audit(ctx, queries, principal, "org.position.update", "org.position", id, "PATCH", "/admin-api/v1/org/positions/{id}", before, position); err != nil {
		return orgdomain.Position{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return orgdomain.Position{}, fmt.Errorf("commit update position: %w", err)
	}
	return position, nil
}

func (repository *Postgres) DeletePosition(ctx context.Context, principal orgdomain.Principal, id uuid.UUID) (int64, error) {
	tx, queries, err := repository.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := queries.GetOrgPositionForUpdate(ctx, db.GetOrgPositionForUpdateParams{TenantID: principal.TenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, orgdomain.ErrPositionNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock position for delete: %w", err)
	}
	count, err := queries.CountOrgPositionMembers(ctx, db.CountOrgPositionMembersParams{TenantID: principal.TenantID, ID: id})
	if err != nil {
		return 0, fmt.Errorf("count position members: %w", err)
	}
	if count > 0 {
		return count, orgdomain.ErrPositionOccupied
	}
	affected, err := queries.SoftDeleteOrgPosition(ctx, db.SoftDeleteOrgPositionParams{TenantID: principal.TenantID, ID: id})
	if err != nil {
		return count, fmt.Errorf("delete position: %w", err)
	}
	if affected != 1 {
		return count, orgdomain.ErrPositionNotFound
	}
	if err = audit(ctx, queries, principal, "org.position.delete", "org.position", id, "DELETE", "/admin-api/v1/org/positions/{id}", before, nil); err != nil {
		return count, err
	}
	if err = tx.Commit(ctx); err != nil {
		return count, fmt.Errorf("commit delete position: %w", err)
	}
	return count, nil
}

func (repository *Postgres) begin(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, nil, fmt.Errorf("begin organization transaction: %w", err)
	}
	return tx, db.New(tx), nil
}
func audit(ctx context.Context, queries *db.Queries, principal orgdomain.Principal, action, resourceType string, id uuid.UUID, method, path string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	resourceID := id.String()
	status := int32(200)
	userAgent := strings.TrimSpace(principal.UserAgent)
	return queries.InsertOrgOperationAudit(ctx, db.InsertOrgOperationAuditParams{TenantID: &principal.TenantID, UserID: &principal.UserID, SessionID: &principal.SessionID, RequestID: principal.RequestID, ActionName: action, ResourceType: &resourceType, ResourceID: &resourceID, HttpMethod: &method, RequestPath: &path, ResponseStatus: &status, ClientIp: principal.IPAddress, UserAgent: pointer(userAgent), BeforeData: beforeJSON, AfterData: afterJSON})
}
func mapUnitWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return orgdomain.ErrUnitCodeConflict
	}
	return fmt.Errorf("write organization unit: %w", err)
}
func mapPositionWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return orgdomain.ErrPositionConflict
	}
	return fmt.Errorf("write organization position: %w", err)
}
func pointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func anyString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return fmt.Sprint(value)
}
