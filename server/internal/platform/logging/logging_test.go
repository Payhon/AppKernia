package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/glog"
)

func TestConfigureWritesStructuredFile(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	path := filepath.Join(t.TempDir(), "logs", "akone.log")
	closer, err := Configure(config.Config{LogLevel: "info", LogFormat: "json", LogFile: path})
	if err != nil {
		t.Fatal(err)
	}
	slog.Info("ready", "component", "test")
	glog.Info(t.Context(), "gf-ready")
	ghttp.GetServer().Logger().Info(t.Context(), "ghttp-ready")
	if err = closer.Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), `"msg":"ready"`) || !strings.Contains(string(content), `"component":"test"`) || !strings.Contains(string(content), "gf-ready") || !strings.Contains(string(content), "ghttp-ready") {
		t.Fatalf("unexpected log content=%q err=%v", content, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("log mode=%v err=%v", info.Mode().Perm(), err)
	}
}
