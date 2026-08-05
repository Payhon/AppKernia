package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

type configCatalog struct {
	Version    int32                      `json:"version"`
	Categories []configCategoryDefinition `json:"categories"`
	Items      []configItemDefinition     `json:"items"`
}

type configCategoryDefinition struct {
	ModuleCode     string `json:"module_code"`
	ConfigGroup    string `json:"config_group"`
	NameKey        string `json:"name_key"`
	DescriptionKey string `json:"description_key"`
	SortOrder      int32  `json:"sort_order"`
}

type configItemDefinition struct {
	ModuleCode       string          `json:"module_code"`
	ConfigGroup      string          `json:"config_group"`
	ConfigKey        string          `json:"config_key"`
	DisplayName      string          `json:"display_name"`
	ValueType        string          `json:"value_type"`
	Value            json.RawMessage `json:"value"`
	DefaultValue     json.RawMessage `json:"default_value"`
	IsSecret         bool            `json:"is_secret"`
	IsPublic         bool            `json:"is_public"`
	ValidationSchema json.RawMessage `json:"validation_schema"`
	Description      string          `json:"description"`
	SortOrder        int32           `json:"sort_order"`
	Status           string          `json:"status"`
}

type regionCatalog struct {
	Version int32               `json:"version"`
	Source  regionCatalogSource `json:"source"`
	Regions []regionDefinition  `json:"regions"`
}

type regionCatalogSource struct {
	Project string `json:"project"`
	Commit  string `json:"commit"`
	License string `json:"license"`
}

type regionDefinition struct {
	Code       string          `json:"code"`
	ParentCode *string         `json:"parent_code"`
	Level      int16           `json:"level"`
	Name       string          `json:"name"`
	FullName   *string         `json:"full_name"`
	PostalCode *string         `json:"postal_code"`
	Longitude  *string         `json:"longitude"`
	Latitude   *string         `json:"latitude"`
	Status     string          `json:"status"`
	Metadata   json.RawMessage `json:"metadata"`
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
	TenantCode        string
	TenantName        string
	Email             string
	DisplayName       string
	Locale            string
	Password          string
	ConfigCatalogPath string
}

func readConfigCatalog(catalogPath string) (configCatalog, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return configCatalog{}, fmt.Errorf("read config catalog: %w", err)
	}
	var catalog configCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return configCatalog{}, fmt.Errorf("decode config catalog: %w", err)
	}
	if catalog.Version < 1 || len(catalog.Categories) == 0 || len(catalog.Items) == 0 {
		return configCatalog{}, fmt.Errorf("config catalog must contain a version, categories, and items")
	}
	categories := make(map[string]struct{}, len(catalog.Categories))
	for _, category := range catalog.Categories {
		key := category.ModuleCode + "/" + category.ConfigGroup
		if category.ModuleCode == "" || category.ConfigGroup == "" || category.NameKey == "" || category.DescriptionKey == "" {
			return configCatalog{}, fmt.Errorf("config category %q is incomplete", key)
		}
		if _, exists := categories[key]; exists {
			return configCatalog{}, fmt.Errorf("duplicate config category %q", key)
		}
		categories[key] = struct{}{}
	}
	items := make(map[string]struct{}, len(catalog.Items))
	for _, item := range catalog.Items {
		categoryKey := item.ModuleCode + "/" + item.ConfigGroup
		itemKey := categoryKey + "/" + item.ConfigKey
		if _, exists := categories[categoryKey]; !exists {
			return configCatalog{}, fmt.Errorf("config item %q references an unknown category", itemKey)
		}
		if _, exists := items[itemKey]; exists {
			return configCatalog{}, fmt.Errorf("duplicate config item %q", itemKey)
		}
		items[itemKey] = struct{}{}
		if item.ConfigKey == "" || item.DisplayName == "" || item.ValueType == "" || item.Status == "" || len(item.ValidationSchema) == 0 {
			return configCatalog{}, fmt.Errorf("config item %q is incomplete", itemKey)
		}
		if item.IsSecret && (len(item.Value) > 0 || len(item.DefaultValue) > 0 || item.IsPublic) {
			return configCatalog{}, fmt.Errorf("secret config item %q cannot define plaintext values or be public", itemKey)
		}
	}
	return catalog, nil
}

func readRegionCatalog(catalogPath string) (regionCatalog, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return regionCatalog{}, fmt.Errorf("read region catalog: %w", err)
	}
	var catalog regionCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return regionCatalog{}, fmt.Errorf("decode region catalog: %w", err)
	}
	if catalog.Version < 1 || catalog.Source.Project == "" || catalog.Source.Commit == "" || catalog.Source.License == "" || len(catalog.Regions) == 0 {
		return regionCatalog{}, fmt.Errorf("region catalog must contain versioned source metadata and regions")
	}
	seen := make(map[string]int16, len(catalog.Regions))
	for _, region := range catalog.Regions {
		if region.Code == "" || region.Name == "" || region.FullName == nil || *region.FullName == "" || len(region.Metadata) == 0 {
			return regionCatalog{}, fmt.Errorf("region %q is incomplete", region.Code)
		}
		var metadata map[string]any
		if err = json.Unmarshal(region.Metadata, &metadata); err != nil || metadata == nil {
			return regionCatalog{}, fmt.Errorf("region %q metadata must be a JSON object", region.Code)
		}
		if region.Level < 0 || region.Level > 10 || (region.Status != "active" && region.Status != "disabled") {
			return regionCatalog{}, fmt.Errorf("region %q has an invalid level or status", region.Code)
		}
		if _, exists := seen[region.Code]; exists {
			return regionCatalog{}, fmt.Errorf("duplicate region %q", region.Code)
		}
		if region.ParentCode == nil {
			if region.Level != 0 {
				return regionCatalog{}, fmt.Errorf("root region %q must use level 0", region.Code)
			}
		} else {
			parentLevel, exists := seen[*region.ParentCode]
			if !exists {
				return regionCatalog{}, fmt.Errorf("region %q references an unknown or unordered parent %q", region.Code, *region.ParentCode)
			}
			if region.Level != parentLevel+1 {
				return regionCatalog{}, fmt.Errorf("region %q must be one level below parent %q", region.Code, *region.ParentCode)
			}
		}
		if _, err = parseRegionCoordinate(region.Longitude, -180, 180); err != nil {
			return regionCatalog{}, fmt.Errorf("region %q longitude: %w", region.Code, err)
		}
		if _, err = parseRegionCoordinate(region.Latitude, -90, 90); err != nil {
			return regionCatalog{}, fmt.Errorf("region %q latitude: %w", region.Code, err)
		}
		seen[region.Code] = region.Level
	}
	return catalog, nil
}

