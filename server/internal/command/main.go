package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/migration"
	"github.com/appkernia/appkernia/server/internal/platform/runtimeassets"
	sqliteplatform "github.com/appkernia/appkernia/server/internal/platform/sqlite"
	"github.com/appkernia/appkernia/server/internal/seed"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"golang.org/x/term"
)

type UsageError struct{ Message string }

func (err *UsageError) Error() string { return err.Message }

var errHelpRequested = errors.New("help requested")

func parseCommandFlags(flags *flag.FlagSet, args []string, usage string) error {
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return errHelpRequested
		}
		return &UsageError{Message: usage}
	}
	if flags.NArg() != 0 {
		return &UsageError{Message: usage}
	}
	return nil
}

func Run(program string, args []string) error {
	if len(args) < 1 {
		Usage(os.Stderr, program)
		return &UsageError{Message: "command is required"}
	}
	var err error
	switch args[0] {
	case "version":
		err = versionCommand(program, args[1:])
	case "doctor":
		err = doctor(program, args[1:])
	case "config":
		err = configCommand(program, args[1:])
	case "auth":
		err = authCommand(program, args[1:])
	case "api":
		err = apiCommand(program, args[1:])
	case "migrate":
		err = migrateDatabase(program, args[1:])
	case "seed":
		err = seedCorePermissions(program, args[1:])
	case "bootstrap-admin":
		err = bootstrapAdmin(program, args[1:])
	case "app-startup":
		err = appStartupCommand(program, args[1:])
	case "app-share":
		err = appShareCommand(program, args[1:])
	case "app-login-provider":
		err = appLoginProviderCommand(program, args[1:])
	default:
		Usage(os.Stderr, program)
		return &UsageError{Message: fmt.Sprintf("unknown command %q", args[0])}
	}
	if errors.Is(err, errHelpRequested) {
		return nil
	}
	return err
}

func Usage(writer io.Writer, program string) {
	fmt.Fprintf(writer, "usage: %s version [--json] | doctor [--json] | config init|validate|show | auth configure | api list|describe|call | migrate up|down [steps] | seed core | bootstrap-admin [flags] | app-startup export --app-id UUID --output DIR [--check] | app-share export --app-id UUID --output DIR [--check] | app-login-provider export --app-id UUID --output DIR [--check]\n", program)
}

func requirePostgreSQL(cfg config.Config, operation string) error {
	if cfg.DatabaseDriver != config.DatabaseDriverPostgreSQL {
		return fmt.Errorf("%s requires database.driver=postgresql in this release", operation)
	}
	return nil
}

func versionCommand(program string, args []string) error {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := parseCommandFlags(flags, args, fmt.Sprintf("usage: %s version [--json]", program)); err != nil {
		return err
	}
	if *jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"version": buildinfo.Version, "commit": buildinfo.Commit, "build_time": buildinfo.BuildTime})
	}
	fmt.Printf("%s version=%s commit=%s build_time=%s\n", program, buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime)
	return nil
}

func doctor(program string, args []string) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := parseCommandFlags(flags, args, fmt.Sprintf("usage: %s doctor [--json]", program)); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if cfg.DatabaseDriver == config.DatabaseDriverSQLite {
		database, openErr := sqliteplatform.Open(ctx, cfg.SQLitePath)
		if openErr != nil {
			return openErr
		}
		defer database.Close()
		if err = database.PingContext(ctx); err != nil {
			return fmt.Errorf("database readiness failed: %w", err)
		}
		if *jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "ready", "database": "sqlite", "path": cfg.SQLitePath, "version": buildinfo.Version})
		}
		fmt.Printf("%s doctor: SQLite ready path=%s\n", program, cfg.SQLitePath)
		return nil
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		return fmt.Errorf("database readiness failed: %w", err)
	}
	if *jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "ready", "database": "postgresql", "version": buildinfo.Version})
	}
	fmt.Printf("%s doctor: PostgreSQL ready\n", program)
	return nil
}

