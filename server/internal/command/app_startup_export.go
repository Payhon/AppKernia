package command

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	storagerepo "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type startupExportRecord struct {
	AppID, TenantID                        uuid.UUID
	ZhName, ZhSubtitle, EnName, EnSubtitle string
	Icon                                   storagedomain.ObjectRef
	IconMediaType                          string
}

func appStartupCommand(program string, args []string) error {
	usage := fmt.Sprintf("usage: %s app-startup export --app-id UUID --output DIR [--check]", program)
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(os.Stdout, usage)
		return nil
	}
	if len(args) == 0 || args[0] != "export" {
		return &UsageError{Message: usage}
	}
	flags := flag.NewFlagSet("app-startup export", flag.ContinueOnError)
	appIDValue := flags.String("app-id", "", "public App UUID")
	output := flags.String("output", "", "mobile project directory")
	check := flags.Bool("check", false, "validate generated files without writing")
	if err := parseCommandFlags(flags, args[1:], usage); err != nil {
		return err
	}
	appID, err := uuid.Parse(strings.TrimSpace(*appIDValue))
	if err != nil || strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--app-id must be a UUID and --output is required")
	}
	outputRoot, err := filepath.Abs(*output)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err = requirePostgreSQL(cfg, "app-startup export"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	record, err := loadStartupExportRecord(ctx, pool, appID)
	if err != nil {
		return err
	}
	objects, err := startupObjectStore(cfg, pool)
	if err != nil {
		return err
	}
	reader, err := objects.Open(ctx, record.Icon)
	if err != nil {
		return fmt.Errorf("open startup icon: %w", err)
	}
	iconBytes, readErr := io.ReadAll(io.LimitReader(reader, storagedomain.MaxAvatarBytes+1))
	closeErr := reader.Close()
	if readErr != nil {
		return fmt.Errorf("read startup icon: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close startup icon: %w", closeErr)
	}
	if len(iconBytes) == 0 || int64(len(iconBytes)) > storagedomain.MaxAvatarBytes {
		return fmt.Errorf("startup icon size is invalid")
	}
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[record.IconMediaType]
	if extension == "" {
		return fmt.Errorf("startup icon must be JPEG, PNG, or WebP")
	}
	snapshot := renderStartupSnapshot(record, extension)
	snapshotPath := filepath.Join(outputRoot, "src", "generated", "startup-snapshot.uts")
	iconPath := filepath.Join(outputRoot, "static", "app-startup", "icon."+extension)
	if *check {
		if err = checkFile(snapshotPath, []byte(snapshot)); err != nil {
			return err
		}
		if err = checkFile(iconPath, iconBytes); err != nil {
			return err
		}
		fmt.Printf("app startup snapshot current app_id=%s output=%s\n", appID, outputRoot)
		return nil
	}
	if err = writeGeneratedFile(snapshotPath, []byte(snapshot), 0o644); err != nil {
		return err
	}
	if err = writeGeneratedFile(iconPath, iconBytes, 0o644); err != nil {
		return err
	}
	fmt.Printf("exported app startup snapshot app_id=%s icon=%s output=%s\n", appID, extension, outputRoot)
	return nil
}

func loadStartupExportRecord(ctx context.Context, pool *pgxpool.Pool, appID uuid.UUID) (startupExportRecord, error) {
	var out startupExportRecord
	var provider, bucket, key, mediaType *string
	err := pool.QueryRow(ctx, `SELECT a.id,a.tenant_id,
zh.display_name,zh.subtitle,en.display_name,en.subtitle,
f.provider,f.bucket_name,f.object_key,f.media_type
FROM app.applications a
JOIN app.application_startup_translations zh ON zh.app_id=a.id AND zh.tenant_id=a.tenant_id AND zh.locale='zh-CN'
JOIN app.application_startup_translations en ON en.app_id=a.id AND en.tenant_id=a.tenant_id AND en.locale='en-US'
JOIN storage.files f ON f.id=a.icon_file_id AND f.tenant_id=a.tenant_id
WHERE a.id=$1 AND a.deleted_at IS NULL AND a.appid IS NOT NULL
  AND f.deleted_at IS NULL AND f.status='ready' AND f.scan_status='clean'
  AND lower(COALESCE(f.media_type,'')) IN ('image/jpeg','image/png','image/webp')`, appID).Scan(
		&out.AppID, &out.TenantID, &out.ZhName, &out.ZhSubtitle, &out.EnName, &out.EnSubtitle,
		&provider, &bucket, &key, &mediaType)
	if err != nil {
		return out, fmt.Errorf("load startup export configuration: %w", err)
	}
	if provider == nil || bucket == nil || key == nil || mediaType == nil || strings.TrimSpace(out.ZhName) == "" || strings.TrimSpace(out.EnName) == "" {
		return out, fmt.Errorf("startup export requires App ID, bilingual names, and a scanned icon")
	}
	out.Icon = storagedomain.ObjectRef{TenantID: out.TenantID, Provider: *provider, Bucket: *bucket, Key: *key}
	out.IconMediaType = strings.ToLower(strings.TrimSpace(*mediaType))
	return out, nil
}

func startupObjectStore(cfg config.Config, pool *pgxpool.Pool) (storagedomain.ObjectStore, error) {
	if cfg.ObjectStorageAdapter == "local" {
		return storagerepo.NewLocalObjectStore(cfg.LocalObjectStorageDir)
	}
	if cfg.ObjectStorageAdapter != "configured" {
		return nil, fmt.Errorf("AK_OBJECT_STORAGE_ADAPTER must be local or configured")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.ConfigMasterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode config master key: %w", err)
	}
	sealer, err := settingsrepo.NewAESGCMSealer(key, cfg.ConfigMasterKeyVersion)
	if err != nil {
		return nil, fmt.Errorf("configure config secret sealer: %w", err)
	}
	return storagerepo.NewConfiguredObjectStore(pool, sealer, cfg.LocalObjectStorageDir, cfg.Environment)
}

func renderStartupSnapshot(record startupExportRecord, extension string) string {
	return fmt.Sprintf(`export type BundledStartupTranslation = { readonly displayName: string; readonly subtitle: string }
export type BundledStartupSnapshot = { readonly appId: string; readonly iconPath: string; readonly zhCN: BundledStartupTranslation; readonly enUS: BundledStartupTranslation }

// Generated by akone app-startup export. Do not edit this file directly.
export const bundledStartupSnapshot: BundledStartupSnapshot = {
  appId: %s,
  iconPath: %s,
  zhCN: { displayName: %s, subtitle: %s },
  enUS: { displayName: %s, subtitle: %s },
}
`, strconv.Quote(record.AppID.String()), strconv.Quote("/static/app-startup/icon."+extension), strconv.Quote(record.ZhName), strconv.Quote(record.ZhSubtitle), strconv.Quote(record.EnName), strconv.Quote(record.EnSubtitle))
}

func checkFile(path string, expected []byte) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("generated startup file missing or unreadable %s: %w", path, err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("generated startup file drifted: %s", path)
	}
	return nil
}

func writeGeneratedFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create startup export directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".app-startup-*")
	if err != nil {
		return fmt.Errorf("create startup export temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err = temporary.Write(content); err == nil {
		err = temporary.Chmod(mode)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write startup export file: %w", err)
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("install startup export file: %w", err)
	}
	return nil
}
