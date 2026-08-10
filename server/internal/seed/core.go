package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
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

type moduleCatalog struct {
	Version int32              `json:"version"`
	Modules []moduleDefinition `json:"modules"`
}

type moduleDefinition struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	NameKey        string          `json:"name_key"`
	Description    string          `json:"description"`
	DescriptionKey string          `json:"description_key"`
	Capabilities   json.RawMessage `json:"capabilities"`
	Status         string          `json:"status"`
}

type configCatalog struct {
	Version    int32                      `json:"version"`
	Categories []configCategoryDefinition `json:"categories"`
	Items      []configItemDefinition     `json:"items"`
}

type dictionaryCatalog struct {
	Version   int32                            `json:"version"`
	Types     []dictionaryTypeDefinition       `json:"types"`
	Items     []dictionaryItemDefinition       `json:"items"`
	Templates []notificationTemplateDefinition `json:"templates"`
}

type notificationTemplateDefinition struct {
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Channel         string          `json:"channel"`
	Locale          string          `json:"locale"`
	SubjectTemplate *string         `json:"subject_template"`
	BodyTemplate    string          `json:"body_template"`
	BodyFormat      string          `json:"body_format"`
	VariablesSchema json.RawMessage `json:"variables_schema"`
	Status          string          `json:"status"`
}

type dictionaryTypeDefinition struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	NameKey         string `json:"name_key"`
	Description     string `json:"description"`
	DescriptionKey  string `json:"description_key"`
	Visibility      string `json:"visibility"`
	ExtensionPolicy string `json:"extension_policy"`
	Status          string `json:"status"`
}

type dictionaryItemDefinition struct {
	TypeCode  string          `json:"type_code"`
	Value     string          `json:"value"`
	Locale    string          `json:"locale"`
	Label     string          `json:"label"`
	SortOrder int32           `json:"sort_order"`
	IsDefault bool            `json:"is_default"`
	Extra     json.RawMessage `json:"extra"`
	Status    string          `json:"status"`
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

func readModuleCatalog(catalogPath string) (moduleCatalog, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return moduleCatalog{}, fmt.Errorf("read module catalog: %w", err)
	}
	var catalog moduleCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return moduleCatalog{}, fmt.Errorf("decode module catalog: %w", err)
	}
	if catalog.Version < 1 || len(catalog.Modules) == 0 {
		return moduleCatalog{}, fmt.Errorf("module catalog must contain a version and modules")
	}
	codePattern := regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)
	seen := make(map[string]struct{}, len(catalog.Modules))
	for _, module := range catalog.Modules {
		if !codePattern.MatchString(module.Code) || module.Name == "" || module.NameKey == "" ||
			module.Description == "" || module.DescriptionKey == "" || module.Status != "enabled" {
			return moduleCatalog{}, fmt.Errorf("module %q is incomplete", module.Code)
		}
		if _, exists := seen[module.Code]; exists {
			return moduleCatalog{}, fmt.Errorf("duplicate module %q", module.Code)
		}
		var capabilities map[string]bool
		if err = json.Unmarshal(module.Capabilities, &capabilities); err != nil || len(capabilities) == 0 {
			return moduleCatalog{}, fmt.Errorf("module %q capabilities must be a non-empty boolean object", module.Code)
		}
		for capability, compiled := range capabilities {
			if !codePattern.MatchString(capability) || !compiled {
				return moduleCatalog{}, fmt.Errorf("module %q capability %q must be a compiled stable code", module.Code, capability)
			}
		}
		seen[module.Code] = struct{}{}
	}
	return catalog, nil
}

