package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	accessadminapp "github.com/appkernia/appkernia/server/internal/modules/accessadmin/application"
	accessadminrepo "github.com/appkernia/appkernia/server/internal/modules/accessadmin/repository"
	accessadminhttp "github.com/appkernia/appkernia/server/internal/modules/accessadmin/transport/http"
	apiclientadminapp "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/application"
	apiclientadminrepo "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/repository"
	apiclientadminhttp "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/transport/http"
	appmanagementapp "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	appmanagementjobdefs "github.com/appkernia/appkernia/server/internal/modules/appmanagement/jobdefs"
	appmanagementrepo "github.com/appkernia/appkernia/server/internal/modules/appmanagement/repository"
	appmanagementhttp "github.com/appkernia/appkernia/server/internal/modules/appmanagement/transport/http"
	auditadminapp "github.com/appkernia/appkernia/server/internal/modules/auditadmin/application"
	auditadminrepo "github.com/appkernia/appkernia/server/internal/modules/auditadmin/repository"
	auditadminhttp "github.com/appkernia/appkernia/server/internal/modules/auditadmin/transport/http"
	blockruleadminapp "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/application"
	blockruleadminrepo "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/repository"
	blockruleadminhttp "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/transport/http"
	contentapp "github.com/appkernia/appkernia/server/internal/modules/content/application"
	contentrepo "github.com/appkernia/appkernia/server/internal/modules/content/repository"
	contenthttp "github.com/appkernia/appkernia/server/internal/modules/content/transport/http"
	dashboardapp "github.com/appkernia/appkernia/server/internal/modules/dashboard/application"
	dashboardrepo "github.com/appkernia/appkernia/server/internal/modules/dashboard/repository"
	dashboardhttp "github.com/appkernia/appkernia/server/internal/modules/dashboard/transport/http"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	iamhttp "github.com/appkernia/appkernia/server/internal/modules/iam/transport/http"
	identitysecurityapp "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/application"
	identitysecurityrepo "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/repository"
	identitysecurityhttp "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/transport/http"
	jobadminapp "github.com/appkernia/appkernia/server/internal/modules/jobadmin/application"
	jobadminrepo "github.com/appkernia/appkernia/server/internal/modules/jobadmin/repository"
	jobadminhttp "github.com/appkernia/appkernia/server/internal/modules/jobadmin/transport/http"
	mobileprofileapp "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/application"
	mobileprofilerepo "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/repository"
	mobileprofilehttp "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/transport/http"
	notificationadminapp "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/application"
	notificationjobdefs "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	notificationadminrepo "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/repository"
	notificationadminhttp "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/transport/http"
	opsadminapp "github.com/appkernia/appkernia/server/internal/modules/opsadmin/application"
	opsadminrepo "github.com/appkernia/appkernia/server/internal/modules/opsadmin/repository"
	opsadminhttp "github.com/appkernia/appkernia/server/internal/modules/opsadmin/transport/http"
	orgapp "github.com/appkernia/appkernia/server/internal/modules/org/application"
	orgrepo "github.com/appkernia/appkernia/server/internal/modules/org/repository"
	orghttp "github.com/appkernia/appkernia/server/internal/modules/org/transport/http"
	platformapp "github.com/appkernia/appkernia/server/internal/modules/platform/application"
	platformhttp "github.com/appkernia/appkernia/server/internal/modules/platform/transport/http"
	pushapp "github.com/appkernia/appkernia/server/internal/modules/push/application"
	pushdomain "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	pushprovider "github.com/appkernia/appkernia/server/internal/modules/push/provider"
	pushrepo "github.com/appkernia/appkernia/server/internal/modules/push/repository"
	pushhttp "github.com/appkernia/appkernia/server/internal/modules/push/transport/http"
	sessionadminapp "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/application"
	sessionadminrepo "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/repository"
	sessionadminhttp "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/transport/http"
	shareconfigapp "github.com/appkernia/appkernia/server/internal/modules/shareconfig/application"
	shareconfigrepo "github.com/appkernia/appkernia/server/internal/modules/shareconfig/repository"
	shareconfighttp "github.com/appkernia/appkernia/server/internal/modules/shareconfig/transport/http"
	storageapp "github.com/appkernia/appkernia/server/internal/modules/storage/application"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	storagerepo "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	storagehttp "github.com/appkernia/appkernia/server/internal/modules/storage/transport/http"
	storageadminapp "github.com/appkernia/appkernia/server/internal/modules/storageadmin/application"
	storageadminrepo "github.com/appkernia/appkernia/server/internal/modules/storageadmin/repository"
	storageadminhttp "github.com/appkernia/appkernia/server/internal/modules/storageadmin/transport/http"
	settingsapp "github.com/appkernia/appkernia/server/internal/modules/systemsettings/application"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	settingshttp "github.com/appkernia/appkernia/server/internal/modules/systemsettings/transport/http"
	tenantadminapp "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/application"
	tenantadminrepo "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/repository"
	tenantadminhttp "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/transport/http"
	useradminapp "github.com/appkernia/appkernia/server/internal/modules/useradmin/application"
	useradminrepo "github.com/appkernia/appkernia/server/internal/modules/useradmin/repository"
	useradminhttp "github.com/appkernia/appkernia/server/internal/modules/useradmin/transport/http"
	webhookadminapp "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/application"
	webhookadmindomain "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
	webhookadminrepo "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/repository"
	webhookadminhttp "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/transport/http"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	platformnotification "github.com/appkernia/appkernia/server/internal/platform/notification"
	platformnotificationhttp "github.com/appkernia/appkernia/server/internal/platform/notification/transport/http"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type API struct {
	server *ghttp.Server
	pool   *pgxpool.Pool
}

