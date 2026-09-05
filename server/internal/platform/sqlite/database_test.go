package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	platformsqlite "github.com/appkernia/appkernia/server/internal/platform/sqlite"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestOpenCreatesMigratesAndReopens(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), "nested", "data")
	databasePath := filepath.Join(directory, "appkernia.db")

	database, err := platformsqlite.Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("open new database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if runtime.GOOS != "windows" {
		assertMode(t, directory, 0o700)
		assertMode(t, databasePath, 0o600)
	}
	var version, foreignKeys, synchronous int
	var journalMode string
	if err = database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatalf("user_version = %d, err = %v; want 1", version, err)
	}
	if err = database.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, err = %v; want 1", foreignKeys, err)
	}
	if err = database.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil || !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, err = %v; want wal", journalMode, err)
	}
	if err = database.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous); err != nil || synchronous != 2 {
		t.Fatalf("synchronous = %d, err = %v; want FULL (2)", synchronous, err)
	}
	if _, err = database.ExecContext(ctx, "CREATE TABLE local_test_data (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatalf("create local data table: %v", err)
	}
	if _, err = database.ExecContext(ctx, "INSERT INTO local_test_data(value) VALUES (?)", "kept"); err != nil {
		t.Fatalf("insert local data: %v", err)
	}
	if err = database.Close(); err != nil {
		t.Fatalf("close first database: %v", err)
	}

	reopened, err := platformsqlite.Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	var value string
	if err = reopened.QueryRowContext(ctx, "SELECT value FROM local_test_data WHERE id = 1").Scan(&value); err != nil || value != "kept" {
		t.Fatalf("reopened value = %q, err = %v; want kept", value, err)
	}
	var migrations int
	if err = reopened.QueryRowContext(ctx, "SELECT count(*) FROM ak_schema_migrations WHERE version = 1").Scan(&migrations); err != nil || migrations != 1 {
		t.Fatalf("schema migration rows = %d, err = %v; want 1", migrations, err)
	}
}

func TestOpenRejectsInvalidPaths(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		database, err := platformsqlite.Open(context.Background(), " \t\n")
		if database != nil {
			_ = database.Close()
		}
		if err == nil {
			t.Fatal("Open accepted an empty path")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink permissions vary on Windows")
		}
		directory := t.TempDir()
		target := filepath.Join(directory, "target.db")
		if err := os.WriteFile(target, nil, 0o600); err != nil {
			t.Fatalf("create symlink target: %v", err)
		}
		link := filepath.Join(directory, "appkernia.db")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		database, err := platformsqlite.Open(context.Background(), link)
		if database != nil {
			_ = database.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("Open symlink error = %v; want regular-file rejection", err)
		}
	})

	t.Run("writable directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("directory permissions are enforced by ACLs on Windows")
		}
		directory := filepath.Join(t.TempDir(), "shared")
		if err := os.Mkdir(directory, 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o777); err != nil {
			t.Fatal(err)
		}
		database, err := platformsqlite.Open(context.Background(), filepath.Join(directory, "appkernia.db"))
		if database != nil {
			_ = database.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "group- or world-writable") {
			t.Fatalf("Open writable directory error = %v", err)
		}
	})
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode %s = %#o, want %#o", path, got, want)
	}
}
