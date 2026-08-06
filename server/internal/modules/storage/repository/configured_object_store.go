package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type SecretOpener interface {
	Open(ciphertext []byte, aad string) ([]byte, error)
}

type ConfiguredObjectStore struct {
	pool        *pgxpool.Pool
	secrets     SecretOpener
	local       *LocalObjectStore
	environment string
}

type storageProfile struct {
	domain.UploadPolicy
	Endpoint        string
	Region          string
	UseSSL          bool
	ForcePathStyle  bool
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type storageDriverMetadata struct {
	Adapter  string `json:"adapter"`
	Provider string `json:"provider"`
}

func NewConfiguredObjectStore(pool *pgxpool.Pool, secrets SecretOpener, localRoot, environment string) (*ConfiguredObjectStore, error) {
	if pool == nil || secrets == nil {
		return nil, errors.New("configured object storage requires PostgreSQL and a secret opener")
	}
	var local *LocalObjectStore
	var err error
	if environment == "development" {
		local, err = NewLocalObjectStore(localRoot)
		if err != nil {
			return nil, err
		}
	}
	return &ConfiguredObjectStore{pool: pool, secrets: secrets, local: local, environment: environment}, nil
}

func (store *ConfiguredObjectStore) ResolvePolicy(ctx context.Context, tenantID uuid.UUID) (domain.UploadPolicy, error) {
	profile, err := store.profile(ctx, tenantID)
	if err != nil {
		return domain.UploadPolicy{}, err
	}
	return profile.UploadPolicy, nil
}

func (store *ConfiguredObjectStore) Put(ctx context.Context, ref domain.ObjectRef, content []byte) error {
	profile, client, err := store.route(ctx, ref)
	if err != nil {
		return err
	}
	if ref.Provider == "local" {
		return store.local.Put(ctx, ref, content)
	}
	_, err = client.PutObject(ctx, profile.Bucket, ref.Key, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (store *ConfiguredObjectStore) Open(ctx context.Context, ref domain.ObjectRef) (io.ReadCloser, error) {
	profile, client, err := store.route(ctx, ref)
	if err != nil {
		return nil, err
	}
	if ref.Provider == "local" {
		return store.local.Open(ctx, ref)
	}
	object, err := client.GetObject(ctx, profile.Bucket, ref.Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	if _, err = object.Stat(); err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return object, nil
}

func (store *ConfiguredObjectStore) Delete(ctx context.Context, ref domain.ObjectRef) error {
	profile, client, err := store.route(ctx, ref)
	if err != nil {
		return err
	}
	if ref.Provider == "local" {
		return store.local.Delete(ctx, ref)
	}
	if err = client.RemoveObject(ctx, profile.Bucket, ref.Key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (store *ConfiguredObjectStore) route(ctx context.Context, ref domain.ObjectRef) (storageProfile, *minio.Client, error) {
	if ref.TenantID == uuid.Nil || !validObjectKey(ref.Key) {
		return storageProfile{}, nil, domain.ErrStorageConfig
	}
	profile, err := store.profile(ctx, ref.TenantID)
	if err != nil {
		return storageProfile{}, nil, err
	}
	if profile.Provider != ref.Provider || profile.Bucket != ref.Bucket {
		return storageProfile{}, nil, domain.ErrStorageConfig
	}
	if ref.Provider == "local" {
		if store.local == nil {
			return storageProfile{}, nil, domain.ErrStorageConfig
		}
		return profile, nil, nil
	}
	client, err := minio.New(profile.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(profile.AccessKeyID, profile.SecretAccessKey, profile.SessionToken),
		Secure: profile.UseSSL, Region: profile.Region,
		BucketLookup: bucketLookup(profile.ForcePathStyle),
	})
	if err != nil {
		return storageProfile{}, nil, fmt.Errorf("create object storage client: %w", err)
	}
	return profile, client, nil
}

func bucketLookup(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func (store *ConfiguredObjectStore) profile(ctx context.Context, tenantID uuid.UUID) (storageProfile, error) {
	if tenantID == uuid.Nil {
		return storageProfile{}, domain.ErrStorageConfig
	}
	rows, err := store.pool.Query(ctx, `
SELECT config_key::text, COALESCE(value_json, default_value_json), is_secret,
       secret_ciphertext
FROM sys.config_items
WHERE tenant_id = $1 AND module_code = 'storage' AND config_group = 'cloud'
  AND status = 'active'`, tenantID)
	if err != nil {
		return storageProfile{}, fmt.Errorf("load object storage configuration: %w", err)
	}
	defer rows.Close()
	values := map[string]json.RawMessage{}
	secrets := map[string]string{}
	for rows.Next() {
		var key string
		var raw json.RawMessage
		var secret bool
		var ciphertext []byte
		if err = rows.Scan(&key, &raw, &secret, &ciphertext); err != nil {
			return storageProfile{}, fmt.Errorf("scan object storage configuration: %w", err)
		}
		if secret {
			if len(ciphertext) == 0 {
				continue
			}
			plain, openErr := store.secrets.Open(ciphertext, tenantID.String())
			if openErr != nil {
				return storageProfile{}, domain.ErrStorageConfig
			}
			secrets[key] = strings.TrimSpace(string(plain))
			continue
		}
		if len(raw) > 0 {
			values[key] = raw
		}
	}
	if err = rows.Err(); err != nil {
		return storageProfile{}, fmt.Errorf("read object storage configuration: %w", err)
	}
	profile := storageProfile{UploadPolicy: domain.UploadPolicy{
		Provider:        stringValue(values, "storage.driver", "local"),
		PathPrefix:      safePrefix(stringValue(values, "storage.path_prefix", "appkernia")),
		MaxImageBytes:   boundedInt64(values, "storage.max_image_bytes", domain.MaxAvatarBytes, domain.MaxAvatarBytes),
		MaxFileBytes:    boundedInt64(values, "storage.max_file_bytes", domain.MaxFileBytes, domain.MaxFileBytes),
		ImageMediaTypes: stringSlice(values, "storage.image_media_types", []string{"image/jpeg", "image/png", "image/webp"}),
		FileMediaTypes:  stringSlice(values, "storage.file_media_types", []string{"application/pdf", "application/json", "application/zip", "application/octet-stream", "text/plain", "text/csv", "image/jpeg", "image/png", "image/webp"}),
	}, Region: stringValue(values, "storage.region", ""), UseSSL: boolValue(values, "storage.use_ssl", true), ForcePathStyle: boolValue(values, "storage.force_path_style", false), AccessKeyID: secrets["storage.access_key_id"], SecretAccessKey: secrets["storage.secret_access_key"], SessionToken: secrets["storage.session_token"]}
	driver, err := store.resolveDriver(ctx, tenantID, profile.Provider)
	if err != nil {
		return storageProfile{}, domain.ErrStorageConfig
	}
	if profile.Provider == "local" {
		if driver.Adapter != "local" || store.environment != "development" || store.local == nil {
			return storageProfile{}, domain.ErrStorageConfig
		}
		profile.Bucket = "appkernia-local"
		profile.ConfigurationSafe = true
		return profile, nil
	}
	if driver.Adapter != "s3_compatible" {
		return storageProfile{}, domain.ErrStorageConfig
	}
	if profile.Provider == "oss" || profile.Provider == "cos" || profile.Provider == "qiniu" {
		profile.ForcePathStyle = false
	}
	profile.Bucket = strings.TrimSpace(stringValue(values, "storage.bucket", ""))
	profile.Endpoint, profile.UseSSL, err = normalizeEndpoint(stringValue(values, "storage.endpoint", ""), profile.UseSSL)
	if err != nil || validateProviderProfile(profile.Provider, profile.Endpoint, profile.Region) != nil || profile.Bucket == "" || profile.AccessKeyID == "" || profile.SecretAccessKey == "" || (!profile.UseSSL && store.environment != "development") {
		return storageProfile{}, domain.ErrStorageConfig
	}
	profile.ConfigurationSafe = true
	return profile, nil
}

func validateProviderProfile(provider, endpoint, region string) error {
	host := strings.ToLower(strings.TrimSpace(endpoint))
	region = strings.ToLower(strings.TrimSpace(region))
	switch provider {
	case "cos":
		if region == "" || !strings.HasPrefix(host, "cos.") || !strings.HasSuffix(host, ".myqcloud.com") || !strings.Contains(host, "."+region+".") {
			return errors.New("Tencent COS endpoint must match the configured region")
		}
	case "oss":
		if region == "" || !strings.HasPrefix(host, "oss-") || !strings.HasSuffix(host, ".aliyuncs.com") || !strings.Contains(host, region) {
			return errors.New("Alibaba OSS endpoint must match the configured region")
		}
	case "qiniu":
		if region == "" || !strings.HasSuffix(host, ".qiniucs.com") {
			return errors.New("Qiniu Kodo endpoint and region are required")
		}
	}
	return nil
}

func (store *ConfiguredObjectStore) resolveDriver(ctx context.Context, tenantID uuid.UUID, provider string) (storageDriverMetadata, error) {
	var raw json.RawMessage
	err := store.pool.QueryRow(ctx, `
WITH ranked AS (
    SELECT i.extra,i.status,
           row_number() OVER (ORDER BY
             CASE WHEN i.locale='zh-CN' THEN 0 WHEN i.locale IS NULL THEN 1 ELSE 2 END,
             CASE WHEN i.tenant_id=$1 THEN 0 ELSE 1 END,
             i.id) AS rank
    FROM sys.dict_types d
    JOIN sys.dict_items i ON i.dict_type_id=d.id
    WHERE d.code='storage.driver' AND d.status='active'
      AND (d.tenant_id IS NULL OR d.tenant_id=$1)
      AND (i.tenant_id IS NULL OR i.tenant_id=$1)
      AND i.item_value=$2
)
SELECT extra FROM ranked WHERE rank=1 AND status='active'`, tenantID, provider).Scan(&raw)
	if err != nil {
		return storageDriverMetadata{}, err
	}
	var metadata storageDriverMetadata
	if json.Unmarshal(raw, &metadata) != nil || metadata.Adapter == "" {
		return storageDriverMetadata{}, domain.ErrStorageConfig
	}
	return metadata, nil
}

func normalizeEndpoint(raw string, secure bool) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", secure, errors.New("endpoint is empty")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", secure, errors.New("endpoint is invalid")
		}
		return parsed.Host, parsed.Scheme == "https", nil
	}
	if strings.ContainsAny(raw, "/?#@") {
		return "", secure, errors.New("endpoint is invalid")
	}
	return raw, secure, nil
}

func safePrefix(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	if !validObjectKey(value) {
		return "appkernia"
	}
	return value
}

func validObjectKey(value string) bool {
	clean := path.Clean(strings.TrimSpace(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !strings.HasPrefix(clean, "/") && !strings.Contains(value, "\\")
}

func stringValue(values map[string]json.RawMessage, key, fallback string) string {
	var value string
	if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	return fallback
}

func boolValue(values map[string]json.RawMessage, key string, fallback bool) bool {
	var value bool
	if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
		return value
	}
	return fallback
}

func boundedInt64(values map[string]json.RawMessage, key string, fallback, ceiling int64) int64 {
	var value int64
	if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil && value > 0 && value <= ceiling {
		return value
	}
	return fallback
}

func stringSlice(values map[string]json.RawMessage, key string, fallback []string) []string {
	var parsed []string
	if raw := values[key]; len(raw) == 0 || json.Unmarshal(raw, &parsed) != nil || len(parsed) == 0 || len(parsed) > 100 {
		return append([]string(nil), fallback...)
	}
	out := make([]string, 0, len(parsed))
	seen := map[string]struct{}{}
	for _, value := range parsed {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || len(value) > 255 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}
