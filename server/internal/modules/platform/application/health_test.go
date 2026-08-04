package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

type probeStub struct {
	err error
}

func (p probeStub) Ping(context.Context) error {
	return p.err
}

func TestHealthServiceReady(t *testing.T) {
	service := NewHealthService(probeStub{}, time.Second)
	if err := service.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestHealthServiceReportsDatabaseFailure(t *testing.T) {
	service := NewHealthService(probeStub{err: errors.New("unavailable")}, time.Second)
	if err := service.Ready(context.Background()); err == nil {
		t.Fatal("Ready() expected an error")
	}
}
