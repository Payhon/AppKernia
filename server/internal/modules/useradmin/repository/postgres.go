package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	userdomain "github.com/appkernia/appkernia/server/internal/modules/useradmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (repository *Postgres) ListRoleOptions(ctx context.Context, tenantID uuid.UUID) ([]userdomain.Reference, error) {
	rows, err := repository.pool.Query(ctx, `SELECT id,code::text,name FROM iam.roles WHERE tenant_id=$1 AND status='active' AND deleted_at IS NULL ORDER BY sort_order,name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant role options: %w", err)
	}
	defer rows.Close()
	items := []userdomain.Reference{}
	for rows.Next() {
		var item userdomain.Reference
		if err = rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const userColumns = `
u.id, COALESCE(u.email::text, ''), COALESCE(u.username::text, ''),
COALESCE(tm.display_name, u.display_name), u.locale, u.time_zone,
CASE WHEN tm.status = 'suspended' THEN 'disabled' WHEN tm.status = 'invited' THEN 'pending' WHEN tm.status = 'left' THEN 'disabled' ELSE u.status END,
u.status, tm.status, u.is_system,
COALESCE((SELECT jsonb_agg(jsonb_build_object('id', r.id, 'code', r.code::text, 'name', r.name) ORDER BY r.sort_order, r.name) FROM iam.user_roles ur JOIN iam.roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id WHERE ur.tenant_id=tm.tenant_id AND ur.user_id=u.id AND r.deleted_at IS NULL), '[]'::jsonb),
COALESCE((SELECT jsonb_agg(jsonb_build_object('id', ou.id, 'code', ou.code::text, 'name', ou.name) ORDER BY uu.is_primary DESC, ou.sort_order, ou.name) FROM org.user_units uu JOIN org.units ou ON ou.tenant_id=uu.tenant_id AND ou.id=uu.unit_id WHERE uu.tenant_id=tm.tenant_id AND uu.user_id=u.id AND ou.deleted_at IS NULL), '[]'::jsonb),
COALESCE((SELECT jsonb_agg(jsonb_build_object('id', p.id, 'code', p.code::text, 'name', p.name) ORDER BY up.is_primary DESC, p.sort_order, p.name) FROM org.user_positions up JOIN org.positions p ON p.tenant_id=up.tenant_id AND p.id=up.position_id WHERE up.tenant_id=tm.tenant_id AND up.user_id=u.id AND p.deleted_at IS NULL), '[]'::jsonb),
(SELECT count(*) FROM iam.sessions s WHERE s.tenant_id=tm.tenant_id AND s.user_id=u.id AND s.status='active' AND s.revoked_at IS NULL AND s.absolute_expires_at > now()),
u.last_login_at, u.last_active_at, u.created_at, u.updated_at`

func (repository *Postgres) ListUsers(ctx context.Context, tenantID uuid.UUID, filters userdomain.Filters) (userdomain.Page, error) {
	args := []any{tenantID}
	where := []string{"tm.tenant_id=$1", "u.deleted_at IS NULL"}
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if filters.Query != "" {
		add("(u.email ILIKE '%%' || $%[1]d || '%%' OR u.display_name ILIKE '%%' || $%[1]d || '%%' OR COALESCE(u.username::text,'') ILIKE '%%' || $%[1]d || '%%')", filters.Query)
	}
	if filters.Status != "" {
		add("CASE WHEN tm.status='suspended' THEN 'disabled' WHEN tm.status='invited' THEN 'pending' WHEN tm.status='left' THEN 'disabled' ELSE u.status END = $%d", filters.Status)
	}
	if filters.UnitID != nil {
		add("EXISTS (SELECT 1 FROM org.user_units fuu WHERE fuu.tenant_id=tm.tenant_id AND fuu.user_id=u.id AND fuu.unit_id=$%d)", *filters.UnitID)
	}
	if filters.PositionID != nil {
		add("EXISTS (SELECT 1 FROM org.user_positions fup WHERE fup.tenant_id=tm.tenant_id AND fup.user_id=u.id AND fup.position_id=$%d)", *filters.PositionID)
	}
	if filters.RoleID != nil {
		add("EXISTS (SELECT 1 FROM iam.user_roles fur WHERE fur.tenant_id=tm.tenant_id AND fur.user_id=u.id AND fur.role_id=$%d)", *filters.RoleID)
	}
	if filters.CreatedFrom != nil {
		add("u.created_at >= $%d", *filters.CreatedFrom)
	}
	if filters.CreatedTo != nil {
		add("u.created_at < $%d", *filters.CreatedTo)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := repository.pool.QueryRow(ctx, "SELECT count(*) FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return userdomain.Page{}, fmt.Errorf("count tenant users: %w", err)
	}
	order := "u.created_at DESC, u.id"
	switch filters.Sort {
	case "created_asc":
		order = "u.created_at, u.id"
	case "name_asc":
		order = "COALESCE(tm.display_name,u.display_name), u.id"
	case "last_login_desc":
		order = "u.last_login_at DESC NULLS LAST, u.id"
	}
	args = append(args, filters.PageSize, (filters.Page-1)*filters.PageSize)
	query := fmt.Sprintf("SELECT %s FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d", userColumns, whereSQL, order, len(args)-1, len(args))
	rows, err := repository.pool.Query(ctx, query, args...)
	if err != nil {
		return userdomain.Page{}, fmt.Errorf("list tenant users: %w", err)
	}
	defer rows.Close()
	items := []userdomain.User{}
	for rows.Next() {
		item, scanErr := scanUser(rows)
		if scanErr != nil {
			return userdomain.Page{}, scanErr
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return userdomain.Page{}, fmt.Errorf("iterate tenant users: %w", err)
	}
	return userdomain.Page{Items: items, Total: total, Page: filters.Page, PageSize: filters.PageSize}, nil
}

func (repository *Postgres) GetUser(ctx context.Context, tenantID, id uuid.UUID) (userdomain.User, error) {
	return getUser(ctx, repository.pool, tenantID, id)
}

func (repository *Postgres) CreateUser(ctx context.Context, principal userdomain.Principal, input userdomain.CreateInput, passwordHash string) (userdomain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return userdomain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,locale,time_zone,status,metadata) VALUES($1,$2,$3,$4,'active','{}') RETURNING id`, input.Email, input.DisplayName, input.Locale, input.TimeZone).Scan(&id)
	if err != nil {
		return userdomain.User{}, mapWriteError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.user_credentials(user_id,password_hash,password_algorithm,force_password_change) VALUES($1,$2,'argon2id',true)`, id, passwordHash); err != nil {
		return userdomain.User{}, fmt.Errorf("create managed credential: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status,invited_by) VALUES($1,$2,'active',$3)`, principal.TenantID, id, principal.UserID); err != nil {
		return userdomain.User{}, fmt.Errorf("create tenant membership: %w", err)
	}
	after, err := getUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if err = insertAudit(ctx, tx, principal, "iam.user.create", "iam.user.create", id, "POST", "/admin-api/v1/users", nil, safeUser(after)); err != nil {
		return userdomain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userdomain.User{}, fmt.Errorf("commit user creation: %w", err)
	}
	return after, nil
}

