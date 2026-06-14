package checker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project/mangachapter/internal/notifier"
	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"
)

const (
	maxRetries      = 2
	retryBackoff    = 2 * time.Second
	contextTimeout  = 30 * time.Second
)

// Result represents the outcome of checking a single manga.
type Result struct {
	MangaID    int64
	Title      string
	Source     string
	NewChapter string
	ChapterURL string
	Checked    bool
	Error      error
}

// Checker orchestrates checking all tracked manga for new chapters.
type Checker struct {
	repo     storage.Repository
	sources  map[string]source.Source
	notifier notifier.Notifier
}

// New creates a new Checker with the given dependencies.
func New(repo storage.Repository, sources map[string]source.Source, n notifier.Notifier) *Checker {
	return &Checker{
		repo:     repo,
		sources:  sources,
		notifier: n,
	}
}

// CheckAll checks all tracked manga for new chapters.
// Returns a summary of results.
func (c *Checker) CheckAll(ctx context.Context) ([]Result, error) {
	mangaList, err := c.repo.ListManga(ctx)
	if err != nil {
		return nil, fmt.Errorf("list manga: %w", err)
	}

	slog.Info("checking all manga", "count", len(mangaList))

	var results []Result
	for _, m := range mangaList {
		r := c.checkOne(ctx, m)
		results = append(results, r)

		// Log result
		if r.Error != nil {
			slog.Error("check failed",
				"manga_id", m.ID,
				"title", m.Title,
				"source", m.Source,
				"error", r.Error,
			)
		} else if r.NewChapter != "" {
			slog.Info("new chapter found",
				"manga_id", m.ID,
				"title", m.Title,
				"chapter", r.NewChapter,
			)
		} else {
			slog.Debug("no new chapter",
				"manga_id", m.ID,
				"title", m.Title,
			)
		}
	}

	// Print summary
	checked, newChapters, errors := 0, 0, 0
	for _, r := range results {
		checked++
		if r.Error != nil {
			errors++
		} else if r.NewChapter != "" {
			newChapters++
		}
	}
	slog.Info("check complete",
		"checked", checked,
		"new_chapters", newChapters,
		"errors", errors,
	)

	return results, nil
}

// CheckOne checks a single manga by ID for new chapters.
func (c *Checker) CheckOne(ctx context.Context, id int64) (*Result, error) {
	m, err := c.repo.GetManga(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get manga %d: %w", id, err)
	}

	r := c.checkOne(ctx, *m)
	return &r, nil
}

// checkOne performs the check for a single manga with retry logic.
func (c *Checker) checkOne(ctx context.Context, m storage.TrackedManga) Result {
	src, ok := c.sources[m.Source]
	if !ok {
		return Result{
			MangaID: m.ID,
			Title:   m.Title,
			Source:  m.Source,
			Error:   fmt.Errorf("unknown source %q", m.Source),
		}
	}

	// Fetch latest chapter with retry
	var ch *source.ChapterInfo
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying fetch",
				"manga_id", m.ID,
				"attempt", attempt,
			)
			select {
			case <-ctx.Done():
				return Result{MangaID: m.ID, Title: m.Title, Source: m.Source, Error: ctx.Err()}
			case <-time.After(retryBackoff):
			}
		}

		// Use a context with timeout per manga
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
			Error:   fmt.Errorf("fetch chapter after %d retries: %w", maxRetries, err),
		}
	}

	// Compare with stored chapter
	if !HasNewChapter(m.LastChapterNum, ch) {
		// No new chapter — just update last checked
		if err := c.repo.UpdateLastChecked(ctx, m.ID); err != nil {
			slog.Error("update last checked failed", "manga_id", m.ID, "error", err)
		}
		return Result{MangaID: m.ID, Title: m.Title, Source: m.Source, Checked: true}
	}

	// New chapter found — notify first, then update DB
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
			slog.Error("send notification failed",
				"manga_id", m.ID,
				"title", m.Title,
				"chapter", ch.Number,
				"error", err,
			)
			// Per ARCHITECTURE.md: update DB even if notification fails
		}
	}

	// Update DB with new chapter
	chUpdate := storage.ChapterUpdate{
		Number:   ch.Number,
		NumValue: ch.NumValue,
	}
	if err := c.repo.UpdateLastChapter(ctx, m.ID, chUpdate); err != nil {
		result.Error = fmt.Errorf("update last chapter: %w", err)
	}

	return result
}