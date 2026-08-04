package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrForbidden = errors.New("operations health permission denied")

type Dependency struct {
	Code       string    `json:"code"`
	Status     string    `json:"status"`
	LatencyMS  int64     `json:"latency_ms"`
	DetailCode string    `json:"detail_code"`
	CheckedAt  time.Time `json:"checked_at"`
}
type Health struct {
	Status       string       `json:"status"`
	Dependencies []Dependency `json:"dependencies"`
	CheckedAt    time.Time    `json:"checked_at"`
}
type Module struct {
	Code    string `json:"code"`
	Version string `json:"version"`
	Status  string `json:"status"`
}
type Queue struct {
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	Available       int64      `json:"available"`
	Running         int64      `json:"running"`
	Retryable       int64      `json:"retryable"`
	Scheduled       int64      `json:"scheduled"`
	Completed       int64      `json:"completed"`
	Failed          int64      `json:"failed"`
}
type RunSummary struct {
	Queued    int64 `json:"queued"`
	Running   int64 `json:"running"`
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
	Skipped   int64 `json:"skipped"`
}
type RuntimeSummary struct {
	AppVersion      string     `json:"app_version"`
	GoVersion       string     `json:"go_version"`
	UptimeSeconds   int64      `json:"uptime_seconds"`
	Modules         []Module   `json:"modules"`
	Queue           Queue      `json:"queue"`
	ScheduleRuns24H RunSummary `json:"schedule_runs_24h"`
	GeneratedAt     time.Time  `json:"generated_at"`
}
type Repository interface {
	Health(context.Context) Health
	Runtime(context.Context, uuid.UUID, time.Time) (RuntimeSummary, error)
}
