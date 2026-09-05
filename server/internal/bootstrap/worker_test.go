package bootstrap

import (
	"testing"

	"github.com/appkernia/appkernia/server/internal/platform/config"
)

func TestNewWorkerRejectsInvalidDatabaseURL(t *testing.T) {
	_, err := NewWorker(t.Context(), config.Config{DatabaseDriver: config.DatabaseDriverPostgreSQL, DatabaseURL: "://", ConfigMasterKeyBase64: "invalid"})
	if err == nil {
		t.Fatal("NewWorker accepted an invalid database URL")
	}
}
