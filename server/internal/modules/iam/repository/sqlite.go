package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const sqliteTimeLayout = "2006-01-02T15:04:05.000000000Z"

type SQLite struct {
	db *sql.DB
}

func NewSQLite(database *sql.DB) *SQLite {
	return &SQLite{db: database}
}

var (
	_ domain.Repository        = (*SQLite)(nil)
	_ domain.SessionRepository = (*SQLite)(nil)
)

type BootstrapPermission struct {
	Code           string
	Name           string
	ModuleCode     string
	ResourceName   string
	ActionName     string
	PermissionKind string
	Status         string
}

type BootstrapMenu struct {
	Code         string
	I18nKey      string
	Title        string
	Type         string
	Path         *string
	Sort         int32
	Parent       *string
	ComponentKey *string
	Permission   *string
	Icon         *string
	Affix        bool
	FeatureFlag  string
}

type BootstrapAdminInput struct {
	TenantCode  string
	TenantName  string
	Email       string
	DisplayName string
	Locale      string
	Password    string
	Permissions []BootstrapPermission
	Menus       []BootstrapMenu
}

type BootstrapAdminResult struct {
	User               domain.User
	Tenant             domain.Tenant
	GrantedPermissions int64
	GrantedMenus       int64
}

var defaultBootstrapPermissions = []BootstrapPermission{
	{Code: "iam.user.read", Name: "Read users", ModuleCode: "iam", ResourceName: "user", ActionName: "read"},
	{Code: "iam.session.read", Name: "Read sessions", ModuleCode: "iam", ResourceName: "session", ActionName: "read"},
	{Code: "jobs.run.read", Name: "Read job runs", ModuleCode: "jobs", ResourceName: "run", ActionName: "read"},
	{Code: "audit.security.read", Name: "Read security events", ModuleCode: "audit", ResourceName: "security", ActionName: "read"},
	{Code: "notify.notice.read", Name: "Read notices", ModuleCode: "notify", ResourceName: "notice", ActionName: "read"},
	{Code: "audit.login.read", Name: "Read login events", ModuleCode: "audit", ResourceName: "login", ActionName: "read"},
	{Code: "audit.operation.read", Name: "Read operation logs", ModuleCode: "audit", ResourceName: "operation", ActionName: "read"},
}

func defaultBootstrapMenus() []BootstrapMenu {
	path, component, icon := "/dashboard", "dashboard", "DashboardOutlined"
	return []BootstrapMenu{{
		Code: "dashboard", I18nKey: "menu.dashboard", Title: "Dashboard", Type: "page",
		Path: &path, ComponentKey: &component, Icon: &icon, Sort: 10, Affix: true,
	}}
}

func (repository *SQLite) BootstrapAdmin(ctx context.Context, raw BootstrapAdminInput) (BootstrapAdminResult, error) {
	identity := domain.CreateIdentity{
		TenantCode: raw.TenantCode, TenantName: raw.TenantName, Email: raw.Email,
		DisplayName: raw.DisplayName, Locale: raw.Locale,
	}.Normalize()
	if identity.TenantCode == "" || identity.TenantName == "" || identity.Email == "" || identity.DisplayName == "" ||
		(identity.Locale != "zh-CN" && identity.Locale != "en-US") {
		return BootstrapAdminResult{}, errors.New("bootstrap administrator identity is incomplete")
	}
	permissions := raw.Permissions
	if len(permissions) == 0 {
		permissions = defaultBootstrapPermissions
	}
	menus := raw.Menus
	if len(menus) == 0 {
		menus = defaultBootstrapMenus()
	}

	tx, err := repository.begin(ctx)
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("begin bootstrap administrator transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())

	tenant, err := resolveBootstrapTenant(ctx, tx, identity, now)
	if err != nil {
		return BootstrapAdminResult{}, err
	}
	user, err := resolveBootstrapUser(ctx, tx, tenant.ID, identity, raw.Password, now)
	if err != nil {
		return BootstrapAdminResult{}, err
	}
	if err = seedBootstrapPermissions(ctx, tx, permissions, now); err != nil {
		return BootstrapAdminResult{}, err
	}
	if err = seedBootstrapMenus(ctx, tx, menus, now); err != nil {
		return BootstrapAdminResult{}, err
	}

	if _, err = ensureRole(ctx, tx, tenant.ID, "member", "Member", "Built-in member role", "self", true, now); err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("upsert member role: %w", err)
	}
	roleID, err := ensureRole(ctx, tx, tenant.ID, "super-admin", "Super Administrator", "Core bootstrap administrator role", "tenant", false, now)
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("upsert bootstrap role: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO iam_user_roles
(tenant_id,user_id,role_id,valid_from,granted_by) VALUES(?,?,?,?,?)`, tenant.ID.String(), user.ID.String(), roleID, now, user.ID.String()); err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("assign bootstrap role: %w", err)
	}
	permissionGrant, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO iam_role_permissions(tenant_id,role_id,permission_id,granted_by)
SELECT ?,?,id,? FROM iam_permissions WHERE status='active'`, tenant.ID.String(), roleID, user.ID.String())
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("grant bootstrap permissions: %w", err)
	}
	menuGrant, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO sys_role_menus(tenant_id,role_id,menu_id)
SELECT ?,?,id FROM sys_menus WHERE tenant_id IS NULL AND deleted_at IS NULL AND status='active'`, tenant.ID.String(), roleID)
	if err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("grant bootstrap menus: %w", err)
	}
	grantedPermissions, _ := permissionGrant.RowsAffected()
	grantedMenus, _ := menuGrant.RowsAffected()
	if err = tx.Commit(); err != nil {
		return BootstrapAdminResult{}, fmt.Errorf("commit bootstrap administrator: %w", err)
	}
	return BootstrapAdminResult{User: user, Tenant: tenant, GrantedPermissions: grantedPermissions, GrantedMenus: grantedMenus}, nil
}

func resolveBootstrapTenant(ctx context.Context, tx *sql.Tx, identity domain.CreateIdentity, now string) (domain.Tenant, error) {
	var id, code, name, status string
	err := tx.QueryRowContext(ctx, `SELECT id,code,name,status FROM iam_tenants
WHERE code=? AND deleted_at IS NULL`, identity.TenantCode).Scan(&id, &code, &name, &status)
	if errors.Is(err, sql.ErrNoRows) {
		value := uuid.New()
		if _, err = tx.ExecContext(ctx, `INSERT INTO iam_tenants(id,code,name,status,created_at,updated_at)
VALUES(?,?,?,'active',?,?)`, value.String(), identity.TenantCode, identity.TenantName, now, now); err != nil {
			return domain.Tenant{}, fmt.Errorf("create bootstrap tenant: %w", err)
		}
		return domain.Tenant{ID: value, Code: identity.TenantCode, Name: identity.TenantName, Status: "active"}, nil
	}
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("resolve bootstrap tenant: %w", err)
	}
	if status != "active" {
		return domain.Tenant{}, errors.New("bootstrap tenant is not active")
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("decode bootstrap tenant ID: %w", err)
	}
	return domain.Tenant{ID: parsed, Code: strings.ToLower(code), Name: name, Status: status}, nil
}

func resolveBootstrapUser(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, identity domain.CreateIdentity, password, now string) (domain.User, error) {
	user, err := queryUser(ctx, tx, `SELECT id,email,mobile,display_name,locale,time_zone,status,avatar_file_id
FROM iam_users WHERE email=? AND deleted_at IS NULL`, identity.Email)
	if errors.Is(err, sql.ErrNoRows) {
		passwordHash, hashErr := application.HashPassword(password)
		if hashErr != nil {
			return domain.User{}, hashErr
		}
		user = domain.User{ID: uuid.New(), Email: identity.Email, DisplayName: identity.DisplayName, Locale: identity.Locale, TimeZone: "UTC", Status: "active"}
		if _, err = tx.ExecContext(ctx, `INSERT INTO iam_users
