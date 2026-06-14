package checker

import (
	"context"
	"testing"
	"time"

	"project/mangachapter/internal/notifier"
	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"
)

// mockSource implements source.Source for testing.
type mockSource struct {
	latest *source.ChapterInfo
	err    error
}

func (m *mockSource) Search(ctx context.Context, query string) ([]source.SearchResult, error) {
	return nil, nil
}

func (m *mockSource) GetLatestChapter(ctx context.Context, mangaURL string) (*source.ChapterInfo, error) {
	return m.latest, m.err
}

// mockNotifier records sent notifications.
type mockNotifier struct {
	sent []notifier.NewChapterNotification
}

func (m *mockNotifier) SendNewChapter(ctx context.Context, n notifier.NewChapterNotification) error {
	m.sent = append(m.sent, n)
	return nil
}

func setupTestRepo(t *testing.T) storage.Repository {
	t.Helper()
	repo, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open test repo: %v", err)
	}
	return repo
}

func TestCheckOne_NewChapter(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	// Add a manga with baseline chapter 100
	m := &storage.TrackedManga{
		Source:         "kiryuu",
		SourceID:       "test-1",
		Title:          "Test Manga",
		URL:            "http://example.com/manga/test",
		LastChapter:    "Chapter 100",
		LastChapterNum: 100,
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("add manga: %v", err)
	}

	// Mock source returns chapter 101
	src := &mockSource{
		latest: &source.ChapterInfo{
			Number:   "Chapter 101",
			NumValue: 101,
			URL:      "http://example.com/manga/test/chapter-101",
		},
	}
	mockNotif := &mockNotifier{}
	sources := map[string]source.Source{"kiryuu": src}

	chk := New(repo, sources, mockNotif)
	result, err := chk.CheckOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("CheckOne error: %v", err)
	}

	if result.NewChapter != "Chapter 101" {
		t.Errorf("NewChapter = %q, want %q", result.NewChapter, "Chapter 101")
	}
	if !result.Checked {
		t.Error("Checked should be true")
	}

	// Verify notification was sent
	if len(mockNotif.sent) != 1 {
		t.Fatalf("notifications sent = %d, want 1", len(mockNotif.sent))
	}
	if mockNotif.sent[0].MangaTitle != "Test Manga" {
		t.Errorf("notification MangaTitle = %q, want %q", mockNotif.sent[0].MangaTitle, "Test Manga")
	}
	if mockNotif.sent[0].Chapter != "Chapter 101" {
		t.Errorf("notification Chapter = %q, want %q", mockNotif.sent[0].Chapter, "Chapter 101")
	}
	if mockNotif.sent[0].PreviousChapter != "Chapter 100" {
		t.Errorf("notification PreviousChapter = %q, want %q", mockNotif.sent[0].PreviousChapter, "Chapter 100")
	}

	// Verify DB was updated
	updated, err := repo.GetManga(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetManga after check: %v", err)
	}
	if updated.LastChapter != "Chapter 101" {
		t.Errorf("DB LastChapter = %q, want %q", updated.LastChapter, "Chapter 101")
	}
	if updated.LastChapterNum != 101 {
		t.Errorf("DB LastChapterNum = %v, want %v", updated.LastChapterNum, 101)
	}
}

func TestCheckOne_NoNewChapter(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	m := &storage.TrackedManga{
		Source:         "kiryuu",
		SourceID:       "test-1",
		Title:          "Test Manga",
		URL:            "http://example.com/manga/test",
		LastChapter:    "Chapter 100",
		LastChapterNum: 100,
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("add manga: %v", err)
	}

	// Mock source returns same chapter
	src := &mockSource{
		latest: &source.ChapterInfo{
			Number:   "Chapter 100",
			NumValue: 100,
		},
	}
	mockNotif := &mockNotifier{}
	sources := map[string]source.Source{"kiryuu": src}

	chk := New(repo, sources, mockNotif)
	result, err := chk.CheckOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("CheckOne error: %v", err)
	}

	if result.NewChapter != "" {
		t.Errorf("NewChapter should be empty, got %q", result.NewChapter)
	}
	if len(mockNotif.sent) != 0 {
		t.Errorf("notifications sent = %d, want 0", len(mockNotif.sent))
	}
}