func NewAPI(ctx context.Context, cfg config.Config) (*API, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	settingsSealer, err := configSecretSealer(cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	riverInsertClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create River insert client: %w", err)
	}
	queueDefinitions := append(notificationjobdefs.Definitions(), appmanagementjobdefs.Definitions()...)
	trackedQueue := jobqueue.NewRiverAdapter(pool, riverInsertClient, jobqueue.MustRegistry(queueDefinitions...))
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("load i18n catalog: %w", err)
	}
	issuer, err := tokenIssuer(cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	iamRepository := iamrepo.NewPostgres(pool)
	var resetNotifier iamapp.PasswordResetNotifier
	if cfg.Environment == "development" && cfg.PasswordRecoveryAdapter == "local" {
		resetNotifier = iamrepo.NewLocalPasswordResetNotifier()
	} else if cfg.PasswordRecoveryAdapter == "notification" {
		resetNotifier, err = notificationadminrepo.NewPasswordResetNotifier(pool, trackedQueue, settingsSealer, cfg.AdminOrigin)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("create password reset notifier: %w", err)
		}
	}
	loginProtectionKey, err := base64.StdEncoding.DecodeString(cfg.LoginProtectionKeyBase64)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("configure login protection key: %w", err)
	}
	authService, err := iamapp.NewAuthService(iamRepository, iamRepository, issuer, iamapp.WithLoginProtectionKey(loginProtectionKey), iamapp.WithAnonymousAuth(
		iamapp.AnonymousAuthConfig{
			AdminRegistrationEnabled: cfg.AdminRegistrationEnabled,
			RegistrationTenantCode:   cfg.AdminRegistrationTenantCode,
			PasswordRecoveryEnabled:  cfg.PasswordRecoveryEnabled,
		},
		resetNotifier,
	))
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	accountDeletionEnabled := true
	featureFlags := map[string]bool{
		"account_deletion":   accountDeletionEnabled,
		"admin_registration": cfg.AdminRegistrationEnabled,
		"password_recovery":  cfg.PasswordRecoveryEnabled,
		"avatar_upload":      cfg.AvatarUploadEnabled,
		"file_storage":       cfg.FileStorageEnabled,
		"multi_tenant":       cfg.MultiTenantEnabled,
		"api_clients":        cfg.APIClientsEnabled,
		"webhooks":           cfg.WebhooksEnabled,
		"push_notifications": cfg.PushEnabled,
		"mfa":                cfg.MFAEnabled,
		"oauth":              cfg.OAuthEnabled,
	}
	authHandler := iamhttp.NewHandler(authService, catalog, cfg.AdminOrigin, cfg.Environment != "development", featureFlags)
	appOTPNotifier, err := notificationadminrepo.NewAppOTPNotifier(trackedQueue, settingsSealer)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create app OTP notifier: %w", err)
	}
	mobileProfileRepository := mobileprofilerepo.NewPostgres(pool)
	storageRepository := storagerepo.NewPostgres(pool)
	var objectStore storagedomain.ObjectStore
	if cfg.AvatarUploadEnabled || cfg.FileStorageEnabled {
		switch cfg.ObjectStorageAdapter {
		case "local":
			objectStore, err = storagerepo.NewLocalObjectStore(cfg.LocalObjectStorageDir)
		case "configured":
			objectStore, err = storagerepo.NewConfiguredObjectStore(pool, settingsSealer, cfg.LocalObjectStorageDir, cfg.Environment)
		}
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("create object storage adapter: %w", err)
		}
	}
	shareConfigRepository := shareconfigrepo.NewPostgres(pool)
	shareConfigService := shareconfigapp.NewService(authService, shareConfigRepository, settingsSealer)
	shareConfigHandler := shareconfighttp.NewHandler(shareConfigService, catalog)
	pushTokenHashKey, err := derivePushTokenHashKey(cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	pushRepository := pushrepo.NewPostgres(pool, trackedQueue)
	var pushPreflighter pushdomain.Preflighter
	if cfg.PushAdapter == "local-mock" {
		pushPreflighter = pushprovider.NewMockSender()
	} else {
		pushPreflighter = pushprovider.NewOfficialSender(pool, settingsSealer, cfg.Environment)
	}
	pushService := pushapp.NewService(authService, pushRepository, settingsSealer, pushTokenHashKey, cfg.Environment, cfg.PushEnabled, pushPreflighter)
	pushHandler := pushhttp.NewHandler(pushService, catalog)
	appManagementService := appmanagementapp.NewService(pool, authService,
		appmanagementapp.WithOTPNotifier(appOTPNotifier),
		appmanagementapp.WithAccountDeletionEnabled(accountDeletionEnabled),
		appmanagementapp.WithObjectErasureQueue(appmanagementrepo.NewObjectErasureQueue(trackedQueue)),
		appmanagementapp.WithObjectStore(objectStore),
		appmanagementapp.WithShareRuntime(func(ctx context.Context, appID uuid.UUID) ([]appmanagementapp.ShareRuntimeProvider, error) {
			providers, loadErr := shareConfigService.Runtime(ctx, appID)
			if loadErr != nil {
				return nil, loadErr
			}
			out := make([]appmanagementapp.ShareRuntimeProvider, 0, len(providers))
			for _, provider := range providers {
				out = append(out, appmanagementapp.ShareRuntimeProvider{ProviderCode: provider.ProviderCode, Enabled: provider.Enabled, Scenes: provider.Scenes, FallbackMode: provider.FallbackMode})
			}
			return out, nil
		}),
		appmanagementapp.WithPushRuntime(func(ctx context.Context, appID uuid.UUID) (appmanagementapp.PushRuntime, error) {
			capability, loadErr := pushService.RuntimeCapability(ctx, appID)
			if loadErr != nil {
				return appmanagementapp.PushRuntime{}, loadErr
			}
			return appmanagementapp.PushRuntime{Enabled: capability.Enabled, Environment: capability.Environment, Providers: capability.Providers, BuildVariants: capability.BuildVariants}, nil
		}),
	)
	appManagementHandler := appmanagementhttp.NewHandler(appManagementService, catalog)
	downloadKeyMaterial, err := base64.StdEncoding.DecodeString(cfg.LoginProtectionKeyBase64)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("configure package download signing: %w", err)
	}
	downloadKeyDeriver := hmac.New(sha256.New, downloadKeyMaterial)
	_, _ = downloadKeyDeriver.Write([]byte("appkernia:mobile-release-download:v1"))
	downloadSigningKey := downloadKeyDeriver.Sum(nil)
	mobileProfileService := mobileprofileapp.NewService(authService, mobileProfileRepository, mobileProfileRepository, mobileprofileapp.WithPackageDownloads(objectStore, downloadSigningKey))
	mobileProfileHandler := mobileprofilehttp.NewHandler(mobileProfileService, catalog)
	storageService := storageapp.NewService(storageRepository, objectStore, cfg.AvatarUploadEnabled)
	storageHandler := storagehttp.NewHandler(authService, storageService, catalog)
	storageAdminRepository := storageadminrepo.NewPostgres(pool)
	storageAdminService := storageadminapp.NewService(authService, storageAdminRepository, objectStore, cfg.FileStorageEnabled)
	storageAdminHandler := storageadminhttp.NewHandler(storageAdminService, catalog)
	dashboardRepository := dashboardrepo.NewPostgres(pool)
	dashboardService := dashboardapp.NewService(authService, dashboardRepository)
	dashboardHandler := dashboardhttp.NewHandler(dashboardService, catalog)
	orgRepository := orgrepo.NewPostgres(pool)
	orgService := orgapp.NewService(authService, orgRepository)
	orgHandler := orghttp.NewHandler(orgService, catalog)
	userAdminRepository := useradminrepo.NewPostgres(pool)
	userAdminService := useradminapp.NewService(authService, userAdminRepository)
	userAdminHandler := useradminhttp.NewHandler(userAdminService, catalog)
	tenantAdminRepository := tenantadminrepo.NewPostgres(pool)
	tenantAdminService := tenantadminapp.NewService(authService, tenantAdminRepository, cfg.MultiTenantEnabled)
	tenantAdminHandler := tenantadminhttp.NewHandler(tenantAdminService, catalog)
	accessAdminRepository := accessadminrepo.NewPostgres(pool)
	accessAdminService := accessadminapp.NewService(authService, accessAdminRepository)
	accessAdminHandler := accessadminhttp.NewHandler(accessAdminService, catalog)
	auditAdminRepository := auditadminrepo.NewPostgres(pool)
	auditAdminService := auditadminapp.NewService(authService, auditAdminRepository)
	auditAdminHandler := auditadminhttp.NewHandler(auditAdminService, catalog)
	sessionAdminRepository := sessionadminrepo.NewPostgres(pool)
	sessionAdminService := sessionadminapp.NewService(authService, sessionAdminRepository)
	sessionAdminHandler := sessionadminhttp.NewHandler(sessionAdminService, catalog)
	settingsRepository := settingsrepo.NewPostgres(pool)
	settingsService := settingsapp.NewService(authService, settingsRepository, settingsSealer)
	settingsHandler := settingshttp.NewHandler(settingsService, catalog)
	identitySecurityRepository := identitysecurityrepo.NewPostgres(pool)
	identitySecurityService := identitysecurityapp.NewService(authService, identitySecurityRepository, settingsSealer, identitysecurityapp.Config{
		MFAEnabled: cfg.MFAEnabled, OAuthEnabled: cfg.OAuthEnabled, OAuthAdapter: cfg.OAuthAdapter, AdminOrigin: cfg.AdminOrigin,
	})
	identitySecurityHandler := identitysecurityhttp.NewHandler(identitySecurityService, catalog)
	notificationAdminRepository := notificationadminrepo.NewPostgres(pool, trackedQueue)
	notificationAdminService := notificationadminapp.NewService(authService, notificationAdminRepository, notificationadminapp.WithDictionaryResolver(settingsRepository), notificationadminapp.WithTargetSealer(settingsSealer))
	notificationAdminHandler := notificationadminhttp.NewHandler(notificationAdminService, catalog)
	contentRepository := contentrepo.NewPostgres(pool, objectStore)
	contentService := contentapp.NewService(authService, contentRepository)
	contentHandler := contenthttp.NewHandler(contentService, catalog)
	jobAdminRepository := jobadminrepo.NewPostgres(pool, riverInsertClient)
	jobAdminService := jobadminapp.NewService(authService, jobAdminRepository)
	jobAdminHandler := jobadminhttp.NewHandler(jobAdminService, catalog)
	apiClientAdminRepository := apiclientadminrepo.NewPostgres(pool)
	apiClientAdminService := apiclientadminapp.NewService(authService, apiClientAdminRepository, issuer)
	apiClientAdminHandler := apiclientadminhttp.NewHandler(apiClientAdminService, catalog)
	machineAuthenticator := apiclientadminapp.NewMachineAuthenticator(apiClientAdminRepository, issuer)
	notificationService := platformnotification.NewPostgresService(pool, trackedQueue)
	notificationAPIHandler := platformnotificationhttp.NewHandler(machineAuthenticator, notificationService, catalog)
	webhookAdminRepository := webhookadminrepo.NewPostgres(pool)
	var webhookAdapter webhookadmindomain.Adapter
	if cfg.WebhookAdapter == "local-mock" {
		webhookAdapter = webhookadminrepo.NewLocalMockAdapter()
	} else {
		webhookAdapter = webhookadminrepo.NewHTTPAdapter()
	}
	webhookAdminService := webhookadminapp.NewService(authService, webhookAdminRepository, settingsSealer, webhookAdapter)
	webhookAdminHandler := webhookadminhttp.NewHandler(webhookAdminService, catalog)
	blockRuleAdminRepository := blockruleadminrepo.NewPostgres(pool)
	blockRuleAdminService := blockruleadminapp.NewService(authService, blockRuleAdminRepository)
	blockRuleAdminHandler := blockruleadminhttp.NewHandler(blockRuleAdminService, catalog)
	opsAdminRepository := opsadminrepo.NewPostgres(pool, opsadminrepo.Config{ObjectStorageConfigured: cfg.AvatarUploadEnabled || cfg.FileStorageEnabled})
	opsAdminService := opsadminapp.NewService(authService, opsAdminRepository)
	opsAdminHandler := opsadminhttp.NewHandler(opsAdminService, catalog)
	health := platformapp.NewHealthService(pool, 2*time.Second)
	handler := platformhttp.NewHandler(health, catalog, featureFlags, settingsRepository, pool)

	server := g.Server("ak-api")
	server.SetAddr(cfg.HTTPAddr)
	server.Use(httpx.RequestContext)
	server.Group("/", func(group *ghttp.RouterGroup) {
		group.GET("/s/{slug}", contentHandler.PublicShare)
		group.GET("/s/assets/{app_id}/{file_id}", contentHandler.PublicShareAsset)
	})
	server.Group("/internal/v1", func(group *ghttp.RouterGroup) {
		group.GET("/health/live", handler.Live)
		group.GET("/health/ready", handler.Ready)
		group.GET("/metrics", handler.Metrics)
	})
	server.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.POST("/auth/client-token", apiClientAdminHandler.Token)
		group.POST("/apps/{app_id}/notifications", notificationAPIHandler.Submit)
		group.GET("/apps/{app_id}/notifications/{message_id}", notificationAPIHandler.Status)
		group.POST("/apps/{app_id}/notifications/{message_id}/cancel", notificationAPIHandler.Cancel)
	})
	server.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(appManagementHandler.RequireMobileApp)
		group.Middleware(appManagementHandler.RequireMobileSessionApp)
		group.GET("/public/config", appManagementHandler.PublicConfig)
		group.GET("/public/startup-assets/{file_id}", appManagementHandler.StartupAsset)
		group.GET("/public/legal/{document_type}", appManagementHandler.Legal)
		group.GET("/public/pages/{slug}", appManagementHandler.Page)
		group.GET("/public/app-version", mobileProfileHandler.AppVersion)
		group.GET("/public/app-version/download/{release_id}/{file_id}", mobileProfileHandler.AppVersionDownload)
		group.GET("/public/dictionaries/{code}", settingsHandler.PublicDictionary)
		group.GET("/public/content/home", contentHandler.PublicHome)
		group.GET("/public/content/items", contentHandler.PublicItems)
		group.GET("/public/content/search", contentHandler.PublicItems)
		group.GET("/public/content/items/{slug}", contentHandler.PublicItem)
		group.GET("/public/content/items/{id}/comments", contentHandler.PublicComments)
		group.GET("/public/content/categories", contentHandler.PublicCategories)
		group.GET("/public/content/topics", contentHandler.PublicTopics)
		group.GET("/public/content/topics/{slug}", contentHandler.PublicTopic)
		group.GET("/public/content/assets/{file_id}", contentHandler.PublicAsset)
		group.GET("/regions", settingsHandler.PublicRegions)
		group.POST("/auth/register", appManagementHandler.Register)
		group.POST("/auth/registration/verify-email", appManagementHandler.VerifyRegistration)
		group.POST("/auth/registration/resend-code", appManagementHandler.ResendRegistration)
		group.POST("/auth/password/forgot", appManagementHandler.ForgotPassword)
		group.POST("/auth/password/reset", appManagementHandler.ResetPassword)
		group.POST("/auth/login/password", authHandler.MobileLogin)
		group.POST("/auth/token/refresh", authHandler.MobileRefresh)
		group.POST("/auth/logout", authHandler.MobileLogout)
		group.GET("/auth/context", authHandler.MobileContext)
		group.POST("/auth/password/change", authHandler.MobileChangeSelfPassword)
		group.POST("/me/legal-consents", appManagementHandler.LegalConsent)
		group.POST("/me/account-deletion/verification-code", appManagementHandler.AccountDeletionVerificationCode)
		group.POST("/me/account-deletion/confirm", appManagementHandler.ConfirmAccountDeletion)
		group.GET("/me", authHandler.MobileMe)
		group.PATCH("/me", authHandler.MobileUpdateMe)
		group.GET("/me/sessions", authHandler.MobileSelfSessions)
		group.DELETE("/me/sessions/{id}", authHandler.MobileRevokeSelfSession)
		group.GET("/me/devices", authHandler.MobileSelfDevices)
		group.DELETE("/me/devices/{id}", authHandler.MobileRemoveSelfDevice)
		group.GET("/me/preferences", mobileProfileHandler.Preferences)
		group.PATCH("/me/preferences", mobileProfileHandler.UpdatePreferences)
		group.GET("/me/notification-preferences", mobileProfileHandler.NotificationPreferences)
		group.PATCH("/me/notification-preferences", mobileProfileHandler.UpdateNotificationPreferences)
		group.GET("/me/notifications/unread-count", mobileProfileHandler.UnreadCount)
		group.GET("/me/notifications", mobileProfileHandler.Notifications)
		group.GET("/me/notifications/{id}", mobileProfileHandler.Notification)
		group.PATCH("/me/notifications/{id}/read", mobileProfileHandler.MarkNotificationRead)
		group.GET("/me/push-devices/current", pushHandler.CurrentDevice)
		group.POST("/me/push-devices", pushHandler.RegisterDevice)
		group.DELETE("/me/push-devices/{push_device_id}", pushHandler.DisableDevice)
		group.POST("/me/push-deliveries/{delivery_id}/opened", pushHandler.MarkOpened)
		group.GET("/me/login-events", mobileProfileHandler.LoginEvents)
		group.GET("/me/security-events", mobileProfileHandler.SecurityEvents)
		group.GET("/article-categories", contentHandler.ArticleCategories)
		group.GET("/article-assets/{file_id}", contentHandler.ArticleAsset)
		group.GET("/articles", contentHandler.Articles)
		group.GET("/articles/{slug}", contentHandler.Article)
		group.PUT("/me/article-bookmarks/{article_id}", contentHandler.Bookmark)
		group.DELETE("/me/article-bookmarks/{article_id}", contentHandler.RemoveBookmark)
		group.GET("/me/content-bookmarks", contentHandler.MyBookmarks)
		group.PUT("/me/content-bookmarks/{article_id}", contentHandler.Bookmark)
		group.DELETE("/me/content-bookmarks/{article_id}", contentHandler.RemoveBookmark)
		group.POST("/content/items/{id}/comments", contentHandler.CreateComment)
		group.DELETE("/me/comments/{id}", contentHandler.DeleteOwnComment)
		group.POST("/comments/{id}/reports", contentHandler.ReportComment)
		group.PUT("/me/blocked-users/{user_id}", contentHandler.BlockUser)
		group.DELETE("/me/blocked-users/{user_id}", contentHandler.UnblockUser)
	})
	server.Group("/admin-api/v1/auth", func(group *ghttp.RouterGroup) {
		group.GET("/public-config", handler.AdminPublicConfig)
		group.POST("/register", authHandler.Register)
		group.POST("/password/forgot", authHandler.ForgotPassword)
		group.POST("/password/reset", authHandler.ResetPassword)
		group.POST("/login", authHandler.Login)
		group.POST("/login/captcha", authHandler.LoginCaptcha)
		group.POST("/switch-tenant", authHandler.SwitchTenant)
		group.POST("/token/refresh", authHandler.Refresh)
		group.POST("/logout", authHandler.Logout)
		group.GET("/context", authHandler.Context)
	})
	server.Group("/admin-api/v1", func(group *ghttp.RouterGroup) {
		group.GET("/share-configs", shareConfigHandler.Configs)
		group.POST("/share-configs", shareConfigHandler.CreateConfig)
		group.GET("/share-configs/{id}", shareConfigHandler.Config)
		group.PATCH("/share-configs/{id}", shareConfigHandler.Config)
		group.DELETE("/share-configs/{id}", shareConfigHandler.Config)
		group.POST("/share-configs/{id}/activate", shareConfigHandler.Activate)
		group.POST("/share-configs/{id}/disable", shareConfigHandler.Disable)
		group.POST("/share-configs/{id}/rotate-secret", shareConfigHandler.RotateSecret)
		group.GET("/apps/{app_id}/share-bindings", shareConfigHandler.Bindings)
		group.PUT("/apps/{app_id}/share-bindings/{provider_code}", shareConfigHandler.Binding)
		group.DELETE("/apps/{app_id}/share-bindings/{provider_code}", shareConfigHandler.Binding)
		group.POST("/apps/{app_id}/share-bindings/{provider_code}/preflight", shareConfigHandler.Preflight)
		group.GET("/apps/{app_id}/push-provider-catalog", pushHandler.Catalog)
		group.GET("/apps/{app_id}/push-provider-configs", pushHandler.Configs)
		group.GET("/apps/{app_id}/push-devices", pushHandler.TestDevices)
		group.GET("/apps/{app_id}/push-delivery-summary", pushHandler.DeliverySummary)
		group.PUT("/apps/{app_id}/push-provider-configs/{provider}", pushHandler.UpsertConfig)
		group.POST("/apps/{app_id}/push-provider-configs/{id}/rotate-secret", pushHandler.RotateSecret)
		group.POST("/apps/{app_id}/push-provider-configs/{id}/preflight", pushHandler.Preflight)
		group.POST("/apps/{app_id}/push-provider-configs/{id}/activate", pushHandler.Activate)
		group.POST("/apps/{app_id}/push-provider-configs/{id}/disable", pushHandler.Disable)
		group.POST("/apps/{app_id}/push-provider-configs/{id}/test", pushHandler.Test)
		group.GET("/apps", appManagementHandler.AdminApps)
		group.POST("/apps", appManagementHandler.AdminApps)
		group.GET("/apps/{app_id}", appManagementHandler.AdminApp)
		group.PATCH("/apps/{app_id}", appManagementHandler.AdminApp)
		group.DELETE("/apps/{app_id}", appManagementHandler.AdminApp)
		group.GET("/apps/{app_id}/scanner-config", appManagementHandler.AdminScannerConfig)
		group.PUT("/apps/{app_id}/scanner-config", appManagementHandler.AdminScannerConfig)
		group.POST("/apps/batch-delete", appManagementHandler.AdminBatchDeleteApps)
		group.POST("/apps/{app_id}/enable", appManagementHandler.AdminEnableApp)
		group.POST("/apps/{app_id}/disable", appManagementHandler.AdminDisableApp)
		group.POST("/apps/{app_id}/startup/onboarding/publish", appManagementHandler.AdminPublishOnboarding)
		group.GET("/apps/{app_id}/mobile/releases", mobileProfileHandler.AdminReleases)
		group.POST("/apps/{app_id}/mobile/releases", mobileProfileHandler.AdminCreateRelease)
		group.POST("/apps/{app_id}/mobile/releases/batch-delete", mobileProfileHandler.AdminBatchDeleteReleases)
		group.GET("/apps/{app_id}/mobile/releases/{id}", mobileProfileHandler.AdminRelease)
		group.PATCH("/apps/{app_id}/mobile/releases/{id}", mobileProfileHandler.AdminUpdateRelease)
		group.DELETE("/apps/{app_id}/mobile/releases/{id}", mobileProfileHandler.AdminRelease)
		group.POST("/apps/{app_id}/mobile/releases/{id}/publish", mobileProfileHandler.AdminPublishRelease)
		group.POST("/apps/{app_id}/mobile/releases/{id}/unpublish", mobileProfileHandler.AdminUnpublishRelease)
		group.GET("/apps/{app_id}/notices", notificationAdminHandler.Notices)
		group.POST("/apps/{app_id}/notices", notificationAdminHandler.CreateNotice)
		group.GET("/apps/{app_id}/notices/{id}", notificationAdminHandler.Notice)
		group.PATCH("/apps/{app_id}/notices/{id}", notificationAdminHandler.UpdateNotice)
		group.POST("/apps/{app_id}/notices/{id}/recipient-preview", notificationAdminHandler.PreviewNotice)
		group.POST("/apps/{app_id}/notices/{id}/publish", notificationAdminHandler.PublishNotice)
		group.POST("/apps/{app_id}/notices/{id}/cancel", notificationAdminHandler.CancelNotice)
		group.GET("/apps/{app_id}/notices/{id}/recipients", notificationAdminHandler.NoticeRecipients)
		group.GET("/apps/{app_id}/messages", notificationAdminHandler.Messages)
		group.POST("/apps/{app_id}/messages", notificationAdminHandler.CreateMessage)
		group.GET("/apps/{app_id}/messages/{id}", notificationAdminHandler.Message)
		group.PATCH("/apps/{app_id}/messages/{id}", notificationAdminHandler.UpdateMessage)
		group.POST("/apps/{app_id}/messages/{id}/recipient-preview", notificationAdminHandler.PreviewMessage)
		group.POST("/apps/{app_id}/messages/{id}/publish", notificationAdminHandler.PublishMessage)
		group.POST("/apps/{app_id}/messages/{id}/cancel", notificationAdminHandler.CancelMessage)
		group.GET("/apps/{app_id}/messages/{id}/recipients", notificationAdminHandler.MessageRecipients)
		group.GET("/apps/{app_id}/notification-operations/summary", notificationAdminHandler.NotificationOperationsSummary)
		group.GET("/apps/{app_id}/notification-operations/trends", notificationAdminHandler.NotificationOperationsTrends)
		group.GET("/apps/{app_id}/notification-runs", notificationAdminHandler.NotificationRuns)
		group.GET("/apps/{app_id}/notification-runs/{run_id}", notificationAdminHandler.NotificationRun)
		group.GET("/apps/{app_id}/notification-tasks", notificationAdminHandler.NotificationTasks)
		group.GET("/apps/{app_id}/notification-tasks/{task_id}", notificationAdminHandler.NotificationTask)
		group.GET("/apps/{app_id}/notification-failures", notificationAdminHandler.NotificationFailures)
		group.POST("/apps/{app_id}/notification-retries", notificationAdminHandler.RetryNotificationTasks)
		group.GET("/apps/{app_id}/content/categories", contentHandler.Categories)
		group.POST("/apps/{app_id}/content/categories", contentHandler.CreateCategory)
		group.GET("/apps/{app_id}/content/categories/{id}", contentHandler.Category)
		group.PATCH("/apps/{app_id}/content/categories/{id}", contentHandler.UpdateCategory)
		group.DELETE("/apps/{app_id}/content/categories/{id}", contentHandler.DeleteCategory)
		group.GET("/apps/{app_id}/content/articles", contentHandler.AdminArticles)
		group.POST("/apps/{app_id}/content/articles", contentHandler.CreateArticle)
		group.GET("/apps/{app_id}/content/articles/{id}", contentHandler.AdminArticle)
		group.PATCH("/apps/{app_id}/content/articles/{id}", contentHandler.UpdateArticle)
		group.DELETE("/apps/{app_id}/content/articles/{id}", contentHandler.DeleteArticle)
		group.POST("/apps/{app_id}/content/articles/{id}/publish", contentHandler.Publish)
		group.POST("/apps/{app_id}/content/articles/{id}/unpublish", contentHandler.Unpublish)
		group.POST("/apps/{app_id}/content/articles/{id}/archive", contentHandler.Archive)
		group.GET("/apps/{app_id}/content/items", contentHandler.AdminArticles)
		group.POST("/apps/{app_id}/content/items", contentHandler.CreateArticle)
		group.GET("/apps/{app_id}/content/items/{id}", contentHandler.AdminArticle)
		group.PATCH("/apps/{app_id}/content/items/{id}", contentHandler.UpdateArticle)
		group.DELETE("/apps/{app_id}/content/items/{id}", contentHandler.DeleteArticle)
		group.POST("/apps/{app_id}/content/items/{id}/publish", contentHandler.Publish)
		group.POST("/apps/{app_id}/content/items/{id}/unpublish", contentHandler.Unpublish)
		group.POST("/apps/{app_id}/content/items/{id}/archive", contentHandler.Archive)
		group.GET("/apps/{app_id}/content/topics", contentHandler.Topics)
		group.POST("/apps/{app_id}/content/topics", contentHandler.CreateTopic)
		group.PATCH("/apps/{app_id}/content/topics/{id}", contentHandler.UpdateTopic)
		group.DELETE("/apps/{app_id}/content/topics/{id}", contentHandler.DeleteTopic)
		group.GET("/apps/{app_id}/content/tags", contentHandler.Tags)
		group.POST("/apps/{app_id}/content/tags", contentHandler.CreateTag)
		group.PATCH("/apps/{app_id}/content/tags/{id}", contentHandler.UpdateTag)
		group.POST("/apps/{app_id}/content/tags/{id}/merge", contentHandler.MergeTag)
		group.DELETE("/apps/{app_id}/content/tags/{id}", contentHandler.DeleteTag)
		group.GET("/apps/{app_id}/content/comments", contentHandler.AdminComments)
		group.POST("/apps/{app_id}/content/comments/batch-moderate", contentHandler.BatchModerateComments)
		group.POST("/apps/{app_id}/content/comments/{id}/moderate", contentHandler.ModerateComment)
		group.GET("/apps/{app_id}/content/comment-reports", contentHandler.CommentReports)
		group.POST("/apps/{app_id}/content/comment-reports/{id}/resolve", contentHandler.ResolveCommentReport)
		group.GET("/apps/{app_id}/content/pages", appManagementHandler.AdminPages)
		group.POST("/apps/{app_id}/content/pages", appManagementHandler.AdminPages)
		group.PATCH("/apps/{app_id}/content/pages/{slug}", appManagementHandler.AdminPage)
		group.POST("/apps/{app_id}/content/pages/{slug}/publish", appManagementHandler.AdminPublishPage)
		group.DELETE("/apps/{app_id}/content/pages/{slug}", appManagementHandler.AdminPage)
		group.GET("/apps/{app_id}/users", appManagementHandler.AdminUsers)
		group.GET("/apps/{app_id}/users/{user_id}", appManagementHandler.AdminUser)
		group.POST("/apps/{app_id}/users", appManagementHandler.AdminCreateUser)
		group.PATCH("/apps/{app_id}/users/{user_id}", appManagementHandler.AdminUpdateUser)
		group.POST("/apps/{app_id}/users/{user_id}/enable", appManagementHandler.AdminEnableUser)
		group.POST("/apps/{app_id}/users/{user_id}/disable", appManagementHandler.AdminDisableUser)
		group.POST("/apps/{app_id}/users/{user_id}/unlock", appManagementHandler.AdminUnlockUser)
		group.POST("/apps/{app_id}/users/{user_id}/reset-password", appManagementHandler.AdminResetUserPassword)
		group.POST("/apps/{app_id}/users/{user_id}/sessions/revoke", appManagementHandler.AdminRevokeUserSessions)
		group.GET("/mobile/releases", mobileProfileHandler.AdminReleases)
		group.POST("/mobile/releases", mobileProfileHandler.AdminCreateRelease)
		group.POST("/mobile/releases/batch-delete", mobileProfileHandler.AdminBatchDeleteReleases)
		group.GET("/mobile/releases/{id}", mobileProfileHandler.AdminRelease)
		group.PATCH("/mobile/releases/{id}", mobileProfileHandler.AdminUpdateRelease)
		group.DELETE("/mobile/releases/{id}", mobileProfileHandler.AdminRelease)
		group.POST("/mobile/releases/{id}/publish", mobileProfileHandler.AdminPublishRelease)
		group.POST("/mobile/releases/{id}/unpublish", mobileProfileHandler.AdminUnpublishRelease)
		group.GET("/me", authHandler.Me)
		group.PATCH("/me", authHandler.UpdateMe)
		group.GET("/me/sessions", authHandler.SelfSessions)
		group.DELETE("/me/sessions/{id}", authHandler.RevokeSelfSession)
		group.GET("/me/devices", authHandler.SelfDevices)
		group.DELETE("/me/devices/{id}", authHandler.RemoveSelfDevice)
		group.POST("/me/password/change", authHandler.ChangeSelfPassword)
		group.GET("/me/mfa", identitySecurityHandler.MFAStatus)
		group.POST("/me/mfa/totp/enroll", identitySecurityHandler.EnrollTOTP)
		group.POST("/me/mfa/totp/verify", identitySecurityHandler.VerifyTOTP)
		group.DELETE("/me/mfa/totp", identitySecurityHandler.DisableTOTP)
		group.POST("/me/mfa/recovery-codes/rotate", identitySecurityHandler.RotateRecoveryCodes)
		group.GET("/me/oauth-accounts", identitySecurityHandler.OAuthAccounts)
		group.POST("/me/oauth/{provider}/start", identitySecurityHandler.StartOAuth)
		group.POST("/me/oauth/{provider}/callback", identitySecurityHandler.CompleteOAuth)
		group.DELETE("/me/oauth/{provider}", identitySecurityHandler.DeleteOAuth)
		group.POST("/me/avatar/upload-session", storageHandler.CreateAvatarUpload)
		group.PUT("/me/avatar/upload-sessions/{id}/content", storageHandler.UploadAvatarContent)
		group.GET("/me/avatar/content", storageHandler.AvatarContent)
		group.POST("/files/upload-sessions", storageAdminHandler.CreateUpload)
		group.GET("/files/upload-policy", storageAdminHandler.UploadPolicy)
		group.GET("/files/upload-sessions/{id}", storageAdminHandler.GetUpload)
		group.PUT("/files/upload-sessions/{id}/parts/{partNumber}", storageAdminHandler.UploadPart)
		group.DELETE("/files/upload-sessions/{id}", storageAdminHandler.CancelUpload)
		group.POST("/files/upload-sessions/{id}/complete", storageAdminHandler.CompleteUpload)
		group.GET("/files", storageAdminHandler.List)
		group.GET("/files/{id}", storageAdminHandler.Get)
		group.POST("/files/{id}/presign-download", storageAdminHandler.PresignDownload)
		group.GET("/files/{id}/content", storageAdminHandler.Download)
		group.GET("/files/{id}/usages", storageAdminHandler.Usages)
		group.DELETE("/files/{id}", storageAdminHandler.Delete)
		group.GET("/dashboard/summary", dashboardHandler.Summary)
		group.GET("/dashboard/trends", dashboardHandler.Trends)
		group.GET("/dashboard/activity", dashboardHandler.Activity)
		group.GET("/org/units/tree", orgHandler.UnitTree)
		group.POST("/org/units", orgHandler.CreateUnit)
		group.PATCH("/org/units/{id}", orgHandler.UpdateUnit)
		group.POST("/org/units/{id}/move", orgHandler.MoveUnit)
		group.DELETE("/org/units/{id}", orgHandler.DeleteUnit)
		group.GET("/org/positions", orgHandler.Positions)
		group.POST("/org/positions", orgHandler.CreatePosition)
		group.PATCH("/org/positions/{id}", orgHandler.UpdatePosition)
		group.DELETE("/org/positions/{id}", orgHandler.DeletePosition)
		group.GET("/users/role-options", userAdminHandler.RoleOptions)
		group.GET("/tenants", tenantAdminHandler.List)
		group.POST("/tenants", tenantAdminHandler.Create)
		group.GET("/tenants/{id}", tenantAdminHandler.Get)
		group.PATCH("/tenants/{id}", tenantAdminHandler.Update)
		group.GET("/tenants/{id}/members", tenantAdminHandler.Members)
		group.POST("/tenants/{id}/members", tenantAdminHandler.AddMember)
		group.PATCH("/tenants/{id}/members/{user_id}", tenantAdminHandler.SetMember)
		group.DELETE("/tenants/{id}/members/{user_id}", tenantAdminHandler.RemoveMember)
		group.GET("/users", userAdminHandler.Users)
		group.POST("/users", userAdminHandler.Create)
		group.GET("/users/{id}", userAdminHandler.User)
		group.PATCH("/users/{id}", userAdminHandler.Update)
		group.POST("/users/{id}/enable", userAdminHandler.Enable)
		group.POST("/users/{id}/disable", userAdminHandler.Disable)
		group.POST("/users/{id}/unlock", userAdminHandler.Unlock)
		group.POST("/users/{id}/reset-password", userAdminHandler.ResetPassword)
		group.PUT("/users/{id}/roles", userAdminHandler.ReplaceRoles)
		group.PUT("/org/users/{user_id}/assignments", userAdminHandler.ReplaceAssignments)
		group.GET("/users/{id}/sessions", userAdminHandler.Sessions)
		group.DELETE("/users/{id}/sessions/{session_id}", userAdminHandler.RevokeSession)
		group.POST("/users/import", userAdminHandler.Import)
		group.POST("/users/export", userAdminHandler.Export)
		group.GET("/roles", accessAdminHandler.Roles)
		group.POST("/roles", accessAdminHandler.CreateRole)
		group.PATCH("/roles/{id}", accessAdminHandler.UpdateRole)
		group.DELETE("/roles/{id}", accessAdminHandler.DeleteRole)
		group.PUT("/roles/{id}/permissions", accessAdminHandler.ReplacePermissions)
		group.PUT("/roles/{id}/menus", accessAdminHandler.ReplaceMenus)
		group.PUT("/roles/{id}/data-scope", accessAdminHandler.ReplaceDataScope)
		group.GET("/permissions", accessAdminHandler.Permissions)
		group.GET("/menus/tree", accessAdminHandler.Menus)
		group.POST("/menus", accessAdminHandler.CreateMenu)
		group.PATCH("/menus/{id}", accessAdminHandler.UpdateMenu)
		group.POST("/menus/{id}/move", accessAdminHandler.MoveMenu)
		group.DELETE("/menus/{id}", accessAdminHandler.DeleteMenu)
		group.GET("/audit/operations", auditAdminHandler.Operations)
		group.GET("/audit/logins", auditAdminHandler.Logins)
		group.GET("/audit/security-events", auditAdminHandler.SecurityEvents)
		group.GET("/audit/security-events/{id}", auditAdminHandler.SecurityEvent)
		group.POST("/audit/security-events/{id}/resolve", auditAdminHandler.ResolveSecurityEvent)
		group.GET("/online-sessions", sessionAdminHandler.List)
		group.DELETE("/online-sessions/{id}", sessionAdminHandler.Revoke)
		group.GET("/configs", settingsHandler.Configs)
		group.POST("/configs", settingsHandler.CreateConfig)
		group.PATCH("/configs/{id}", settingsHandler.UpdateConfig)
		group.POST("/configs/{id}/rotate-secret", settingsHandler.RotateSecret)
		group.GET("/dictionaries/{code}", settingsHandler.AdminDictionary)
		group.GET("/dict-types", settingsHandler.DictTypes)
		group.POST("/dict-types", settingsHandler.CreateDictType)
		group.PATCH("/dict-types/{id}", settingsHandler.UpdateDictType)
		group.GET("/dict-types/{id}/items", settingsHandler.DictItems)
		group.POST("/dict-types/{id}/items", settingsHandler.CreateDictItem)
		group.PATCH("/dict-items/{id}", settingsHandler.UpdateDictItem)
		group.DELETE("/dict-items/{id}", settingsHandler.DeleteDictItem)
		group.GET("/regions", settingsHandler.Regions)
		group.POST("/regions", settingsHandler.CreateRegion)
		group.PATCH("/regions/{code}", settingsHandler.UpdateRegion)
		group.DELETE("/regions/{code}", settingsHandler.DeleteRegion)
		group.GET("/notices", notificationAdminHandler.Notices)
		group.POST("/notices", notificationAdminHandler.CreateNotice)
		group.GET("/notices/{id}", notificationAdminHandler.Notice)
		group.PATCH("/notices/{id}", notificationAdminHandler.UpdateNotice)
		group.POST("/notices/{id}/recipient-preview", notificationAdminHandler.PreviewNotice)
		group.POST("/notices/{id}/publish", notificationAdminHandler.PublishNotice)
		group.POST("/notices/{id}/cancel", notificationAdminHandler.CancelNotice)
		group.GET("/content/categories", contentHandler.Categories)
		group.POST("/content/categories", contentHandler.CreateCategory)
		group.GET("/content/categories/{id}", contentHandler.Category)
		group.PATCH("/content/categories/{id}", contentHandler.UpdateCategory)
		group.DELETE("/content/categories/{id}", contentHandler.DeleteCategory)
		group.GET("/content/articles", contentHandler.AdminArticles)
		group.POST("/content/articles", contentHandler.CreateArticle)
		group.GET("/content/articles/{id}", contentHandler.AdminArticle)
		group.PATCH("/content/articles/{id}", contentHandler.UpdateArticle)
		group.DELETE("/content/articles/{id}", contentHandler.DeleteArticle)
		group.POST("/content/articles/{id}/publish", contentHandler.Publish)
		group.POST("/content/articles/{id}/unpublish", contentHandler.Unpublish)
		group.POST("/content/articles/{id}/archive", contentHandler.Archive)
		group.GET("/notices/{id}/recipients", notificationAdminHandler.NoticeRecipients)
		group.GET("/messages", notificationAdminHandler.Messages)
		group.POST("/messages", notificationAdminHandler.CreateMessage)
		group.GET("/messages/{id}", notificationAdminHandler.Message)
		group.PATCH("/messages/{id}", notificationAdminHandler.UpdateMessage)
		group.POST("/messages/{id}/recipient-preview", notificationAdminHandler.PreviewMessage)
		group.POST("/messages/{id}/publish", notificationAdminHandler.PublishMessage)
		group.POST("/messages/{id}/cancel", notificationAdminHandler.CancelMessage)
		group.GET("/messages/{id}/recipients", notificationAdminHandler.MessageRecipients)
		group.GET("/notification-templates", notificationAdminHandler.Templates)
		group.POST("/notification-templates", notificationAdminHandler.CreateTemplate)
		group.PATCH("/notification-templates/{id}", notificationAdminHandler.UpdateTemplate)
		group.GET("/notification-templates/{id}/sms-bindings", notificationAdminHandler.SMSTemplateBindings)
		group.PUT("/notification-templates/{id}/sms-bindings/{provider}", notificationAdminHandler.UpsertSMSTemplateBinding)
		group.DELETE("/notification-templates/{id}/sms-bindings/{provider}", notificationAdminHandler.DeleteSMSTemplateBinding)
		group.POST("/notification-templates/{id}/test", notificationAdminHandler.TestTemplate)
		group.GET("/notification-deliveries", notificationAdminHandler.Deliveries)
		group.GET("/notification-deliveries/{id}", notificationAdminHandler.Delivery)
		group.POST("/notification-deliveries/{id}/retry", notificationAdminHandler.RetryDelivery)
		group.GET("/job-handlers", jobAdminHandler.Handlers)
		group.POST("/job-schedules/preview", jobAdminHandler.Preview)
		group.GET("/job-schedules", jobAdminHandler.List)
		group.POST("/job-schedules", jobAdminHandler.Create)
		group.PATCH("/job-schedules/{id}", jobAdminHandler.Update)
		group.POST("/job-schedules/{id}/pause", jobAdminHandler.Pause)
		group.POST("/job-schedules/{id}/resume", jobAdminHandler.Resume)
		group.POST("/job-schedules/{id}/execute", jobAdminHandler.Execute)
		group.GET("/job-schedules/{id}/runs", jobAdminHandler.Runs)
		group.GET("/api-clients", apiClientAdminHandler.List)
		group.POST("/api-clients", apiClientAdminHandler.Create)
		group.GET("/api-clients/{id}", apiClientAdminHandler.Get)
		group.PATCH("/api-clients/{id}", apiClientAdminHandler.Update)
		group.POST("/api-clients/{id}/secrets", apiClientAdminHandler.CreateSecret)
		group.DELETE("/api-clients/{id}/secrets/{secret_id}", apiClientAdminHandler.RevokeSecret)
		group.PUT("/api-clients/{id}/permissions", apiClientAdminHandler.Permissions)
		group.PUT("/api-clients/{id}/apps", apiClientAdminHandler.Applications)
		group.GET("/webhooks", webhookAdminHandler.List)
		group.POST("/webhooks", webhookAdminHandler.Create)
		group.PATCH("/webhooks/{id}", webhookAdminHandler.Update)
		group.POST("/webhooks/{id}/test", webhookAdminHandler.Test)
		group.GET("/webhooks/{id}/deliveries", webhookAdminHandler.Deliveries)
		group.GET("/block-rules", blockRuleAdminHandler.List)
		group.POST("/block-rules", blockRuleAdminHandler.Create)
		group.PATCH("/block-rules/{id}", blockRuleAdminHandler.Update)
		group.DELETE("/block-rules/{id}", blockRuleAdminHandler.Revoke)
		group.GET("/ops/health", opsAdminHandler.Health)
		group.GET("/ops/runtime-summary", opsAdminHandler.Runtime)
	})
	return &API{server: server, pool: pool}, nil
}

