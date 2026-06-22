package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Checker mendefinisikan interface untuk memeriksa manga.
type Checker interface {
	CheckAll(ctx context.Context) error
}

// Scheduler menjalankan pengecekan berkala menggunakan cron/v3.
type Scheduler struct {
	checker  Checker
	cronExpr string        // cron expression, e.g. "0 * * * *"
	interval time.Duration // fallback jika cronExpr kosong
	cron     *cron.Cron
	mu       sync.Mutex
	running  bool
}

// New membuat Scheduler baru dengan interval (dikonversi ke cron expression).
func New(checker Checker, interval time.Duration) *Scheduler {
	return &Scheduler{
		checker:  checker,
		interval: interval,
	}
}

// NewWithCron membuat Scheduler baru dengan cron expression.
func NewWithCron(checker Checker, cronExpr string) *Scheduler {
	return &Scheduler{
		checker:  checker,
		cronExpr: cronExpr,
	}
}

// Run menjalankan scheduler dan memblokir sampai context dibatalkan.
// Menjalankan pengecekan pertama segera, lalu terjadwal sesuai cron/interval.
func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()

	expr := s.cronExpr
	if expr == "" {
		expr = durationToCron(s.interval)
	}

	c := cron.New()
	s.cron = c
	s.running = true
	s.mu.Unlock()

	slog.Info("scheduler dimulai", "cron", expr)

	// Jalankan segera saat start
	s.runCheck(ctx)

	_, err := c.AddFunc(expr, func() {
		s.runCheck(ctx)
	})
	if err != nil {
		slog.Error("scheduler: gagal menambahkan job cron", "expr", expr, "error", err)
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return
	}

	c.Start()
	defer func() {
		<-c.Stop().Done()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		slog.Info("scheduler dihentikan")
	}()

	<-ctx.Done()
}

// Stop menghentikan scheduler secara graceful.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	c := s.cron
	s.mu.Unlock()

	if c != nil {
		<-c.Stop().Done()
	}
}

func (s *Scheduler) runCheck(ctx context.Context) {
	slog.Info("scheduler: menjalankan pengecekan")

	if err := s.checker.CheckAll(ctx); err != nil {
		slog.Error("scheduler: pengecekan gagal", "error", err)
	}

	slog.Info("scheduler: pengecekan selesai")
}

// durationToCron mengonversi time.Duration ke cron expression sederhana.
// Mendukung kelipatan jam dan menit; durasi di bawah 1 menit dibulatkan ke 1 menit.
func durationToCron(d time.Duration) string {
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		h := int(d.Hours())
		if h == 1 {
			return "0 * * * *" // setiap jam
		}
		return fmt.Sprintf("0 */%d * * *", h)
	case d >= time.Minute:
		m := int(d.Minutes())
		if m == 1 {
			return "* * * * *" // setiap menit
		}
		return fmt.Sprintf("*/%d * * * *", m)
	default:
		return "* * * * *" // fallback: setiap menit
	}
}

// CheckAllFunc mengadaptasi fungsi ke interface Checker.
type CheckAllFunc func(ctx context.Context) error

func (f CheckAllFunc) CheckAll(ctx context.Context) error {
	return f(ctx)
}