func migrateDatabase(program string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s migrate up|down [steps]\n", program)
		return nil
	}
	if len(args) < 1 || len(args) > 2 || (args[0] != "up" && args[0] != "down") {
		return errorsForUsage(program, "migration direction must be up or down")
	}
	var steps int
	if len(args) == 2 {
		parsed, err := strconv.Atoi(args[1])
		if err != nil || parsed <= 0 {
			return errorsForUsage(program, "migration steps must be a positive integer")
		}
		steps = parsed
	}
	if args[0] == "down" && steps == 0 {
		steps = -1
	} else if args[0] == "down" {
		steps = -steps
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseDriver == config.DatabaseDriverSQLite {
		if args[0] == "down" {
			return errors.New("SQLite migrations are forward-only; restore a backup to roll back")
		}
		database, openErr := sqliteplatform.Open(context.Background(), cfg.SQLitePath)
		if openErr != nil {
			return openErr
		}
		if closeErr := database.Close(); closeErr != nil {
			return closeErr
		}
		fmt.Println("migration version=1 dirty=false")
		return nil
	}
	migrationPath := os.Getenv("AK_MIGRATION_PATH")
	var cleanup func()
	if migrationPath == "" {
		root, release, materializeErr := runtimeassets.Materialize()
		if materializeErr != nil {
			return materializeErr
		}
		cleanup = release
		migrationPath = filepath.Join(root, "blueprint", "backend", "db", "migrations")
	}
	if cleanup != nil {
		defer cleanup()
	}
	runner, err := migration.NewRunner(cfg.DatabaseURL, migrationPath)
	if err != nil {
		return err
	}
	if args[0] == "up" && steps == 0 {
		if err = runner.Up(); err != nil {
			return err
		}
		if err = migrateRiverUp(cfg.DatabaseURL); err != nil {
			return err
		}
	} else if err = runner.Steps(steps); err != nil {
		return err
	}
	version, dirty, err := runner.Version()
	if err != nil {
		return err
	}
	fmt.Printf("migration version=%d dirty=%t\n", version, dirty)
	return nil
}

func migrateRiverUp(databaseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("create River migration pool: %w", err)
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create River migrator: %w", err)
	}
	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("apply River official migrations: %w", err)
	}
	fmt.Printf("river migrations applied=%d\n", len(result.Versions))
	return nil
}

func errorsForUsage(program, message string) error {
	return &UsageError{Message: fmt.Sprintf("%s; usage: %s migrate up|down [steps]", message, program)}
}

func seedCorePermissions(program string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s seed core\n", program)
		return nil
	}
	if len(args) != 1 || (args[0] != "core" && args[0] != "core-permissions") {
		return &UsageError{Message: fmt.Sprintf("usage: %s seed core", program)}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseDriver == config.DatabaseDriverSQLite {
		database, openErr := sqliteplatform.Open(context.Background(), cfg.SQLitePath)
		if openErr != nil {
			return openErr
		}
		if closeErr := database.Close(); closeErr != nil {
			return closeErr
		}
		fmt.Println("SQLite schema is current; bootstrap-admin seeds local permissions")
		return nil
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	assetRoot, cleanup, err := runtimeassets.Materialize()
	if err != nil {
		return err
	}
	defer cleanup()
	assetPath := func(parts ...string) string { return filepath.Join(append([]string{assetRoot}, parts...)...) }
	permissionPaths := []string{
		assetPath("blueprint", "backend", "spec", "core-permissions.json"),
		assetPath("blueprint", "admin-frontend", "integration", "core-permissions.delta.json"),
		assetPath("blueprint", "mobile", "integration", "app-permissions.delta.json"),
	}
	permissionCount := 0
	for _, catalogPath := range permissionPaths {
		count, seedErr := seed.CorePermissions(context.Background(), pool, catalogPath)
		if seedErr != nil {
			return seedErr
		}
		permissionCount += count
	}
	menuCount, err := seed.CoreMenus(context.Background(), pool, assetPath("blueprint", "admin-frontend", "spec", "admin-menu-seed.json"))
	if err != nil {
		return err
	}
	moduleCount, err := seed.CoreModules(context.Background(), pool, assetPath("blueprint", "backend", "spec", "core-modules.json"))
	if err != nil {
		return err
	}
	configCatalogPath := assetPath("blueprint", "backend", "spec", "core-configs.json")
	configCount, err := seed.CoreConfigs(context.Background(), pool, configCatalogPath)
	if err != nil {
		return err
	}
	regionCount, err := seed.CoreRegions(context.Background(), pool, assetPath("blueprint", "backend", "spec", "core-regions.json"))
	if err != nil {
		return err
	}
	dictionaryCount, err := seed.CoreDictionaries(context.Background(), pool, assetPath("blueprint", "backend", "spec", "core-dictionaries.json"))
	if err != nil {
		return err
	}
	adminSeeded, err := seedDevelopmentAdmin(context.Background(), pool, cfg.Environment, configCatalogPath)
	if err != nil {
		return err
	}
	fmt.Printf("seeded core permissions=%d menus=%d modules=%d tenant_configs=%d regions=%d dictionaries=%d development_admin=%t build_version=%s\n", permissionCount, menuCount, moduleCount, configCount, regionCount, dictionaryCount, adminSeeded, buildinfo.Version)
	return nil
}

func seedDevelopmentAdmin(ctx context.Context, pool *pgxpool.Pool, environment, configCatalogPath string) (bool, error) {
	passwordFile := strings.TrimSpace(os.Getenv("AK_SEED_ADMIN_PASSWORD_FILE"))
	if passwordFile == "" {
		return false, nil
	}
	if environment != "development" {
		return false, errorsForSeedAdmin("AK_SEED_ADMIN_PASSWORD_FILE is allowed only in development")
	}
	passwordBytes, err := os.ReadFile(passwordFile)
	if err != nil {
		return false, fmt.Errorf("read seed administrator password file: %w", err)
	}
	defer func() {
		for index := range passwordBytes {
			passwordBytes[index] = 0
		}
	}()
	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return false, errorsForSeedAdmin("seed administrator password file is empty")
	}
	valueOrDefault := func(name, fallback string) string {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
		return fallback
	}
	_, _, _, _, err = seed.BootstrapAdmin(ctx, pool, seed.BootstrapAdminInput{
		TenantCode:        valueOrDefault("AK_BOOTSTRAP_TENANT_CODE", "local"),
		TenantName:        valueOrDefault("AK_BOOTSTRAP_TENANT_NAME", "Local Workspace"),
		Email:             valueOrDefault("AK_SEED_ADMIN_EMAIL", "admin@appkernia.local"),
		DisplayName:       valueOrDefault("AK_BOOTSTRAP_DISPLAY_NAME", "Local Admin"),
		Locale:            valueOrDefault("AK_BOOTSTRAP_LOCALE", "zh-CN"),
		Password:          password,
		ConfigCatalogPath: configCatalogPath,
	})
	if err != nil {
		return false, fmt.Errorf("seed development administrator: %w", err)
	}
	return true, nil
}