(id,email,display_name,locale,time_zone,status,created_at,updated_at) VALUES(?,?,?,?,?,'active',?,?)`,
			user.ID.String(), user.Email, user.DisplayName, user.Locale, user.TimeZone, now, now); err != nil {
			return domain.User{}, classifySQLiteCreateError(err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO iam_user_credentials
(user_id,password_hash,password_version,password_updated_at,created_at,updated_at) VALUES(?,?,1,?,?,?)`,
			user.ID.String(), passwordHash, now, now, now); err != nil {
			return domain.User{}, fmt.Errorf("create bootstrap credential: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO iam_tenant_members
(tenant_id,user_id,display_name,status,joined_at,created_at,updated_at) VALUES(?,?,?,'active',?,?,?)`,
			tenantID.String(), user.ID.String(), user.DisplayName, now, now, now); err != nil {
			return domain.User{}, fmt.Errorf("create bootstrap membership: %w", err)
		}
		return user, nil
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("resolve bootstrap user: %w", err)
	}
	if user.Status != "active" {
		return domain.User{}, errors.New("bootstrap user is not active")
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM iam_user_credentials WHERE user_id=?`, user.ID.String()).Scan(&count); err != nil || count != 1 {
		return domain.User{}, errors.New("bootstrap user credential is unavailable")
	}
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM iam_tenant_members
WHERE tenant_id=? AND user_id=? AND status='active'`, tenantID.String(), user.ID.String()).Scan(&count); err != nil || count != 1 {
		return domain.User{}, errors.New("bootstrap user is not an active member of the target tenant")
	}
	return user, nil
}

func seedBootstrapPermissions(ctx context.Context, tx *sql.Tx, permissions []BootstrapPermission, now string) error {
	seen := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		permission.Code = strings.ToLower(strings.TrimSpace(permission.Code))
		permission.Name = strings.TrimSpace(permission.Name)
		permission.ModuleCode = strings.TrimSpace(permission.ModuleCode)
		permission.ResourceName = strings.TrimSpace(permission.ResourceName)
		permission.ActionName = strings.TrimSpace(permission.ActionName)
		if permission.PermissionKind == "" {
			permission.PermissionKind = "api"
		}
		if permission.Status == "" {
			permission.Status = "active"
		}
		if permission.Code == "" || permission.Name == "" || permission.ModuleCode == "" || permission.ResourceName == "" || permission.ActionName == "" {
			return errors.New("bootstrap permission is incomplete")
		}
		if _, duplicate := seen[permission.Code]; duplicate {
			return fmt.Errorf("duplicate bootstrap permission %s", permission.Code)
		}
		seen[permission.Code] = struct{}{}
		if _, err := tx.ExecContext(ctx, `INSERT INTO iam_permissions
(id,code,name,module_code,resource_name,action_name,permission_kind,status,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(code) DO UPDATE SET name=excluded.name,module_code=excluded.module_code,
resource_name=excluded.resource_name,action_name=excluded.action_name,permission_kind=excluded.permission_kind,
status=excluded.status,updated_at=excluded.updated_at`, uuid.NewString(), permission.Code, permission.Name,
			permission.ModuleCode, permission.ResourceName, permission.ActionName, permission.PermissionKind,
			permission.Status, now, now); err != nil {
			return fmt.Errorf("upsert bootstrap permission %s: %w", permission.Code, err)
		}
	}
	return nil
}

func seedBootstrapMenus(ctx context.Context, tx *sql.Tx, menus []BootstrapMenu, now string) error {
	remaining := append([]BootstrapMenu(nil), menus...)
	ids := make(map[string]string, len(menus))
	for len(remaining) > 0 {
		pending := remaining[:0]
		progress := false
		for _, menu := range remaining {
			menu.Code = strings.ToLower(strings.TrimSpace(menu.Code))
			if menu.Code == "" || strings.TrimSpace(menu.I18nKey) == "" || strings.TrimSpace(menu.Title) == "" || strings.TrimSpace(menu.Type) == "" {
				return errors.New("bootstrap menu is incomplete")
			}
			var parentID any
			if menu.Parent != nil {
				resolved, ok := ids[strings.ToLower(strings.TrimSpace(*menu.Parent))]
				if !ok {
					pending = append(pending, menu)
					continue
				}
				parentID = resolved
			}
			var permissionID any
			if menu.Permission != nil {
				var resolved string
				if err := tx.QueryRowContext(ctx, `SELECT id FROM iam_permissions WHERE code=? AND status='active'`, strings.TrimSpace(*menu.Permission)).Scan(&resolved); errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("menu %s references unknown permission %s", menu.Code, *menu.Permission)
				} else if err != nil {
					return fmt.Errorf("resolve menu %s permission: %w", menu.Code, err)
				}
				permissionID = resolved
			}
			var id string
			err := tx.QueryRowContext(ctx, `SELECT id FROM sys_menus WHERE tenant_id IS NULL AND code=? AND deleted_at IS NULL`, menu.Code).Scan(&id)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				id = uuid.NewString()
				_, err = tx.ExecContext(ctx, `INSERT INTO sys_menus
(id,tenant_id,parent_id,permission_id,code,i18n_key,title,menu_type,route_path,component_key,icon,affix,sort_order,feature_flag,status,created_at,updated_at)
VALUES(?,NULL,?,?,?,?,?,?,?,?,?,?,?,?, 'active',?,?)`, id, parentID, permissionID, menu.Code,
					menu.I18nKey, menu.Title, menu.Type, stringPointerValue(menu.Path), stringPointerValue(menu.ComponentKey),
					stringPointerValue(menu.Icon), boolInt(menu.Affix), menu.Sort, menu.FeatureFlag, now, now)
			case err == nil:
				_, err = tx.ExecContext(ctx, `UPDATE sys_menus SET parent_id=?,permission_id=?,i18n_key=?,title=?,menu_type=?,
route_path=?,component_key=?,icon=?,affix=?,sort_order=?,feature_flag=?,status='active',updated_at=? WHERE id=?`,
					parentID, permissionID, menu.I18nKey, menu.Title, menu.Type, stringPointerValue(menu.Path),
					stringPointerValue(menu.ComponentKey), stringPointerValue(menu.Icon), boolInt(menu.Affix), menu.Sort,
					menu.FeatureFlag, now, id)
			}
			if err != nil {
				return fmt.Errorf("upsert bootstrap menu %s: %w", menu.Code, err)
			}
			ids[menu.Code] = id
			progress = true
		}
		if !progress {
			return errors.New("bootstrap menu parents contain a cycle or unknown code")
		}
		remaining = pending
	}
	return nil
}

func ensureRole(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, code, name, description, scope string, isDefault bool, now string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM iam_roles WHERE tenant_id=? AND code=? AND deleted_at IS NULL`, tenantID.String(), code).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		id = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO iam_roles
(id,tenant_id,code,name,description,role_type,data_scope,is_default,is_system,status,created_at,updated_at)
VALUES(?,?,?,?,?,'system',?,?,1,'active',?,?)`, id, tenantID.String(), code, name, description, scope, boolInt(isDefault), now, now)
	} else if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE iam_roles SET name=?,description=?,data_scope=?,is_default=?,is_system=1,status='active',updated_at=? WHERE id=?`,
			name, description, scope, boolInt(isDefault), now, id)
	}
	return id, err
}

func (repository *SQLite) CreateIdentity(ctx context.Context, raw domain.CreateIdentity) (domain.User, domain.Tenant, error) {
	input := raw.Normalize()
	tx, err := repository.begin(ctx)
	if err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("begin identity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())
	tenant := domain.Tenant{ID: uuid.New(), Code: input.TenantCode, Name: input.TenantName, Status: "active"}
	user := domain.User{ID: uuid.New(), Email: input.Email, DisplayName: input.DisplayName, Locale: input.Locale, TimeZone: "UTC", Status: "active"}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_tenants(id,code,name,status,created_at,updated_at) VALUES(?,?,?,'active',?,?)`,
		tenant.ID.String(), tenant.Code, tenant.Name, now, now); err != nil {
		return domain.User{}, domain.Tenant{}, classifySQLiteCreateError(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_users(id,email,display_name,locale,time_zone,status,created_at,updated_at)
VALUES(?,?,?,?,?,'active',?,?)`, user.ID.String(), user.Email, user.DisplayName, user.Locale, user.TimeZone, now, now); err != nil {
		return domain.User{}, domain.Tenant{}, classifySQLiteCreateError(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_user_credentials
(user_id,password_hash,password_version,password_updated_at,created_at,updated_at) VALUES(?,?,1,?,?,?)`, user.ID.String(), input.PasswordHash, now, now, now); err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("create credential: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_tenant_members
(tenant_id,user_id,display_name,status,joined_at,created_at,updated_at) VALUES(?,?,?,'active',?,?,?)`, tenant.ID.String(), user.ID.String(), user.DisplayName, now, now, now); err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("create tenant membership: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return domain.User{}, domain.Tenant{}, classifySQLiteCreateError(err)
	}
	return user, tenant, nil
}

func (repository *SQLite) RegisterAdmin(ctx context.Context, raw domain.RegisterAdmin) error {
	input := domain.CreateIdentity{TenantCode: raw.TenantCode, Email: raw.Email, DisplayName: raw.DisplayName, Locale: raw.Locale}.Normalize()
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin admin registration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var tenantID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM iam_tenants WHERE code=? AND status='active' AND deleted_at IS NULL`, input.TenantCode).Scan(&tenantID); errors.Is(err, sql.ErrNoRows) {
		return domain.ErrRegistrationTenant
	} else if err != nil {
		return fmt.Errorf("get registration tenant: %w", err)
	}
	now := sqliteTime(time.Now())
	userID := uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_users(id,email,display_name,locale,time_zone,status,metadata,created_at,updated_at)
VALUES(?,?,?,?,?,'active',?,?,?)`, userID, input.Email, input.DisplayName, input.Locale, "UTC", `{"registration_source":"admin_self_service"}`, now, now); err != nil {
		return classifySQLiteCreateError(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_user_credentials
(user_id,password_hash,password_version,password_updated_at,created_at,updated_at) VALUES(?,?,1,?,?,?)`, userID, raw.PasswordHash, now, now, now); err != nil {
		return fmt.Errorf("create registered credential: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_tenant_members
(tenant_id,user_id,display_name,status,joined_at,created_at,updated_at) VALUES(?,?,?,'active',?,?,?)`, tenantID, userID, input.DisplayName, now, now, now); err != nil {
		return fmt.Errorf("create registered membership: %w", err)
	}
	var roleID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM iam_roles WHERE tenant_id=? AND is_default=1 AND status='active' AND deleted_at IS NULL ORDER BY code LIMIT 1`, tenantID).Scan(&roleID); errors.Is(err, sql.ErrNoRows) {
		return domain.ErrRegistrationTenant
	} else if err != nil {
		return fmt.Errorf("get registration role: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO iam_user_roles(tenant_id,user_id,role_id,valid_from) VALUES(?,?,?,?)`, tenantID, userID, roleID, now); err != nil {
		return fmt.Errorf("assign registration role: %w", err)
	}
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: tenantID, UserID: userID, RequestID: raw.RequestID, ModuleCode: "iam",
		ActionName: "iam.auth.register", ResourceType: "iam.user", ResourceID: userID,
		RequestPath: "/admin-api/v1/auth/register", IPAddress: raw.IPAddress, UserAgent: raw.UserAgent,
	}); err != nil {
		return fmt.Errorf("audit admin registration: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return classifySQLiteCreateError(err)
	}
	return nil
}