func configSecretSealer(cfg config.Config) (*settingsrepo.AESGCMSealer, error) {
	var key []byte
	var err error
	if cfg.ConfigMasterKeyBase64 != "" {
		key, err = base64.StdEncoding.DecodeString(cfg.ConfigMasterKeyBase64)
	}
	if err != nil {
		return nil, fmt.Errorf("configure config secret sealer: %w", err)
	}
	sealer, err := settingsrepo.NewAESGCMSealer(key, cfg.ConfigMasterKeyVersion)
	if err != nil {
		return nil, fmt.Errorf("configure config secret sealer: %w", err)
	}
	return sealer, nil
}

func derivePushTokenHashKey(cfg config.Config) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(cfg.ConfigMasterKeyBase64)
	if err != nil || len(key) != 32 {
		return nil, errors.New("configure push token hash key: AK_CONFIG_MASTER_KEY_BASE64 must encode exactly 32 bytes")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("appkernia:push-token-hash:v1"))
	return mac.Sum(nil), nil
}

func tokenIssuer(cfg config.Config) (*iamapp.TokenIssuer, error) {
	if cfg.JWTPrivateKey == "" {
		return iamapp.NewDevelopmentTokenIssuer()
	}
	issuer, err := iamapp.NewTokenIssuerFromBase64("appkernia", cfg.JWTKeyID, cfg.JWTPrivateKey, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("configure access token issuer: %w", err)
	}
	return issuer, nil
}

func (a *API) Start() error {
	return a.server.Start()
}

func (a *API) Shutdown() error {
	defer a.pool.Close()
	return a.server.Shutdown()
}
