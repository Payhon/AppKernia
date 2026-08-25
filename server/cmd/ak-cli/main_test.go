package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSeedDevelopmentAdminSkipsWithoutPasswordFile(t *testing.T) {
	t.Setenv("AK_SEED_ADMIN_PASSWORD_FILE", "")
	seeded, err := seedDevelopmentAdmin(t.Context(), nil, "development")
	if err != nil || seeded {
		t.Fatalf("seeded=%t err=%v", seeded, err)
	}
}

func TestRenderStartupSnapshotIsDeterministicAndEscapesValues(t *testing.T) {
	record := startupExportRecord{AppID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), ZhName: "应用'名称", ZhSubtitle: "副标题", EnName: "App", EnSubtitle: "Explore"}
	first := renderStartupSnapshot(record, "png")
	second := renderStartupSnapshot(record, "png")
	if first != second || !strings.Contains(first, `iconPath: "/static/app-startup/icon.png"`) || !strings.Contains(first, `displayName: "应用'名称"`) {
		t.Fatalf("unexpected generated snapshot:\n%s", first)
	}
}

func TestCheckFileDetectsStartupSnapshotDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup-snapshot.uts")
	if err := os.WriteFile(path, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkFile(path, []byte("current")); err != nil {
		t.Fatalf("current file rejected: %v", err)
	}
	if err := checkFile(path, []byte("expected")); err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("drift was not detected: %v", err)
	}
}

func TestSeedDevelopmentAdminRejectsSecretOutsideDevelopment(t *testing.T) {
	passwordFile := filepath.Join(t.TempDir(), "seed-admin-password")
	if err := os.WriteFile(passwordFile, []byte("integration seed password 2026!\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AK_SEED_ADMIN_PASSWORD_FILE", passwordFile)
	seeded, err := seedDevelopmentAdmin(t.Context(), nil, "production")
	if err == nil || seeded || !strings.Contains(err.Error(), "allowed only in development") {
		t.Fatalf("seeded=%t err=%v", seeded, err)
	}
}

func TestSeedDevelopmentAdminRejectsEmptyPasswordFile(t *testing.T) {
	passwordFile := filepath.Join(t.TempDir(), "seed-admin-password")
	if err := os.WriteFile(passwordFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AK_SEED_ADMIN_PASSWORD_FILE", passwordFile)
	seeded, err := seedDevelopmentAdmin(t.Context(), nil, "development")
	if err == nil || seeded || !strings.Contains(err.Error(), "password file is empty") {
		t.Fatalf("seeded=%t err=%v", seeded, err)
	}
}