func TestCheckOne_SourceError(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	m := &storage.TrackedManga{
		Source:         "kiryuu",
		SourceID:       "test-1",
		Title:          "Test Manga",
		URL:            "http://example.com/manga/test",
		LastChapter:    "Chapter 100",
		LastChapterNum: 100,
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("add manga: %v", err)
	}

	// Mock source returns error
	src := &mockSource{
		err: context.DeadlineExceeded,
	}
	sources := map[string]source.Source{"kiryuu": src}

	chk := New(repo, sources, nil)
	result, err := chk.CheckOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("CheckOne error: %v", err)
	}

	if result.Error == nil {
		t.Error("expected error in result")
	}
}

func TestCheckOne_UnknownSource(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	m := &storage.TrackedManga{
		Source:   "mangaplus",
		SourceID: "test-1",
		Title:    "Test Manga",
		URL:      "http://example.com/manga/test",
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("add manga: %v", err)
	}

	sources := map[string]source.Source{}
	chk := New(repo, sources, nil)
	result, err := chk.CheckOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("CheckOne error: %v", err)
	}

	if result.Error == nil {
		t.Error("expected error for unknown source")
	}
}

func TestCheckAll(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	// Add two manga
	m1 := &storage.TrackedManga{
		Source: "kiryuu", SourceID: "1", Title: "Manga A",
		URL: "http://a.com", LastChapter: "Chapter 10", LastChapterNum: 10,
	}
	m2 := &storage.TrackedManga{
		Source: "kiryuu", SourceID: "2", Title: "Manga B",
		URL: "http://b.com", LastChapter: "Chapter 5", LastChapterNum: 5,
	}
	if err := repo.AddManga(ctx, m1); err != nil {
		t.Fatalf("add m1: %v", err)
	}
	if err := repo.AddManga(ctx, m2); err != nil {
		t.Fatalf("add m2: %v", err)
	}

	// Mock source always returns chapter 20
	src := &mockSource{
		latest: &source.ChapterInfo{
			Number:   "Chapter 20",
			NumValue: 20,
			URL:      "http://example.com/chapter-20",
		},
	}
	mockNotif := &mockNotifier{}
	sources := map[string]source.Source{"kiryuu": src}

	chk := New(repo, sources, mockNotif)
	results, err := chk.CheckAll(ctx)
	if err != nil {
		t.Fatalf("CheckAll error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}

	newCount := 0
	for _, r := range results {
		if r.NewChapter != "" {
			newCount++
		}
	}
	if newCount != 2 {
		t.Errorf("new chapters = %d, want 2", newCount)
	}
	if len(mockNotif.sent) != 2 {
		t.Errorf("notifications sent = %d, want 2", len(mockNotif.sent))
	}
}

func TestCheckOne_NilNotifier(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	m := &storage.TrackedManga{
		Source: "kiryuu", SourceID: "1", Title: "Test",
		URL: "http://test.com", LastChapter: "Chapter 1", LastChapterNum: 1,
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("add manga: %v", err)
	}

	src := &mockSource{
		latest: &source.ChapterInfo{Number: "Chapter 2", NumValue: 2},
	}
	sources := map[string]source.Source{"kiryuu": src}

	// Nil notifier — should not panic
	chk := New(repo, sources, nil)
	result, err := chk.CheckOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("CheckOne error: %v", err)
	}
	if result.NewChapter != "Chapter 2" {
		t.Errorf("NewChapter = %q, want %q", result.NewChapter, "Chapter 2")
	}
}

func TestCheckAll_EmptyList(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	sources := map[string]source.Source{}
	chk := New(repo, sources, nil)

	results, err := chk.CheckAll(context.Background())
	if err != nil {
		t.Fatalf("CheckAll error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
}

// Suppress unused import
var _ = time.Now