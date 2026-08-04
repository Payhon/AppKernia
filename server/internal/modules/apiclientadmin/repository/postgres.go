package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(p *pgxpool.Pool) *Postgres { return &Postgres{p} }

type scanner interface{ Scan(...any) error }

const selectClient = `SELECT c.id,c.client_id::text,c.name,COALESCE(c.description,''),COALESCE(array_to_json(c.allowed_cidrs)::text,'[]'),c.status,c.expires_at,c.created_at,c.updated_at,
COALESCE((SELECT jsonb_agg(jsonb_build_object('id',s.id,'prefix',s.secret_prefix,'created_at',s.created_at,'expires_at',s.expires_at,'revoked_at',s.revoked_at,'last_used_at',s.last_used_at) ORDER BY s.created_at DESC) FROM sys.api_client_secrets s WHERE s.api_client_id=c.id),'[]'::jsonb),
COALESCE(ARRAY(SELECT p.code::text FROM sys.api_client_permissions cp JOIN iam.permissions p ON p.id=cp.permission_id WHERE cp.tenant_id=c.tenant_id AND cp.api_client_id=c.id ORDER BY p.code::text),'{}'::text[]) FROM sys.api_clients c`

func scanClient(row scanner) (clients.Client, error) {
	var c clients.Client
	var cidrs string
	var secrets []byte
	if e := row.Scan(&c.ID, &c.ClientID, &c.Name, &c.Description, &cidrs, &c.Status, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt, &secrets, &c.Permissions); e != nil {
		return c, e
	}
	if e := json.Unmarshal([]byte(cidrs), &c.AllowedCIDRs); e != nil {
		return c, e
	}
	if e := json.Unmarshal(secrets, &c.Secrets); e != nil {
		return c, e
	}
	if c.AllowedCIDRs == nil {
		c.AllowedCIDRs = []string{}
	}
	if c.Permissions == nil {
		c.Permissions = []string{}
	}
	if c.Secrets == nil {
		c.Secrets = []clients.Secret{}
	}
	return c, nil
}
func (r *Postgres) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenant, id uuid.UUID) (clients.Client, error) {
	c, e := scanClient(q.QueryRow(ctx, selectClient+` WHERE c.tenant_id=$1 AND c.id=$2`, tenant, id))
	if errors.Is(e, pgx.ErrNoRows) {
		e = clients.ErrNotFound
	}
	return c, e
}
func (r *Postgres) List(ctx context.Context, tenant uuid.UUID, f clients.Filter) (clients.Page, error) {
	args := []any{tenant}
	where := []string{"c.tenant_id=$1"}
	if f.Query != "" {
		args = append(args, f.Query)
		where = append(where, fmt.Sprintf("(c.name ILIKE '%%'||$%d||'%%' OR c.client_id::text ILIKE '%%'||$%d||'%%')", len(args), len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("c.status=$%d", len(args)))
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if e := r.pool.QueryRow(ctx, "SELECT count(*) FROM sys.api_clients c WHERE "+cond, args...).Scan(&total); e != nil {
		return clients.Page{}, e
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, e := r.pool.Query(ctx, selectClient+" WHERE "+cond+fmt.Sprintf(" ORDER BY c.updated_at DESC,c.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if e != nil {
		return clients.Page{}, e
	}
	defer rows.Close()
	items := []clients.Client{}
	for rows.Next() {
		c, se := scanClient(rows)
		if se != nil {
			return clients.Page{}, se
		}
		items = append(items, c)
	}
	return clients.Page{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}
func (r *Postgres) Get(ctx context.Context, tenant, id uuid.UUID) (clients.Client, error) {
	return r.get(ctx, r.pool, tenant, id)
}
func (r *Postgres) Create(ctx context.Context, p clients.Principal, clientID string, in clients.Input) (clients.Client, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return clients.Client{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	e = tx.QueryRow(ctx, `INSERT INTO sys.api_clients(tenant_id,client_id,name,description,allowed_cidrs,status,expires_at,created_by) VALUES($1,$2,$3,NULLIF($4,''),$5::cidr[],$6,$7,$8) RETURNING id`, p.TenantID, clientID, in.Name, in.Description, in.AllowedCIDRs, in.Status, in.ExpiresAt, p.UserID).Scan(&id)
	if unique(e) {
		return clients.Client{}, clients.ErrConflict
	}
	if e != nil {
		return clients.Client{}, e
	}
	if e = audit(ctx, tx, p, "sys.api_client.create", id, "POST", map[string]any{"client_id": clientID, "status": in.Status, "cidr_count": len(in.AllowedCIDRs)}); e != nil {
		return clients.Client{}, e
	}
	c, e := r.get(ctx, tx, p.TenantID, id)
	if e != nil {
		return c, e
	}
	return c, tx.Commit(ctx)
}
func (r *Postgres) Update(ctx context.Context, p clients.Principal, id uuid.UUID, in clients.Input) (clients.Client, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return clients.Client{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `UPDATE sys.api_clients SET name=$1,description=NULLIF($2,''),allowed_cidrs=$3::cidr[],status=$4,expires_at=$5 WHERE tenant_id=$6 AND id=$7`, in.Name, in.Description, in.AllowedCIDRs, in.Status, in.ExpiresAt, p.TenantID, id)
	if e != nil {
		return clients.Client{}, e
	}
	if tag.RowsAffected() != 1 {
		return clients.Client{}, clients.ErrNotFound
	}
	if e = audit(ctx, tx, p, "sys.api_client.update", id, "PATCH", map[string]any{"status": in.Status, "cidr_count": len(in.AllowedCIDRs)}); e != nil {
		return clients.Client{}, e
	}
	c, e := r.get(ctx, tx, p.TenantID, id)
	if e != nil {
		return c, e
	}
	return c, tx.Commit(ctx)
}
func (r *Postgres) CreateSecret(ctx context.Context, p clients.Principal, id uuid.UUID, prefix string, hash []byte, expires *time.Time) (clients.Secret, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return clients.Secret{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	var active int
	e = tx.QueryRow(ctx, `SELECT true,(SELECT count(*) FROM sys.api_client_secrets s WHERE s.api_client_id=c.id AND s.revoked_at IS NULL AND (s.expires_at IS NULL OR s.expires_at>now())) FROM sys.api_clients c WHERE c.tenant_id=$1 AND c.id=$2 FOR UPDATE`, p.TenantID, id).Scan(&exists, &active)
	if errors.Is(e, pgx.ErrNoRows) {
		return clients.Secret{}, clients.ErrNotFound
	}
	if e != nil {
		return clients.Secret{}, e
	}
	if active >= 2 {
		return clients.Secret{}, clients.ErrConflict
	}
	var s clients.Secret
	e = tx.QueryRow(ctx, `INSERT INTO sys.api_client_secrets(api_client_id,secret_prefix,secret_hash,expires_at) VALUES($1,$2,$3,$4) RETURNING id,secret_prefix,created_at,expires_at,revoked_at,last_used_at`, id, prefix, hash, expires).Scan(&s.ID, &s.Prefix, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt, &s.LastUsedAt)
	if e != nil {
		return s, e
	}
	if e = audit(ctx, tx, p, "sys.api_client.rotate_secret", id, "POST", map[string]any{"secret_id": s.ID, "prefix": s.Prefix, "expires_at": s.ExpiresAt}); e != nil {
		return s, e
	}
	return s, tx.Commit(ctx)
}
func (r *Postgres) RevokeSecret(ctx context.Context, p clients.Principal, id, sid uuid.UUID) error {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `UPDATE sys.api_client_secrets s SET revoked_at=now() FROM sys.api_clients c WHERE c.id=s.api_client_id AND c.tenant_id=$1 AND c.id=$2 AND s.id=$3 AND s.revoked_at IS NULL`, p.TenantID, id, sid)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return clients.ErrNotFound
	}
	if e = audit(ctx, tx, p, "sys.api_client.revoke_secret", id, "DELETE", map[string]any{"secret_id": sid}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (r *Postgres) ReplacePermissions(ctx context.Context, p clients.Principal, id uuid.UUID, codes []string) (clients.Client, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return clients.Client{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sys.api_clients WHERE tenant_id=$1 AND id=$2)`, p.TenantID, id).Scan(&exists); e != nil {
		return clients.Client{}, e
	}
	if !exists {
		return clients.Client{}, clients.ErrNotFound
	}
	var matched int
	if len(codes) > 0 {
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM iam.permissions WHERE status='active' AND code=ANY($1::citext[])`, codes).Scan(&matched); e != nil {
			return clients.Client{}, e
		}
		if matched != len(codes) {
			return clients.Client{}, clients.ErrInvalid
		}
	}
	if _, e = tx.Exec(ctx, `DELETE FROM sys.api_client_permissions WHERE tenant_id=$1 AND api_client_id=$2`, p.TenantID, id); e != nil {
		return clients.Client{}, e
	}
	if len(codes) > 0 {
		if _, e = tx.Exec(ctx, `INSERT INTO sys.api_client_permissions(tenant_id,api_client_id,permission_id) SELECT $1,$2,id FROM iam.permissions WHERE code=ANY($3::citext[])`, p.TenantID, id, codes); e != nil {
			return clients.Client{}, e
		}
	}
	if e = audit(ctx, tx, p, "sys.api_client.assign_permission", id, "PUT", map[string]any{"permission_codes": codes}); e != nil {
		return clients.Client{}, e
	}
	c, e := r.get(ctx, tx, p.TenantID, id)
	if e != nil {
		return c, e
	}
	return c, tx.Commit(ctx)
}
func (r *Postgres) Authenticate(ctx context.Context, cred clients.Credential) (clients.Client, error) {
	c, e := scanClient(r.pool.QueryRow(ctx, selectClient+` JOIN sys.api_client_secrets auth_secret ON auth_secret.api_client_id=c.id WHERE c.client_id=$1 AND c.status='active' AND (c.expires_at IS NULL OR c.expires_at>now()) AND auth_secret.secret_hash=$2 AND auth_secret.revoked_at IS NULL AND (auth_secret.expires_at IS NULL OR auth_secret.expires_at>now())`, cred.ClientID, cred.SecretHash))
	if errors.Is(e, pgx.ErrNoRows) {
		return c, clients.ErrCredential
	}
	if e != nil {
		return c, e
	}
	if len(c.AllowedCIDRs) > 0 {
		ip, pe := netip.ParseAddr(cred.IPAddress)
		if pe != nil {
			return clients.Client{}, clients.ErrCredential
		}
		allowed := false
		for _, raw := range c.AllowedCIDRs {
			prefix, _ := netip.ParsePrefix(raw)
			if prefix.Contains(ip) {
				allowed = true
				break
			}
		}
		if !allowed {
			return clients.Client{}, clients.ErrCredential
		}
	}
	_, e = r.pool.Exec(ctx, `UPDATE sys.api_client_secrets SET last_used_at=now() WHERE api_client_id=$1 AND secret_hash=$2`, c.ID, cred.SecretHash)
	return c, e
}
func audit(ctx context.Context, tx pgx.Tx, p clients.Principal, action string, id uuid.UUID, method string, after any) error {
	raw, e := json.Marshal(after)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,user_agent,after_data,succeeded) VALUES($1,$2,NULLIF($3,'00000000-0000-0000-0000-000000000000'::uuid),$4,'sys',$5,$5,'api_client',$6,$7,$8,200,NULLIF($9,''),$10,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, id.String(), method, "/admin-api/v1/api-clients/"+id.String(), strings.TrimSpace(p.UserAgent), raw)
	return e
}
func unique(e error) bool { return e != nil && strings.Contains(e.Error(), "SQLSTATE 23505") }
