package checker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"project/mangachapter/internal/notifier"
	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"
)

const (
	maxRetries     = 2
	retryBackoff   = 2 * time.Second
	contextTimeout = 30 * time.Second
	maxConcurrent  = 10 // maksimum pengecekan bersamaan
)

// Result merepresentasikan hasil pengecekan satu manga.
type Result struct {
	MangaID    int64
	Title      string
	Source     string
	NewChapter string
	ChapterURL string
	Checked    bool
	Error      error
}

// Checker mengatur pengecekan semua manga yang dilacak untuk chapter baru.
type Checker struct {
	repo     storage.Repository
	sources  map[string]source.Source
	notifier notifier.Notifier
}

// New membuat Checker baru dengan dependensi yang diberikan.
func New(repo storage.Repository, sources map[string]source.Source, n notifier.Notifier) *Checker {
	return &Checker{
		repo:     repo,
		sources:  sources,
		notifier: n,
	}
}

// CheckAll memeriksa semua manga yang dilacak untuk chapter baru secara bersamaan menggunakan errgroup.
// Menunggu semua pengecekan selesai sebelum mengembalikan hasil.
func (c *Checker) CheckAll(ctx context.Context) ([]Result, error) {
	mangaList, err := c.repo.ListManga(ctx)
	if err != nil {
		return nil, fmt.Errorf("daftar manga: %w", err)
	}

	slog.Info("memeriksa semua manga", "jumlah", len(mangaList))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	var (
		mu      sync.Mutex
		results []Result
	)

	for _, m := range mangaList {
		g.Go(func() error {
			r := c.checkOne(gctx, m)

			mu.Lock()
			results = append(results, r)
			mu.Unlock()

			// Log hasil
			if r.Error != nil {
				slog.Error("pemeriksaan gagal",
					"manga_id", m.ID,
					"judul", m.Title,
					"sumber", m.Source,
					"error", r.Error,
				)
			} else if r.NewChapter != "" {
				slog.Info("chapter baru ditemukan",
					"manga_id", m.ID,
					"judul", m.Title,
					"chapter", r.NewChapter,
				)
			} else {
				slog.Debug("tidak ada chapter baru",
					"manga_id", m.ID,
					"judul", m.Title,
				)
			}

			return nil // jangan propagasi error individual ke errgroup
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup: %w", err)
	}

	// Cetak ringkasan
	newChapters, errs := 0, 0
	for _, r := range results {
		if r.Error != nil {
			errs++
		} else if r.NewChapter != "" {
			newChapters++
		}
	}
	slog.Info("pemeriksaan selesai",
		"diperiksa", len(results),
		"chapter_baru", newChapters,
		"error", errs,
	)

	return results, nil
}

// CheckOne memeriksa satu manga berdasarkan ID untuk chapter baru.
func (c *Checker) CheckOne(ctx context.Context, id int64) (*Result, error) {
	m, err := c.repo.GetManga(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ambil manga %d: %w", id, err)
	}

	r := c.checkOne(ctx, *m)
	return &r, nil
}

// checkOne melakukan pengecekan satu manga dengan logika retry.
func (c *Checker) checkOne(ctx context.Context, m storage.TrackedManga) Result {
	src, ok := c.sources[m.Source]
	if !ok {
		return Result{
			MangaID: m.ID,
			Title:   m.Title,
			Source:  m.Source,
			Error:   fmt.Errorf("sumber tidak dikenal %q", m.Source),
		}
	}

	// Ambil chapter terbaru dengan retry
	var ch *source.ChapterInfo
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("mencoba ulang fetch",
				"manga_id", m.ID,
				"percobaan", attempt,
			)
			select {
			case <-ctx.Done():
				return Result{MangaID: m.ID, Title: m.Title, Source: m.Source, Error: ctx.Err()}
			case <-time.After(retryBackoff):
			}
		}

		// Gunakan context dengan timeout per manga
		fetchCtx, cancel := context.WithTimeout(ctx, contextTimeout)
		ch, err = src.GetLatestChapter(fetchCtx, m.URL)
		cancel()

		if err == nil {
			break
		}
	}

	if err != nil {
		return Result{
			MangaID: m.ID,
			Title:   m.Title,
			Source:  m.Source,
			Error:   fmt.Errorf("ambil chapter setelah %d retry: %w", maxRetries, err),
		}
	}

	// Bandingkan dengan chapter yang tersimpan
	if !HasNewChapter(m.LastChapterNum, ch) {
		// Tidak ada chapter baru — hanya update last checked
		if err := c.repo.UpdateLastChecked(ctx, m.ID); err != nil {
			slog.Error("update last checked gagal", "manga_id", m.ID, "error", err)
		}
		return Result{MangaID: m.ID, Title: m.Title, Source: m.Source, Checked: true}
	}

	// Chapter baru ditemukan — notifikasi dulu, lalu update DB
	result := Result{
		MangaID:    m.ID,
		Title:      m.Title,
		Source:     m.Source,
		NewChapter: ch.Number,
		ChapterURL: ch.URL,
		Checked:    true,
	}

	if c.notifier != nil {
		n := notifier.NewChapterNotification{
			MangaTitle:      m.Title,
			Source:          m.Source,
			Chapter:         ch.Number,
			ChapterURL:      ch.URL,
			PreviousChapter: m.LastChapter,
		}
		if err := c.notifier.SendNewChapter(ctx, n); err != nil {
			slog.Error("kirim notifikasi gagal",
				"manga_id", m.ID,
				"judul", m.Title,
				"chapter", ch.Number,
				"error", err,
			)
			// Berdasarkan ARCHITECTURE.md: update DB meskipun notifikasi gagal
		}
	}

	// Update DB dengan chapter baru
	chUpdate := storage.ChapterUpdate{
		Number:   ch.Number,
		NumValue: ch.NumValue,
	}
	if err := c.repo.UpdateLastChapter(ctx, m.ID, chUpdate); err != nil {
		result.Error = fmt.Errorf("update chapter terakhir: %w", err)
	}

	return result
}
