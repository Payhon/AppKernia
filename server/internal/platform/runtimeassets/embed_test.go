package runtimeassets

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEmbeddedRuntimeAssets(t *testing.T) {
	openAPI, err := OpenAPI()
	if err != nil || !strings.Contains(string(openAPI), "openapi:") {
		t.Fatalf("OpenAPI() missing canonical document: %v", err)
	}
	root, cleanup, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	for _, name := range []string{
		"blueprint/backend/db/migrations/000001_extensions_and_schemas.up.sql",
		"blueprint/backend/spec/core-regions.json",
		"blueprint/admin-frontend/spec/admin-menu-seed.json",
	} {
		if info, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(name))); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("embedded asset %s missing: %v", name, statErr)
		}
	}
}

func TestBundleMatchesRepositorySources(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", ".."))
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	migrationCount := 0
	err = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		sourceName := name
		if strings.HasPrefix(name, "openapi/") {
			sourceName = "server/" + name
		}
		embedded, readErr := fs.ReadFile(files, name)
		if readErr != nil {
			return readErr
		}
		source, readErr := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(sourceName)))
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(embedded, source) {
			return &assetDriftError{name: name}
		}
		if strings.HasPrefix(name, "blueprint/backend/db/migrations/") {
			migrationCount++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceMigrations, err := filepath.Glob(filepath.Join(repositoryRoot, "blueprint", "backend", "db", "migrations", "*.sql"))
	if err != nil || migrationCount != len(sourceMigrations) {
		t.Fatalf("embedded migrations=%d source migrations=%d err=%v", migrationCount, len(sourceMigrations), err)
	}
}

type assetDriftError struct{ name string }

func (err *assetDriftError) Error() string { return "embedded runtime asset drifted: " + err.name }
