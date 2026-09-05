// Command generate creates the deterministic runtime asset bundle embedded in
// akone. It is invoked by go generate and release preflight.
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "generate runtime assets:", err)
		os.Exit(1)
	}
}

func run() error {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve generator source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "..", "..", ".."))
	output := filepath.Join(repositoryRoot, "server", "internal", "platform", "runtimeassets", "bundle.zip")
	files := []string{
		"server/openapi/openapi.yaml",
		"blueprint/backend/spec/core-permissions.json",
		"blueprint/backend/spec/core-modules.json",
		"blueprint/backend/spec/core-configs.json",
		"blueprint/backend/spec/core-regions.json",
		"blueprint/backend/spec/core-dictionaries.json",
		"blueprint/admin-frontend/spec/admin-menu-seed.json",
		"blueprint/admin-frontend/integration/core-permissions.delta.json",
		"blueprint/mobile/integration/app-permissions.delta.json",
	}
	migrations, err := filepath.Glob(filepath.Join(repositoryRoot, "blueprint", "backend", "db", "migrations", "*.sql"))
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return fmt.Errorf("no database migrations found")
	}
	for _, migration := range migrations {
		relative, relErr := filepath.Rel(repositoryRoot, migration)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(relative))
	}
	sort.Strings(files)
	temporary, err := os.CreateTemp(filepath.Dir(output), ".bundle-*.zip")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	archive := zip.NewWriter(temporary)
	fixedTime := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range files {
		source := filepath.Join(repositoryRoot, filepath.FromSlash(name))
		input, openErr := os.Open(source)
		if openErr != nil {
			return openErr
		}
		archiveName := strings.TrimPrefix(name, "server/")
		header := &zip.FileHeader{Name: archiveName, Method: zip.Deflate}
		header.SetMode(0o644)
		header.SetModTime(fixedTime)
		writer, createErr := archive.CreateHeader(header)
		if createErr == nil {
			_, createErr = io.Copy(writer, input)
		}
		closeErr := input.Close()
		if createErr != nil {
			return createErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err = archive.Close(); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, output)
}
