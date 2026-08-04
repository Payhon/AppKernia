package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	tenantdomain "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) List(ctx context.Context, current uuid.UUID, f tenantdomain.Filters) (tenantdomain.Page, error) {
	q := `SELECT t.id,t.code::text,t.name,t.status,coalesce(t.plan_code,''),t.created_at,t.updated_at,(SELECT count(*) FROM iam.tenant_members tm WHERE tm.tenant_id=t.id AND tm.status='active') FROM iam.tenants t WHERE t.id=$1 AND t.deleted_at IS NULL`
	args := []any{current}
	n := 2
	if f.Query != "" {
		q += fmt.Sprintf(" AND (t.name ILIKE $%d OR t.code::text ILIKE $%d)", n, n)
		args = append(args, "%"+f.Query+"%")
		n++
	}
	if f.Status != "" {
		q += fmt.Sprintf(" AND t.status=$%d", n)
		args = append(args, f.Status)
	}
	q += " ORDER BY t.created_at DESC"
	rows, e := r.pool.Query(ctx, q, args...)
	if e != nil {
		return tenantdomain.Page{}, e
	}
	defer rows.Close()
	items := []tenantdomain.Tenant{}
	for rows.Next() {
		var x tenantdomain.Tenant
		if e = rows.Scan(&x.ID, &x.Code, &x.Name, &x.Status, &x.PlanCode, &x.CreatedAt, &x.UpdatedAt, &x.MemberCount); e != nil {
			return tenantdomain.Page{}, e
		}
		items = append(items, x)
	}
	return tenantdomain.Page{Items: items, Total: int64(len(items)), Page: f.Page, PageSize: f.PageSize}, rows.Err()
}
func (r *Postgres) Get(ctx context.Context, current, id uuid.UUID) (tenantdomain.Tenant, error) {
	if current != id {
		return tenantdomain.Tenant{}, tenantdomain.ErrNotFound
	}
	return getTenant(ctx, r.pool, id)
}
func getTenant(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id uuid.UUID) (tenantdomain.Tenant, error) {
	var x tenantdomain.Tenant
	e := q.QueryRow(ctx, `SELECT t.id,t.code::text,t.name,t.status,coalesce(t.plan_code,''),t.created_at,t.updated_at,(SELECT count(*) FROM iam.tenant_members tm WHERE tm.tenant_id=t.id AND tm.status='active') FROM iam.tenants t WHERE t.id=$1 AND t.deleted_at IS NULL`, id).Scan(&x.ID, &x.Code, &x.Name, &x.Status, &x.PlanCode, &x.CreatedAt, &x.UpdatedAt, &x.MemberCount)
	if errors.Is(e, pgx.ErrNoRows) {
		return x, tenantdomain.ErrNotFound
	}
	return x, e
}
func (r *Postgres) Create(ctx context.Context, p tenantdomain.Principal, in tenantdomain.CreateInput) (tenantdomain.Tenant, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id, roleID uuid.UUID
	e = tx.QueryRow(ctx, `INSERT INTO iam.tenants(code,name,status,settings) VALUES($1,$2,'active','{}') RETURNING id`, in.Code, in.Name).Scan(&id)
	if e != nil {
		return tenantdomain.Tenant{}, classify(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,display_name,status,invited_by) VALUES($1,$2,NULL,'active',$2)`, id, p.UserID); e != nil {
		return tenantdomain.Tenant{}, e
	}
	e = tx.QueryRow(ctx, `INSERT INTO iam.roles(tenant_id,code,name,description,role_type,data_scope,is_system,status) VALUES($1,'super-admin','Super Administrator','Tenant creator administrator','system','tenant',true,'active') RETURNING id`, id).Scan(&roleID)
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO iam.user_roles(tenant_id,user_id,role_id,granted_by) VALUES($1,$2,$3,$2)`, id, p.UserID, roleID); e != nil {
		return tenantdomain.Tenant{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO iam.role_permissions(tenant_id,role_id,permission_id,granted_by) SELECT $1,$2,id,$3 FROM iam.permissions WHERE status='active' ON CONFLICT DO NOTHING`, id, roleID, p.UserID); e != nil {
		return tenantdomain.Tenant{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO sys.role_menus(tenant_id,role_id,menu_id) SELECT $1,$2,id FROM sys.menus WHERE tenant_id IS NULL AND status='active' AND deleted_at IS NULL ON CONFLICT DO NOTHING`, id, roleID); e != nil {
		return tenantdomain.Tenant{}, e
	}
	after, e := getTenant(ctx, tx, id)
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	if e = audit(ctx, tx, p, id, "iam.tenant.create", "POST", "/admin-api/v1/tenants", nil, after); e != nil {
		return tenantdomain.Tenant{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return tenantdomain.Tenant{}, e
	}
	return after, nil
}
func (r *Postgres) Update(ctx context.Context, p tenantdomain.Principal, id uuid.UUID, in tenantdomain.UpdateInput) (tenantdomain.Tenant, error) {
	if p.TenantID != id {
		return tenantdomain.Tenant{}, tenantdomain.ErrNotFound
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, e := getTenant(ctx, tx, id)
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE iam.tenants SET name=$1,status=$2,updated_at=now() WHERE id=$3`, in.Name, in.Status, id); e != nil {
		return tenantdomain.Tenant{}, e
	}
	after, e := getTenant(ctx, tx, id)
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	if in.Status != "active" {
		_, e = tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=coalesce(revoked_at,now()),revoke_reason='tenant_disabled',access_token_version=access_token_version+1 WHERE tenant_id=$1 AND status='active'`, id)
		if e != nil {
			return tenantdomain.Tenant{}, e
		}
	}
	if e = audit(ctx, tx, p, id, "iam.tenant.update", "PATCH", "/admin-api/v1/tenants/"+id.String(), before, after); e != nil {
		return tenantdomain.Tenant{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return tenantdomain.Tenant{}, e
	}
	return after, nil
}
func (r *Postgres) Members(ctx context.Context, current, id uuid.UUID) ([]tenantdomain.Member, error) {
	if current != id {
		return nil, tenantdomain.ErrNotFound
	}
	rows, e := r.pool.Query(ctx, `SELECT u.id,coalesce(u.email::text,''),coalesce(tm.display_name,u.display_name),tm.status,tm.joined_at,coalesce(array_agg(ro.code::text ORDER BY ro.code) FILTER(WHERE ro.id IS NOT NULL),'{}') FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id LEFT JOIN iam.user_roles ur ON ur.tenant_id=tm.tenant_id AND ur.user_id=tm.user_id LEFT JOIN iam.roles ro ON ro.tenant_id=ur.tenant_id AND ro.id=ur.role_id WHERE tm.tenant_id=$1 GROUP BY u.id,u.email,tm.display_name,u.display_name,tm.status,tm.joined_at ORDER BY tm.joined_at`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []tenantdomain.Member{}
	for rows.Next() {
		var m tenantdomain.Member
		if e = rows.Scan(&m.UserID, &m.Email, &m.DisplayName, &m.Status, &m.JoinedAt, &m.RoleCodes); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *Postgres) AddMember(ctx context.Context, p tenantdomain.Principal, id uuid.UUID, in tenantdomain.AddMemberInput) (tenantdomain.Member, error) {
	if p.TenantID != id {
		return tenantdomain.Member{}, tenantdomain.ErrNotFound
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return tenantdomain.Member{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var uid uuid.UUID
	e = tx.QueryRow(ctx, `SELECT id FROM iam.users WHERE email=$1 AND deleted_at IS NULL`, in.Email).Scan(&uid)
	if errors.Is(e, pgx.ErrNoRows) {
		return tenantdomain.Member{}, tenantdomain.ErrNotFound
	}
	if e != nil {
		return tenantdomain.Member{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,display_name,status,invited_by,invited_at) VALUES($1,$2,nullif($3,''),'active',$4,now()) ON CONFLICT(tenant_id,user_id) DO UPDATE SET status='active',left_at=NULL,updated_at=now()`, id, uid, in.DisplayName, p.UserID)
	if e != nil {
		return tenantdomain.Member{}, classify(e)
	}
	if e = audit(ctx, tx, p, uid, "iam.tenant.member.invite", "POST", "/admin-api/v1/tenants/"+id.String()+"/members", nil, map[string]any{"user_id": uid, "status": "active"}); e != nil {
		return tenantdomain.Member{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return tenantdomain.Member{}, e
	}
	return r.member(ctx, id, uid)
}
func (r *Postgres) SetMemberStatus(ctx context.Context, p tenantdomain.Principal, id, uid uuid.UUID, status string) (tenantdomain.Member, error) {
	if p.TenantID != id {
		return tenantdomain.Member{}, tenantdomain.ErrNotFound
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return tenantdomain.Member{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if status != "active" {
		var isAdmin bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.user_roles ur JOIN iam.roles ro ON ro.id=ur.role_id AND ro.tenant_id=ur.tenant_id WHERE ur.tenant_id=$1 AND ur.user_id=$2 AND ro.code='super-admin' AND ro.status='active')`, id, uid).Scan(&isAdmin)
		if isAdmin {
			var count int
			_ = tx.QueryRow(ctx, `SELECT count(DISTINCT ur.user_id) FROM iam.user_roles ur JOIN iam.roles ro ON ro.id=ur.role_id AND ro.tenant_id=ur.tenant_id JOIN iam.tenant_members tm ON tm.tenant_id=ur.tenant_id AND tm.user_id=ur.user_id WHERE ur.tenant_id=$1 AND ro.code='super-admin' AND ro.status='active' AND tm.status='active'`, id).Scan(&count)
			if count <= 1 {
				return tenantdomain.Member{}, tenantdomain.ErrLastAdmin
			}
		}
	}
	tag, e := tx.Exec(ctx, `UPDATE iam.tenant_members SET status=$1,left_at=CASE WHEN $2 THEN now() ELSE NULL END,updated_at=now() WHERE tenant_id=$3 AND user_id=$4`, status, status == "left", id, uid)
	if e != nil {
		return tenantdomain.Member{}, e
	}
	if tag.RowsAffected() == 0 {
		return tenantdomain.Member{}, tenantdomain.ErrNotFound
	}
	if status != "active" {
		_, e = tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=coalesce(revoked_at,now()),revoke_reason='tenant_membership_changed',access_token_version=access_token_version+1 WHERE tenant_id=$1 AND user_id=$2 AND status='active'`, id, uid)
		if e != nil {
			return tenantdomain.Member{}, e
		}
	}
	action := "iam.tenant.member.update"
	method := "PATCH"
	if status == "left" {
		action = "iam.tenant.member.remove"
		method = "DELETE"
	}
	if e = audit(ctx, tx, p, uid, action, method, "/admin-api/v1/tenants/"+id.String()+"/members/"+uid.String(), nil, map[string]any{"user_id": uid, "status": status}); e != nil {
		return tenantdomain.Member{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return tenantdomain.Member{}, e
	}
	return r.member(ctx, id, uid)
}
func (r *Postgres) member(ctx context.Context, id, uid uuid.UUID) (tenantdomain.Member, error) {
	var m tenantdomain.Member
	e := r.pool.QueryRow(ctx, `SELECT u.id,coalesce(u.email::text,''),coalesce(tm.display_name,u.display_name),tm.status,tm.joined_at,coalesce(array_agg(ro.code::text ORDER BY ro.code) FILTER(WHERE ro.id IS NOT NULL),'{}') FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id LEFT JOIN iam.user_roles ur ON ur.tenant_id=tm.tenant_id AND ur.user_id=tm.user_id LEFT JOIN iam.roles ro ON ro.tenant_id=ur.tenant_id AND ro.id=ur.role_id WHERE tm.tenant_id=$1 AND tm.user_id=$2 GROUP BY u.id,u.email,tm.display_name,u.display_name,tm.status,tm.joined_at`, id, uid).Scan(&m.UserID, &m.Email, &m.DisplayName, &m.Status, &m.JoinedAt, &m.RoleCodes)
	if errors.Is(e, pgx.ErrNoRows) {
		return m, tenantdomain.ErrNotFound
	}
	return m, e
}
func audit(ctx context.Context, tx pgx.Tx, p tenantdomain.Principal, id uuid.UUID, action, method, path string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	resourceType := "iam.tenant"
	if strings.HasPrefix(action, "iam.tenant.member.") {
		resourceType = "iam.tenant_member"
	}
	_, e := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'iam',$5,$6,$7,$8,$9,200,$10,nullif($11,''),$12,$13,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, resourceType, id, method, path, p.IPAddress, p.UserAgent, b, a)
	return e
}
func classify(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) && (p.Code == "23505" || p.Code == "23503") {
		return tenantdomain.ErrConflict
	}
	return fmt.Errorf("tenant write: %w", e)
}
