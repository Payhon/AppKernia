//go:build integration

package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSystemSettingsTenantLocksSecretsAndDictionaries(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	hash, err := iamapp.HashPassword("settings integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identities := iamrepo.NewPostgres(pool)
	owner, tenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "settings-source-" + suffix, TenantName: "Settings Source", Email: "settings-owner-" + suffix + "@example.test", DisplayName: "Settings Owner", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	otherOwner, otherTenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "settings-other-" + suffix, TenantName: "Settings Other", Email: "settings-other-" + suffix + "@example.test", DisplayName: "Other", Locale: "en-US", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create other identity: %v", err)
	}

	var globalConfigID, globalTypeID, globalItemID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO sys.config_items(module_code,config_group,config_key,display_name,value_type,value_json,is_public,status) VALUES('core','integration',$1,'Global setting','string','"global"',true,'active') RETURNING id`, "global."+suffix).Scan(&globalConfigID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO sys.dict_types(code,name,is_system,status) VALUES($1,'Global dictionary',true,'active') RETURNING id`, "global."+suffix).Scan(&globalTypeID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO sys.dict_items(dict_type_id,item_value,label,locale,status) VALUES($1,'ready','Ready','en-US','active') RETURNING id`, globalTypeID).Scan(&globalItemID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sys.dict_types WHERE code = ANY($1::text[])`, []string{
			"global." + suffix,
			"registered." + suffix,
			"s3." + suffix,
		})
	}()

	repository := NewPostgres(pool)
	principal := settings.Principal{TenantID: tenant.ID, UserID: owner.ID, RequestID: "settings-" + suffix}
	config, err := repository.CreateConfig(ctx, principal, settings.ConfigInput{ModuleCode: "core", ConfigGroup: "integration", ConfigKey: "limit." + suffix, DisplayName: "Limit", ValueType: "integer", Value: []byte(`8`), DefaultValue: []byte(`5`), ValidationSchema: []byte(`{"minimum":1}`), IsPublic: true, Status: "active"}, nil, 0)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}
	page, err := repository.ListConfigs(ctx, tenant.ID, settings.PageFilter{Query: suffix, Page: 1, PageSize: 20, Sort: "key"})
	if err != nil || page.Total != 2 {
		t.Fatalf("visible configs=%#v error=%v", page, err)
	}
	if _, err = repository.UpdateConfig(ctx, principal, globalConfigID, settings.ConfigInput{Version: 1}); !errors.Is(err, settings.ErrLocked) {
		t.Fatalf("global config update error=%v", err)
	}
	if _, err = repository.UpdateConfig(ctx, settings.Principal{TenantID: otherTenant.ID, UserID: otherOwner.ID}, config.ID, settings.ConfigInput{Version: config.Version}); !errors.Is(err, settings.ErrNotFound) {
		t.Fatalf("cross-tenant config update error=%v", err)
	}
	configInput := settings.ConfigInput{ModuleCode: config.ModuleCode, ConfigGroup: config.ConfigGroup, ConfigKey: config.ConfigKey, DisplayName: config.DisplayName, ValueType: config.ValueType, Value: []byte(`9`), DefaultValue: config.DefaultValue, ValidationSchema: config.ValidationSchema, IsPublic: config.IsPublic, Description: config.Description, SortOrder: config.SortOrder, Status: config.Status, Version: config.Version}
	config, err = repository.UpdateConfig(ctx, principal, config.ID, configInput)
	if err != nil || config.Version != 2 {
		t.Fatalf("update config=%#v error=%v", config, err)
	}
	if _, err = repository.UpdateConfig(ctx, principal, config.ID, configInput); !errors.Is(err, settings.ErrConflict) {
		t.Fatalf("stale config update error=%v", err)
	}

	plaintext := []byte("integration-secret-" + suffix)
	sealer, err := NewAESGCMSealer(bytes.Repeat([]byte{0xa5}, 32), 3)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, _, err := sealer.Seal(plaintext, tenant.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	secret, err := repository.CreateConfig(ctx, principal, settings.ConfigInput{ModuleCode: "core", ConfigGroup: "integration", ConfigKey: "secret." + suffix, DisplayName: "Secret", ValueType: "string", IsSecret: true, ValidationSchema: []byte(`{}`), Status: "active"}, ciphertext, 3)
	if err != nil || !secret.SecretConfigured || secret.Value != nil || secret.DefaultValue != nil {
		t.Fatalf("secret config=%#v error=%v", secret, err)
	}
	var stored []byte
	if err = pool.QueryRow(ctx, `SELECT secret_ciphertext FROM sys.config_items WHERE id=$1`, secret.ID).Scan(&stored); err != nil || bytes.Equal(stored, plaintext) {
		t.Fatalf("stored secret error=%v plaintext=%v", err, bytes.Equal(stored, plaintext))
	}
	var auditText string
	if err = pool.QueryRow(ctx, `SELECT coalesce(after_data::text,'') FROM audit.operation_logs WHERE request_id=$1 AND action_name='sys.config.create' AND resource_id=$2 ORDER BY occurred_at DESC LIMIT 1`, principal.RequestID, secret.ID.String()).Scan(&auditText); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(auditText, string(plaintext)) || strings.Contains(auditText, string(ciphertext)) || strings.Contains(auditText, "secret_ciphertext") {
		t.Fatalf("audit leaked secret data: %s", auditText)
	}
	publicValues, err := repository.ListPublicConfigs(ctx)
	if err != nil {
		t.Fatalf("list public configs: %v", err)
	}
	globalKey := "core.integration.global." + suffix
	if string(publicValues[globalKey]) != `"global"` {
		t.Fatalf("global public config missing: key=%s values=%#v", globalKey, publicValues)
	}
	if _, exists := publicValues["core.integration."+config.ConfigKey]; exists {
		t.Fatalf("tenant-scoped public config leaked: %#v", publicValues)
	}
	if _, exists := publicValues["core.integration."+secret.ConfigKey]; exists {
		t.Fatalf("secret config leaked: %#v", publicValues)
	}

	types, err := repository.ListDictTypes(ctx, tenant.ID, settings.PageFilter{Query: suffix, Page: 1, PageSize: 20, Sort: "key"})
	if err != nil || types.Total != 1 || !types.Items[0].IsLocked {
		t.Fatalf("global dictionary types=%#v error=%v", types, err)
	}
	if _, err = repository.UpdateDictType(ctx, principal, globalTypeID, settings.DictTypeInput{Code: "locked", Name: "Locked", Status: "active"}); !errors.Is(err, settings.ErrLocked) {
		t.Fatalf("global dictionary update error=%v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE sys.dict_types SET visibility='public',extension_policy='open' WHERE id=$1`, globalTypeID); err != nil {
		t.Fatal(err)
	}
	localeEN := "en-US"
	if _, err = repository.CreateDictItem(ctx, principal, globalTypeID, settings.DictItemInput{ItemValue: "ready", Label: "Tenant ready", Locale: &localeEN, Extra: []byte(`{}`), Status: "active"}); err != nil {
		t.Fatalf("create tenant overlay on global dictionary: %v", err)
	}
	resolved, err := repository.ResolveDictionary(ctx, &tenant.ID, "global."+suffix, "en-US", false)
	if err != nil || len(resolved.Items) != 1 || resolved.Items[0].Label != "Tenant ready" {
		t.Fatalf("tenant dictionary overlay=%#v error=%v", resolved, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE sys.dict_items SET status='disabled' WHERE dict_type_id=$1 AND tenant_id=$2 AND item_value='ready' AND locale='en-US'`, globalTypeID, tenant.ID); err != nil {
		t.Fatal(err)
	}
	resolved, err = repository.ResolveDictionary(ctx, &tenant.ID, "global."+suffix, "en-US", false)
	if err != nil || len(resolved.Items) != 0 {
		t.Fatalf("disabled tenant overlay must hide the global option: %#v error=%v", resolved, err)
	}
	publicResolved, err := repository.ResolveDictionary(ctx, nil, "global."+suffix, "en-US", true)
	if err != nil || len(publicResolved.Items) != 1 || publicResolved.Items[0].Label != "Ready" {
		t.Fatalf("public dictionary isolation=%#v error=%v", publicResolved, err)
	}
	typ, err := repository.CreateDictType(ctx, principal, settings.DictTypeInput{Code: "tenant." + suffix, Name: "Tenant Dictionary", Status: "active"})
	if err != nil {
		t.Fatalf("create dictionary type: %v", err)
	}
	localeZH := "zh-CN"
	item, err := repository.CreateDictItem(ctx, principal, typ.ID, settings.DictItemInput{ItemValue: "ready", Label: "就绪", Locale: &localeZH, Extra: []byte(`{}`), Status: "active"})
	if err != nil {
		t.Fatalf("create dictionary item: %v", err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, typ.ID, settings.DictItemInput{ItemValue: "ready", Label: "重复", Locale: &localeZH, Extra: []byte(`{}`), Status: "active"}); !errors.Is(err, settings.ErrConflict) {
		t.Fatalf("duplicate locale item error=%v", err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, typ.ID, settings.DictItemInput{ItemValue: "ready", Label: "Ready", Locale: &localeEN, Extra: []byte(`{}`), Status: "active"}); err != nil {
		t.Fatalf("second locale item: %v", err)
	}
	concurrentErrors := make(chan error, 2)
	for range 2 {
		go func() {
			_, createErr := repository.CreateDictItem(ctx, principal, typ.ID, settings.DictItemInput{ItemValue: "concurrent", Label: "Concurrent", Locale: &localeEN, Extra: []byte(`{}`), Status: "active"})
			concurrentErrors <- createErr
		}()
	}
	var concurrentSuccess, concurrentConflict int
	for range 2 {
		createErr := <-concurrentErrors
		if createErr == nil {
			concurrentSuccess++
		} else if errors.Is(createErr, settings.ErrConflict) {
			concurrentConflict++
		} else {
			t.Fatalf("unexpected concurrent dictionary error: %v", createErr)
		}
	}
	if concurrentSuccess != 1 || concurrentConflict != 1 {
		t.Fatalf("concurrent dictionary uniqueness successes=%d conflicts=%d", concurrentSuccess, concurrentConflict)
	}
	items, err := repository.ListDictItems(ctx, tenant.ID, typ.ID, settings.DictItemFilter{Page: 1, PageSize: 20, Sort: "sort_order"})
	if err != nil || items.Total != 3 || items.Type.IsLocked {
		t.Fatalf("dictionary items=%#v error=%v", items, err)
	}
	if err = repository.DeleteDictItem(ctx, principal, item.ID); err != nil {
		t.Fatalf("delete dictionary item: %v", err)
	}
	if err = repository.DeleteDictItem(ctx, principal, globalItemID); !errors.Is(err, settings.ErrLocked) {
		t.Fatalf("global dictionary item delete error=%v", err)
	}

	var registeredTypeID, s3TypeID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO sys.dict_types(code,name,is_system,status,extension_policy) VALUES($1,'Registered',true,'active','registered') RETURNING id`, "registered."+suffix).Scan(&registeredTypeID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO sys.dict_items(dict_type_id,item_value,label,locale,extra,status) VALUES($1,'compiled','Compiled','en-US','{"adapter":"compiled"}'::jsonb,'active')`, registeredTypeID); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, registeredTypeID, settings.DictItemInput{ItemValue: "compiled", Label: "Tenant compiled", Locale: &localeEN, Extra: []byte(`{"adapter":"compiled"}`), Status: "active"}); err != nil {
		t.Fatalf("registered built-in override error=%v", err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, registeredTypeID, settings.DictItemInput{ItemValue: "unknown", Label: "Unknown", Locale: &localeZH, Extra: []byte(`{"adapter":"unknown"}`), Status: "active"}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("unknown registered capability error=%v", err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO sys.dict_types(code,name,is_system,status,extension_policy) VALUES($1,'S3 compatible',true,'active','s3_compatible') RETURNING id`, "s3."+suffix).Scan(&s3TypeID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO sys.dict_items(dict_type_id,item_value,label,locale,extra,status) VALUES($1,'local','Local','en-US','{"adapter":"local"}'::jsonb,'active')`, s3TypeID); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, s3TypeID, settings.DictItemInput{ItemValue: "local", Label: "Tenant local", Locale: &localeEN, Extra: []byte(`{"adapter":"local"}`), Status: "active"}); err != nil {
		t.Fatalf("built-in non-S3 metadata override error=%v", err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, s3TypeID, settings.DictItemInput{ItemValue: "custom-s3", Label: "Custom S3", Locale: &localeZH, Extra: []byte(`{"adapter":"s3_compatible","provider":"custom-s3"}`), Status: "active"}); err != nil {
		t.Fatalf("custom S3-compatible driver error=%v", err)
	}
	if _, err = repository.CreateDictItem(ctx, principal, s3TypeID, settings.DictItemInput{ItemValue: "custom-code", Label: "Custom code", Locale: &localeEN, Extra: []byte(`{"adapter":"plugin"}`), Status: "active"}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("unsafe storage adapter error=%v", err)
	}

	rootCode := "root-" + suffix[:8]
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM audit.operation_logs WHERE resource_type='sys.region' AND resource_id LIKE $1`, rootCode+"%")
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sys.regions WHERE code LIKE $1 AND parent_code IS NOT NULL`, rootCode+"%")
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sys.regions WHERE code=$1`, rootCode)
	}()
	if _, err = pool.Exec(ctx, `INSERT INTO sys.regions(code,level,name,full_name,status) VALUES($1,0,'Root','Root','active')`, rootCode); err != nil {
		t.Fatal(err)
	}
	rows := make([][]any, 250)
	for index := range rows {
		rows[index] = []any{fmt.Sprintf("%s-%03d", rootCode, index), rootCode, int16(1), "Child", "Root / Child", "active"}
	}
	if copied, copyErr := pool.CopyFrom(ctx, pgx.Identifier{"sys", "regions"}, []string{"code", "parent_code", "level", "name", "full_name", "status"}, pgx.CopyFromRows(rows)); copyErr != nil || copied != 250 {
		t.Fatalf("large region fixture copied=%d error=%v", copied, copyErr)
	}
	roots, err := repository.ListRegions(ctx, settings.RegionFilter{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	var rootFound bool
	for _, region := range roots {
		if region.Code == rootCode {
			rootFound = region.HasChildren
		}
	}
	if !rootFound {
		t.Fatalf("lazy root missing or has_children=false: %#v", roots)
	}
	children, err := repository.ListRegions(ctx, settings.RegionFilter{ParentCode: rootCode, Limit: 100})
	if err != nil || len(children) != 100 {
		t.Fatalf("lazy child limit=%d error=%v", len(children), err)
	}

	managedCode := rootCode + "-managed"
	managed, err := repository.CreateRegion(ctx, principal, settings.RegionCreateInput{Code: managedCode, ParentCode: rootCode, Name: "Managed City", FullName: "Root / Managed City", Status: "active"})
	if err != nil || managed.Level != 1 || managed.Version != 1 {
		t.Fatalf("managed region=%+v error=%v", managed, err)
	}
	managed, err = repository.UpdateRegion(ctx, principal, managedCode, settings.RegionUpdateInput{Name: "Managed City Updated", FullName: "Root / Managed City Updated", Status: "active", Version: managed.Version})
	if err != nil || managed.Name != "Managed City Updated" || managed.Version != 2 {
		t.Fatalf("updated managed region=%+v error=%v", managed, err)
	}
	if _, err = repository.UpdateRegion(ctx, principal, managedCode, settings.RegionUpdateInput{Name: "Stale", FullName: "Root / Stale", Status: "active", Version: 1}); !errors.Is(err, settings.ErrConflict) {
		t.Fatalf("stale update error=%v", err)
	}

	leafCode := managedCode + "-leaf"
	leaf, err := repository.CreateRegion(ctx, principal, settings.RegionCreateInput{Code: leafCode, ParentCode: managedCode, Name: "Managed County", FullName: "Root / Managed City Updated / Managed County", Status: "active"})
	if err != nil || leaf.Level != 2 {
		t.Fatalf("leaf region=%+v error=%v", leaf, err)
	}
	if _, err = repository.CreateRegion(ctx, principal, settings.RegionCreateInput{Code: leafCode + "-deep", ParentCode: leafCode, Name: "Too Deep", FullName: "Too Deep", Status: "active"}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("deep child error=%v", err)
	}
	if err = repository.DeleteRegion(ctx, principal, managedCode); !errors.Is(err, settings.ErrConflict) {
		t.Fatalf("parent delete error=%v", err)
	}
	if err = repository.DeleteRegion(ctx, principal, leafCode); err != nil {
		t.Fatalf("delete leaf: %v", err)
	}
	managedChildren, err := repository.ListRegions(ctx, settings.RegionFilter{ParentCode: managedCode, Limit: 100})
	if err != nil || len(managedChildren) != 0 {
		t.Fatalf("soft-deleted children=%+v error=%v", managedChildren, err)
	}

	seedName := "Seed Must Not Replace Manual Edit"
	seedFullName := "Root / Seed Must Not Replace Manual Edit"
	queries := db.New(pool)
	if err = queries.UpsertCoreRegion(ctx, db.UpsertCoreRegionParams{Code: managedCode, ParentCode: &rootCode, Level: 1, Name: seedName, FullName: &seedFullName, Status: "disabled", Metadata: []byte(`{"source":"seed"}`)}); err != nil {
		t.Fatalf("seed managed region: %v", err)
	}
	if err = queries.UpsertCoreRegion(ctx, db.UpsertCoreRegionParams{Code: leafCode, ParentCode: &managedCode, Level: 2, Name: seedName, FullName: &seedFullName, Status: "active", Metadata: []byte(`{"source":"seed"}`)}); err != nil {
		t.Fatalf("seed deleted region: %v", err)
	}
	managedAfterSeed, err := getRegion(ctx, pool, managedCode, false)
	if err != nil || managedAfterSeed.Name != "Managed City Updated" || managedAfterSeed.Status != "active" {
		t.Fatalf("seed overwrote managed region=%+v error=%v", managedAfterSeed, err)
	}
	deletedAfterSeed, err := repository.ListRegions(ctx, settings.RegionFilter{Query: leafCode, Limit: 100})
	if err != nil || len(deletedAfterSeed) != 0 {
		t.Fatalf("seed restored deleted region=%+v error=%v", deletedAfterSeed, err)
	}
	var regionAuditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE resource_type='sys.region' AND resource_id LIKE $1`, rootCode+"%").Scan(&regionAuditCount); err != nil || regionAuditCount != 4 {
		t.Fatalf("region audit count=%d error=%v", regionAuditCount, err)
	}
}