// CoreModules synchronizes the exact compile-time module registry. Runtime
// plugin installation is intentionally unsupported, so rows outside the
// versioned catalog are stale fixtures or data from a different build.
func CoreModules(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	catalog, err := readModuleCatalog(catalogPath)
	if err != nil {
		return 0, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin module seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	codes := make([]string, 0, len(catalog.Modules))
	for _, module := range catalog.Modules {
		description := module.Description
		if err = queries.UpsertCoreModule(ctx, db.UpsertCoreModuleParams{
			Code: module.Code, Name: module.Name, NameKey: module.NameKey,
			Version: buildinfo.Version, Description: &description,
			DescriptionKey: module.DescriptionKey, Capabilities: module.Capabilities,
			Status: module.Status,
		}); err != nil {
			return 0, fmt.Errorf("upsert module %s: %w", module.Code, err)
		}
		codes = append(codes, module.Code)
	}
	if _, err = queries.DeleteModulesOutsideCoreCatalog(ctx, codes); err != nil {
		return 0, fmt.Errorf("remove modules outside core catalog: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit module seed: %w", err)
	}
	return len(catalog.Modules), nil
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

func readDictionaryCatalog(catalogPath string) (dictionaryCatalog, error) {
	rawCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return dictionaryCatalog{}, fmt.Errorf("read dictionary catalog: %w", err)
	}
	var catalog dictionaryCatalog
	if err = json.Unmarshal(rawCatalog, &catalog); err != nil {
		return dictionaryCatalog{}, fmt.Errorf("decode dictionary catalog: %w", err)
	}
	if catalog.Version < 1 || len(catalog.Types) == 0 || len(catalog.Items) == 0 {
		return dictionaryCatalog{}, fmt.Errorf("dictionary catalog must contain a version, types, and items")
	}
	types := make(map[string]struct{}, len(catalog.Types))
	for _, definition := range catalog.Types {
		if definition.Code == "" || definition.Name == "" || definition.NameKey == "" || definition.DescriptionKey == "" ||
			(definition.Visibility != "internal" && definition.Visibility != "public") ||
			(definition.ExtensionPolicy != "fixed" && definition.ExtensionPolicy != "open" && definition.ExtensionPolicy != "registered" && definition.ExtensionPolicy != "s3_compatible") ||
			(definition.Status != "active" && definition.Status != "disabled") {
			return dictionaryCatalog{}, fmt.Errorf("dictionary type %q is incomplete", definition.Code)
		}
		if _, exists := types[definition.Code]; exists {
			return dictionaryCatalog{}, fmt.Errorf("duplicate dictionary type %q", definition.Code)
		}
		types[definition.Code] = struct{}{}
	}
	items := make(map[string]struct{}, len(catalog.Items))
	for index := range catalog.Items {
		item := &catalog.Items[index]
		if _, exists := types[item.TypeCode]; !exists || item.Value == "" || item.Label == "" || (item.Locale != "zh-CN" && item.Locale != "en-US") {
			return dictionaryCatalog{}, fmt.Errorf("dictionary item %q/%q is incomplete", item.TypeCode, item.Value)
		}
		if item.Status == "" {
			item.Status = "active"
		}
		if len(item.Extra) == 0 {
			item.Extra = json.RawMessage(`{}`)
		}
		if !json.Valid(item.Extra) || (item.Status != "active" && item.Status != "disabled") {
			return dictionaryCatalog{}, fmt.Errorf("dictionary item %q/%q has invalid metadata", item.TypeCode, item.Value)
		}
		key := item.TypeCode + "/" + item.Value + "/" + item.Locale
		if _, exists := items[key]; exists {
			return dictionaryCatalog{}, fmt.Errorf("duplicate dictionary item %q", key)
		}
		items[key] = struct{}{}
	}
	templates := make(map[string]struct{}, len(catalog.Templates))
	for _, item := range catalog.Templates {
		key := item.Channel + "/" + item.Code + "/" + item.Locale
		if item.Code == "" || item.Name == "" || (item.Channel != "email" && item.Channel != "sms") ||
			(item.Locale != "zh-CN" && item.Locale != "en-US") || item.BodyTemplate == "" ||
			(item.BodyFormat != "plain" && item.BodyFormat != "html") || !json.Valid(item.VariablesSchema) ||
			(item.Status != "active" && item.Status != "disabled") {
			return dictionaryCatalog{}, fmt.Errorf("notification template %q is incomplete", key)
		}
		if item.Channel == "sms" && item.BodyFormat != "plain" {
			return dictionaryCatalog{}, fmt.Errorf("SMS template %q must be plain text", key)
		}
		if _, exists := templates[key]; exists {
			return dictionaryCatalog{}, fmt.Errorf("duplicate notification template %q", key)
		}
		templates[key] = struct{}{}
	}
	return catalog, nil
}

// CoreDictionaries idempotently installs locked global dictionary definitions.
// Tenant overrides are never touched by the core seed.
func CoreDictionaries(ctx context.Context, pool *pgxpool.Pool, catalogPath string) (int, error) {
	catalog, err := readDictionaryCatalog(catalogPath)
	if err != nil {
		return 0, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin dictionary seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	typeIDs := make(map[string]uuid.UUID, len(catalog.Types))
	for _, definition := range catalog.Types {
		description := definition.Description
		descriptionKey := definition.DescriptionKey
		id, upsertErr := queries.UpsertCoreDictionaryType(ctx, db.UpsertCoreDictionaryTypeParams{
			Code: definition.Code, Name: definition.Name, NameKey: &definition.NameKey,
			Description: &description, DescriptionKey: &descriptionKey,
			Visibility: definition.Visibility, ExtensionPolicy: definition.ExtensionPolicy, Status: definition.Status,
		})
		if upsertErr != nil {
			return 0, fmt.Errorf("upsert dictionary type %s: %w", definition.Code, upsertErr)
		}
		typeIDs[definition.Code] = id
	}
	for _, item := range catalog.Items {
		locale := item.Locale
		if err = queries.UpsertCoreDictionaryItem(ctx, db.UpsertCoreDictionaryItemParams{
			DictTypeID: typeIDs[item.TypeCode], ItemValue: item.Value, Label: item.Label,
			Locale: &locale, SortOrder: item.SortOrder, IsDefault: item.IsDefault,
			Extra: item.Extra, Status: item.Status,
		}); err != nil {
			return 0, fmt.Errorf("upsert dictionary item %s/%s/%s: %w", item.TypeCode, item.Value, item.Locale, err)
		}
	}
	for _, item := range catalog.Templates {
		locale := item.Locale
		if err = queries.UpsertCoreNotificationTemplate(ctx, db.UpsertCoreNotificationTemplateParams{
			Code: item.Code, Name: item.Name, Channel: item.Channel, Locale: &locale,
			SubjectTemplate: item.SubjectTemplate, BodyTemplate: item.BodyTemplate,
			BodyFormat: item.BodyFormat, VariablesSchema: item.VariablesSchema, Status: item.Status,
		}); err != nil {
			return 0, fmt.Errorf("upsert notification template %s/%s/%s: %w", item.Channel, item.Code, item.Locale, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit dictionary seed: %w", err)
	}
	return len(catalog.Types) + len(catalog.Items) + len(catalog.Templates), nil
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
	normalized := domain.CreateIdentity{
		TenantCode: input.TenantCode, TenantName: input.TenantName, Email: input.Email,
		DisplayName: input.DisplayName, Locale: input.Locale,
	}.Normalize()
	if normalized.TenantCode == "" || normalized.TenantName == "" || normalized.Email == "" ||
		normalized.DisplayName == "" || (normalized.Locale != "zh-CN" && normalized.Locale != "en-US") {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("bootstrap administrator identity is incomplete")
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("begin bootstrap administrator transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)

	tenantRow, err := queries.GetActiveTenantByCode(ctx, normalized.TenantCode)
	if errors.Is(err, pgx.ErrNoRows) {
		tenantRow, err = queries.CreateTenant(ctx, db.CreateTenantParams{
			Code: normalized.TenantCode, Name: normalized.TenantName,
			Status: "active", Settings: json.RawMessage(`{}`),
		})
	}
	if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("resolve bootstrap tenant: %w", err)
	}

	email := normalized.Email
	userRow, err := queries.GetUserByEmail(ctx, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		passwordHash, hashErr := application.HashPassword(input.Password)
		if hashErr != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, hashErr
		}
		userRow, err = queries.CreateUser(ctx, db.CreateUserParams{
			Email: &email, DisplayName: normalized.DisplayName, Locale: normalized.Locale,
			TimeZone: "UTC", Status: "active", Metadata: json.RawMessage(`{}`),
		})
		if err == nil {
			_, err = queries.CreateUserCredential(ctx, db.CreateUserCredentialParams{
				UserID: userRow.ID, PasswordHash: passwordHash,
			})
		}
		if err == nil {
			_, err = queries.CreateTenantMember(ctx, db.CreateTenantMemberParams{
				TenantID: tenantRow.ID, UserID: userRow.ID,
				DisplayName: &normalized.DisplayName, Status: "active",
			})
		}
		if err != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("create bootstrap identity: %w", err)
		}
	} else if err != nil {
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("resolve bootstrap user: %w", err)
	} else {
		if userRow.Status != "active" {
			return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("bootstrap user is not active")
		}
		if _, credentialErr := queries.GetCredentialByEmail(ctx, &email); credentialErr != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("bootstrap user credential is unavailable: %w", credentialErr)
		}
		if _, memberErr := queries.GetActiveTenantMember(ctx, db.GetActiveTenantMemberParams{
			TenantID: tenantRow.ID, UserID: userRow.ID,
		}); memberErr != nil {
			return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("bootstrap user is not an active member of the target tenant: %w", memberErr)
		}
	}

	user := domain.User{
		ID: userRow.ID, Email: email, DisplayName: userRow.DisplayName,
		Locale: userRow.Locale, TimeZone: userRow.TimeZone, Status: userRow.Status,
		AvatarFileID: userRow.AvatarFileID,
	}
	tenant := domain.Tenant{
		ID: tenantRow.ID, Code: strings.ToLower(tenantRow.Code), Name: tenantRow.Name, Status: tenantRow.Status,
	}
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
		return domain.User{}, domain.Tenant{}, 0, 0, fmt.Errorf("commit bootstrap administrator: %w", err)
	}
	return user, tenant, granted, grantedMenus, nil
}
