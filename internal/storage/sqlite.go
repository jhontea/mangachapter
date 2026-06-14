package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS tracked_manga (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    source           TEXT NOT NULL CHECK(source IN ('kiryuu', 'mangaplus')),
    source_id        TEXT NOT NULL,
    title            TEXT NOT NULL,
    url              TEXT NOT NULL,
    last_chapter     TEXT,
    last_chapter_num REAL,
    last_checked     DATETIME,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source, source_id)
);

CREATE INDEX IF NOT EXISTS idx_tracked_manga_source ON tracked_manga(source);

CREATE TABLE IF NOT EXISTS notifications (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    manga_id    INTEGER NOT NULL REFERENCES tracked_manga(id) ON DELETE CASCADE,
    chapter     TEXT NOT NULL,
    chapter_url TEXT,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_manga_id ON notifications(manga_id);
`

type SQLiteRepository struct {
	db *sql.DB
}

func Open(path string) (*SQLiteRepository, error) {
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db directory: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *SQLiteRepository) migrate() error {
	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLiteRepository) AddManga(ctx context.Context, m *TrackedManga) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num, last_checked)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, m.Source, m.SourceID, m.Title, m.URL, nullString(m.LastChapter), m.LastChapterNum)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert manga: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	m.ID = id
	return nil
}

func (r *SQLiteRepository) RemoveManga(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tracked_manga WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete manga: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) ListManga(ctx context.Context) ([]TrackedManga, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, source, source_id, title, url, last_chapter, last_chapter_num, last_checked, created_at
		FROM tracked_manga
		ORDER BY title ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list manga: %w", err)
	}
	defer rows.Close()

	var items []TrackedManga
	for rows.Next() {
		m, err := scanTrackedManga(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manga rows: %w", err)
	}
	return items, nil
}

func (r *SQLiteRepository) GetManga(ctx context.Context, id int64) (*TrackedManga, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source, source_id, title, url, last_chapter, last_chapter_num, last_checked, created_at
		FROM tracked_manga
		WHERE id = ?
	`, id)

	m, err := scanTrackedManga(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *SQLiteRepository) UpdateLastChapter(ctx context.Context, id int64, ch ChapterUpdate) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tracked_manga
		SET last_chapter = ?, last_chapter_num = ?, last_checked = CURRENT_TIMESTAMP
		WHERE id = ?
	`, ch.Number, ch.NumValue, id)
	if err != nil {
		return fmt.Errorf("update last chapter: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) UpdateLastChecked(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tracked_manga
		SET last_checked = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("update last checked: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) LogNotification(ctx context.Context, mangaID int64, chapter, chapterURL string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (manga_id, chapter, chapter_url)
		VALUES (?, ?, ?)
	`, mangaID, chapter, nullString(chapterURL))
	if err != nil {
		return fmt.Errorf("log notification: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrackedManga(row rowScanner) (*TrackedManga, error) {
	var m TrackedManga
	var lastChapter sql.NullString
	var lastChecked sql.NullString
	var createdAt string

	err := row.Scan(
		&m.ID,
		&m.Source,
		&m.SourceID,
		&m.Title,
		&m.URL,
		&lastChapter,
		&m.LastChapterNum,
		&lastChecked,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}

	if lastChapter.Valid {
		m.LastChapter = lastChapter.String
	}
	if lastChecked.Valid {
		t, err := parseSQLiteTime(lastChecked.String)
		if err != nil {
			return nil, fmt.Errorf("parse last_checked: %w", err)
		}
		m.LastChecked = &t
	}
	m.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	return &m, nil
}

func parseSQLiteTime(value string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
