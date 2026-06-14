# Arsitektur

## Diagram Komponen

```
┌─────────────────────────────────────────────────────────────┐
│                        cmd/manga                             │
│                     (Cobra CLI)                              │
└──────────┬──────────────────────────────────────────────────┘
           │
     ┌─────┴─────┬─────────────┬──────────────┐
     ▼           ▼             ▼              ▼
┌─────────┐ ┌─────────┐ ┌──────────┐ ┌────────────┐
│ config  │ │ storage │ │ checker  │ │ scheduler  │
└─────────┘ └────┬────┘ └────┬─────┘ └─────┬──────┘
                 │           │              │
                 │     ┌─────┴─────┐        │
                 │     ▼           ▼        │
                 │ ┌─────────┐ ┌─────────┐  │
                 │ │ source  │ │notifier │  │
                 │ │ adapters│ │ (email) │  │
                 │ └────┬────┘ └─────────┘  │
                 │      │                   │
                 ▼      ▼                   ▼
            ┌────────┐ ┌──────────┐    cron tick
            │ SQLite │ │ Kiryuu   │    (1h default)
            └────────┘ │ MangaPlus│
                       └──────────┘
```

## Modul

### `internal/config`

- Load `config.yaml` dari working directory atau path via flag/env
- Env override: `MANGA_SMTP_PASSWORD`, `MANGA_CONFIG_PATH`, dll.
- Validasi: interval valid, SMTP host/port wajib jika notifier aktif

### `internal/storage`

- SQLite via `database/sql`
- Migration sederhana (CREATE IF NOT EXISTS) saat startup
- Repository interface:

```go
type Repository interface {
    AddManga(ctx context.Context, m *TrackedManga) error
    RemoveManga(ctx context.Context, id int64) error
    ListManga(ctx context.Context) ([]TrackedManga, error)
    GetManga(ctx context.Context, id int64) (*TrackedManga, error)
    UpdateLastChapter(ctx context.Context, id int64, ch ChapterInfo) error
    UpdateLastChecked(ctx context.Context, id int64) error
    LogNotification(ctx context.Context, mangaID int64, chapter string) error
}
```

### `internal/source`

- Registry: `map[string]Source` keyed by source name
- Factory dari config
- HTTP client shared dengan timeout, User-Agent, rate limiter

### `internal/checker`

- Orchestrator: loop semua manga, panggil source, compare, notify, persist
- Retry: 2-3x dengan backoff untuk network error
- Context timeout per manga (mis. 30s)

### `internal/notifier`

- Interface:

```go
type Notifier interface {
    SendNewChapter(ctx context.Context, n NewChapterNotification) error
}

type NewChapterNotification struct {
    MangaTitle   string
    Source       string
    Chapter      string
    ChapterURL   string
    PreviousChapter string
}
```

### `internal/scheduler`

- Wrap `robfig/cron`
- Parse interval dari config (`1h` atau cron `0 * * * *`)
- Graceful shutdown on SIGINT/SIGTERM

## Sequence: Scheduled Check

```
Scheduler          Checker           Source           Storage          Notifier
    |                 |                 |                 |                 |
    |-- tick -------->|                 |                 |                 |
    |                 |-- ListManga --->|                 |                 |
    |                 |<-- manga[] -----|                 |                 |
    |                 |                 |                 |                 |
    |                 |-- GetLatest --->|                 |                 |
    |                 |<-- chapter -----|                 |                 |
    |                 |                 |                 |                 |
    |                 | [if new chapter]|                 |                 |
    |                 |-----------------------------------|-- Send -------->|
    |                 |-- UpdateLastChapter ------------->|                 |
    |                 |                 |                 |                 |
    |                 | [else]          |                 |                 |
    |                 |-- UpdateLastChecked ------------->|                 |
```

## Error Handling

| Error | Perilaku |
|-------|----------|
| Network timeout | Retry 2x, log warning, skip manga ini |
| HTML parse fail (Kiryuu) | Log error, skip; mungkin structure berubah |
| API error (MangaPlus) | Retry, log |
| SMTP fail | Log error, chapter tetap di-update DB (hindari duplicate notif) atau rollback — **pilih: update DB dulu, log notif gagal** |
| Duplicate add | UNIQUE constraint → error ke user |

## Concurrency

- Checker: sequential per manga (rate limit friendly)
- Bisa parallel nanti dengan worker pool + semaphore (max 3 concurrent)
- Fase awal: sequential cukup

## Logging

- Package: `log/slog`
- Level: INFO default, DEBUG via flag
- Fields: `manga_id`, `source`, `title`, `chapter`, `duration`
