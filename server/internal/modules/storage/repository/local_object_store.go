package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
)

type LocalObjectStore struct {
	root string
}

func NewLocalObjectStore(root string) (*LocalObjectStore, error) {
	absolute, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, errors.New("local object storage root is invalid")
	}
	if err = os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create local object storage root: %w", err)
	}
	return &LocalObjectStore{root: absolute}, nil
}

func (store *LocalObjectStore) Put(_ context.Context, objectKey string, content []byte) error {
	target, err := store.resolve(objectKey)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create local object directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ak-upload-*")
	if err != nil {
		return fmt.Errorf("create local object temporary file: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if err = temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(content)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err != nil {
		return fmt.Errorf("write local object: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close local object: %w", closeErr)
	}
	if err = os.Rename(temporaryName, target); err != nil {
		return fmt.Errorf("commit local object: %w", err)
	}
	return nil
}

func (store *LocalObjectStore) Open(_ context.Context, objectKey string) (io.ReadCloser, error) {
	target, err := store.resolve(objectKey)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domain.ErrObjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open local object: %w", err)
	}
	return file, nil
}

func (store *LocalObjectStore) Delete(_ context.Context, objectKey string) error {
	target, err := store.resolve(objectKey)
	if err != nil {
		return err
	}
	if err = os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete local object: %w", err)
	}
	return nil
}

func (store *LocalObjectStore) resolve(objectKey string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(objectKey)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("local object key is invalid")
	}
	target := filepath.Join(store.root, clean)
	if target == store.root || !strings.HasPrefix(target, store.root+string(filepath.Separator)) {
		return "", errors.New("local object key escapes storage root")
	}
	return target, nil
}