func (repository *SQLite) FindCredentialByEmail(ctx context.Context, email string) (domain.Credential, error) {
	return repository.findCredential(ctx, `SELECT u.id,u.email,u.mobile,u.display_name,u.locale,u.time_zone,u.status,u.avatar_file_id,c.password_hash
FROM iam_users u JOIN iam_user_credentials c ON c.user_id=u.id
WHERE u.email=? AND u.deleted_at IS NULL`, strings.ToLower(strings.TrimSpace(email)))
}

func (repository *SQLite) FindCredentialByAppIdentifier(ctx context.Context, appID uuid.UUID, identifierType, value string) (domain.Credential, error) {
	return repository.findCredential(ctx, `SELECT u.id,u.email,u.mobile,u.display_name,u.locale,u.time_zone,u.status,u.avatar_file_id,c.password_hash
FROM app_user_login_identifiers i
JOIN app_user_memberships m ON m.app_id=i.app_id AND m.user_id=i.user_id AND m.tenant_id=i.tenant_id AND m.status='active'
JOIN iam_users u ON u.id=i.user_id AND u.deleted_at IS NULL
JOIN iam_user_credentials c ON c.user_id=u.id
WHERE i.app_id=? AND i.identifier_type=? AND i.normalized_value=? AND i.status='active' AND i.verified_at IS NOT NULL`,
		appID.String(), strings.ToLower(strings.TrimSpace(identifierType)), strings.ToLower(strings.TrimSpace(value)))
}

func (repository *SQLite) findCredential(ctx context.Context, query string, args ...any) (domain.Credential, error) {
	var id string
	var email, mobile, avatar sql.NullString
	var user domain.User
	var passwordHash string
	err := repository.db.QueryRowContext(ctx, query, args...).Scan(&id, &email, &mobile, &user.DisplayName, &user.Locale, &user.TimeZone, &user.Status, &avatar, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Credential{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Credential{}, fmt.Errorf("find credential: %w", err)
	}
	if err = decodeUserIDs(&user, id, avatar); err != nil {
		return domain.Credential{}, err
	}
	user.Email, user.Mobile = email.String, mobile.String
	return domain.Credential{User: user, PasswordHash: passwordHash}, nil
}

func (repository *SQLite) AppProfileIdentifiers(ctx context.Context, appID, userID uuid.UUID) (string, string, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT identifier_type,normalized_value FROM app_user_login_identifiers
WHERE app_id=? AND user_id=? AND status='active' AND verified_at IS NOT NULL ORDER BY identifier_type`, appID.String(), userID.String())
	if err != nil {
		return "", "", fmt.Errorf("load app profile identifiers: %w", err)
	}
	defer rows.Close()
	var email, mobile string
	for rows.Next() {
		var kind, value string
		if err = rows.Scan(&kind, &value); err != nil {
			return "", "", fmt.Errorf("scan app profile identifier: %w", err)
		}
		switch kind {
		case "email":
			email = value
		case "mobile":
			mobile = value
		}
	}
	return email, mobile, rows.Err()
}

func (repository *SQLite) ResolveActiveMobileAppMembership(ctx context.Context, appID, userID uuid.UUID) (domain.Tenant, error) {
	var id, code, name, status string
	err := repository.db.QueryRowContext(ctx, `SELECT t.id,t.code,t.name,t.status FROM app_user_memberships m
JOIN app_applications a ON a.id=m.app_id AND a.tenant_id=m.tenant_id AND a.status='active'
JOIN iam_tenants t ON t.id=m.tenant_id AND t.status='active' AND t.deleted_at IS NULL
JOIN iam_users u ON u.id=m.user_id AND u.status='active' AND u.deleted_at IS NULL
WHERE m.app_id=? AND m.user_id=? AND m.status='active'`, appID.String(), userID.String()).Scan(&id, &code, &name, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Tenant{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("resolve mobile app membership: %w", err)
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("decode mobile tenant ID: %w", err)
	}
	return domain.Tenant{ID: parsed, Code: code, Name: name, Status: status}, nil
}

func (repository *SQLite) ListUserTenants(ctx context.Context, userID uuid.UUID) ([]domain.Tenant, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT t.id,t.code,t.name,t.status FROM iam_tenant_members m
JOIN iam_tenants t ON t.id=m.tenant_id AND t.deleted_at IS NULL AND t.status='active'
WHERE m.user_id=? AND m.status='active' ORDER BY t.name,t.id`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list user tenants: %w", err)
	}
	defer rows.Close()
	result := make([]domain.Tenant, 0)
	for rows.Next() {
		var value domain.Tenant
		var id string
		if err = rows.Scan(&id, &value.Code, &value.Name, &value.Status); err != nil {
			return nil, fmt.Errorf("scan user tenant: %w", err)
		}
		if value.ID, err = uuid.Parse(id); err != nil {
			return nil, fmt.Errorf("decode user tenant ID: %w", err)
		}
		result = append(result, value)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list user tenants: %w", err)
	}
	return result, nil
}

func (repository *SQLite) UpdateSelfProfile(ctx context.Context, input domain.UpdateSelfProfile) (domain.User, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin self profile transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	before, err := queryUser(ctx, tx, `SELECT id,email,mobile,display_name,locale,time_zone,status,avatar_file_id
FROM iam_users WHERE id=? AND deleted_at IS NULL`, input.UserID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get self profile: %w", err)
	}
	now := sqliteTime(time.Now())
	result, err := tx.ExecContext(ctx, `UPDATE iam_users SET display_name=COALESCE(?,display_name),locale=COALESCE(?,locale),
time_zone=COALESCE(?,time_zone),updated_at=? WHERE id=? AND deleted_at IS NULL`,
		stringPointerValue(input.DisplayName), stringPointerValue(input.Locale), stringPointerValue(input.TimeZone), now, input.UserID.String())
	if err != nil {
		return domain.User{}, fmt.Errorf("update self profile: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.User{}, domain.ErrIdentityNotFound
	}
	after, err := queryUser(ctx, tx, `SELECT id,email,mobile,display_name,locale,time_zone,status,avatar_file_id
FROM iam_users WHERE id=? AND deleted_at IS NULL`, input.UserID.String())
	if err != nil {
		return domain.User{}, fmt.Errorf("get updated self profile: %w", err)
	}
	beforeJSON, _ := json.Marshal(map[string]string{"display_name": before.DisplayName, "locale": before.Locale, "time_zone": before.TimeZone})
	afterJSON, _ := json.Marshal(map[string]string{"display_name": after.DisplayName, "locale": after.Locale, "time_zone": after.TimeZone})
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: input.TenantID.String(), UserID: input.UserID.String(), SessionID: input.SessionID.String(),
		RequestID: input.RequestID, ModuleCode: "iam", ActionName: "iam.me.profile.update",
		ResourceType: "iam.user", ResourceID: input.UserID.String(), RequestPath: "/admin-api/v1/me/profile",
		IPAddress: input.IPAddress, UserAgent: input.UserAgent, BeforeData: string(beforeJSON), AfterData: string(afterJSON),
	}); err != nil {
		return domain.User{}, fmt.Errorf("audit self profile update: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return domain.User{}, fmt.Errorf("commit self profile update: %w", err)
	}
	return after, nil
}

