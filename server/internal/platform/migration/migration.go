package migration

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Runner struct {
	databaseURL string
	sourcePath  string
}

func NewRunner(databaseURL, sourcePath string) (*Runner, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	absolutePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve migration path: %w", err)
	}
	return &Runner{databaseURL: databaseURL, sourcePath: absolutePath}, nil
}

func (runner *Runner) Steps(count int) error {
	if count == 0 {
		return errors.New("migration step count must not be zero")
	}
	sourceURL := (&url.URL{Scheme: "file", Path: runner.sourcePath}).String()
	instance, err := migrate.New(sourceURL, runner.databaseURL)
	if err != nil {
		return fmt.Errorf("open migration runner: %w", err)
	}
	defer func() {
		_, _ = instance.Close()
	}()
	if err = instance.Steps(count); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply %d migration step(s): %w", count, err)
	}
	return nil
}

func (runner *Runner) Up() error {
	sourceURL := (&url.URL{Scheme: "file", Path: runner.sourcePath}).String()
	instance, err := migrate.New(sourceURL, runner.databaseURL)
	if err != nil {
		return fmt.Errorf("open migration runner: %w", err)
	}
	defer func() {
		_, _ = instance.Close()
	}()
	if err = instance.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply all pending migrations: %w", err)
	}
	return nil
}

func (runner *Runner) Version() (uint, bool, error) {
	sourceURL := (&url.URL{Scheme: "file", Path: runner.sourcePath}).String()
	instance, err := migrate.New(sourceURL, runner.databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("open migration runner: %w", err)
	}
	defer func() {
		_, _ = instance.Close()
	}()
	version, dirty, err := instance.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read migration version: %w", err)
	}
	return version, dirty, nil
}
