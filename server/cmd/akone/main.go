package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/appkernia/appkernia/server/internal/bootstrap"
	"github.com/appkernia/appkernia/server/internal/command"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/logging"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "akone:", err)
		var usageErr *command.UsageError
		if errors.As(err, &usageErr) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	configPath, args, err := globalConfigPath(args)
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		usage()
		return nil
	}
	if args[0] == "serve" {
		return serve(configPath, args[1:])
	}
	if configPath != "" {
		previous, existed := os.LookupEnv("AK_CONFIG_FILE")
		if err = os.Setenv("AK_CONFIG_FILE", configPath); err != nil {
			return fmt.Errorf("select configuration file: %w", err)
		}
		defer func() {
			if existed {
				_ = os.Setenv("AK_CONFIG_FILE", previous)
			} else {
				_ = os.Unsetenv("AK_CONFIG_FILE")
			}
		}()
	}
	return command.Run("akone", args)
}

func globalConfigPath(args []string) (string, []string, error) {
	if len(args) == 0 || args[0] != "--config" {
		return "", args, nil
	}
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return "", nil, &command.UsageError{Message: "--config requires a YAML file path"}
	}
	return strings.TrimSpace(args[1]), args[2:], nil
}

func usage() {
	fmt.Fprintln(os.Stdout, "AppKernia single-binary server and command-line client")
	fmt.Fprintln(os.Stdout, "usage: akone [--config FILE] serve [flags]")
	command.Usage(os.Stdout, "akone")
}

func serve(globalPath string, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	configPath := flags.String("config", "", "YAML configuration file")
	listen := flags.String("listen", "", "HTTP listen address")
	adminPath := flags.String("admin-path", "", "Admin URL path")
	adminStaticDir := flags.String("admin-static-dir", "", "external Admin static directory")
	sqlitePath := flags.String("sqlite", "", "SQLite database file path")
	logLevel := flags.String("log-level", "", "debug, info, warn, or error")
	logFormat := flags.String("log-format", "", "text or json")
	logFile := flags.String("log-file", "", "log file path")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil || flags.NArg() != 0 {
		return &command.UsageError{Message: "usage: akone [--config FILE] serve [--listen ADDR] [--sqlite FILE] [--admin-path PATH] [--admin-static-dir DIR] [--log-level LEVEL] [--log-format FORMAT] [--log-file FILE]"}
	}
	if globalPath != "" && strings.TrimSpace(*configPath) != "" {
		return &command.UsageError{Message: "--config may be specified either before or after serve, not both"}
	}
	if globalPath == "" {
		globalPath = strings.TrimSpace(*configPath)
	}
	databaseDriver := ""
	if strings.TrimSpace(*sqlitePath) != "" {
		databaseDriver = config.DatabaseDriverSQLite
	}
	cfg, err := config.Load(config.Options{Path: globalPath, Overrides: config.Overrides{
		HTTPAddr: strings.TrimSpace(*listen), AdminPath: strings.TrimSpace(*adminPath),
		AdminStaticDir: strings.TrimSpace(*adminStaticDir), LogLevel: strings.TrimSpace(*logLevel),
		LogFormat: strings.TrimSpace(*logFormat), LogFile: strings.TrimSpace(*logFile),
		DatabaseDriver: databaseDriver, SQLitePath: strings.TrimSpace(*sqlitePath),
	}})
	if err != nil {
		return err
	}
	logCloser, err := logging.Configure(cfg)
	if err != nil {
		return err
	}
	defer logCloser.Close()
	logFields := []any{"version", buildinfo.Version, "commit", buildinfo.Commit, "listen", cfg.HTTPAddr, "database", cfg.DatabaseDriver}
	if cfg.DatabaseDriver == config.DatabaseDriverSQLite {
		logFields = append(logFields, "sqlite_path", cfg.SQLitePath)
	}
	slog.Info("starting AppKernia", logFields...)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	api, err := bootstrap.NewAPI(ctx, cfg)
	if err != nil {
		return err
	}
	var worker *bootstrap.Worker
	if cfg.DatabaseDriver == config.DatabaseDriverPostgreSQL {
		worker, err = bootstrap.NewWorker(ctx, cfg)
		if err != nil {
			return errors.Join(err, api.Shutdown())
		}
		if err = worker.Start(ctx); err != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			return errors.Join(err, worker.Shutdown(shutdownCtx), api.Shutdown())
		}
	}
	if err = api.Start(); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if worker != nil {
			return errors.Join(err, worker.Shutdown(shutdownCtx), api.Shutdown())
		}
		return errors.Join(err, api.Shutdown())
	}
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if worker != nil {
		return errors.Join(api.Shutdown(), worker.Shutdown(shutdownCtx))
	}
	return api.Shutdown()
}
