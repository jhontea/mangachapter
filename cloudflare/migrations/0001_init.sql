-- Schema database untuk Manga Chapter Notifier
-- Cloudflare D1 (SQLite-compatible)

-- Tabel manga yang dilacak
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

-- Index untuk pencarian berdasarkan sumber
CREATE INDEX IF NOT EXISTS idx_tracked_manga_source ON tracked_manga(source);

-- Tabel notifikasi yang telah dikirim
CREATE TABLE IF NOT EXISTS notifications (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    manga_id    INTEGER NOT NULL REFERENCES tracked_manga(id) ON DELETE CASCADE,
    chapter     TEXT NOT NULL,
    chapter_url TEXT,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_manga_id ON notifications(manga_id);