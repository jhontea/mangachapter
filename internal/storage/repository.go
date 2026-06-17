package storage

import (
	"context"
	"errors"
	"strings"
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

// Open membuka koneksi ke database berdasarkan DSN.
// Jika DSN mengandung "postgres" atau "postgresql", buka PostgreSQL.
// Jika tidak, buka SQLite (backward compatibility).
func Open(dsn string) (Repository, error) {
	if dsn == "" {
		return nil, errors.New("dsn tidak boleh kosong")
	}

	// PostgreSQL DSN
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return OpenPostgres(dsn)
	}

	// SQLite fallback
	return OpenSQLite(dsn)
}