func (repository *SQLite) GetSelfPasswordState(ctx context.Context, userID uuid.UUID) (domain.SelfPasswordState, error) {
	var state domain.SelfPasswordState
	err := repository.db.QueryRowContext(ctx, `SELECT password_hash,password_version FROM iam_user_credentials WHERE user_id=?`, userID.String()).
		Scan(&state.CurrentHash, &state.CurrentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SelfPasswordState{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.SelfPasswordState{}, fmt.Errorf("get self password state: %w", err)
	}
	state.HistoryHashes, err = listPasswordHistory(ctx, repository.db, userID)
	if err != nil {
		return domain.SelfPasswordState{}, err
	}
	return state, nil
}

func (repository *SQLite) ChangeSelfPassword(ctx context.Context, input domain.ChangeSelfPassword) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin self password transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())
	result, err := tx.ExecContext(ctx, `UPDATE iam_user_credentials SET password_hash=?,password_version=password_version+1,
password_updated_at=?,updated_at=? WHERE user_id=? AND password_hash=? AND password_version=?`, input.NewHash, now, now,
		input.UserID.String(), input.ExpectedHash, input.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("update self password: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.ErrPasswordChanged
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_password_history(id,user_id,password_hash,created_at) VALUES(?,?,?,?)`,
		uuid.NewString(), input.UserID.String(), input.ExpectedHash, now); err != nil {
		return fmt.Errorf("insert password history: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?)
WHERE revoked_at IS NULL AND session_id IN(SELECT id FROM iam_sessions WHERE user_id=? AND id<>?)`,
		now, input.UserID.String(), input.SessionID.String()); err != nil {
		return fmt.Errorf("revoke other session refresh tokens: %w", err)
	}
	revoked, err := tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=COALESCE(revoked_at,?),
revoke_reason='password_changed',access_token_version=access_token_version+1,updated_at=?
WHERE user_id=? AND id<>? AND status='active' AND revoked_at IS NULL`, now, now, input.UserID.String(), input.SessionID.String())
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}
	revokedCount, _ := revoked.RowsAffected()
	afterVersion := input.ExpectedVersion + 1
	beforeJSON, _ := json.Marshal(map[string]any{"password_version": input.ExpectedVersion})
	afterJSON, _ := json.Marshal(map[string]any{"password_version": afterVersion, "other_sessions_revoked": revokedCount})
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: input.TenantID.String(), UserID: input.UserID.String(), SessionID: input.SessionID.String(),
		RequestID: input.RequestID, ModuleCode: "iam", ActionName: "iam.me.password.change",
		ResourceType: "iam.user_credential", ResourceID: input.UserID.String(), RequestPath: "/admin-api/v1/me/password/change",
		IPAddress: input.IPAddress, UserAgent: input.UserAgent, BeforeData: string(beforeJSON), AfterData: string(afterJSON),
	}); err != nil {
		return fmt.Errorf("audit self password change: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit self password change: %w", err)
	}
	return nil
}

func (repository *SQLite) PreparePasswordReset(ctx context.Context, input domain.PreparePasswordReset) (*domain.PasswordResetRecipient, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin password reset challenge transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var userID, email, locale, status string
	err = tx.QueryRowContext(ctx, `SELECT u.id,COALESCE(u.email,''),u.locale,u.status FROM iam_users u
JOIN iam_user_credentials c ON c.user_id=u.id WHERE u.email=? AND u.deleted_at IS NULL`, strings.ToLower(strings.TrimSpace(input.Email))).
		Scan(&userID, &email, &locale, &status)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && status != "active") {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find password reset identity: %w", err)
	}
	var tenantID string
	if err = tx.QueryRowContext(ctx, `SELECT tenant_id FROM iam_tenant_members WHERE user_id=? AND status='active'
ORDER BY created_at,tenant_id LIMIT 1`, userID).Scan(&tenantID); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("resolve password reset tenant: %w", err)
	}
	now := time.Now().UTC()
	var recent int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM iam_verification_challenges
WHERE target_hash=? AND challenge_type='password_reset' AND consumed_at IS NULL AND expires_at>? AND created_at>?`,
		input.TargetHash, sqliteTime(now), sqliteTime(now.Add(-time.Minute))).Scan(&recent); err != nil {
		return nil, fmt.Errorf("check password reset cooldown: %w", err)
	}
	if recent > 0 {
		return nil, nil
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_verification_challenges
(id,challenge_type,user_id,target_hash,secret_hash,expires_at,created_ip,created_at) VALUES(?,'password_reset',?,?,?,?,?,?)`,
		uuid.NewString(), userID, input.TargetHash, input.SecretHash, sqliteTime(input.ExpiresAt), ipValue(input.IPAddress), sqliteTime(now)); err != nil {
		return nil, fmt.Errorf("create password reset challenge: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit password reset challenge: %w", err)
	}
	parsedTenant, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("decode password reset tenant ID: %w", err)
	}
	return &domain.PasswordResetRecipient{TenantID: parsedTenant, Email: email, Locale: locale}, nil
}

func (repository *SQLite) GetPasswordResetState(ctx context.Context, tokenHash []byte) (domain.PasswordResetState, error) {
	var state domain.PasswordResetState
	var userID string
	err := repository.db.QueryRowContext(ctx, `SELECT vc.user_id,c.password_hash,c.password_version
FROM iam_verification_challenges vc JOIN iam_user_credentials c ON c.user_id=vc.user_id
JOIN iam_users u ON u.id=vc.user_id WHERE vc.secret_hash=? AND vc.challenge_type='password_reset'
AND vc.consumed_at IS NULL AND vc.expires_at>? AND vc.attempt_count<5 AND u.status='active' AND u.deleted_at IS NULL
ORDER BY vc.created_at DESC LIMIT 1`, tokenHash, sqliteTime(time.Now())).Scan(&userID, &state.CurrentHash, &state.CurrentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PasswordResetState{}, domain.ErrResetTokenInvalid
	}
	if err != nil {
		return domain.PasswordResetState{}, fmt.Errorf("get password reset state: %w", err)
	}
	state.UserID, err = uuid.Parse(userID)
	if err != nil {
		return domain.PasswordResetState{}, fmt.Errorf("decode password reset user ID: %w", err)
	}
	state.HistoryHashes, err = listPasswordHistory(ctx, repository.db, state.UserID)
	if err != nil {
		return domain.PasswordResetState{}, err
	}
	return state, nil
}

func (repository *SQLite) ResetPassword(ctx context.Context, input domain.ResetPassword) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password reset transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())
	var challengeID, userID string
	err = tx.QueryRowContext(ctx, `SELECT id,user_id FROM iam_verification_challenges WHERE secret_hash=?
AND challenge_type='password_reset' AND consumed_at IS NULL AND expires_at>? AND attempt_count<5`, input.TokenHash, now).Scan(&challengeID, &userID)
	if errors.Is(err, sql.ErrNoRows) || userID != input.UserID.String() {
		return domain.ErrResetTokenInvalid
	}
	if err != nil {
		return fmt.Errorf("lock password reset challenge: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE iam_user_credentials SET password_hash=?,password_version=password_version+1,
password_updated_at=?,updated_at=? WHERE user_id=? AND password_hash=? AND password_version=?`, input.NewHash, now, now,
		input.UserID.String(), input.ExpectedHash, input.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("update reset password: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.ErrPasswordChanged
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_password_history(id,user_id,password_hash,created_at) VALUES(?,?,?,?)`,
		uuid.NewString(), input.UserID.String(), input.ExpectedHash, now); err != nil {
		return fmt.Errorf("insert reset password history: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?)
WHERE revoked_at IS NULL AND session_id IN(SELECT id FROM iam_sessions WHERE user_id=?)`, now, input.UserID.String()); err != nil {
		return fmt.Errorf("revoke reset refresh tokens: %w", err)
	}
	revoked, err := tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=COALESCE(revoked_at,?),
revoke_reason='password_reset',access_token_version=access_token_version+1,updated_at=?
WHERE user_id=? AND status='active' AND revoked_at IS NULL`, now, now, input.UserID.String())
	if err != nil {
		return fmt.Errorf("revoke reset sessions: %w", err)
	}
	revokedCount, _ := revoked.RowsAffected()
	if _, err = tx.ExecContext(ctx, `UPDATE iam_verification_challenges SET consumed_at=COALESCE(consumed_at,?)
WHERE user_id=? AND challenge_type='password_reset' AND consumed_at IS NULL`, now, input.UserID.String()); err != nil {
		return fmt.Errorf("consume password reset challenges: %w", err)
	}
	var tenantID string
	_ = tx.QueryRowContext(ctx, `SELECT tenant_id FROM iam_tenant_members WHERE user_id=? AND status='active' ORDER BY created_at,tenant_id LIMIT 1`, input.UserID.String()).Scan(&tenantID)
	beforeJSON, _ := json.Marshal(map[string]any{"password_version": input.ExpectedVersion})
	afterJSON, _ := json.Marshal(map[string]any{"password_version": input.ExpectedVersion + 1, "sessions_revoked": revokedCount})
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: tenantID, UserID: input.UserID.String(), RequestID: input.RequestID,
		ModuleCode: "iam", ActionName: "iam.auth.password.reset", ResourceType: "iam.user_credential",
		ResourceID: input.UserID.String(), RequestPath: "/admin-api/v1/auth/password/reset",
		IPAddress: input.IPAddress, UserAgent: input.UserAgent, BeforeData: string(beforeJSON), AfterData: string(afterJSON),
	}); err != nil {
		return fmt.Errorf("audit password reset: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit password reset: %w", err)
	}
	return nil
}

func (repository *SQLite) LoginCaptchaRequired(ctx context.Context, scopeHash []byte, now time.Time) (bool, error) {
	var count int32
	err := repository.db.QueryRowContext(ctx, `SELECT failure_count FROM iam_login_failure_states
WHERE scope_hash=? AND expires_at>?`, scopeHash, sqliteTime(now)).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get login failure state: %w", err)
	}
	return count >= 3, nil
}

func (repository *SQLite) CheckLoginCaptchaGeneration(ctx context.Context, scopeHash []byte, now time.Time) error {
	required, err := repository.LoginCaptchaRequired(ctx, scopeHash, now)
	if err != nil {
		return fmt.Errorf("check login captcha generation: %w", err)
	}
	if !required {
		return domain.ErrLoginCaptchaNotRequired
	}
	var coolingDown int
	if err = repository.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM iam_interactive_captcha_challenges
WHERE scope_hash=? AND created_at>?)`, scopeHash, sqliteTime(now.Add(-2*time.Second))).Scan(&coolingDown); err != nil {
		return fmt.Errorf("check login captcha generation: %w", err)
	}
	if coolingDown == 1 {
		return domain.ErrLoginCaptchaCooldown
	}
	return nil
}

