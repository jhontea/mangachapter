package storage

import (
	"context"
	"errors"
)

var (
	ErrAlreadyExists = errors.New("manga sudah dilacak")
	ErrNotFound      = errors.New("manga tidak ditemukan")
)

// Repository mendefinisikan interface untuk penyimpanan data manga.
type Repository interface {
	AddManga(ctx context.Context, m *TrackedManga) error
	RemoveManga(ctx context.Context, id int64) error
	ListManga(ctx context.Context) ([]TrackedManga, error)
	GetManga(ctx context.Context, id int64) (*TrackedManga, error)
	UpdateLastChapter(ctx context.Context, id int64, ch ChapterUpdate) error
	UpdateLastChecked(ctx context.Context, id int64) error
	LogNotification(ctx context.Context, mangaID int64, chapter, chapterURL string) error
	Close() error
}