package bootstrap

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	dashboardapp "github.com/appkernia/appkernia/server/internal/modules/dashboard/application"
	dashboardrepo "github.com/appkernia/appkernia/server/internal/modules/dashboard/repository"
	dashboardhttp "github.com/appkernia/appkernia/server/internal/modules/dashboard/transport/http"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	iamhttp "github.com/appkernia/appkernia/server/internal/modules/iam/transport/http"
	platformapp "github.com/appkernia/appkernia/server/internal/modules/platform/application"
	platformhttp "github.com/appkernia/appkernia/server/internal/modules/platform/transport/http"
	"github.com/appkernia/appkernia/server/internal/platform/adminui"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	sqliteplatform "github.com/appkernia/appkernia/server/internal/platform/sqlite"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type sqliteProbe struct{ database *sql.DB }

func (probe sqliteProbe) Ping(ctx context.Context) error { return probe.database.PingContext(ctx) }

type emptyPublicConfigStore struct{}

func (emptyPublicConfigStore) ListPublicConfigs(context.Context) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}

func newSQLiteAPI(ctx context.Context, cfg config.Config) (*API, error) {
	database, err := sqliteplatform.Open(ctx, cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	closeOnError := func(err error) (*API, error) {
		_ = database.Close()
		return nil, err
	}
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		return closeOnError(fmt.Errorf("load i18n catalog: %w", err))
	}
	issuer, err := tokenIssuer(cfg)
	if err != nil {
		return closeOnError(err)
	}
	loginProtectionKey, err := base64.StdEncoding.DecodeString(cfg.LoginProtectionKeyBase64)
	if err != nil {
		return closeOnError(fmt.Errorf("configure login protection key: %w", err))
	}
	repository := iamrepo.NewSQLite(database)
	passwordRecoveryEnabled := cfg.PasswordRecoveryEnabled && cfg.Environment == "development" && cfg.PasswordRecoveryAdapter == "local"
	var resetNotifier iamapp.PasswordResetNotifier
	if passwordRecoveryEnabled {
		resetNotifier = iamrepo.NewLocalPasswordResetNotifier()
	}
	authService, err := iamapp.NewAuthService(
		repository,
		repository,
		issuer,
		iamapp.WithLoginProtectionKey(loginProtectionKey),
		iamapp.WithLoginCaptchaTypeProvider(func(context.Context) (platformcaptcha.Type, error) {
			return platformcaptcha.TypeSlide, nil
		}),
		iamapp.WithAnonymousAuth(iamapp.AnonymousAuthConfig{
			AdminRegistrationEnabled: cfg.AdminRegistrationEnabled,
			RegistrationTenantCode:   cfg.AdminRegistrationTenantCode,
			PasswordRecoveryEnabled:  passwordRecoveryEnabled,
		}, resetNotifier),
	)
	if err != nil {
		return closeOnError(fmt.Errorf("create SQLite auth service: %w", err))
	}
	featureFlags := map[string]bool{
		"account_deletion": false, "admin_registration": cfg.AdminRegistrationEnabled,
		"password_recovery": passwordRecoveryEnabled, "avatar_upload": false,
		"file_storage": false, "multi_tenant": cfg.MultiTenantEnabled,
		"api_clients": false, "webhooks": false, "push_notifications": false,
		"mfa": false, "oauth": false,
	}
	authHandler := iamhttp.NewHandler(authService, catalog, cfg.AdminOrigin, cfg.Environment != "development", featureFlags)
	dashboardHandler := dashboardhttp.NewHandler(
		dashboardapp.NewService(authService, dashboardrepo.NewSQLite(database)),
		catalog,
	)
	platformHandler := platformhttp.NewHandler(
		platformapp.NewHealthService(sqliteProbe{database: database}, 2*time.Second),
		catalog,
		featureFlags,
		emptyPublicConfigStore{},
	)
	adminUI, err := adminui.New(cfg.AdminPath, cfg.AdminStaticDir, cfg.PublicWebBaseURL)
	if err != nil {
		return closeOnError(fmt.Errorf("configure Admin UI: %w", err))
	}
	server := g.Server("ak-api")
	server.SetAddr(cfg.HTTPAddr)
	server.Use(httpx.RequestContext)
	server.Group("/internal/v1", func(group *ghttp.RouterGroup) {
		group.GET("/health/live", platformHandler.Live)
		group.GET("/health/ready", platformHandler.Ready)
		group.GET("/metrics", platformHandler.Metrics)
	})
	server.Group("/admin-api/v1/auth", func(group *ghttp.RouterGroup) {
		group.GET("/public-config", platformHandler.AdminPublicConfig)
		group.POST("/register", authHandler.Register)
		group.POST("/password/forgot", authHandler.ForgotPassword)
		group.POST("/password/reset", authHandler.ResetPassword)
		group.POST("/login", authHandler.Login)
		group.POST("/login/captcha", authHandler.LoginCaptcha)
		group.GET("/csrf-token", authHandler.CSRFToken)
		group.POST("/switch-tenant", authHandler.SwitchTenant)
		group.POST("/token/refresh", authHandler.Refresh)
		group.POST("/logout", authHandler.Logout)
		group.GET("/context", authHandler.Context)
	})
	server.Group("/admin-api/v1", func(group *ghttp.RouterGroup) {
		group.GET("/me", authHandler.Me)
		group.PATCH("/me", authHandler.UpdateMe)
		group.GET("/me/sessions", authHandler.SelfSessions)
		group.DELETE("/me/sessions/{id}", authHandler.RevokeSelfSession)
		group.GET("/me/devices", authHandler.SelfDevices)
		group.DELETE("/me/devices/{id}", authHandler.RemoveSelfDevice)
		group.POST("/me/password/change", authHandler.ChangeSelfPassword)
		group.GET("/dashboard/summary", dashboardHandler.Summary)
		group.GET("/dashboard/trends", dashboardHandler.Trends)
		group.GET("/dashboard/activity", dashboardHandler.Activity)
	})
	if adminUI != nil {
		adminUI.Register(server)
	}
	return &API{server: server, adminUI: adminUI, closeDatabase: database.Close}, nil
}