func (repository *SQLite) RecordLoginFailure(ctx context.Context, input domain.LoginFailure) (int32, error) {
	if len(input.ScopeHash) != sha256.Size {
		return 0, errors.New("login failure scope hash must be SHA-256")
	}
	tx, err := repository.begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin failed login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now, expires := sqliteTime(input.FailedAt), sqliteTime(input.ExpiresAt)
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_login_failure_states
(scope_hash,failure_count,first_failed_at,last_failed_at,expires_at) VALUES(?,1,?,?,?)
ON CONFLICT(scope_hash) DO UPDATE SET
failure_count=CASE WHEN iam_login_failure_states.expires_at>excluded.last_failed_at
THEN min(iam_login_failure_states.failure_count+1,1000000) ELSE 1 END,
first_failed_at=CASE WHEN iam_login_failure_states.expires_at>excluded.last_failed_at
THEN iam_login_failure_states.first_failed_at ELSE excluded.first_failed_at END,
last_failed_at=excluded.last_failed_at,expires_at=excluded.expires_at`, input.ScopeHash, now, now, expires); err != nil {
		return 0, fmt.Errorf("update login failure state: %w", err)
	}
	var count int32
	if err = tx.QueryRowContext(ctx, `SELECT failure_count FROM iam_login_failure_states WHERE scope_hash=?`, input.ScopeHash).Scan(&count); err != nil {
		return 0, fmt.Errorf("read login failure state: %w", err)
	}
	var tenantID any
	if input.UserID != nil {
		var id string
		if lookupErr := tx.QueryRowContext(ctx, `SELECT tenant_id FROM iam_tenant_members WHERE user_id=? AND status='active'
ORDER BY created_at,tenant_id LIMIT 1`, input.UserID.String()).Scan(&id); lookupErr == nil {
			tenantID = id
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_login_events
(id,tenant_id,user_id,app_id,request_id,auth_method,audience,succeeded,client_ip,user_agent,occurred_at)
VALUES(?,?,?,?,?,'password',?,0,?,?,?)`, uuid.NewString(), tenantID, uuidPointerValue(input.UserID),
		uuidPointerValue(input.AppID), emptyToNil(input.RequestID), input.Audience, ipValue(input.IPAddress), emptyToNil(input.UserAgent), now); err != nil {
		return 0, fmt.Errorf("record failed login: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit failed login transaction: %w", err)
	}
	return count, nil
}

func (repository *SQLite) ResetLoginFailures(ctx context.Context, scopeHash []byte) error {
	if _, err := repository.db.ExecContext(ctx, `DELETE FROM iam_login_failure_states WHERE scope_hash=?`, scopeHash); err != nil {
		return fmt.Errorf("reset login failure state: %w", err)
	}
	return nil
}

func (repository *SQLite) CreateLoginCaptcha(ctx context.Context, input domain.LoginCaptchaChallenge) (uuid.UUID, error) {
	return repository.createInteractiveCaptcha(ctx, input, true)
}

func (repository *SQLite) CreateInteractiveCaptcha(ctx context.Context, input domain.LoginCaptchaChallenge) (uuid.UUID, error) {
	return repository.createInteractiveCaptcha(ctx, input, false)
}

