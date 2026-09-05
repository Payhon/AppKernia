package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/glog"
)

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// Configure applies one process-wide log level/format/output to both stdlib
// slog and GoFrame. File output is mirrored to stderr for service managers.
func Configure(cfg config.Config) (io.Closer, error) {
	var writer io.Writer = os.Stderr
	var closer io.Closer = nopCloser{}
	if cfg.LogFile != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0o700); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		if err = file.Chmod(0o600); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("secure log file: %w", err)
		}
		writer, closer = io.MultiWriter(os.Stderr, file), file
	}
	level := map[string]slog.Level{"debug": slog.LevelDebug, "info": slog.LevelInfo, "warn": slog.LevelWarn, "error": slog.LevelError}[cfg.LogLevel]
	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler = slog.NewTextHandler(writer, options)
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(writer, options)
	}
	slog.SetDefault(slog.New(handler))
	goFrame := glog.Instance()
	goFrame.SetWriter(writer)
	goFrame.SetStdoutPrint(false)
	goFrame.SetWriterColorEnable(false)
	if err := goFrame.SetLevelStr(cfg.LogLevel); err != nil {
		_ = closer.Close()
		return nil, fmt.Errorf("configure GoFrame log level: %w", err)
	}
	if cfg.LogFormat == "json" {
		goFrame.SetHandlers(glog.HandlerJson)
	} else {
		goFrame.SetHandlers(glog.HandlerStructure)
	}
	// Package-level GoFrame logs and ghttp use distinct defaults. Point both at
	// the configured logger before bootstrap constructs the default server.
	glog.SetDefaultLogger(goFrame)
	ghttp.GetServer().SetLogger(goFrame)
	return closer, nil
}
