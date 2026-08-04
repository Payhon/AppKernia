package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	blocks "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(p *pgxpool.Pool) *Postgres { return &Postgres{p} }

type scanner interface{ Scan(...any) error }

const selectRule = `SELECT id,subject_type,subject_value,action,COALESCE(reason,''),starts_at,expires_at,status,created_at,updated_at FROM iam.block_rules`

func hint(kind, v string) string {
	r := []rune(v)
	if len(r) <= 8 {
		return strings.Repeat("*", len(r))
	}
	switch kind {
	case "ip":
		p := strings.Split(v, ".")
		if len(p) == 4 {
			return strings.Join(p[:3], ".") + ".***"
		}
	case "cidr":
		if prefix, err := netip.ParsePrefix(v); err == nil {
			address := prefix.Addr().String()
			if x := strings.LastIndex(address, "."); x > 0 {
				return address[:x] + ".***/" + fmt.Sprint(prefix.Bits())
			}
		}
	}
	return string(r[:4]) + "***" + string(r[len(r)-4:])
}
func scan(row scanner) (blocks.Rule, error) {
	var v blocks.Rule
	var raw string
	e := row.Scan(&v.ID, &v.SubjectType, &raw, &v.Action, &v.Reason, &v.StartsAt, &v.ExpiresAt, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	v.SubjectHint = hint(v.SubjectType, raw)
	v.Scope = "tenant"
	return v, e
}
func (r *Postgres) List(ctx context.Context, tenant uuid.UUID, f blocks.Filter) (blocks.Page, error) {
	args := []any{tenant}
	where := []string{"tenant_id=$1"}
	add := func(sql string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(sql, len(args))) }
	if f.SubjectType != "" {
		add("subject_type=$%d", f.SubjectType)
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	switch f.Expiry {
	case "active":
		where = append(where, "(expires_at IS NULL OR expires_at>now())")
	case "expired":
		where = append(where, "expires_at<=now()")
	case "never":
		where = append(where, "expires_at IS NULL")
	}
	cond := strings.Join(where, " AND ")
	if f.SubjectHint != "" {
		rows, e := r.pool.Query(ctx, selectRule+" WHERE "+cond+" ORDER BY updated_at DESC,id DESC", args...)
		if e != nil {
			return blocks.Page{}, e
		}
		defer rows.Close()
		matches := []blocks.Rule{}
		needle := strings.ToLower(f.SubjectHint)
		for rows.Next() {
			v, scanErr := scan(rows)
			if scanErr != nil {
				return blocks.Page{}, scanErr
			}
			if strings.Contains(strings.ToLower(v.SubjectHint), needle) {
				matches = append(matches, v)
			}
		}
		if e = rows.Err(); e != nil {
			return blocks.Page{}, e
		}
		start := int((f.Page - 1) * f.PageSize)
		end := start + int(f.PageSize)
		if start > len(matches) {
			start = len(matches)
		}
		if end > len(matches) {
			end = len(matches)
		}
		return blocks.Page{Items: matches[start:end], Page: f.Page, PageSize: f.PageSize, Total: int64(len(matches))}, nil
	}
	var total int64
	if e := r.pool.QueryRow(ctx, "SELECT count(*) FROM iam.block_rules WHERE "+cond, args...).Scan(&total); e != nil {
		return blocks.Page{}, e
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, e := r.pool.Query(ctx, selectRule+" WHERE "+cond+fmt.Sprintf(" ORDER BY updated_at DESC,id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if e != nil {
		return blocks.Page{}, e
	}
	defer rows.Close()
	items := []blocks.Rule{}
	for rows.Next() {
		v, se := scan(rows)
		if se != nil {
			return blocks.Page{}, se
		}
		items = append(items, v)
	}
	return blocks.Page{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}
func (r *Postgres) Create(ctx context.Context, p blocks.Principal, in blocks.CreateInput) (blocks.Rule, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return blocks.Rule{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	e = tx.QueryRow(ctx, `INSERT INTO iam.block_rules(tenant_id,subject_type,subject_value,action,reason,starts_at,expires_at,status,created_by) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9) RETURNING id`, p.TenantID, in.SubjectType, in.SubjectValue, in.Action, in.Reason, in.StartsAt, in.ExpiresAt, in.Status, p.UserID).Scan(&id)
	if e != nil {
		return blocks.Rule{}, e
	}
	if e = audit(ctx, tx, p, "iam.block_rule.create", id, "POST", map[string]any{"subject_type": in.SubjectType, "subject_hint": hint(in.SubjectType, in.SubjectValue), "action": in.Action, "status": in.Status}); e != nil {
		return blocks.Rule{}, e
	}
	v, e := scan(tx.QueryRow(ctx, selectRule+` WHERE tenant_id=$1 AND id=$2`, p.TenantID, id))
	if e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}
func (r *Postgres) Update(ctx context.Context, p blocks.Principal, id uuid.UUID, in blocks.UpdateInput) (blocks.Rule, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return blocks.Rule{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `UPDATE iam.block_rules SET action=$1,reason=NULLIF($2,''),starts_at=$3,expires_at=$4,status=$5 WHERE tenant_id=$6 AND id=$7`, in.Action, in.Reason, in.StartsAt, in.ExpiresAt, in.Status, p.TenantID, id)
	if e != nil {
		return blocks.Rule{}, e
	}
	if tag.RowsAffected() != 1 {
		return blocks.Rule{}, blocks.ErrNotFound
	}
	if e = audit(ctx, tx, p, "iam.block_rule.update", id, "PATCH", map[string]any{"action": in.Action, "status": in.Status, "expires_at": in.ExpiresAt}); e != nil {
		return blocks.Rule{}, e
	}
	v, e := scan(tx.QueryRow(ctx, selectRule+` WHERE tenant_id=$1 AND id=$2`, p.TenantID, id))
	if e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}
func (r *Postgres) Revoke(ctx context.Context, p blocks.Principal, id uuid.UUID) (blocks.RevokeResult, error) {
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return blocks.RevokeResult{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var kind, raw, action string
	e = tx.QueryRow(ctx, `SELECT subject_type,subject_value,action FROM iam.block_rules WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, p.TenantID, id).Scan(&kind, &raw, &action)
	if errors.Is(e, pgx.ErrNoRows) {
		return blocks.RevokeResult{}, blocks.ErrNotFound
	}
	if e != nil {
		return blocks.RevokeResult{}, e
	}
	if e = audit(ctx, tx, p, "iam.block_rule.delete", id, "DELETE", map[string]any{"subject_type": kind, "subject_hint": hint(kind, raw), "action": action, "revoked": true}); e != nil {
		return blocks.RevokeResult{}, e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM iam.block_rules WHERE tenant_id=$1 AND id=$2`, p.TenantID, id); e != nil {
		return blocks.RevokeResult{}, e
	}
	return blocks.RevokeResult{ID: id, Revoked: true}, tx.Commit(ctx)
}
func audit(ctx context.Context, tx pgx.Tx, p blocks.Principal, action string, id uuid.UUID, method string, after any) error {
	raw, e := json.Marshal(after)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,user_agent,after_data,succeeded) VALUES($1,$2,NULLIF($3,'00000000-0000-0000-0000-000000000000'::uuid),$4,'iam',$5,$5,'block_rule',$6,$7,$8,200,NULLIF($9,''),$10,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, id.String(), method, "/admin-api/v1/block-rules/"+id.String(), strings.TrimSpace(p.UserAgent), raw)
	return e
}