func (repository *SQLite) createInteractiveCaptcha(ctx context.Context, input domain.LoginCaptchaChallenge, requireLoginFailures bool) (uuid.UUID, error) {
	if input.ID == uuid.Nil || len(input.ScopeHash) != sha256.Size || len(input.ProofHash) != sha256.Size ||
		!validLoginCaptchaType(input.CaptchaType) || !input.ExpiresAt.After(input.CreatedAt) {
		return uuid.Nil, errors.New("invalid interactive captcha challenge")
	}
	tx, err := repository.begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin login captcha transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	createdAt := sqliteTime(input.CreatedAt)
	if requireLoginFailures {
		var failures int32
		var expiresAt string
		err = tx.QueryRowContext(ctx, `SELECT failure_count,expires_at FROM iam_login_failure_states WHERE scope_hash=?`, input.ScopeHash).
			Scan(&failures, &expiresAt)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (failures < 3 || expiresAt <= createdAt)) {
			return uuid.Nil, domain.ErrLoginCaptchaNotRequired
		}
		if err != nil {
			return uuid.Nil, fmt.Errorf("lock login failure state: %w", err)
		}
	}
	var coolingDown int
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM iam_interactive_captcha_challenges
WHERE scope_hash=? AND created_at>?)`, input.ScopeHash, sqliteTime(input.CreatedAt.Add(-2*time.Second))).Scan(&coolingDown); err != nil {
		return uuid.Nil, fmt.Errorf("check login captcha cooldown: %w", err)
	}
	if coolingDown == 1 {
		return uuid.Nil, domain.ErrLoginCaptchaCooldown
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_interactive_captcha_challenges SET consumed_at=COALESCE(consumed_at,?)
WHERE scope_hash=? AND consumed_at IS NULL`, createdAt, input.ScopeHash); err != nil {
		return uuid.Nil, fmt.Errorf("invalidate active login captcha: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_interactive_captcha_challenges
(id,scope_hash,captcha_type,proof_hash,attempt_count,expires_at,created_at) VALUES(?,?,?,?,0,?,?)`, input.ID.String(),
		input.ScopeHash, input.CaptchaType, input.ProofHash, sqliteTime(input.ExpiresAt), createdAt); err != nil {
		return uuid.Nil, fmt.Errorf("insert login captcha: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit login captcha transaction: %w", err)
	}
	return input.ID, nil
}

func (repository *SQLite) VerifyLoginCaptcha(ctx context.Context, input domain.LoginCaptchaAttempt) error {
	return repository.verifyInteractiveCaptcha(ctx, input)
}

func (repository *SQLite) VerifyInteractiveCaptcha(ctx context.Context, input domain.LoginCaptchaAttempt) error {
	return repository.verifyInteractiveCaptcha(ctx, input)
}

func (repository *SQLite) verifyInteractiveCaptcha(ctx context.Context, input domain.LoginCaptchaAttempt) error {
	if input.ID == uuid.Nil || len(input.ScopeHash) != sha256.Size || len(input.ProofHash) != sha256.Size || !validLoginCaptchaType(input.CaptchaType) {
		return domain.ErrLoginCaptchaInvalid
	}
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin captcha verification: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var captchaType string
	var proofHash []byte
	var attemptCount int
	now := sqliteTime(input.Now)
	err = tx.QueryRowContext(ctx, `SELECT captcha_type,proof_hash,attempt_count FROM iam_interactive_captcha_challenges
WHERE id=? AND scope_hash=? AND consumed_at IS NULL AND expires_at>? AND attempt_count<5`, input.ID.String(), input.ScopeHash, now).
		Scan(&captchaType, &proofHash, &attemptCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrLoginCaptchaInvalid
	}
	if err != nil {
		return fmt.Errorf("lock login captcha: %w", err)
	}
	if !loginCaptchaProofMatches(captchaType, proofHash, input) {
		return domain.ErrLoginCaptchaInvalid
	}
	consume := input.Valid || attemptCount+1 >= 5
	if _, err = tx.ExecContext(ctx, `UPDATE iam_interactive_captcha_challenges SET attempt_count=attempt_count+1,
consumed_at=CASE WHEN ?=1 THEN ? ELSE consumed_at END WHERE id=? AND consumed_at IS NULL AND attempt_count<5`,
		boolInt(consume), now, input.ID.String()); err != nil {
		return fmt.Errorf("complete login captcha attempt: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit captcha verification: %w", err)
	}
	if !input.Valid {
		return domain.ErrLoginCaptchaInvalid
	}
	return nil
}

func (repository *SQLite) CreateSession(ctx context.Context, input domain.CreateSession) (domain.Session, error) {
	if len(input.RefreshTokenHash) != sha256.Size {
		return domain.Session{}, errors.New("refresh token hash must be SHA-256")
	}
	tx, err := repository.begin(ctx)
	if err != nil {
		return domain.Session{}, fmt.Errorf("begin session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())
	var deviceID any
	if input.DeviceKey != "" {
		var id string
		err = tx.QueryRowContext(ctx, `SELECT id FROM iam_devices WHERE user_id=? AND device_key=?`, input.UserID.String(), input.DeviceKey).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			id = uuid.NewString()
			_, err = tx.ExecContext(ctx, `INSERT INTO iam_devices
(id,user_id,device_key,platform,last_ip,last_seen_at,created_at,updated_at) VALUES(?,?,?,'web',?,?,?,?)`,
				id, input.UserID.String(), input.DeviceKey, ipValue(input.IPAddress), now, now, now)
		} else if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE iam_devices SET last_ip=?,last_seen_at=?,updated_at=? WHERE id=?`, ipValue(input.IPAddress), now, now, id)
		}
		if err != nil {
			return domain.Session{}, fmt.Errorf("upsert web device: %w", err)
		}
		deviceID = id
	}
	session := domain.Session{
		ID: uuid.New(), UserID: input.UserID, TenantID: input.TenantID, AppID: input.AppID,
		Audience: input.Audience, AccessTokenVersion: 1, AbsoluteExpiresAt: input.AbsoluteExpiresAt.UTC(),
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_sessions
(id,user_id,tenant_id,app_id,device_id,audience,status,access_token_version,absolute_expires_at,idle_expires_at,
ip_address,user_agent,last_seen_at,created_at,updated_at) VALUES(?,?,?,?,?,?,'active',1,?,?,?,?,?,?,?)`,
		session.ID.String(), input.UserID.String(), input.TenantID.String(), uuidPointerValue(input.AppID), deviceID,
		input.Audience, sqliteTime(input.AbsoluteExpiresAt), sqliteTime(input.IdleExpiresAt), ipValue(input.IPAddress),
		emptyToNil(input.UserAgent), now, now, now); err != nil {
		return domain.Session{}, fmt.Errorf("create session: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_refresh_tokens
(id,session_id,token_hash,expires_at,created_ip,created_at) VALUES(?,?,?,?,?,?)`, uuid.NewString(), session.ID.String(),
		input.RefreshTokenHash, sqliteTime(input.RefreshExpiresAt), ipValue(input.IPAddress), now); err != nil {
		return domain.Session{}, fmt.Errorf("create refresh token: %w", err)
	}
	authMethod := strings.TrimSpace(input.AuthMethod)
	if authMethod == "" {
		authMethod = "password"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_login_events
(id,tenant_id,user_id,session_id,app_id,request_id,auth_method,audience,succeeded,client_ip,user_agent,device_registered,occurred_at)
VALUES(?,?,?,?,?,?,?,?,1,?,?,?,?)`, uuid.NewString(), input.TenantID.String(), input.UserID.String(), session.ID.String(),
		uuidPointerValue(input.AppID), emptyToNil(input.RequestID), authMethod, input.Audience, ipValue(input.IPAddress),
		emptyToNil(input.UserAgent), boolInt(input.DeviceKey != ""), now); err != nil {
		return domain.Session{}, fmt.Errorf("record successful login: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return domain.Session{}, fmt.Errorf("commit session transaction: %w", err)
	}
	return session, nil
}

func (repository *SQLite) RotateRefreshToken(ctx context.Context, oldHash, newHash []byte, ipAddress *netip.Addr) (domain.Session, error) {
	if len(oldHash) != sha256.Size || len(newHash) != sha256.Size {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	tx, err := repository.begin(ctx)
	if err != nil {
		return domain.Session{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var tokenID, sessionID, userID, tenantID, audience, status, refreshExpires, absoluteExpires, idleExpires string
	var appID, consumedAt, refreshRevokedAt, sessionRevokedAt sql.NullString
	var tokenVersion int32
	err = tx.QueryRowContext(ctx, `SELECT rt.id,rt.session_id,rt.expires_at,rt.consumed_at,rt.revoked_at,
s.user_id,s.tenant_id,s.app_id,s.audience,s.status,s.access_token_version,s.absolute_expires_at,s.idle_expires_at,s.revoked_at
FROM iam_refresh_tokens rt JOIN iam_sessions s ON s.id=rt.session_id WHERE rt.token_hash=?`, oldHash).
		Scan(&tokenID, &sessionID, &refreshExpires, &consumedAt, &refreshRevokedAt, &userID, &tenantID, &appID,
			&audience, &status, &tokenVersion, &absoluteExpires, &idleExpires, &sessionRevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("lock refresh token: %w", err)
	}
	now := sqliteTime(time.Now())
	if consumedAt.Valid {
		if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET reuse_detected_at=COALESCE(reuse_detected_at,?) WHERE id=?`, now, tokenID); err != nil {
			return domain.Session{}, fmt.Errorf("mark refresh token reuse: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=COALESCE(revoked_at,?),
revoke_reason='refresh_token_reuse',access_token_version=access_token_version+1,updated_at=? WHERE id=? AND revoked_at IS NULL`, now, now, sessionID); err != nil {
			return domain.Session{}, fmt.Errorf("revoke reused session: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE session_id=?`, now, sessionID); err != nil {
			return domain.Session{}, fmt.Errorf("revoke refresh token family: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO audit_security_events
(id,tenant_id,user_id,session_id,app_id,event_type,severity,source,status,client_ip,metadata,occurred_at)
VALUES(?,?,?,?,?,'iam.refresh_token.reuse','high','auth','open',?,'{"action":"session_revoked"}',?)`, uuid.NewString(),
			tenantID, userID, sessionID, nullStringValue(appID), ipValue(ipAddress), now); err != nil {
			return domain.Session{}, fmt.Errorf("record refresh reuse security event: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return domain.Session{}, fmt.Errorf("commit refresh reuse revocation: %w", err)
		}
		return domain.Session{}, domain.ErrRefreshReused
	}
	if refreshRevokedAt.Valid || sessionRevokedAt.Valid || status != "active" || refreshExpires <= now || absoluteExpires <= now || idleExpires <= now {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	newID := uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO iam_refresh_tokens
(id,session_id,token_hash,parent_token_id,expires_at,created_ip,created_at) VALUES(?,?,?,?,?,?,?)`, newID, sessionID, newHash,
		tokenID, refreshExpires, ipValue(ipAddress), now); err != nil {
		return domain.Session{}, fmt.Errorf("create rotated refresh token: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET consumed_at=?,replaced_by_token_id=?
WHERE id=? AND consumed_at IS NULL AND revoked_at IS NULL`, now, newID, tokenID)
	if err != nil {
		return domain.Session{}, fmt.Errorf("consume refresh token: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_sessions SET last_seen_at=?,ip_address=COALESCE(?,ip_address),updated_at=? WHERE id=?`,
		now, ipValue(ipAddress), now, sessionID); err != nil {
		return domain.Session{}, fmt.Errorf("touch refreshed session: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return domain.Session{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	resultSession := domain.Session{Audience: audience, AccessTokenVersion: tokenVersion}
	if resultSession.ID, err = uuid.Parse(sessionID); err != nil {
		return domain.Session{}, fmt.Errorf("decode refresh session ID: %w", err)
	}
	if resultSession.UserID, err = uuid.Parse(userID); err != nil {
		return domain.Session{}, fmt.Errorf("decode refresh user ID: %w", err)
	}
	if resultSession.TenantID, err = uuid.Parse(tenantID); err != nil {
		return domain.Session{}, fmt.Errorf("decode refresh tenant ID: %w", err)
	}
	if appID.Valid {
		value, parseErr := uuid.Parse(appID.String)
		if parseErr != nil {
			return domain.Session{}, fmt.Errorf("decode refresh app ID: %w", parseErr)
		}
		resultSession.AppID = &value
	}
	resultSession.AbsoluteExpiresAt, err = parseSQLiteTime(absoluteExpires)
	if err != nil {
		return domain.Session{}, fmt.Errorf("decode refresh expiry: %w", err)
	}
	return resultSession, nil
}

func (repository *SQLite) RevokeSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin revoke transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := sqliteTime(time.Now())
	if _, err = tx.ExecContext(ctx, `UPDATE notify_push_devices SET status='disabled',invalidated_at=COALESCE(invalidated_at,?),
invalid_reason=COALESCE(invalid_reason,'session_revoked'),updated_at=? WHERE status='active' AND EXISTS(
SELECT 1 FROM iam_sessions s WHERE s.id=? AND s.audience='ak-mobile' AND s.app_id=notify_push_devices.app_id
AND s.device_id=notify_push_devices.device_id AND s.tenant_id=notify_push_devices.tenant_id AND s.user_id=notify_push_devices.user_id)`, now, now, sessionID.String()); err != nil {
		return fmt.Errorf("disable push binding for revoked mobile session: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=COALESCE(revoked_at,?),
revoke_reason=?,access_token_version=access_token_version+1,updated_at=? WHERE id=? AND revoked_at IS NULL`, now, reason, now, sessionID.String()); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE session_id=?`, now, sessionID.String()); err != nil {
		return fmt.Errorf("revoke session refresh tokens: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit session revocation: %w", err)
	}
	return nil
}

func (repository *SQLite) ValidateSession(ctx context.Context, session domain.Session) error {
	var count int
	appID := uuidPointerValue(session.AppID)
	err := repository.db.QueryRowContext(ctx, `SELECT count(*) FROM iam_sessions WHERE id=? AND user_id=? AND tenant_id=?
AND audience=? AND access_token_version=? AND ((app_id IS NULL AND ? IS NULL) OR app_id=?)
AND status='active' AND revoked_at IS NULL AND absolute_expires_at>? AND idle_expires_at>?`, session.ID.String(), session.UserID.String(),
		session.TenantID.String(), session.Audience, session.AccessTokenVersion, appID, appID, sqliteTime(time.Now()), sqliteTime(time.Now())).Scan(&count)
	if err != nil {
		return fmt.Errorf("validate active session: %w", err)
	}
	if count != 1 {
		return domain.ErrRefreshInvalid
	}
	return nil
}

func (repository *SQLite) GetAuthContext(ctx context.Context, userID, tenantID uuid.UUID) (domain.AuthContext, error) {
	var result domain.AuthContext
	var userIDText, tenantIDText string
	var email, mobile, avatar sql.NullString
	err := repository.db.QueryRowContext(ctx, `SELECT u.id,u.email,u.mobile,u.display_name,u.locale,u.time_zone,u.status,u.avatar_file_id,
t.id,t.code,t.name,t.status FROM iam_users u JOIN iam_tenant_members tm ON tm.user_id=u.id AND tm.tenant_id=? AND tm.status='active'
JOIN iam_tenants t ON t.id=tm.tenant_id AND t.deleted_at IS NULL AND t.status='active'
WHERE u.id=? AND u.deleted_at IS NULL AND u.status='active'`, tenantID.String(), userID.String()).Scan(
		&userIDText, &email, &mobile, &result.User.DisplayName, &result.User.Locale, &result.User.TimeZone, &result.User.Status,
		&avatar, &tenantIDText, &result.Tenant.Code, &result.Tenant.Name, &result.Tenant.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthContext{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("get auth context user: %w", err)
	}
	if err = decodeUserIDs(&result.User, userIDText, avatar); err != nil {
		return domain.AuthContext{}, err
	}
	result.User.Email, result.User.Mobile = email.String, mobile.String
	if result.Tenant.ID, err = uuid.Parse(tenantIDText); err != nil {
		return domain.AuthContext{}, fmt.Errorf("decode auth tenant ID: %w", err)
	}
	result.TimeZone = result.User.TimeZone
	now := sqliteTime(time.Now())
	result.Roles, err = queryStrings(ctx, repository.db, `SELECT r.code FROM iam_user_roles ur JOIN iam_roles r
ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.deleted_at IS NULL
WHERE ur.tenant_id=? AND ur.user_id=? AND r.status='active' AND ur.valid_from<=?
AND (ur.valid_until IS NULL OR ur.valid_until>?) ORDER BY r.code`, tenantID.String(), userID.String(), now, now)
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context roles: %w", err)
	}
	result.Permissions, err = queryStrings(ctx, repository.db, `SELECT DISTINCT p.code FROM iam_user_roles ur
JOIN iam_roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.deleted_at IS NULL
JOIN iam_role_permissions rp ON rp.tenant_id=ur.tenant_id AND rp.role_id=ur.role_id
JOIN iam_permissions p ON p.id=rp.permission_id AND p.status='active'
WHERE ur.tenant_id=? AND ur.user_id=? AND r.status='active' AND ur.valid_from<=?
AND (ur.valid_until IS NULL OR ur.valid_until>?) ORDER BY p.code`, tenantID.String(), userID.String(), now, now)
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context permissions: %w", err)
	}
	rows, err := repository.db.QueryContext(ctx, `SELECT DISTINCT m.id,m.parent_id,m.code,m.i18n_key,m.title,m.menu_type,
m.route_path,m.component_key,m.icon,m.affix,m.sort_order,m.feature_flag FROM iam_user_roles ur
JOIN iam_roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.status='active' AND r.deleted_at IS NULL
JOIN sys_role_menus rm ON rm.tenant_id=ur.tenant_id AND rm.role_id=ur.role_id
JOIN sys_menus m ON m.id=rm.menu_id AND m.status='active' AND m.deleted_at IS NULL
WHERE ur.tenant_id=? AND ur.user_id=? AND ur.valid_from<=? AND (ur.valid_until IS NULL OR ur.valid_until>?)
AND (m.permission_id IS NULL OR EXISTS(SELECT 1 FROM iam_role_permissions rp
WHERE rp.tenant_id=ur.tenant_id AND rp.role_id=ur.role_id AND rp.permission_id=m.permission_id))
ORDER BY m.sort_order,m.code`, tenantID.String(), userID.String(), now, now)
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context menus: %w", err)
	}
	defer rows.Close()
	result.Menus = make([]domain.Menu, 0)
	for rows.Next() {
		var menu domain.Menu
		var id string
		var parent, path, component, icon sql.NullString
		var affix int
		if err = rows.Scan(&id, &parent, &menu.Code, &menu.I18nKey, &menu.Title, &menu.Type, &path, &component, &icon,
			&affix, &menu.SortOrder, &menu.FeatureFlag); err != nil {
			return domain.AuthContext{}, fmt.Errorf("scan auth context menu: %w", err)
		}
		if menu.ID, err = uuid.Parse(id); err != nil {
			return domain.AuthContext{}, fmt.Errorf("decode menu ID: %w", err)
		}
		if menu.ParentID, err = parseOptionalUUID(parent); err != nil {
			return domain.AuthContext{}, fmt.Errorf("decode menu parent ID: %w", err)
		}
		menu.RoutePath, menu.ComponentKey, menu.Icon = nullStringPointer(path), nullStringPointer(component), nullStringPointer(icon)
		menu.Affix = affix == 1
		result.Menus = append(result.Menus, menu)
	}
	if err = rows.Err(); err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context menus: %w", err)
	}
	return result, nil
}

func (repository *SQLite) ListSelfSessions(ctx context.Context, scope domain.SelfSessionScope) ([]domain.SelfSession, error) {
	now := sqliteTime(time.Now())
	rows, err := repository.db.QueryContext(ctx, `SELECT id,audience,status,ip_address,user_agent,last_seen_at,absolute_expires_at,created_at
FROM iam_sessions WHERE user_id=? AND tenant_id=? AND audience=? AND status='active' AND revoked_at IS NULL
AND absolute_expires_at>? AND idle_expires_at>? ORDER BY last_seen_at DESC,created_at DESC,id`,
		scope.UserID.String(), scope.TenantID.String(), scope.Audience, now, now)
	if err != nil {
		return nil, fmt.Errorf("list self sessions: %w", err)
	}
	defer rows.Close()
	result := make([]domain.SelfSession, 0)
	for rows.Next() {
		var value domain.SelfSession
		var id, lastSeen, absolute, created string
		var ip, agent sql.NullString
		if err = rows.Scan(&id, &value.Audience, &value.Status, &ip, &agent, &lastSeen, &absolute, &created); err != nil {
			return nil, fmt.Errorf("scan self session: %w", err)
		}
		if value.ID, err = uuid.Parse(id); err != nil {
			return nil, fmt.Errorf("decode self session ID: %w", err)
		}
		value.IPAddress = parseIP(ip)
		value.UserAgent = agent.String
		if value.LastSeenAt, err = parseSQLiteTime(lastSeen); err != nil {
			return nil, fmt.Errorf("decode self session last seen time: %w", err)
		}
		if value.AbsoluteExpiresAt, err = parseSQLiteTime(absolute); err != nil {
			return nil, fmt.Errorf("decode self session expiry: %w", err)
		}
		if value.CreatedAt, err = parseSQLiteTime(created); err != nil {
			return nil, fmt.Errorf("decode self session creation time: %w", err)
		}
		result = append(result, value)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list self sessions: %w", err)
	}
	return result, nil
}

func (repository *SQLite) RevokeSelfSession(ctx context.Context, input domain.RevokeSelfSession) error {
	tx, err := repository.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin self session revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM iam_sessions WHERE id=? AND user_id=? AND tenant_id=?
AND audience=? AND status='active' AND revoked_at IS NULL`, input.SessionID.String(), input.UserID.String(),
		input.TenantID.String(), input.Audience).Scan(&count); err != nil {
		return fmt.Errorf("lock self session: %w", err)
	}
	if count != 1 {
		return domain.ErrSessionNotFound
	}
	now := sqliteTime(time.Now())
	if _, err = tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=?,revoke_reason=?,
access_token_version=access_token_version+1,updated_at=? WHERE id=?`, now, input.RevocationReason, now, input.SessionID.String()); err != nil {
		return fmt.Errorf("revoke self session: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE session_id=?`, now, input.SessionID.String()); err != nil {
		return fmt.Errorf("revoke self session refresh tokens: %w", err)
	}
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: input.TenantID.String(), UserID: input.UserID.String(), SessionID: input.ActorSessionID.String(),
		RequestID: input.RequestID, ModuleCode: "iam", ActionName: "iam.me.session.revoke", ResourceType: "iam.session",
		ResourceID: input.SessionID.String(), RequestPath: "/admin-api/v1/me/sessions/" + input.SessionID.String(),
		IPAddress: input.IPAddress, UserAgent: input.UserAgent, BeforeData: `{"status":"active"}`, AfterData: `{"status":"revoked"}`,
	}); err != nil {
		return fmt.Errorf("audit self session revocation: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit self session revocation: %w", err)
	}
	return nil
}

