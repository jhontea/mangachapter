package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Checker defines the interface for checking manga.
type Checker interface {
	CheckAll(ctx context.Context) error
}

// Scheduler runs periodic checks.
type Scheduler struct {
	checker  Checker
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// New creates a new Scheduler.
func New(checker Checker, interval time.Duration) *Scheduler {
	return &Scheduler{
		checker:  checker,
		interval: interval,
	}
}

// Run starts the scheduler and blocks until the context is cancelled.
// It runs an immediate first check, then waits the interval before the next check.
func (s *Scheduler) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	slog.Info("scheduler started",
		"interval", s.interval,
	)

	// Run immediately on start
	s.runCheck(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.runCheck(ctx)
		}
	}
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) runCheck(ctx context.Context) {
	slog.Info("scheduler: running check")

	if err := s.checker.CheckAll(ctx); err != nil {
		slog.Error("scheduler: check failed", "error", err)
	}

	slog.Info("scheduler: check complete")
}

// CheckAllFunc adapts a function to the Checker interface.
type CheckAllFunc func(ctx context.Context) error

func (f CheckAllFunc) CheckAll(ctx context.Context) error {
	return f(ctx)
}