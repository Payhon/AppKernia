package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/migration"
	"github.com/appkernia/appkernia/server/internal/seed"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "doctor":
		err = doctor()
	case "migrate":
		err = migrateDatabase(os.Args[2:])
	case "seed":
		err = seedCorePermissions(os.Args[2:])
	case "bootstrap-admin":
		err = bootstrapAdmin(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ak-cli doctor | migrate up|down [steps] | seed core-permissions | bootstrap-admin [flags]")
}

func doctor() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		return fmt.Errorf("database readiness failed: %w", err)
	}
	fmt.Println("ak-cli doctor: PostgreSQL ready")
	return nil
}

func migrateDatabase(args []string) error {
	if len(args) < 1 || len(args) > 2 || (args[0] != "up" && args[0] != "down") {
		return errorsForUsage("migration direction must be up or down")
	}
	var steps int
	if len(args) == 2 {
		parsed, err := strconv.Atoi(args[1])
		if err != nil || parsed <= 0 {
			return errorsForUsage("migration steps must be a positive integer")
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
	migrationPath := os.Getenv("AK_MIGRATION_PATH")
	if migrationPath == "" {
		migrationPath = "../blueprint/backend/db/migrations"
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

func errorsForUsage(message string) error {
	return fmt.Errorf("%s; usage: ak-cli migrate up|down [steps]", message)
}

func seedCorePermissions(args []string) error {
	if len(args) != 1 || (args[0] != "core" && args[0] != "core-permissions") {
		return fmt.Errorf("usage: ak-cli seed core")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	permissionPaths := []string{
		"../blueprint/backend/spec/core-permissions.json",
		"../blueprint/admin-frontend/integration/core-permissions.delta.json",
	}
	permissionCount := 0
	for _, catalogPath := range permissionPaths {
		count, seedErr := seed.CorePermissions(context.Background(), pool, catalogPath)
		if seedErr != nil {
			return seedErr
		}
		permissionCount += count
	}
	menuCount, err := seed.CoreMenus(context.Background(), pool, "../blueprint/admin-frontend/spec/admin-menu-seed.json")
	if err != nil {
		return err
	}
	fmt.Printf("seeded core permissions=%d menus=%d\n", permissionCount, menuCount)
	return nil
}

func bootstrapAdmin(args []string) error {
	flags := flag.NewFlagSet("bootstrap-admin", flag.ContinueOnError)
	email := flags.String("email", "", "administrator email")
	tenantCode := flags.String("tenant-code", "", "initial tenant code")
	tenantName := flags.String("tenant-name", "", "initial tenant name")
	displayName := flags.String("display-name", "", "administrator display name")
	locale := flags.String("locale", "zh-CN", "administrator locale: zh-CN or en-US")
	if err := flags.Parse(args); err != nil {
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
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	user, tenant, granted, grantedMenus, err := seed.BootstrapAdmin(context.Background(), pool, seed.BootstrapAdminInput{
		TenantCode: *tenantCode, TenantName: *tenantName, Email: *email,
		DisplayName: *displayName, Locale: *locale, Password: password,
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
