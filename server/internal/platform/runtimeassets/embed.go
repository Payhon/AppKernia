package runtimeassets

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:generate go run ./cmd/generate

//go:embed bundle.zip
var bundle []byte

func Files() (fs.FS, error) {
	reader, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return nil, fmt.Errorf("open embedded runtime assets: %w", err)
	}
	return reader, nil
}

func ReadFile(name string) ([]byte, error) {
	files, err := Files()
	if err != nil {
		return nil, err
	}
	content, err := fs.ReadFile(files, strings.TrimPrefix(name, "/"))
	if err != nil {
		return nil, fmt.Errorf("read embedded runtime asset %s: %w", name, err)
	}
	return content, nil
}

func OpenAPI() ([]byte, error) {
	return ReadFile("openapi/openapi.yaml")
}

// Materialize writes the trusted embedded blueprint subset into an owner-only
// temporary directory for existing migration and seed code that accepts paths.
func Materialize() (string, func(), error) {
	files, err := Files()
	if err != nil {
		return "", nil, err
	}
	root, err := os.MkdirTemp("", "akone-assets-*")
	if err != nil {
		return "", nil, fmt.Errorf("create runtime asset directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	err = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if !fs.ValidPath(name) || strings.HasPrefix(name, ".") || strings.Contains(name, "/.") {
			return fmt.Errorf("embedded runtime asset has unsafe path %q", name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("embedded runtime asset is not a regular file: %s", name)
		}
		content, readErr := fs.ReadFile(files, name)
		if readErr != nil {
			return readErr
		}
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o700); mkdirErr != nil {
			return mkdirErr
		}
		return os.WriteFile(target, content, 0o600)
	})
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("materialize runtime assets: %w", err)
	}
	return root, cleanup, nil
}
