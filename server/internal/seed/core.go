package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type permissionCatalog struct {
	Permissions []permissionDefinition `json:"permissions"`
}

type permissionDefinition struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	ModuleCode     string `json:"module_code"`
	ResourceName   string `json:"resource_name"`
	ActionName     string `json:"action_name"`
	PermissionKind string `json:"permission_kind"`
	Status         string `json:"status"`
}

type menuCatalog struct {
	Menus []menuDefinition `json:"menus"`
}

type menuDefinition struct {
	Code          string  `json:"code"`
	I18nKey       string  `json:"i18n_key"`
	Title         string  `json:"title"`
	Type          string  `json:"type"`
	Path          *string `json:"path"`
	Sort          int32   `json:"sort"`
	Parent        *string `json:"parent"`
	ComponentKey  *string `json:"component_key"`
	Permission    *string `json:"permission"`
	Icon          *string `json:"icon"`
	Affix         bool    `json:"affix"`
	Phase         string  `json:"phase"`
	FeatureFlag   string  `json:"feature_flag"`
	BackendStatus string  `json:"backend_status"`
}

type BootstrapAdminInput struct {
	TenantCode  string
	TenantName  string
	Email       string
	DisplayName string
	Locale      string
	Password    string
}

func CorePermissions(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return 0, fmt.Errorf("read permission catalog: %w", err)
	}
	var catalog permissionCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return 0, fmt.Errorf("decode permission catalog: %w", err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin permission seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	for _, permission := range catalog.Permissions {
		if _, err = queries.UpsertCorePermission(ctx, db.UpsertCorePermissionParams{
			Code: permission.Code, Name: permission.Name, ModuleCode: permission.ModuleCode,
			ResourceName: permission.ResourceName, ActionName: permission.ActionName,
			PermissionKind: permission.PermissionKind, Status: permission.Status,
		}); err != nil {
			return 0, fmt.Errorf("upsert permission %s: %w", permission.Code, err)
		}
	}
	if _, err = queries.SyncSystemAdminPermissions(ctx); err != nil {
		return 0, fmt.Errorf("sync system administrator permissions: %w", err)
	}
	if _, err = queries.SyncDefaultTenantRoles(ctx); err != nil {
		return 0, fmt.Errorf("sync default tenant roles: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit permission seed: %w", err)
	}
	return len(catalog.Permissions), nil
}

func CoreMenus(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return 0, fmt.Errorf("read menu catalog: %w", err)
	}
	var catalog menuCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return 0, fmt.Errorf("decode menu catalog: %w", err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin menu seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	menuIDs := make(map[string]uuid.UUID, len(catalog.Menus))
	for _, menu := range catalog.Menus {
		var parentID *uuid.UUID
		if menu.Parent != nil {
			resolved, found := menuIDs[*menu.Parent]
			if !found {
				return 0, fmt.Errorf("menu %s references unknown or unordered parent %s", menu.Code, *menu.Parent)
			}
			parentID = &resolved
		}
		var permissionID *uuid.UUID
		if menu.Permission != nil {
			resolved, lookupErr := queries.GetActivePermissionIDByCode(ctx, *menu.Permission)
			if lookupErr != nil {
				return 0, fmt.Errorf("menu %s references unknown permission %s: %w", menu.Code, *menu.Permission, lookupErr)
			}
			permissionID = &resolved
		}
		metadata, err := json.Marshal(map[string]string{
			"i18n_key": menu.I18nKey, "phase": menu.Phase,
			"feature_flag": menu.FeatureFlag, "backend_status": menu.BackendStatus,
		})
		if err != nil {
			return 0, fmt.Errorf("encode menu metadata: %w", err)
		}
		row, err := queries.UpsertCoreMenu(ctx, db.UpsertCoreMenuParams{
			ParentID: parentID, PermissionID: permissionID, Code: menu.Code, Title: menu.Title,
			MenuType: menu.Type, RoutePath: menu.Path, ComponentKey: menu.ComponentKey,
			Icon: menu.Icon, Affix: menu.Affix, SortOrder: menu.Sort, Metadata: metadata,
		})
		if err != nil {
			return 0, fmt.Errorf("upsert menu %s: %w", menu.Code, err)
		}
		menuIDs[menu.Code] = row.ID
	}
	if _, err = queries.SyncSystemAdminMenus(ctx); err != nil {
		return 0, fmt.Errorf("sync system administrator menus: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit menu seed: %w", err)
	}
	return len(catalog.Menus), nil
}

func BootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, input BootstrapAdminInput) (domain.User, domain.Tenant, int64, int64, error) {
	passwordHash, err := application.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, err
	}
	postgresRepository := repository.NewPostgres(pool)
	user, tenant, err := postgresRepository.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: input.TenantCode, TenantName: input.TenantName, Email: input.Email,
		DisplayName: input.DisplayName, Locale: input.Locale, PasswordHash: passwordHash,
	})
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("begin bootstrap role transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	roleID, err := queries.UpsertSystemAdminRole(ctx, tenant.ID)
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("upsert bootstrap role: %w", err)
	}
	if err = queries.AssignUserRole(ctx, db.AssignUserRoleParams{
		TenantID: tenant.ID, UserID: user.ID, RoleID: roleID, GrantedBy: &user.ID,
	}); err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("assign bootstrap role: %w", err)
	}
	granted, err := queries.GrantAllActivePermissionsToRole(ctx, db.GrantAllActivePermissionsToRoleParams{
		TenantID: tenant.ID, RoleID: roleID, GrantedBy: &user.ID,
	})
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("grant bootstrap permissions: %w", err)
	}
	grantedMenus, err := queries.GrantAllCoreMenusToRole(ctx, db.GrantAllCoreMenusToRoleParams{TenantID: tenant.ID, RoleID: roleID})
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("grant bootstrap menus: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("commit bootstrap role: %w", err)
	}
	return user, tenant, granted, grantedMenus, nil
}