func (repository *SQLite) ListSelfDevices(ctx context.Context, scope domain.SelfDeviceScope) ([]domain.SelfDevice, error) {
	now := sqliteTime(time.Now())
	rows, err := repository.db.QueryContext(ctx, `SELECT d.id,d.platform,d.device_name,d.model,d.os_version,d.app_version,
d.last_ip,d.last_seen_at,d.created_at,COALESCE((SELECT s.user_agent FROM iam_sessions s WHERE s.user_id=d.user_id
AND s.device_id=d.id AND s.tenant_id=? AND s.audience=? ORDER BY s.last_seen_at DESC,s.created_at DESC,s.id LIMIT 1),''),
(SELECT count(*) FROM iam_sessions s WHERE s.user_id=d.user_id AND s.device_id=d.id AND s.tenant_id=? AND s.audience=?
AND s.status='active' AND s.revoked_at IS NULL AND s.absolute_expires_at>? AND s.idle_expires_at>?),
EXISTS(SELECT 1 FROM iam_sessions s WHERE s.id=? AND s.user_id=d.user_id AND s.device_id=d.id)
FROM iam_devices d WHERE d.user_id=? AND EXISTS(SELECT 1 FROM iam_sessions s WHERE s.user_id=d.user_id
AND s.device_id=d.id AND s.tenant_id=? AND s.audience=?) ORDER BY 12 DESC,d.last_seen_at DESC,d.created_at DESC,d.id`,
		scope.TenantID.String(), scope.Audience, scope.TenantID.String(), scope.Audience, now, now, scope.SessionID.String(),
		scope.UserID.String(), scope.TenantID.String(), scope.Audience)
	if err != nil {
		return nil, fmt.Errorf("list self devices: %w", err)
	}
	defer rows.Close()
	result := make([]domain.SelfDevice, 0)
	for rows.Next() {
		var value domain.SelfDevice
		var id, created string
		var name, model, osVersion, appVersion, lastIP, lastSeen sql.NullString
		var current int
		if err = rows.Scan(&id, &value.Platform, &name, &model, &osVersion, &appVersion, &lastIP, &lastSeen,
			&created, &value.LatestUserAgent, &value.ActiveSessionCount, &current); err != nil {
			return nil, fmt.Errorf("scan self device: %w", err)
		}
		if value.ID, err = uuid.Parse(id); err != nil {
			return nil, fmt.Errorf("decode self device ID: %w", err)
		}
		value.DeviceName, value.Model, value.OSVersion, value.AppVersion = name.String, model.String, osVersion.String, appVersion.String
		value.LastIP, value.Current = parseIP(lastIP), current == 1
		if lastSeen.Valid {
			parsed, parseErr := parseSQLiteTime(lastSeen.String)
			if parseErr != nil {
				return nil, fmt.Errorf("decode self device last seen time: %w", parseErr)
			}
			value.LastSeenAt = &parsed
		}
		if value.CreatedAt, err = parseSQLiteTime(created); err != nil {
			return nil, fmt.Errorf("decode self device creation time: %w", err)
		}
		result = append(result, value)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list self devices: %w", err)
	}
	return result, nil
}

