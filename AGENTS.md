# AGENTS.md — Panduan untuk AI Agent

Instruksi untuk melanjutkan development proyek **Manga Chapter Notifier**.

## Baca Terlebih Dahulu

1. [docs/CONTEXT.md](docs/CONTEXT.md) — ringkasan proyek, keputusan desain, interface
2. [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) — checklist fase; lihat task `[ ]` berikutnya
3. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — modul & alur data

## State Saat Ini

- **Fase 0 (Dokumentasi):** selesai
- **Fase 1 (Foundation):** selesai — config, storage, CLI skeleton
- **Fase 2 (Source Adapters):** selesai — Kiryuu + Manga Plus adapters, CLI add/search
- **Fase 3 (Checker & Notifier):** selesai — chapter comparison, checker orchestrator, email SMTP notifier, CLI check
- **Fase 4 (Scheduler & Polish):** selesai — scheduler daemon, graceful shutdown, structured slog
- **Fase 5 (Opsional):** Web UI selesai, parallel checker selesai; sisanya: webhook notifier, Docker, etc.

## Konvensi Kode

- Go 1.22+, module path sesuaikan saat `go mod init`
- Package layout: `cmd/` + `internal/` (no exported packages kecuali perlu)
- Logging: `log/slog` structured
- Context: pass `context.Context` ke semua I/O (HTTP, DB)
- Error: wrap dengan `%w`; pesan user-friendly di CLI layer
- Comments: minimal, hanya untuk non-obvious logic
- Tests: httptest + fixture di `testdata/`; no live HTTP in unit tests

## Prinsip Implementasi

1. **Minimal scope** — implement task fase saat ini saja
2. **Interface source** — Kiryuu dan MangaPlus implement `source.Source`
3. **Baseline on add** — saat `manga add`, fetch & save chapter tanpa email
4. **Sequential check** — rate limit friendly; parallel opsional nanti
5. **Config secrets** — password via env, jangan hardcode

## Struktur Target

```
cmd/manga/main.go
internal/config/
internal/storage/
internal/source/     # kiryuu.go, mangaplus.go, source.go
internal/checker/
internal/notifier/
internal/scheduler/
```

## Task Priority

Ikuti urutan di [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md):

```
Fase 1 → Fase 2 → Fase 3 → Fase 4
```

Jangan loncat fase kecuali user minta.

## Source-Specific Notes

- **Kiryuu:** inspect HTML dulu, update selector di [docs/SOURCES.md](docs/SOURCES.md)
- **MangaPlus:** gunakan `github.com/luevano/mangoplus`; API unofficial & unstable
- **MangaPlus limitation:** hanya first + latest chapter visible

## Database

Schema di [docs/DATABASE.md](docs/DATABASE.md). SQLite pure Go preferred: `modernc.org/sqlite`.

## Config

Reference: [docs/CONFIG.md](docs/CONFIG.md). Example: [config.yaml.example](config.yaml.example).

## CLI Commands (harus diimplement)

```
manga add <source> <title> [--url URL | --id ID]
manga list
manga remove <id>
manga search <source> <query>
manga check [--id ID]
manga run
```

## Dependencies (approved)

```
github.com/PuerkitoBio/goquery
github.com/luevano/mangoplus
github.com/robfig/cron/v3
modernc.org/sqlite
github.com/spf13/cobra
gopkg.in/yaml.v3
```

## Jangan

- Commit `config.yaml` dengan credentials
- Hit live manga sites di unit test default
- Over-engineer (Web UI, Docker) kecuali user minta
- Buat commit kecuali user explicitly request

## Saat Menyelesaikan Task

1. Update checklist di `docs/IMPLEMENTATION.md` (`[ ]` → `[x]`)
2. Update status table di `README.md` jika fase selesai
3. Update `docs/SOURCES.md` jika selector/API diverifikasi
4. Run `go test ./...` dan `go build ./cmd/manga`

## Prompt Lanjutan (contoh untuk user)

```
"Lanjutkan Fase 1 dari docs/IMPLEMENTATION.md"
"Implement Kiryuu adapter — inspect HTML dulu"
"Wire manga check command dengan email notifier"
```
