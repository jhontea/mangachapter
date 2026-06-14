# Manga Chapter Notifier

Aplikasi Go untuk memantau chapter manga baru dari beberapa sumber web, lalu mengirim notifikasi email saat ada update.

## Fitur

- Menambah manga ke daftar pantauan (watchlist)
- Menyimpan chapter terakhir yang diketahui per manga
- Pengecekan otomatis berkala (default: setiap 1 jam)
- Notifikasi email saat chapter baru terdeteksi
- Dua sumber awal:
  - [Kiryuu](https://v6.kiryuu.to/) — WordPress REST API
  - [Manga Plus](https://mangaplus.shueisha.co.jp/updates) — unofficial API

## Status Proyek

| Fase | Status | Deskripsi |
|------|--------|-----------|
| Fase 0 — Dokumentasi | Selesai | Docs, context, config example |
| Fase 1 — Foundation | Selesai | Go module, config, storage, CLI dasar |
| Fase 2 — Source Adapters | Selesai | Kiryuu + Manga Plus |
| Fase 3 — Checker & Notifier | Selesai | Logika cek + email SMTP |
| Fase 4 — Scheduler | Belum | Cron daemon + polish |

Detail implementasi: [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)

## Quick Start

### Persyaratan

- Go 1.22+
- SMTP server (Gmail App Password, Outlook, atau SMTP custom) — opsional
- Koneksi internet untuk scraping/API

### 1. Setup Konfigurasi

```bash
cp config.yaml.example config.yaml
# Edit config.yaml sesuai kebutuhan (SMTP, dll.)
```

### 2. Build

```bash
go build -o manga ./cmd/manga
```

### 3. Jalankan CLI (Backend)

```bash
# Lihat daftar manga
./manga list

# Cari manga dari sumber
./manga search kiryuu "one piece"
./manga search mangaplus "dandadan"

# Tambah manga ke watchlist
./manga add kiryuu "One Piece" --url https://v6.kiryuu.to/manga/one-piece/
./manga add mangaplus "Dandadan" --id 400007

# Cek update manual
./manga check
./manga check --id 1        # cek satu manga spesifik

# Jalankan scheduler (berjalan terus, cek setiap 1 jam)
./manga run
```

### 4. Jalankan Web UI (Frontend + API)

Web server menyajikan **frontend** sekaligus **REST API** dalam satu proses:

```bash
go run ./cmd/web/
```

Kemudian buka browser: **http://localhost:8080**

Web UI menyediakan:
- 📖 **My Manga** — daftar manga yang dilacak
- 🔍 **Search** — cari manga dari Kiryuu / MangaPlus
- ➕ **Add Manga** — tambah manga ke watchlist
- 🔄 **Check Updates** — cek update manual dari browser

Web API endpoints (REST):
| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/manga` | List semua manga |
| POST | `/api/manga` | Tambah manga baru |
| GET | `/api/manga/{id}` | Detail manga |
| DELETE | `/api/manga/{id}` | Hapus manga |
| POST | `/api/manga/check-all` | Cek semua update |
| GET | `/api/manga/search?source=...&query=...` | Cari manga |
| GET | `/api/sources` | Daftar sumber tersedia |

### 5. Environment Variables

| Variable | Deskripsi |
|----------|-----------|
| `MANGA_SMTP_PASSWORD` | Password SMTP (jangan simpan di config.yaml) |
| `MANGA_SMTP_USERNAME` | Username SMTP override |
| `MANGA_DB_PATH` | Path database SQLite |
| `MANGA_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `PORT` | Port web server (default: 8080) |

## Struktur Proyek

```
mangachapter/
├── cmd/
│   ├── manga/           # CLI entry point
│   └── web/             # Web server (API + frontend)
├── web/
│   └── index.html       # Frontend single-page app
├── internal/
│   ├── config/          # Load YAML + env
│   ├── storage/         # SQLite repository
│   ├── source/          # Adapter per sumber (kiryuu, mangaplus)
│   ├── checker/         # Logika deteksi chapter baru
│   ├── notifier/        # Email SMTP
│   └── scheduler/       # Cron job
├── docs/                # Dokumentasi proyek
├── config.yaml.example
└── go.mod
```

## Dokumentasi

| File | Isi |
|------|-----|
| [docs/CONTEXT.md](docs/CONTEXT.md) | Context lengkap untuk melanjutkan development |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Arsitektur, diagram, interface |
| [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) | Roadmap fase + checklist task |
| [docs/SOURCES.md](docs/SOURCES.md) | Detail teknis Kiryuu & Manga Plus |
| [docs/DATABASE.md](docs/DATABASE.md) | Schema SQLite + query |
| [docs/CONFIG.md](docs/CONFIG.md) | Referensi konfigurasi |
| [AGENTS.md](AGENTS.md) | Panduan singkat untuk AI agent |

## Requirements

- Go 1.22+
- SMTP server (Gmail App Password, Outlook, atau SMTP custom)
- Koneksi internet untuk scraping/API

## Lisensi

Private / personal use. Scraping hanya untuk keperluan pribadi; patuhi ToS masing-masing situs.
