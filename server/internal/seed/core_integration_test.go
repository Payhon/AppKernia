//go:build integration

package seed

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCoreModulesIsExactIdempotentAndVersioned(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-modules.json")
	previousVersion := buildinfo.Version
	buildinfo.Version = "integration-module-catalog"
	defer func() {
		buildinfo.Version = previousVersion
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer restoreCancel()
		_, _ = CoreModules(restoreCtx, pool, catalogPath)
	}()

	if _, err = pool.Exec(ctx, `
		INSERT INTO sys.modules(code,name,name_key,version,description,description_key,capabilities,status)
		VALUES('e2e.catalog','fixture','fixture.name','fixture','fixture','fixture.description','{"fixture":true}','enabled')
		ON CONFLICT (code) DO UPDATE SET version='fixture'
	`); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		count, seedErr := CoreModules(ctx, pool, catalogPath)
		if seedErr != nil || count != 8 {
			t.Fatalf("run=%d count=%d error=%v", run+1, count, seedErr)
		}
	}

	rows, err := pool.Query(ctx, `SELECT code::text, version FROM sys.modules ORDER BY code`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code, version string
		if err = rows.Scan(&code, &version); err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
		if version != buildinfo.Version {
			t.Fatalf("module %s version=%q want=%q", code, version, buildinfo.Version)
		}
	}
	want := []string{"audit", "iam", "jobs", "notify", "ops", "org", "storage", "sys"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("module codes=%v want=%v", codes, want)
	}
}
