//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentIdentityCreationEnforcesUniqueEmail(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	email := fmt.Sprintf("race-%s@example.test", suffix)
	repo := repository.NewPostgres(pool)
	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for index := range 2 {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			<-start
			user, tenant, createErr := repo.CreateIdentity(ctx, domain.CreateIdentity{
				TenantCode:   fmt.Sprintf("race-%d-%s", index, suffix),
				TenantName:   fmt.Sprintf("Race Tenant %d", index),
				Email:        email,
				DisplayName:  "Race User",
				Locale:       "zh-CN",
				PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$test$test",
			})
			if createErr == nil && (user.ID.Version() != 7 || tenant.ID.Version() != 7) {
				createErr = fmt.Errorf("expected UUIDv7 identifiers")
			}
			results <- createErr
		}(index)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes := 0
	duplicateFailures := 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case errors.Is(result, domain.ErrEmailAlreadyExists):
			duplicateFailures++
		default:
			t.Fatalf("unexpected concurrent create result: %v", result)
		}
	}
	if successes != 1 || duplicateFailures != 1 {
		t.Fatalf("expected one success and one duplicate failure, got success=%d duplicate=%d", successes, duplicateFailures)
	}
}

func TestUpdateSelfProfilePersistsAndAudits(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	repo := repository.NewPostgres(pool)
	user, tenant, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "profile-" + suffix, TenantName: "Profile Tenant",
		Email: "profile-" + suffix + "@example.test", DisplayName: "Before Name",
		Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$test$test",
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	displayName, locale, timeZone := "After Name", "en-US", "Asia/Shanghai"
	updated, err := repo.UpdateSelfProfile(ctx, domain.UpdateSelfProfile{
		UserID: user.ID, TenantID: tenant.ID, DisplayName: &displayName, Locale: &locale,
		TimeZone: &timeZone, RequestID: "profile-update-" + suffix, UserAgent: "integration-test",
	})
	if err != nil {
		t.Fatalf("update self profile: %v", err)
	}
	if updated.DisplayName != displayName || updated.Locale != locale || updated.TimeZone != timeZone {
		t.Fatalf("unexpected updated profile: %#v", updated)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE user_id = $1 AND tenant_id = $2 AND request_id = $3
		  AND action_name = 'iam.me.update' AND succeeded = true
		  AND before_data->>'locale' = 'zh-CN' AND after_data->>'locale' = 'en-US'
	`, user.ID, tenant.ID, "profile-update-"+suffix).Scan(&auditCount); err != nil {
		t.Fatalf("count profile audit records: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one profile audit record, got %d", auditCount)
	}
}