func (repository *Postgres) UpdateUser(ctx context.Context, principal userdomain.Principal, id uuid.UUID, input userdomain.UpdateInput) (userdomain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return userdomain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := lockUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.users SET display_name=$1,locale=$2,time_zone=$3 WHERE id=$4 AND deleted_at IS NULL`, input.DisplayName, input.Locale, input.TimeZone, id); err != nil {
		return userdomain.User{}, fmt.Errorf("update managed user: %w", err)
	}
	after, err := getUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if err = insertAudit(ctx, tx, principal, "iam.user.update", "iam.user.update", id, "PATCH", "/admin-api/v1/users/{id}", safeUser(before), safeUser(after)); err != nil {
		return userdomain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userdomain.User{}, fmt.Errorf("commit user update: %w", err)
	}
	return after, nil
}

func (repository *Postgres) SetMemberStatus(ctx context.Context, principal userdomain.Principal, id uuid.UUID, status string) (userdomain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return userdomain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := lockUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if status == "suspended" {
		var count int64
		err = tx.QueryRow(ctx, `SELECT count(*) FROM iam.tenant_members tm JOIN iam.user_roles ur ON ur.tenant_id=tm.tenant_id AND ur.user_id=tm.user_id JOIN iam.roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id WHERE tm.tenant_id=$1 AND tm.status='active' AND r.code='super-admin' AND r.status='active' AND r.deleted_at IS NULL`, principal.TenantID).Scan(&count)
		if err != nil {
			return userdomain.User{}, fmt.Errorf("count active tenant administrators: %w", err)
		}
		isAdmin := false
		for _, role := range before.Roles {
			if role.Code == "super-admin" {
				isAdmin = true
			}
		}
		if isAdmin && count <= 1 {
			return userdomain.User{}, userdomain.ErrLastAdmin
		}
	}
	result, err := tx.Exec(ctx, `UPDATE iam.tenant_members SET status=$1,left_at=NULL WHERE tenant_id=$2 AND user_id=$3`, status, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, fmt.Errorf("set membership status: %w", err)
	}
	if result.RowsAffected() != 1 {
		return userdomain.User{}, userdomain.ErrNotFound
	}
	if status == "suspended" {
		if _, err = tx.Exec(ctx, `UPDATE iam.refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE revoked_at IS NULL AND session_id IN (SELECT id FROM iam.sessions WHERE tenant_id=$1 AND user_id=$2)`, principal.TenantID, id); err != nil {
			return userdomain.User{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=COALESCE(revoked_at,now()),revoke_reason='tenant_member_suspended',access_token_version=access_token_version+1 WHERE tenant_id=$1 AND user_id=$2 AND status='active' AND revoked_at IS NULL`, principal.TenantID, id); err != nil {
			return userdomain.User{}, err
		}
	}
	after, err := getUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	action, permission, path := "iam.user.disable", "iam.user.disable", "/admin-api/v1/users/{id}/disable"
	if status == "active" {
		action, permission, path = "iam.user.enable", "iam.user.enable", "/admin-api/v1/users/{id}/enable"
	}
	if err = insertAudit(ctx, tx, principal, action, permission, id, "POST", path, safeUser(before), safeUser(after)); err != nil {
		return userdomain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userdomain.User{}, fmt.Errorf("commit membership status: %w", err)
	}
	return after, nil
}

