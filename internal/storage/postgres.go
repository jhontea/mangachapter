package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

// PostgresRepository mengimplementasikan Repository menggunakan PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// OpenPostgres membuka koneksi ke database PostgreSQL dan menjalankan migrasi.
func OpenPostgres(dsn string) (*PostgresRepository, error) {
	// Parse config dan gunakan SimpleProtocol untuk hindari prepared statement cache
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Register sebagai named driver
	connStr := stdlib.RegisterConnConfig(config)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("buka postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	repo := &PostgresRepository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *PostgresRepository) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS tracked_manga (
			id               BIGSERIAL PRIMARY KEY,
			source           TEXT NOT NULL CHECK(source IN ('kiryuu', 'mangaplus')),
			source_id        TEXT NOT NULL,
			title            TEXT NOT NULL,
			url              TEXT NOT NULL,
			last_chapter     TEXT,
			last_chapter_num DOUBLE PRECISION,
			last_checked     TIMESTAMPTZ,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(source, source_id)
		);

		CREATE INDEX IF NOT EXISTS idx_tracked_manga_source ON tracked_manga(source);

		CREATE TABLE IF NOT EXISTS notifications (
			id          BIGSERIAL PRIMARY KEY,
			manga_id    BIGINT NOT NULL REFERENCES tracked_manga(id) ON DELETE CASCADE,
			chapter     TEXT NOT NULL,
			chapter_url TEXT,
			sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_notifications_manga_id ON notifications(manga_id);
	`)
	if err != nil {
		return fmt.Errorf("migrasi schema: %w", err)
	}
	return nil
}

// Close menutup koneksi database.
func (r *PostgresRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

// AddManga menambahkan manga baru ke daftar yang dilacak.
func (r *PostgresRepository) AddManga(ctx context.Context, m *TrackedManga) error {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num, last_checked)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id
	`, m.Source, m.SourceID, m.Title, m.URL, nullString(m.LastChapter), m.LastChapterNum).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert manga: %w", err)
	}
	m.ID = id
	return nil
}

// RemoveManga menghapus manga dari daftar yang dilacak.
func (r *PostgresRepository) RemoveManga(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tracked_manga WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("hapus manga: %w", err)
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

// ListManga mengembalikan semua manga yang dilacak.
func (r *PostgresRepository) ListManga(ctx context.Context) ([]TrackedManga, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, source, source_id, title, url, last_chapter, last_chapter_num, last_checked, created_at
		FROM tracked_manga
		ORDER BY title ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("daftar manga: %w", err)
	}
	defer rows.Close()

	var items []TrackedManga
	for rows.Next() {
		m, err := scanTrackedMangaPG(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi baris manga: %w", err)
	}
	return items, nil
}

// GetManga mengembalikan manga berdasarkan ID.
func (r *PostgresRepository) GetManga(ctx context.Context, id int64) (*TrackedManga, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source, source_id, title, url, last_chapter, last_chapter_num, last_checked, created_at
		FROM tracked_manga
		WHERE id = $1
	`, id)

	m, err := scanTrackedMangaPG(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

// UpdateLastChapter memperbarui chapter terakhir yang diketahui.
func (r *PostgresRepository) UpdateLastChapter(ctx context.Context, id int64, ch ChapterUpdate) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tracked_manga
		SET last_chapter = $1, last_chapter_num = $2, last_checked = NOW()
		WHERE id = $3
	`, ch.Number, ch.NumValue, id)
	if err != nil {
		return fmt.Errorf("update chapter terakhir: %w", err)
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

// UpdateLastChecked memperbarui waktu pengecekan terakhir.
func (r *PostgresRepository) UpdateLastChecked(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tracked_manga
		SET last_checked = NOW()
		WHERE id = $1
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

// LogNotification mencatat notifikasi yang telah dikirim.
func (r *PostgresRepository) LogNotification(ctx context.Context, mangaID int64, chapter, chapterURL string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (manga_id, chapter, chapter_url)
		VALUES ($1, $2, $3)
	`, mangaID, chapter, nullString(chapterURL))
	if err != nil {
		return fmt.Errorf("catat notifikasi: %w", err)
	}
	return nil
}

// scanTrackedMangaPG scans a row into TrackedManga for PostgreSQL.
// PostgreSQL with pgx returns timestamps differently than SQLite.
func scanTrackedMangaPG(row rowScanner) (*TrackedManga, error) {
	var m TrackedManga
	var lastChapter sql.NullString
	var lastChecked sql.NullTime
	var createdAt time.Time

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
		m.LastChecked = &lastChecked.Time
	}
	m.CreatedAt = createdAt

	return &m, nil
}

// init registers the pgx driver
func init() {
	_ = stdlib.GetDefaultDriver()
}
