package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialsAreOwnerOnlyAndEnvironmentOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	stored := clientCredentials{Server: "https://stored.example.test", ClientID: "ak_stored", ClientSecret: "aks_stored-secret-that-is-long-enough"}
	if err := writeCredentials(path, stored); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials mode=%v err=%v", info.Mode().Perm(), err)
	}
	t.Setenv("AK_SERVER_URL", "https://env.example.test")
	t.Setenv("AK_CLIENT_ID", "ak_environment")
	t.Setenv("AK_CLIENT_SECRET", "aks_environment-secret-that-is-long-enough")
	loaded, err := loadCredentials(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server != "https://env.example.test" || loaded.ClientID != "ak_environment" || !strings.HasPrefix(loaded.ClientSecret, "aks_environment") {
		t.Fatalf("unexpected credentials: %#v", loaded)
	}
}

func TestCredentialsRejectInsecureRemoteServer(t *testing.T) {
	credentials := clientCredentials{Server: "http://example.test", ClientID: "ak_test", ClientSecret: "aks_secret-value-that-is-long-enough"}
	if err := credentials.validate(); err == nil {
		t.Fatal("insecure remote server was accepted")
	}
}