func errorsForSeedAdmin(message string) error {
	return fmt.Errorf("%s; omit AK_SEED_ADMIN_PASSWORD_FILE and use bootstrap-admin interactively outside local development", message)
}

func bootstrapAdmin(program string, args []string) error {
	flags := flag.NewFlagSet("bootstrap-admin", flag.ContinueOnError)
	email := flags.String("email", "", "administrator email")
	tenantCode := flags.String("tenant-code", "", "initial tenant code")
	tenantName := flags.String("tenant-name", "", "initial tenant name")
	displayName := flags.String("display-name", "", "administrator display name")
	locale := flags.String("locale", "zh-CN", "administrator locale: zh-CN or en-US")
	if err := parseCommandFlags(flags, args, fmt.Sprintf("usage: %s bootstrap-admin --email EMAIL --tenant-code CODE --tenant-name NAME --display-name NAME [--locale LOCALE]", program)); err != nil {
		return err
	}
	if *email == "" || *tenantCode == "" || *tenantName == "" || *displayName == "" || (*locale != "zh-CN" && *locale != "en-US") {
		return fmt.Errorf("email, tenant-code, tenant-name, display-name and a supported locale are required")
	}
	password, err := readPassword()
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseDriver == config.DatabaseDriverSQLite {
		database, openErr := sqliteplatform.Open(context.Background(), cfg.SQLitePath)
		if openErr != nil {
			return openErr
		}
		defer database.Close()
		result, bootstrapErr := iamrepo.NewSQLite(database).BootstrapAdmin(context.Background(), iamrepo.BootstrapAdminInput{
			TenantCode: *tenantCode, TenantName: *tenantName, Email: *email,
			DisplayName: *displayName, Locale: *locale, Password: password,
		})
		if bootstrapErr != nil {
			return bootstrapErr
		}
		fmt.Printf("bootstrapped user_id=%s tenant_id=%s permissions_granted=%d menus_granted=%d\n",
			result.User.ID, result.Tenant.ID, result.GrantedPermissions, result.GrantedMenus)
		return nil
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	assetRoot, cleanup, err := runtimeassets.Materialize()
	if err != nil {
		return err
	}
	defer cleanup()
	user, tenant, granted, grantedMenus, err := seed.BootstrapAdmin(context.Background(), pool, seed.BootstrapAdminInput{
		TenantCode: *tenantCode, TenantName: *tenantName, Email: *email,
		DisplayName: *displayName, Locale: *locale, Password: password,
		ConfigCatalogPath: filepath.Join(assetRoot, "blueprint", "backend", "spec", "core-configs.json"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("bootstrapped user_id=%s tenant_id=%s permissions_granted=%d menus_granted=%d\n", user.ID, tenant.ID, granted, grantedMenus)
	return nil
}

func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return strings.TrimSpace(string(value)), nil
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("read password from stdin: %w", scanner.Err())
	}
	return strings.TrimSpace(scanner.Text()), nil
}