func parseRegionCoordinate(value *string, minimum, maximum float64) (pgtype.Numeric, error) {
	if value == nil {
		return pgtype.Numeric{}, nil
	}
	parsed, err := strconv.ParseFloat(*value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < minimum || parsed > maximum {
		return pgtype.Numeric{}, fmt.Errorf("must be a decimal between %g and %g", minimum, maximum)
	}
	var numeric pgtype.Numeric
	if err = numeric.Scan(*value); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("parse decimal: %w", err)
	}
	return numeric, nil
}

func seedTenantConfigs(ctx context.Context, queries *db.Queries, tenantID uuid.UUID, catalog configCatalog) (int, error) {
	for _, item := range catalog.Items {
		description := item.Description
		var schema map[string]any
		if err := json.Unmarshal(item.ValidationSchema, &schema); err != nil {
			return 0, fmt.Errorf("decode validation schema for %s: %w", item.ConfigKey, err)
		}
		schema["x-appkernia-catalog"] = true
		validationSchema, err := json.Marshal(schema)
		if err != nil {
			return 0, fmt.Errorf("encode validation schema for %s: %w", item.ConfigKey, err)
		}
		if _, err := queries.UpsertTenantCoreConfig(ctx, db.UpsertTenantCoreConfigParams{
			TenantID: &tenantID, ModuleCode: item.ModuleCode, ConfigGroup: item.ConfigGroup,
			ConfigKey: item.ConfigKey, DisplayName: item.DisplayName, ValueType: item.ValueType,
			ValueJson: item.Value, DefaultValueJson: item.DefaultValue, IsSecret: item.IsSecret,
			IsPublic: item.IsPublic, ValidationSchema: validationSchema,
			Description: &description, SortOrder: item.SortOrder, Status: item.Status,
		}); err != nil {
			return 0, fmt.Errorf("upsert config %s/%s/%s: %w", item.ModuleCode, item.ConfigGroup, item.ConfigKey, err)
		}
	}
	return len(catalog.Items), nil
}

// CoreConfigs idempotently installs catalog metadata and safe initial values for
// every active tenant. Existing tenant values and encrypted secrets are preserved.
func CoreConfigs(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	catalog, err := readConfigCatalog(catalogPath)
	if err != nil {
		return 0, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin config seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	tenantIDs, err := queries.ListActiveTenantIDsForConfigSeed(ctx)
	if err != nil {
		return 0, fmt.Errorf("list active tenants for config seed: %w", err)
	}
	seeded := 0
	for _, tenantID := range tenantIDs {
		count, seedErr := seedTenantConfigs(ctx, queries, tenantID, catalog)
		if seedErr != nil {
			return 0, seedErr
		}
		seeded += count
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit config seed: %w", err)
	}
	return seeded, nil
}

// CoreRegions idempotently installs the versioned global region catalog. The
// catalog must be ordered parent-first so the database foreign key is always valid.
func CoreRegions(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	catalog, err := readRegionCatalog(catalogPath)
	if err != nil {
		return 0, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin region seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	for _, region := range catalog.Regions {
		longitude, coordinateErr := parseRegionCoordinate(region.Longitude, -180, 180)
		if coordinateErr != nil {
			return 0, fmt.Errorf("region %s longitude: %w", region.Code, coordinateErr)
		}
		latitude, coordinateErr := parseRegionCoordinate(region.Latitude, -90, 90)
		if coordinateErr != nil {
			return 0, fmt.Errorf("region %s latitude: %w", region.Code, coordinateErr)
		}
		if err = queries.UpsertCoreRegion(ctx, db.UpsertCoreRegionParams{
			Code: region.Code, ParentCode: region.ParentCode, Level: region.Level,
			Name: region.Name, FullName: region.FullName, PostalCode: region.PostalCode,
			Longitude: longitude, Latitude: latitude, Status: region.Status,
			Metadata: region.Metadata,
		}); err != nil {
			return 0, fmt.Errorf("upsert region %s: %w", region.Code, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit region seed: %w", err)
	}
	return len(catalog.Regions), nil
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
	var catalog configCatalog
	var err error
	if input.ConfigCatalogPath != "" {
		catalog, err = readConfigCatalog(input.ConfigCatalogPath)
		if err != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, err
		}
	}
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
	if input.ConfigCatalogPath != "" {
		if _, err = seedTenantConfigs(ctx, queries, tenant.ID, catalog); err != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, err
		}
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
