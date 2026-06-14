package storage

import (
	"context"
	"errors"
)

var (
	ErrAlreadyExists = errors.New("manga already tracked")
	ErrNotFound      = errors.New("manga not found")
)

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
