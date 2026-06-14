# Implementation Roadmap

Checklist task per fase. Centang `[x]` saat selesai.

---

## Fase 0 — Dokumentasi ✅

- [x] README.md
- [x] docs/CONTEXT.md
- [x] docs/ARCHITECTURE.md
- [x] docs/IMPLEMENTATION.md (file ini)
- [x] docs/SOURCES.md
- [x] docs/DATABASE.md
- [x] docs/CONFIG.md
- [x] config.yaml.example
- [x] AGENTS.md

---

## Fase 1 — Foundation

### 1.1 Init project

- [x] `go mod init project/mangachapter`
- [x] Buat struktur folder sesuai ARCHITECTURE.md
- [x] `.gitignore` (data/, config.yaml, *.db, bin/)

### 1.2 Config

- [x] `internal/config/config.go` — struct + Load()
- [x] Parse YAML, env override
- [x] Unit test load config

### 1.3 Storage

- [x] `internal/storage/models.go` — TrackedManga, Notification
- [x] `internal/storage/sqlite.go` — Open, Migrate, Repository impl
- [x] Migration SQL dari DATABASE.md
- [x] Unit test CRUD dengan `:memory:` SQLite

### 1.4 CLI skeleton

- [x] `cmd/manga/main.go`
- [x] Cobra root + subcommands stub: add, list, remove, search, check, run
- [x] Global flags: `--config`, `--debug`

### 1.5 Deliverable Fase 1

```bash
go build ./cmd/manga
./manga list   # returns empty list, no panic
```

---

## Fase 2 — Source Adapters ✅

### 2.1 Source interface

- [x] `internal/source/source.go` — interface + types + registry
- [x] Shared HTTP client helper (timeout, User-Agent, rate limit)

### 2.2 Kiryuu adapter

- [x] Inspect HTML v6.kiryuu.to (search page, manga detail, chapter list)
- [x] Document selectors di SOURCES.md
- [x] `internal/source/kiryuu.go`:
  - [x] Search()
  - [x] GetLatestChapter()
  - [x] Parse chapter number helper
- [x] Test dengan httptest mock HTML

### 2.3 Manga Plus adapter

- [x] Pilih: library `luevano/mangoplus` vs HTTP manual
- [x] `internal/source/mangaplus.go`:
  - [x] Search() (jika API support)
  - [x] GetLatestChapter() by title ID
- [x] Test dengan mock

### 2.4 Wire CLI add/search

- [x] `manga add kiryuu ...` — fetch baseline chapter, save DB
- [x] `manga add mangaplus ... --id N`
- [x] `manga search <source> <query>`

### 2.5 Deliverable Fase 2

```bash
./manga add kiryuu "Test" --url https://v6.kiryuu.to/manga/...
./manga list   # shows manga with last chapter
./manga search kiryuu "one piece"
```

---

## Fase 3 — Checker & Notifier ✅

### 3.1 Chapter comparison

- [x] `internal/checker/compare.go` — parse & compare chapter numbers
- [x] Handle edge cases: empty baseline, special chapters, decimals

### 3.2 Checker

- [x] `internal/checker/checker.go` — CheckAll(), CheckOne(id)
- [x] Integrate source registry + storage
- [x] Retry logic

### 3.3 Email notifier

- [x] `internal/notifier/notifier.go` — Notifier interface
- [x] `internal/notifier/email.go` — SMTP send
- [x] Plain text template
- [x] Test dengan mock SMTP atau integration skip

### 3.4 Wire CLI check

- [x] `manga check` — run checker once, print results
- [x] `manga check --id 1` — single manga

### 3.5 Deliverable Fase 3

```bash
./manga check
# logs: checked N manga, M new chapters, emails sent
```

---

## Fase 4 — Scheduler & Polish

### 4.1 Scheduler

- [ ] `internal/scheduler/scheduler.go`
- [ ] Parse interval from config
- [ ] `manga run` — block, handle SIGINT

### 4.2 Logging & UX

- [ ] Structured slog throughout
- [ ] `--debug` flag
- [ ] Friendly error messages di CLI

### 4.3 README update

- [ ] Update status table di README
- [ ] Troubleshooting section

### 4.4 Deliverable Fase 4

```bash
./manga run
# runs indefinitely, checks every 1h
```

---

## Fase 5 — Opsional (future)

- [ ] Web UI (embed atau separate)
- [ ] Discord/Telegram webhook notifier
- [ ] Docker + docker-compose
- [ ] Windows service wrapper
- [ ] Parallel checker dengan rate limit
- [ ] Source tambahan (MangaDex, dll.)

---

## Urutan file yang dibuat (suggested)

```
1. go.mod
2. internal/config/config.go
3. internal/storage/models.go
4. internal/storage/sqlite.go
5. cmd/manga/main.go
6. internal/source/source.go
7. internal/source/kiryuu.go
8. internal/source/mangaplus.go
9. internal/checker/checker.go
10. internal/notifier/email.go
11. internal/scheduler/scheduler.go
12. cmd/manga/commands/*.go (optional split)
```

---

## Definition of Done (whole project)

- [ ] Add/list/remove manga via CLI
- [ ] Baseline chapter saved on add (no email)
- [ ] Manual check detects new chapter
- [ ] Email sent on new chapter
- [ ] Scheduler runs hourly checks
- [ ] Config via YAML + env
- [ ] README accurate