func (repository *Postgres) UnlockUser(ctx context.Context, principal userdomain.Principal, id uuid.UUID) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = lockUser(ctx, tx, principal.TenantID, id); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE iam.user_credentials SET failed_attempts=0,locked_until=NULL WHERE user_id=$1`, id)
	if err != nil {
		return fmt.Errorf("unlock managed user: %w", err)
	}
	if result.RowsAffected() != 1 {
		return userdomain.ErrNotFound
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.users SET status='active' WHERE id=$1 AND status='locked'`, id); err != nil {
		return err
	}
	if err = insertAudit(ctx, tx, principal, "iam.user.unlock", "iam.user.unlock", id, "POST", "/admin-api/v1/users/{id}/unlock", nil, map[string]any{"unlocked": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Postgres) ResetPassword(ctx context.Context, principal userdomain.Principal, id uuid.UUID, passwordHash string) (int64, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = lockUser(ctx, tx, principal.TenantID, id); err != nil {
		return 0, err
	}
	var oldHash string
	var oldVersion int32
	if err = tx.QueryRow(ctx, `SELECT password_hash,password_version FROM iam.user_credentials WHERE user_id=$1 FOR UPDATE`, id).Scan(&oldHash, &oldVersion); errors.Is(err, pgx.ErrNoRows) {
		return 0, userdomain.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock managed credential: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.password_history(user_id,password_hash,password_algorithm) VALUES($1,$2,'argon2id')`, id, oldHash); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.user_credentials SET password_hash=$1,password_version=password_version+1,password_changed_at=now(),force_password_change=true,failed_attempts=0,locked_until=NULL WHERE user_id=$2`, passwordHash, id); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE revoked_at IS NULL AND session_id IN (SELECT id FROM iam.sessions WHERE user_id=$1)`, id); err != nil {
		return 0, err
	}
	result, err := tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=COALESCE(revoked_at,now()),revoke_reason='admin_password_reset',access_token_version=access_token_version+1 WHERE user_id=$1 AND status='active' AND revoked_at IS NULL`, id)
	if err != nil {
		return 0, err
	}
	revoked := result.RowsAffected()
	if err = insertAudit(ctx, tx, principal, "iam.user.reset_password", "iam.user.reset_password", id, "POST", "/admin-api/v1/users/{id}/reset-password", map[string]any{"password_version": oldVersion}, map[string]any{"password_version": oldVersion + 1, "sessions_revoked": revoked, "force_password_change": true}); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return revoked, nil
}

