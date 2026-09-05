package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

const (
	driverName    = "sqlite3"
	schemaVersion = 1
)

// schemaSQL is intentionally owned by the standalone SQLite runtime. The
// PostgreSQL migrations remain the source for the full server deployment.
//
//go:embed schema.sql
var schemaSQL string

// Open opens (or creates) the SQLite database at path and applies the embedded
// schema.
func Open(ctx context.Context, path string) (_ *sql.DB, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("SQLite database path is required")
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite database path: %w", err)
	}
	if err = prepareFile(absolutePath); err != nil {
		return nil, err
	}

	databaseURL := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := databaseURL.Query()
	query.Set("_txlock", "immediate")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(on)")
	query.Add("_pragma", "journal_mode(wal)")
	query.Add("_pragma", "synchronous(full)")
	databaseURL.RawQuery = query.Encode()

	database, err := sql.Open(driverName, databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	defer func() {
		if err != nil {
			_ = database.Close()
		}
	}()

	// SQLite permits concurrent readers, but a single pooled connection avoids
	// writer upgrade races and makes transaction boundaries deterministic.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if err = database.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect SQLite database: %w", err)
	}
	if err = migrate(ctx, database); err != nil {
		return nil, err
	}
	if err = verifyPragmas(ctx, database); err != nil {
		return nil, err
	}
	if err = os.Chmod(absolutePath, 0o600); err != nil {
		return nil, fmt.Errorf("secure SQLite database file: %w", err)
	}
	return database, nil
}

func prepareFile(path string) error {
	directory := filepath.Dir(path)
	directoryInfo, statErr := os.Stat(directory)
	createdDirectory := errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !createdDirectory {
		return fmt.Errorf("inspect SQLite database directory: %w", statErr)
	}
	if !createdDirectory && !directoryInfo.IsDir() {
		return fmt.Errorf("SQLite database directory is not a directory: %s", directory)
	}
	if runtime.GOOS != "windows" && !createdDirectory && directoryInfo.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("SQLite database directory must not be group- or world-writable: %s", directory)
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create SQLite database directory: %w", err)
	}
	if createdDirectory {
		if err := os.Chmod(directory, 0o700); err != nil {
			return fmt.Errorf("secure SQLite database directory: %w", err)
		}
	}

	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("SQLite database path must be a regular file: %s", path)
		}
		if err = os.Chmod(path, 0o600); err != nil {
			return fmt.Errorf("secure SQLite database file: %w", err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect SQLite database file: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("create SQLite database file: %w", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("close new SQLite database file: %w", closeErr)
	}
	return nil
}

func migrate(ctx context.Context, database *sql.DB) error {
	var version int
	if err := database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read SQLite schema version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("SQLite schema version %d is newer than supported version %d", version, schemaVersion)
	}
	if version == schemaVersion {
		return nil
	}

	tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin SQLite schema migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply SQLite schema version %d: %w", schemaVersion, err)
	}
	if _, err = tx.ExecContext(ctx,
		"INSERT OR IGNORE INTO ak_schema_migrations(version, name, applied_at) VALUES (?, ?, ?)",
		schemaVersion, "standalone_core", sqliteTimeNow(),
	); err != nil {
		return fmt.Errorf("record SQLite schema version %d: %w", schemaVersion, err)
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return fmt.Errorf("set SQLite schema version %d: %w", schemaVersion, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit SQLite schema version %d: %w", schemaVersion, err)
	}
	return nil
}

func verifyPragmas(ctx context.Context, database *sql.DB) error {
	var foreignKeys int
	if err := database.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return fmt.Errorf("verify SQLite foreign key enforcement: %w", err)
	}
	if foreignKeys != 1 {
		return errors.New("SQLite foreign key enforcement is disabled")
	}
	var journalMode string
	if err := database.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("verify SQLite journal mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("SQLite WAL mode is unavailable: journal_mode=%s", journalMode)
	}
	return nil
}

func sqliteTimeNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
}
