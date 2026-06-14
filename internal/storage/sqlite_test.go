package storage

import (
	"context"
	"errors"
	"testing"
)

func openTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()

	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestCRUD(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	m := &TrackedManga{
		Source:         "kiryuu",
		SourceID:       "one-piece",
		Title:          "One Piece",
		URL:            "https://v6.kiryuu.to/manga/one-piece/",
		LastChapter:    "Chapter 1100",
		LastChapterNum: 1100,
	}
	if err := repo.AddManga(ctx, m); err != nil {
		t.Fatalf("AddManga() error = %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected manga ID to be set")
	}

	if err := repo.AddManga(ctx, m); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate AddManga() error = %v, want ErrAlreadyExists", err)
	}

	got, err := repo.GetManga(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetManga() error = %v", err)
	}
	if got.Title != m.Title {
		t.Errorf("Title = %q, want %q", got.Title, m.Title)
	}

	list, err := repo.ListManga(ctx)
	if err != nil {
		t.Fatalf("ListManga() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListManga() len = %d, want 1", len(list))
	}

	if err := repo.UpdateLastChapter(ctx, m.ID, ChapterUpdate{Number: "Chapter 1101", NumValue: 1101}); err != nil {
		t.Fatalf("UpdateLastChapter() error = %v", err)
	}

	got, err = repo.GetManga(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetManga() after update error = %v", err)
	}
	if got.LastChapter != "Chapter 1101" {
		t.Errorf("LastChapter = %q, want Chapter 1101", got.LastChapter)
	}
	if got.LastChecked == nil {
		t.Error("expected LastChecked to be set after update")
	}

	if err := repo.UpdateLastChecked(ctx, m.ID); err != nil {
		t.Fatalf("UpdateLastChecked() error = %v", err)
	}

	if err := repo.LogNotification(ctx, m.ID, "Chapter 1101", "https://example.com/ch1101"); err != nil {
		t.Fatalf("LogNotification() error = %v", err)
	}

	if err := repo.RemoveManga(ctx, m.ID); err != nil {
		t.Fatalf("RemoveManga() error = %v", err)
	}

	if _, err := repo.GetManga(ctx, m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetManga() after delete error = %v, want ErrNotFound", err)
	}
}

func TestRemoveNotFound(t *testing.T) {
	repo := openTestRepo(t)
	if err := repo.RemoveManga(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("RemoveManga() error = %v, want ErrNotFound", err)
	}
}
