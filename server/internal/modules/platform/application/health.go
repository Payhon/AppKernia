package application

import (
	"context"
	"time"
)

type DatabaseProber interface {
	Ping(context.Context) error
}

type HealthService struct {
	database DatabaseProber
	timeout  time.Duration
}

func NewHealthService(database DatabaseProber, timeout time.Duration) *HealthService {
	return &HealthService{database: database, timeout: timeout}
}

func (s *HealthService) Ready(ctx context.Context) error {
	timeoutContext, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.database.Ping(timeoutContext)
}
