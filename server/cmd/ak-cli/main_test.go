package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedDevelopmentAdminSkipsWithoutPasswordFile(t *testing.T) {
	t.Setenv("AK_SEED_ADMIN_PASSWORD_FILE", "")
	seeded, err := seedDevelopmentAdmin(t.Context(), nil, "development")
	if err != nil || seeded {
		t.Fatalf("seeded=%t err=%v", seeded, err)
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
