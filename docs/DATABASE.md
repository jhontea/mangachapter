# Database Schema

SQLite database untuk menyimpan watchlist dan log notifikasi.

## Lokasi File

Default: `./data/manga.db` (configurable via `storage.path`)

Buat direktori `data/` otomatis saat startup jika belum ada.

---

## Schema

### `tracked_manga`

Manga yang dipantau.

```sql
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
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Primary key |
| `source` | TEXT | `kiryuu` atau `mangaplus` |
| `source_id` | TEXT | Slug (kiryuu) atau title ID string (mangaplus) |
| `title` | TEXT | Display name |
| `url` | TEXT | Full URL halaman manga |
| `last_chapter` | TEXT | Display chapter terakhir, e.g. "Chapter 123" |
| `last_chapter_num` | REAL | Numeric value untuk comparison |
| `last_checked` | DATETIME | Timestamp cek terakhir (UTC) |
| `created_at` | DATETIME | Waktu ditambahkan |

### `notifications`

Log email/notifikasi terkirim (audit trail).

```sql
CREATE TABLE IF NOT EXISTS notifications (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    manga_id   INTEGER NOT NULL REFERENCES tracked_manga(id) ON DELETE CASCADE,
    chapter    TEXT NOT NULL,
    chapter_url TEXT,
    sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_manga_id ON notifications(manga_id);
```

---

## Go Models

```go
package storage

import "time"

type TrackedManga struct {
    ID             int64
    Source         string
    SourceID       string
    Title          string
    URL            string
    LastChapter    string
    LastChapterNum float64
    LastChecked    *time.Time
    CreatedAt      time.Time
}

type Notification struct {
    ID         int64
    MangaID    int64
    Chapter    string
    ChapterURL string
    SentAt     time.Time
}
```

---

## Repository Queries

### Add manga

```sql
INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num, last_checked)
VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);
```

On conflict (`UNIQUE(source, source_id)`): return error `ErrAlreadyExists`.

### List all

```sql
SELECT id, source, source_id, title, url, last_chapter, last_chapter_num, last_checked, created_at
FROM tracked_manga
ORDER BY title ASC;
```

### Get by ID

```sql
SELECT ... FROM tracked_manga WHERE id = ?;
```

### Update last chapter (after new chapter detected)

```sql
UPDATE tracked_manga
SET last_chapter = ?, last_chapter_num = ?, last_checked = CURRENT_TIMESTAMP
WHERE id = ?;
```

### Update last checked only (no new chapter)

```sql
UPDATE tracked_manga
SET last_checked = CURRENT_TIMESTAMP
WHERE id = ?;
```

### Remove

```sql
DELETE FROM tracked_manga WHERE id = ?;
```

### Log notification

```sql
INSERT INTO notifications (manga_id, chapter, chapter_url)
VALUES (?, ?, ?);
```

---

## Migration Strategy

Fase awal: run all `CREATE TABLE IF NOT EXISTS` on startup — no versioning.

Future: tambah `schema_migrations` table jika schema evolve.

---

## Example Data

```sql
INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num)
VALUES
  ('kiryuu', 'one-piece', 'One Piece', 'https://v6.kiryuu.to/manga/one-piece/', 'Chapter 1100', 1100),
  ('mangaplus', '100026', 'Jujutsu Kaisen', 'https://mangaplus.shueisha.co.jp/titles/100026', 'Chapter 250', 250);
```
