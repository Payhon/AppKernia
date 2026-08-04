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
	typ, err := repository.CreateDictType(ctx, principal, settings.DictTypeInput{Code: "tenant." + suffix, Name: "Tenant Dictionary", Status: "active"})
	if err != nil {
		t.Fatalf("create dictionary type: %v", err)
	}
	localeZH, localeEN := "zh-CN", "en-US"
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
	items, err := repository.ListDictItems(ctx, tenant.ID, typ.ID, settings.DictItemFilter{Page: 1, PageSize: 20, Sort: "sort_order"})
	if err != nil || items.Total != 2 || items.Type.IsLocked {
		t.Fatalf("dictionary items=%#v error=%v", items, err)
	}
	if err = repository.DeleteDictItem(ctx, principal, item.ID); err != nil {
		t.Fatalf("delete dictionary item: %v", err)
	}
	if err = repository.DeleteDictItem(ctx, principal, globalItemID); !errors.Is(err, settings.ErrLocked) {
		t.Fatalf("global dictionary item delete error=%v", err)
	}

	rootCode := "root-" + suffix[:8]
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
	moduleCode := "compiled." + suffix
	if _, err = pool.Exec(ctx, `INSERT INTO sys.modules(code,name,version,description,capabilities,status) VALUES($1,'Compiled module','1.2.3','Read only','{"events":true}','enabled')`, moduleCode); err != nil {
		t.Fatal(err)
	}
	modules, err := repository.ListModules(ctx, settings.ModuleFilter{Query: suffix, Status: "enabled"})
	if err != nil || len(modules) != 1 || modules[0].Code != moduleCode {
		t.Fatalf("compiled modules=%#v error=%v", modules, err)
	}
}
