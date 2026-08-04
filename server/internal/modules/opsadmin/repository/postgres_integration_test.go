//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHealthAndRuntimeSummaryAreBoundedAndSecretFree(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	pool, e := pgxpool.New(context.Background(), dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	repo := NewPostgres(pool, Config{ObjectStorageConfigured: true})
	health := repo.Health(context.Background())
	if health.Status != "ready" || len(health.Dependencies) != 4 {
		t.Fatalf("health=%#v", health)
	}
	summary, e := repo.Runtime(context.Background(), uuid.New(), time.Now().Add(-time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(summary)
	text := strings.ToLower(string(raw))
	for _, forbidden := range []string{"postgres://", "password", "secret", "/users/", "appkernia-dev-only"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("runtime leaked %q: %s", forbidden, raw)
		}
	}
	if summary.Queue.Status != "ready" && summary.Queue.Status != "unknown" {
		t.Fatalf("queue=%#v", summary.Queue)
	}
}