func (repository *SQLite) RemoveSelfDevice(ctx context.Context, input domain.RemoveSelfDevice) (bool, error) {
	tx, err := repository.begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin self device removal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var current int
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM iam_sessions current_session WHERE current_session.id=?
AND current_session.user_id=d.user_id AND current_session.device_id=d.id) FROM iam_devices d WHERE d.id=? AND d.user_id=?
AND EXISTS(SELECT 1 FROM iam_sessions s WHERE s.user_id=d.user_id AND s.device_id=d.id AND s.tenant_id=? AND s.audience=?)`,
		input.SessionID.String(), input.DeviceID.String(), input.UserID.String(), input.TenantID.String(), input.Audience).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return false, domain.ErrDeviceNotFound
	}
	if err != nil {
		return false, fmt.Errorf("lock self device: %w", err)
	}
	now := sqliteTime(time.Now())
	if _, err = tx.ExecContext(ctx, `UPDATE iam_refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE revoked_at IS NULL
AND session_id IN(SELECT id FROM iam_sessions WHERE user_id=? AND device_id=?)`, now, input.UserID.String(), input.DeviceID.String()); err != nil {
		return false, fmt.Errorf("revoke self device refresh tokens: %w", err)
	}
	revoked, err := tx.ExecContext(ctx, `UPDATE iam_sessions SET status='revoked',revoked_at=COALESCE(revoked_at,?),
revoke_reason='device_removed',access_token_version=access_token_version+1,updated_at=?
WHERE user_id=? AND device_id=? AND status='active' AND revoked_at IS NULL`, now, now, input.UserID.String(), input.DeviceID.String())
	if err != nil {
		return false, fmt.Errorf("revoke self device sessions: %w", err)
	}
	revokedCount, _ := revoked.RowsAffected()
	afterJSON, _ := json.Marshal(map[string]any{"status": "removed", "sessions_revoked": revokedCount})
	if err = insertOperationAudit(ctx, tx, operationAudit{
		TenantID: input.TenantID.String(), UserID: input.UserID.String(), SessionID: input.SessionID.String(),
		RequestID: input.RequestID, ModuleCode: "iam", ActionName: "iam.me.device.remove", ResourceType: "iam.device",
		ResourceID: input.DeviceID.String(), RequestPath: "/admin-api/v1/me/devices/" + input.DeviceID.String(),
		IPAddress: input.IPAddress, UserAgent: input.UserAgent, BeforeData: `{"status":"active"}`, AfterData: string(afterJSON),
	}); err != nil {
		return false, fmt.Errorf("audit self device removal: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM iam_devices WHERE id=? AND user_id=?`, input.DeviceID.String(), input.UserID.String()); err != nil {
		return false, fmt.Errorf("delete self device: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("commit self device removal: %w", err)
	}
	return current == 1, nil
}

type sqliteQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (repository *SQLite) begin(ctx context.Context) (*sql.Tx, error) {
	if repository == nil || repository.db == nil {
		return nil, errors.New("SQLite repository database is nil")
	}
	return repository.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

func queryUser(ctx context.Context, queryer sqliteQueryRower, query string, args ...any) (domain.User, error) {
	var user domain.User
	var id string
	var email, mobile, avatar sql.NullString
	err := queryer.QueryRowContext(ctx, query, args...).Scan(&id, &email, &mobile, &user.DisplayName, &user.Locale,
		&user.TimeZone, &user.Status, &avatar)
	if err != nil {
		return domain.User{}, err
	}
	if err = decodeUserIDs(&user, id, avatar); err != nil {
		return domain.User{}, err
	}
	user.Email, user.Mobile = email.String, mobile.String
	return user, nil
}

func decodeUserIDs(user *domain.User, id string, avatar sql.NullString) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("decode user ID: %w", err)
	}
	user.ID = parsed
	if avatar.Valid {
		value, parseErr := uuid.Parse(avatar.String)
		if parseErr != nil {
			return fmt.Errorf("decode avatar file ID: %w", parseErr)
		}
		user.AvatarFileID = &value
	}
	return nil
}

func queryStrings(ctx context.Context, queryer sqliteQueryer, query string, args ...any) ([]string, error) {
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func listPasswordHistory(ctx context.Context, queryer sqliteQueryer, userID uuid.UUID) ([]string, error) {
	result, err := queryStrings(ctx, queryer, `SELECT password_hash FROM iam_password_history
WHERE user_id=? ORDER BY created_at DESC,id DESC LIMIT 5`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list recent password hashes: %w", err)
	}
	return result, nil
}

func classifySQLiteCreateError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint failed") && strings.Contains(message, "iam_users.email") {
		return domain.ErrEmailAlreadyExists
	}
	return fmt.Errorf("create identity: %w", err)
}

type operationAudit struct {
	TenantID     string
	UserID       string
	SessionID    string
	RequestID    string
	ModuleCode   string
	ActionName   string
	ResourceType string
	ResourceID   string
	RequestPath  string
	IPAddress    *netip.Addr
	UserAgent    string
	BeforeData   string
	AfterData    string
}

func insertOperationAudit(ctx context.Context, execer sqliteExecer, value operationAudit) error {
	_, err := execer.ExecContext(ctx, `INSERT INTO audit_operation_logs
(id,tenant_id,user_id,session_id,request_id,module_code,action_name,resource_type,resource_id,request_path,
client_ip,user_agent,succeeded,before_data,after_data,metadata,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?,?)`,
		uuid.NewString(), emptyToNil(value.TenantID), emptyToNil(value.UserID), emptyToNil(value.SessionID),
		emptyToNil(value.RequestID), value.ModuleCode, value.ActionName, emptyToNil(value.ResourceType),
		emptyToNil(value.ResourceID), emptyToNil(value.RequestPath), ipValue(value.IPAddress), emptyToNil(value.UserAgent),
		emptyToNil(value.BeforeData), emptyToNil(value.AfterData), `{}`, sqliteTime(time.Now()))
	return err
}

func sqliteTime(value time.Time) string {
	return value.UTC().Format(sqliteTimeLayout)
}

func parseSQLiteTime(value string) (time.Time, error) {
	parsed, err := time.Parse(sqliteTimeLayout, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

func ipValue(value *netip.Addr) any {
	if value == nil || !value.IsValid() {
		return nil
	}
	return value.String()
}

func parseIP(value sql.NullString) *netip.Addr {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed, err := netip.ParseAddr(value.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func uuidPointerValue(value *uuid.UUID) any {
	if value == nil || *value == uuid.Nil {
		return nil
	}
	return value.String()
}

func stringPointerValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func parseOptionalUUID(value sql.NullString) (*uuid.UUID, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := uuid.Parse(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func emptyToNil(value string) any {
	value = strings.TrimSpace(value)
	if value == "" || value == uuid.Nil.String() {
		return nil
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
