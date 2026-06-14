package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Checker mendefinisikan interface untuk memeriksa manga.
type Checker interface {
	CheckAll(ctx context.Context) error
}

// Scheduler menjalankan pengecekan berkala.
type Scheduler struct {
	checker  Checker
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// New membuat Scheduler baru.
func New(checker Checker, interval time.Duration) *Scheduler {
	return &Scheduler{
		checker:  checker,
		interval: interval,
	}
}

// Run menjalankan scheduler dan memblokir sampai context dibatalkan.
// Menjalankan pengecekan pertama segera, lalu menunggu interval sebelum pengecekan berikutnya.
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

	slog.Info("scheduler dimulai",
		"interval", s.interval,
	)

	// Jalankan segera saat start
	s.runCheck(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler dihentikan")
			return
		case <-ticker.C:
			s.runCheck(ctx)
		}
	}
}

// Stop menghentikan scheduler secara graceful.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// IsRunning mengembalikan apakah scheduler sedang berjalan.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) runCheck(ctx context.Context) {
	slog.Info("scheduler: menjalankan pengecekan")

	if err := s.checker.CheckAll(ctx); err != nil {
		slog.Error("scheduler: pengecekan gagal", "error", err)
	}

	slog.Info("scheduler: pengecekan selesai")
}

// CheckAllFunc mengadaptasi fungsi ke interface Checker.
type CheckAllFunc func(ctx context.Context) error

func (f CheckAllFunc) CheckAll(ctx context.Context) error {
	return f(ctx)
}