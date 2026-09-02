package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	accessdomain "github.com/appkernia/appkernia/server/internal/modules/accessadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

var registeredComponentKeys = map[string]struct{}{
	"dashboard": {}, "system.settings.configs": {}, "system.settings.dictionaries": {}, "system.settings.regions": {}, "system.settings.login-providers": {},
	"system.users.departments": {}, "system.users.accounts": {}, "system.users.positions": {}, "system.users.tenants": {},
	"system.access.roles": {}, "system.access.permissions": {}, "system.access.menus": {}, "system.storage.files": {},
	"system.notifications.notices": {}, "system.notifications.messages": {}, "system.notifications.templates": {}, "system.notifications.operations": {}, "system.notifications.push-channels": {},
	"system.integrations.schedules": {}, "system.integrations.api-clients": {}, "system.integrations.webhooks": {},
	"system.security.operation-logs": {}, "system.security.login-logs": {}, "system.security.events": {}, "system.security.block-rules": {},
	"system.monitoring.sessions": {}, "system.monitoring.health": {},
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Postgres) ListRoles(ctx context.Context, tenantID uuid.UUID, f accessdomain.Filters) (accessdomain.RolePage, error) {
	where := ` WHERE r.tenant_id=$1 AND r.deleted_at IS NULL`
	args := []any{tenantID}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += fmt.Sprintf(` AND (r.name ILIKE $%d OR r.code::text ILIKE $%d)`, len(args), len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND r.status=$%d`, len(args))
	}
	if f.RoleType != "" {
		args = append(args, f.RoleType)
		where += fmt.Sprintf(` AND r.role_type=$%d`, len(args))
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM iam.roles r`+where, args...).Scan(&total); err != nil {
		return accessdomain.RolePage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, roleSelect+where+fmt.Sprintf(` ORDER BY r.sort_order,r.name LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return accessdomain.RolePage{}, err
	}
	defer rows.Close()
	items := []accessdomain.Role{}
	for rows.Next() {
		item, scanErr := scanRole(rows)
		if scanErr != nil {
			return accessdomain.RolePage{}, scanErr
		}
		items = append(items, item)
	}
	return accessdomain.RolePage{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, rows.Err()
}

const roleSelect = `SELECT r.id,r.parent_id,r.code::text,r.name,coalesce(r.description,''),r.role_type,r.data_scope,r.sort_order,r.is_default,r.is_system,r.status,r.created_at,r.updated_at,
 (SELECT count(*) FROM iam.user_roles ur WHERE ur.tenant_id=r.tenant_id AND ur.role_id=r.id AND (ur.valid_until IS NULL OR ur.valid_until>now())),
 coalesce((SELECT array_agg(rp.permission_id ORDER BY rp.permission_id) FROM iam.role_permissions rp WHERE rp.tenant_id=r.tenant_id AND rp.role_id=r.id),'{}'::uuid[]),
 coalesce((SELECT array_agg(rm.menu_id ORDER BY rm.menu_id) FROM sys.role_menus rm WHERE rm.tenant_id=r.tenant_id AND rm.role_id=r.id),'{}'::uuid[]),
 coalesce((SELECT array_agg(rs.unit_id ORDER BY rs.unit_id) FROM iam.role_scope_units rs WHERE rs.tenant_id=r.tenant_id AND rs.role_id=r.id),'{}'::uuid[])
 FROM iam.roles r`

type roleScanner interface{ Scan(...any) error }

func scanRole(row roleScanner) (accessdomain.Role, error) {
	var x accessdomain.Role
	err := row.Scan(&x.ID, &x.ParentID, &x.Code, &x.Name, &x.Description, &x.RoleType, &x.DataScope, &x.SortOrder, &x.IsDefault, &x.IsSystem, &x.Status, &x.CreatedAt, &x.UpdatedAt, &x.MemberCount, &x.PermissionIDs, &x.MenuIDs, &x.ScopeUnitIDs)
	return x, err
}
func getRole(ctx context.Context, q rowQuerier, tenantID, id uuid.UUID, lock bool) (accessdomain.Role, error) {
	suffix := ` WHERE r.tenant_id=$1 AND r.id=$2 AND r.deleted_at IS NULL`
	if lock {
		suffix += ` FOR UPDATE OF r`
	}
	x, err := scanRole(q.QueryRow(ctx, roleSelect+suffix, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, accessdomain.ErrNotFound
	}
	return x, err
}

func (r *Postgres) CreateRole(ctx context.Context, p accessdomain.Principal, in accessdomain.RoleInput) (accessdomain.Role, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Role{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = validateRoleParent(ctx, tx, p.TenantID, uuid.Nil, in.ParentID); err != nil {
		return accessdomain.Role{}, err
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO iam.roles(tenant_id,parent_id,code,name,description,role_type,data_scope,sort_order,is_system,status) VALUES($1,$2,$3,$4,nullif($5,''),'custom','self',$6,false,$7) RETURNING id`, p.TenantID, in.ParentID, in.Code, in.Name, in.Description, in.SortOrder, in.Status).Scan(&id)
	if err != nil {
		return accessdomain.Role{}, classify(err)
	}
	after, err := getRole(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return accessdomain.Role{}, err
	}
	if err = audit(ctx, tx, p, "iam.role.create", "iam.role", id, "POST", "/admin-api/v1/roles", nil, after); err != nil {
		return accessdomain.Role{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Role{}, err
	}
	return after, nil
}

func (r *Postgres) UpdateRole(ctx context.Context, p accessdomain.Principal, id uuid.UUID, in accessdomain.RoleInput) (accessdomain.Role, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Role{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getRole(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return accessdomain.Role{}, err
	}
	if before.IsSystem {
		return accessdomain.Role{}, accessdomain.ErrSystemRole
	}
	if err = validateRoleParent(ctx, tx, p.TenantID, id, in.ParentID); err != nil {
		return accessdomain.Role{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE iam.roles SET parent_id=$1,code=$2,name=$3,description=nullif($4,''),sort_order=$5,status=$6 WHERE tenant_id=$7 AND id=$8`, in.ParentID, in.Code, in.Name, in.Description, in.SortOrder, in.Status, p.TenantID, id)
	if err != nil {
		return accessdomain.Role{}, classify(err)
	}
	after, err := getRole(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return accessdomain.Role{}, err
	}
	if err = audit(ctx, tx, p, "iam.role.update", "iam.role", id, "PATCH", "/admin-api/v1/roles/"+id.String(), before, after); err != nil {
		return accessdomain.Role{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Role{}, err
	}
	return after, nil
}

func validateRoleParent(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	if *parentID == id {
		return accessdomain.ErrConflict
	}
	var cycle bool
	err := tx.QueryRow(ctx, `WITH RECURSIVE a AS (SELECT id,parent_id FROM iam.roles WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL UNION ALL SELECT r.id,r.parent_id FROM iam.roles r JOIN a ON r.id=a.parent_id WHERE r.tenant_id=$1 AND r.deleted_at IS NULL) SELECT EXISTS(SELECT 1 FROM a WHERE id=$3)`, tenantID, *parentID, id).Scan(&cycle)
	if err != nil {
		return err
	}
	if cycle {
		return accessdomain.ErrConflict
	}
	return nil
}

func (r *Postgres) DeleteRole(ctx context.Context, p accessdomain.Principal, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getRole(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return err
	}
	if before.IsSystem {
		return accessdomain.ErrSystemRole
	}
	if before.MemberCount > 0 {
		return accessdomain.ErrRoleOccupied
	}
	var children int64
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM iam.roles WHERE tenant_id=$1 AND parent_id=$2 AND deleted_at IS NULL`, p.TenantID, id).Scan(&children); err != nil {
		return err
	}
	if children > 0 {
		return accessdomain.ErrRoleOccupied
	}
	tag, err := tx.Exec(ctx, `UPDATE iam.roles SET deleted_at=now(),status='disabled' WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, p.TenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return accessdomain.ErrNotFound
	}
	if err = audit(ctx, tx, p, "iam.role.delete", "iam.role", id, "DELETE", "/admin-api/v1/roles/"+id.String(), before, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ReplaceRolePermissions(ctx context.Context, p accessdomain.Principal, id uuid.UUID, ids []uuid.UUID) (accessdomain.Role, error) {
	return r.replaceRole(ctx, p, id, "permissions", ids, "")
}
func (r *Postgres) ReplaceRoleMenus(ctx context.Context, p accessdomain.Principal, id uuid.UUID, ids []uuid.UUID) (accessdomain.Role, error) {
	return r.replaceRole(ctx, p, id, "menus", ids, "")
}
func (r *Postgres) ReplaceRoleDataScope(ctx context.Context, p accessdomain.Principal, id uuid.UUID, scope string, ids []uuid.UUID) (accessdomain.Role, error) {
	return r.replaceRole(ctx, p, id, "data_scope", ids, scope)
}
func (r *Postgres) replaceRole(ctx context.Context, p accessdomain.Principal, id uuid.UUID, kind string, ids []uuid.UUID, scope string) (accessdomain.Role, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Role{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getRole(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return accessdomain.Role{}, err
	}
	if before.IsSystem {
		return accessdomain.Role{}, accessdomain.ErrSystemRole
	}
	switch kind {
	case "permissions":
		if err = ensureCount(ctx, tx, `SELECT count(*) FROM iam.permissions WHERE id=ANY($1::uuid[]) AND status='active'`, ids, len(ids)); err != nil {
			return accessdomain.Role{}, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM iam.role_permissions WHERE tenant_id=$1 AND role_id=$2`, p.TenantID, id); err == nil && len(ids) > 0 {
			_, err = tx.Exec(ctx, `INSERT INTO iam.role_permissions(tenant_id,role_id,permission_id,granted_by) SELECT $1,$2,unnest($3::uuid[]),$4`, p.TenantID, id, ids, p.UserID)
		}
	case "menus":
		if err = ensureCount(ctx, tx, `SELECT count(*) FROM sys.menus WHERE id=ANY($1::uuid[]) AND (tenant_id IS NULL OR tenant_id=$2) AND status='active' AND deleted_at IS NULL`, ids, len(ids), p.TenantID); err != nil {
			return accessdomain.Role{}, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM sys.role_menus WHERE tenant_id=$1 AND role_id=$2`, p.TenantID, id); err == nil && len(ids) > 0 {
			_, err = tx.Exec(ctx, `INSERT INTO sys.role_menus(tenant_id,role_id,menu_id) SELECT $1,$2,unnest($3::uuid[])`, p.TenantID, id, ids)
		}
	case "data_scope":
		if err = ensureCount(ctx, tx, `SELECT count(*) FROM org.units WHERE tenant_id=$2 AND id=ANY($1::uuid[]) AND deleted_at IS NULL`, ids, len(ids), p.TenantID); err != nil {
			return accessdomain.Role{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE iam.roles SET data_scope=$1 WHERE tenant_id=$2 AND id=$3`, scope, p.TenantID, id); err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM iam.role_scope_units WHERE tenant_id=$1 AND role_id=$2`, p.TenantID, id)
		}
		if err == nil && len(ids) > 0 {
			_, err = tx.Exec(ctx, `INSERT INTO iam.role_scope_units(tenant_id,role_id,unit_id,include_descendants) SELECT $1,$2,unnest($3::uuid[]),false`, p.TenantID, id, ids)
		}
	}
	if err != nil {
		return accessdomain.Role{}, classify(err)
	}
	after, err := getRole(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return accessdomain.Role{}, err
	}
	action := map[string]string{"permissions": "iam.role.assign_permission", "menus": "iam.role.assign_menu", "data_scope": "iam.role.update_data_scope"}[kind]
	if err = audit(ctx, tx, p, action, "iam.role", id, "PUT", "/admin-api/v1/roles/"+id.String()+"/"+strings.ReplaceAll(kind, "_", "-"), before, after); err != nil {
		return accessdomain.Role{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Role{}, err
	}
	return after, nil
}
func ensureCount(ctx context.Context, q rowQuerier, sql string, ids []uuid.UUID, want int, extra ...any) error {
	if len(ids) == 0 {
		return nil
	}
	args := []any{ids}
	args = append(args, extra...)
	var count int
	if err := q.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return err
	}
	if count != want {
		return accessdomain.ErrNotFound
	}
	return nil
}

func (r *Postgres) ListPermissions(ctx context.Context, f accessdomain.PermissionFilters) ([]accessdomain.Permission, error) {
	q := `SELECT id,code::text,name,module_code,resource_name,action_name,permission_kind,http_methods,coalesce(route_pattern,''),coalesce(description,''),status FROM iam.permissions WHERE true`
	args := []any{}
	add := func(column, value string) {
		if value != "" {
			args = append(args, value)
			q += fmt.Sprintf(` AND %s=$%d`, column, len(args))
		}
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		q += fmt.Sprintf(` AND (code::text ILIKE $%d OR name ILIKE $%d OR coalesce(description,'') ILIKE $%d)`, len(args), len(args), len(args))
	}
	add("module_code", f.ModuleCode)
	add("resource_name", f.ResourceName)
	add("action_name", f.ActionName)
	add("permission_kind", f.PermissionKind)
	add("status", f.Status)
	q += ` ORDER BY module_code,resource_name,action_name LIMIT 1000`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []accessdomain.Permission{}
	for rows.Next() {
		var x accessdomain.Permission
		if err = rows.Scan(&x.ID, &x.Code, &x.Name, &x.ModuleCode, &x.ResourceName, &x.ActionName, &x.PermissionKind, &x.HTTPMethods, &x.RoutePattern, &x.Description, &x.Status); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

const menuSelect = `SELECT m.id,m.tenant_id,m.parent_id,m.permission_id,coalesce(p.code::text,''),m.code::text,m.title,coalesce(m.metadata->>'i18n_key',''),m.menu_type,coalesce(m.route_path,''),coalesce(m.component_key,''),coalesce(m.icon,''),coalesce(m.external_url,''),m.open_mode,m.hidden,m.affix,m.sort_order,m.status,(m.tenant_id IS NULL),m.updated_at,(SELECT count(*) FROM sys.role_menus rm WHERE rm.menu_id=m.id) FROM sys.menus m LEFT JOIN iam.permissions p ON p.id=m.permission_id`

func (r *Postgres) ListMenus(ctx context.Context, tenantID uuid.UUID) ([]accessdomain.Menu, error) {
	rows, err := r.pool.Query(ctx, menuSelect+` WHERE (m.tenant_id IS NULL OR m.tenant_id=$1) AND m.deleted_at IS NULL ORDER BY m.sort_order,m.title`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	flat := []accessdomain.Menu{}
	for rows.Next() {
		x, e := scanMenu(rows)
		if e != nil {
			return nil, e
		}
		flat = append(flat, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return buildTree(flat), nil
}

type menuScanner interface{ Scan(...any) error }

func scanMenu(row menuScanner) (accessdomain.Menu, error) {
	var x accessdomain.Menu
	err := row.Scan(&x.ID, &x.TenantID, &x.ParentID, &x.PermissionID, &x.PermissionCode, &x.Code, &x.Title, &x.I18nKey, &x.Type, &x.Path, &x.ComponentKey, &x.Icon, &x.ExternalURL, &x.OpenMode, &x.Hidden, &x.Affix, &x.SortOrder, &x.Status, &x.IsCore, &x.UpdatedAt, &x.RoleCount)
	x.Children = []accessdomain.Menu{}
	return x, err
}
func getMenu(ctx context.Context, q rowQuerier, tenantID, id uuid.UUID, owned, lock bool) (accessdomain.Menu, error) {
	where := ` WHERE m.id=$2 AND m.deleted_at IS NULL AND `
	if owned {
		where += `m.tenant_id=$1`
	} else {
		where += `(m.tenant_id IS NULL OR m.tenant_id=$1)`
	}
	if lock {
		where += ` FOR UPDATE OF m`
	}
	x, err := scanMenu(q.QueryRow(ctx, menuSelect+where, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, accessdomain.ErrNotFound
	}
	return x, err
}
func buildTree(flat []accessdomain.Menu) []accessdomain.Menu {
	byParent := map[uuid.UUID][]accessdomain.Menu{}
	roots := []accessdomain.Menu{}
	ids := map[uuid.UUID]struct{}{}
	for _, x := range flat {
		ids[x.ID] = struct{}{}
	}
	for _, x := range flat {
		if x.ParentID == nil {
			roots = append(roots, x)
			continue
		}
		if _, ok := ids[*x.ParentID]; !ok {
			roots = append(roots, x)
			continue
		}
		byParent[*x.ParentID] = append(byParent[*x.ParentID], x)
	}
	var attach func(accessdomain.Menu) accessdomain.Menu
	attach = func(x accessdomain.Menu) accessdomain.Menu {
		x.Children = []accessdomain.Menu{}
		for _, child := range byParent[x.ID] {
			x.Children = append(x.Children, attach(child))
		}
		return x
	}
	out := make([]accessdomain.Menu, 0, len(roots))
	for _, root := range roots {
		out = append(out, attach(root))
	}
	return out
}

func (r *Postgres) CreateMenu(ctx context.Context, p accessdomain.Principal, in accessdomain.MenuInput) (accessdomain.Menu, error) {
	if in.ComponentKey != "" {
		if _, ok := registeredComponentKeys[in.ComponentKey]; !ok {
			return accessdomain.Menu{}, accessdomain.ErrComponentKey
		}
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Menu{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = validateMenuLinks(ctx, tx, p.TenantID, uuid.Nil, in.ParentID, in.PermissionID, 1); err != nil {
		return accessdomain.Menu{}, err
	}
	metadata, _ := json.Marshal(map[string]string{"i18n_key": in.I18nKey})
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO sys.menus(tenant_id,parent_id,permission_id,code,title,menu_type,route_path,component_key,icon,external_url,open_mode,hidden,affix,sort_order,status,metadata) VALUES($1,$2,$3,$4,$5,$6,nullif($7,''),nullif($8,''),nullif($9,''),nullif($10,''),$11,$12,$13,$14,$15,$16) RETURNING id`, p.TenantID, in.ParentID, in.PermissionID, in.Code, in.Title, in.Type, in.Path, in.ComponentKey, in.Icon, in.ExternalURL, in.OpenMode, in.Hidden, in.Affix, in.SortOrder, in.Status, metadata).Scan(&id)
	if err != nil {
		return accessdomain.Menu{}, classify(err)
	}
	after, err := getMenu(ctx, tx, p.TenantID, id, true, false)
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if err = audit(ctx, tx, p, "sys.menu.create", "sys.menu", id, "POST", "/admin-api/v1/menus", nil, after); err != nil {
		return accessdomain.Menu{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Menu{}, err
	}
	return after, nil
}
func (r *Postgres) UpdateMenu(ctx context.Context, p accessdomain.Principal, id uuid.UUID, in accessdomain.MenuInput) (accessdomain.Menu, error) {
	if in.ComponentKey != "" {
		if _, ok := registeredComponentKeys[in.ComponentKey]; !ok {
			return accessdomain.Menu{}, accessdomain.ErrComponentKey
		}
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Menu{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getMenu(ctx, tx, p.TenantID, id, true, true)
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if !sameOptionalUUID(before.ParentID, in.ParentID) {
		return accessdomain.Menu{}, accessdomain.ErrInvalid
	}
	if err = validateMenuLinks(ctx, tx, p.TenantID, id, in.ParentID, in.PermissionID, subtreeDepth(ctx, tx, p.TenantID, id)); err != nil {
		return accessdomain.Menu{}, err
	}
	metadata, _ := json.Marshal(map[string]string{"i18n_key": in.I18nKey})
	_, err = tx.Exec(ctx, `UPDATE sys.menus SET parent_id=$1,permission_id=$2,code=$3,title=$4,menu_type=$5,route_path=nullif($6,''),component_key=nullif($7,''),icon=nullif($8,''),external_url=nullif($9,''),open_mode=$10,hidden=$11,affix=$12,sort_order=$13,status=$14,metadata=$15 WHERE tenant_id=$16 AND id=$17`, in.ParentID, in.PermissionID, in.Code, in.Title, in.Type, in.Path, in.ComponentKey, in.Icon, in.ExternalURL, in.OpenMode, in.Hidden, in.Affix, in.SortOrder, in.Status, metadata, p.TenantID, id)
	if err != nil {
		return accessdomain.Menu{}, classify(err)
	}
	after, err := getMenu(ctx, tx, p.TenantID, id, true, false)
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if err = audit(ctx, tx, p, "sys.menu.update", "sys.menu", id, "PATCH", "/admin-api/v1/menus/"+id.String(), before, after); err != nil {
		return accessdomain.Menu{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Menu{}, err
	}
	return after, nil
}
func (r *Postgres) MoveMenu(ctx context.Context, p accessdomain.Principal, id uuid.UUID, in accessdomain.MenuMove) (accessdomain.Menu, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return accessdomain.Menu{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getMenu(ctx, tx, p.TenantID, id, true, true)
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if err = validateMenuLinks(ctx, tx, p.TenantID, id, in.ParentID, nil, subtreeDepth(ctx, tx, p.TenantID, id)); err != nil {
		return accessdomain.Menu{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE sys.menus SET parent_id=$1,sort_order=$2 WHERE tenant_id=$3 AND id=$4`, in.ParentID, in.SortOrder, p.TenantID, id)
	if err != nil {
		return accessdomain.Menu{}, classify(err)
	}
	after, err := getMenu(ctx, tx, p.TenantID, id, true, false)
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if err = audit(ctx, tx, p, "sys.menu.move", "sys.menu", id, "POST", "/admin-api/v1/menus/"+id.String()+"/move", before, after); err != nil {
		return accessdomain.Menu{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return accessdomain.Menu{}, err
	}
	return after, nil
}
func validateMenuLinks(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, parentID, permissionID *uuid.UUID, subtree int) error {
	parentDepth := 0
	if parentID != nil {
		if *parentID == id {
			return accessdomain.ErrMenuCycle
		}
		var cycle bool
		err := tx.QueryRow(ctx, `WITH RECURSIVE a AS (SELECT id,parent_id,1 depth FROM sys.menus WHERE id=$2 AND (tenant_id IS NULL OR tenant_id=$1) AND deleted_at IS NULL UNION ALL SELECT m.id,m.parent_id,a.depth+1 FROM sys.menus m JOIN a ON m.id=a.parent_id WHERE (m.tenant_id IS NULL OR m.tenant_id=$1) AND m.deleted_at IS NULL) SELECT coalesce(max(depth),0),coalesce(bool_or(id=$3),false) FROM a`, tenantID, *parentID, id).Scan(&parentDepth, &cycle)
		if err != nil {
			return err
		}
		if parentDepth == 0 {
			return accessdomain.ErrNotFound
		}
		if cycle {
			return accessdomain.ErrMenuCycle
		}
	}
	if parentDepth+subtree > 3 {
		return accessdomain.ErrMenuDepth
	}
	if permissionID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.permissions WHERE id=$1 AND status='active')`, *permissionID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return accessdomain.ErrNotFound
		}
	}
	return nil
}
func subtreeDepth(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) int {
	var depth int
	err := tx.QueryRow(ctx, `WITH RECURSIVE d AS (SELECT id,1 depth FROM sys.menus WHERE id=$2 AND tenant_id=$1 AND deleted_at IS NULL UNION ALL SELECT m.id,d.depth+1 FROM sys.menus m JOIN d ON m.parent_id=d.id WHERE m.tenant_id=$1 AND m.deleted_at IS NULL) SELECT coalesce(max(depth),1) FROM d`, tenantID, id).Scan(&depth)
	if err != nil {
		return 4
	}
	return depth
}
func (r *Postgres) DeleteMenu(ctx context.Context, p accessdomain.Principal, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getMenu(ctx, tx, p.TenantID, id, true, true)
	if err != nil {
		return err
	}
	var occupied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sys.menus WHERE parent_id=$1 AND deleted_at IS NULL) OR EXISTS(SELECT 1 FROM sys.role_menus WHERE menu_id=$1)`, id).Scan(&occupied); err != nil {
		return err
	}
	if occupied {
		return accessdomain.ErrMenuOccupied
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.menus SET deleted_at=now(),status='disabled' WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, p.TenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return accessdomain.ErrNotFound
	}
	if err = audit(ctx, tx, p, "sys.menu.delete", "sys.menu", id, "DELETE", "/admin-api/v1/menus/"+id.String(), before, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func audit(ctx context.Context, tx pgx.Tx, p accessdomain.Principal, action, resource string, id uuid.UUID, method, path string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	module := strings.Split(action, ".")[0]
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),$5,$6,$6,$7,$8,$9,$10,200,$11,nullif($12,''),$13,$14,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, module, action, resource, id.String(), method, path, p.IPAddress, p.UserAgent, b, a)
	return err
}
func classify(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "23505" || p.Code == "23503" || p.Code == "23514") {
		return accessdomain.ErrConflict
	}
	return fmt.Errorf("access administration write: %w", err)
}

func sameOptionalUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