func (repository *Postgres) ReplaceRoles(ctx context.Context, principal userdomain.Principal, id uuid.UUID, roleIDs []uuid.UUID) (userdomain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return userdomain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := lockUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	var count int64
	if len(roleIDs) > 0 {
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM iam.roles WHERE tenant_id=$1 AND id=ANY($2::uuid[]) AND status='active' AND deleted_at IS NULL`, principal.TenantID, roleIDs).Scan(&count); err != nil {
			return userdomain.User{}, err
		}
		if count != int64(len(roleIDs)) {
			return userdomain.User{}, userdomain.ErrRoleInvalid
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.user_roles WHERE tenant_id=$1 AND user_id=$2`, principal.TenantID, id); err != nil {
		return userdomain.User{}, err
	}
	for _, roleID := range roleIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO iam.user_roles(tenant_id,user_id,role_id,granted_by) VALUES($1,$2,$3,$4)`, principal.TenantID, id, roleID, principal.UserID); err != nil {
			return userdomain.User{}, err
		}
	}
	after, err := getUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if err = insertAudit(ctx, tx, principal, "iam.user.assign_role", "iam.user.assign_role", id, "PUT", "/admin-api/v1/users/{id}/roles", map[string]any{"roles": before.Roles}, map[string]any{"roles": after.Roles}); err != nil {
		return userdomain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userdomain.User{}, err
	}
	return after, nil
}

func (repository *Postgres) ReplaceAssignments(ctx context.Context, principal userdomain.Principal, id uuid.UUID, input userdomain.AssignmentInput) (userdomain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return userdomain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := lockUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if err = validateTenantIDs(ctx, tx, "org.units", principal.TenantID, input.UnitIDs); err != nil {
		return userdomain.User{}, err
	}
	if err = validateTenantIDs(ctx, tx, "org.positions", principal.TenantID, input.PositionIDs); err != nil {
		return userdomain.User{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM org.user_positions WHERE tenant_id=$1 AND user_id=$2`, principal.TenantID, id); err != nil {
		return userdomain.User{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM org.user_units WHERE tenant_id=$1 AND user_id=$2`, principal.TenantID, id); err != nil {
		return userdomain.User{}, err
	}
	for _, unitID := range input.UnitIDs {
		primary := input.PrimaryUnitID != nil && unitID == *input.PrimaryUnitID
		if _, err = tx.Exec(ctx, `INSERT INTO org.user_units(tenant_id,user_id,unit_id,is_primary) VALUES($1,$2,$3,$4)`, principal.TenantID, id, unitID, primary); err != nil {
			return userdomain.User{}, err
		}
	}
	for _, positionID := range input.PositionIDs {
		primary := input.PrimaryPositionID != nil && positionID == *input.PrimaryPositionID
		if _, err = tx.Exec(ctx, `INSERT INTO org.user_positions(tenant_id,user_id,position_id,unit_id,is_primary) VALUES($1,$2,$3,$4,$5)`, principal.TenantID, id, positionID, input.PrimaryUnitID, primary); err != nil {
			return userdomain.User{}, err
		}
	}
	after, err := getUser(ctx, tx, principal.TenantID, id)
	if err != nil {
		return userdomain.User{}, err
	}
	if err = insertAudit(ctx, tx, principal, "org.assignment.update", "org.assignment.update", id, "PUT", "/admin-api/v1/org/users/{user_id}/assignments", map[string]any{"units": before.Units, "positions": before.Positions}, map[string]any{"units": after.Units, "positions": after.Positions}); err != nil {
		return userdomain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userdomain.User{}, err
	}
	return after, nil
}

func (repository *Postgres) ListSessions(ctx context.Context, tenantID, target, current uuid.UUID) ([]userdomain.Session, error) {
	exists, err := db.New(repository.pool).TenantManagedUserExists(ctx, db.TenantManagedUserExistsParams{TenantID: tenantID, UserID: target})
	if err != nil {
		return nil, fmt.Errorf("check tenant user before listing sessions: %w", err)
	}
	if !exists {
		return nil, userdomain.ErrNotFound
	}
	rows, err := repository.pool.Query(ctx, `SELECT id,audience,status,COALESCE(host(ip_address),''),COALESCE(user_agent,''),last_seen_at,absolute_expires_at,revoked_at,id=$3 FROM iam.sessions WHERE tenant_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 100`, tenantID, target, current)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []userdomain.Session{}
	for rows.Next() {
		var item userdomain.Session
		if err = rows.Scan(&item.ID, &item.Audience, &item.Status, &item.IPAddress, &item.UserAgent, &item.LastSeenAt, &item.ExpiresAt, &item.RevokedAt, &item.Current); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repository *Postgres) RevokeSession(ctx context.Context, principal userdomain.Principal, target, sessionID uuid.UUID) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = lockUser(ctx, tx, principal.TenantID, target); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=COALESCE(revoked_at,now()),revoke_reason='admin_revoke',access_token_version=access_token_version+1 WHERE id=$1 AND tenant_id=$2 AND user_id=$3 AND status='active' AND revoked_at IS NULL`, sessionID, principal.TenantID, target)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return userdomain.ErrSessionAbsent
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE session_id=$1 AND revoked_at IS NULL`, sessionID); err != nil {
		return err
	}
	if err = insertAudit(ctx, tx, principal, "iam.session.revoke", "iam.session.revoke", sessionID, "DELETE", "/admin-api/v1/users/{id}/sessions/{session_id}", nil, map[string]any{"target_user_id": target, "revoked": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type rowScanner interface{ Scan(...any) error }
type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getUser(ctx context.Context, q querier, tenantID, id uuid.UUID) (userdomain.User, error) {
	row := q.QueryRow(ctx, "SELECT "+userColumns+" FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE tm.tenant_id=$1 AND u.id=$2 AND u.deleted_at IS NULL", tenantID, id)
	item, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userdomain.User{}, userdomain.ErrNotFound
	}
	return item, err
}
func scanUser(row rowScanner) (userdomain.User, error) {
	var item userdomain.User
	var roles, units, positions []byte
	err := row.Scan(&item.ID, &item.Email, &item.Username, &item.DisplayName, &item.Locale, &item.TimeZone, &item.Status, &item.GlobalStatus, &item.MemberStatus, &item.IsSystem, &roles, &units, &positions, &item.ActiveSessionCount, &item.LastLoginAt, &item.LastActiveAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(roles, &item.Roles); err != nil {
		return item, err
	}
	if err = json.Unmarshal(units, &item.Units); err != nil {
		return item, err
	}
	if err = json.Unmarshal(positions, &item.Positions); err != nil {
		return item, err
	}
	return item, nil
}
func lockUser(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) (userdomain.User, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE tm.tenant_id=$1 AND tm.user_id=$2 AND u.deleted_at IS NULL FOR UPDATE OF tm,u`, tenantID, id).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return userdomain.User{}, userdomain.ErrNotFound
	} else if err != nil {
		return userdomain.User{}, err
	}
	return getUser(ctx, tx, tenantID, id)
}
func validateTenantIDs(ctx context.Context, tx pgx.Tx, table string, tenantID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	if table != "org.units" && table != "org.positions" {
		return userdomain.ErrOrgInvalid
	}
	var count int64
	query := "SELECT count(*) FROM " + table + " WHERE tenant_id=$1 AND id=ANY($2::uuid[]) AND status='active' AND deleted_at IS NULL"
	if err := tx.QueryRow(ctx, query, tenantID, ids).Scan(&count); err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return userdomain.ErrOrgInvalid
	}
	return nil
}
func (repository *Postgres) begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin user administration transaction: %w", err)
	}
	return tx, nil
}
func insertAudit(ctx context.Context, tx pgx.Tx, p userdomain.Principal, action, permission string, id uuid.UUID, method, path string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,$3,$4,'iam',$5,$6,'iam.user',$7,$8,$9,200,$10,$11,$12,$13,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, permission, id.String(), method, path, p.IPAddress, nullable(p.UserAgent), beforeJSON, afterJSON)
	if err != nil {
		return fmt.Errorf("insert user administration audit: %w", err)
	}
	return nil
}
func safeUser(user userdomain.User) any {
	return map[string]any{"id": user.ID, "email": user.Email, "display_name": user.DisplayName, "locale": user.Locale, "time_zone": user.TimeZone, "status": user.Status, "member_status": user.MemberStatus, "roles": user.Roles, "units": user.Units, "positions": user.Positions}
}
func nullable(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return userdomain.ErrEmailConflict
	}
	return fmt.Errorf("write managed user: %w", err)
}
