# Manga Chapter Notifier

Aplikasi Go untuk memantau chapter manga baru dari beberapa sumber web, lalu mengirim notifikasi (Telegram atau email) saat ada update.

## Fitur

- Menambah manga ke daftar pantauan
- Menyimpan chapter terakhir yang diketahui per manga
- Pengecekan otomatis berkala (default: setiap 1 jam)
- Notifikasi via **Telebot Bot** atau **email SMTP**
- Dua sumber awal:
  - [Kiryuu](https://v6.kiryuu.to/) — WordPress REST API
  - [Manga Plus](https://mangaplus.shueisha.co.jp/updates) — unofficial API
- Web UI untuk manajemen manga via browser

## Status Proyek

| Fase | Status | Deskripsi |
|------|--------|-----------|
| Fase 0 — Dokumentasi | ✅ Selesai | Docs, context, config example |
| Fase 1 — Foundation | ✅ Selesai | Go module, config, storage, CLI dasar |
| Fase 2 — Source Adapters | ✅ Selesai | Kiryuu + Manga Plus |
| Fase 3 — Checker & Notifier | ✅ Selesai | Logika cek + email SMTP + Telegram |
| Fase 4 — Scheduler | ✅ Selesai | Cron daemon + polish |
| Fase 5 — Web UI & Telegram | ✅ Selesai | Web UI + Telegram notifier + parallel checker |

Detail implementasi: [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)

## Quick Start

### Persyaratan

- Go 1.22+
- Telegram Bot Token + Chat ID (direkomendasikan) atau SMTP server
- Koneksi internet untuk scraping/API

### 1. Setup Konfigurasi

```bash
cp config.yaml.example config.yaml
# Edit config.yaml sesuai kebutuhan
```

Untuk Telegram, buat file `.env`:
```env
TELEGRAM_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here
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

# Tambah manga ke daftar pantau
./manga add kiryuu "One Piece" --url https://v6.kiryuu.to/manga/one-piece/
./manga add mangaplus "Dandadan" --id 400007

# Hapus manga dari daftar pantau
./manga remove 1

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
- 📖 **Daftar Manga** — lihat semua manga yang dilacak
- 🔍 **Cari** — cari manga dari Kiryuu / MangaPlus
- ➕ **Tambah Manga** — tambah manga ke daftar pantau
- 🔄 **Cek Update** — cek update manual dari browser

### 5. API Endpoints (REST)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/manga` | List semua manga |
| POST | `/api/manga` | Tambah manga baru |
| GET | `/api/manga/{id}` | Detail manga |
| DELETE | `/api/manga/{id}` | Hapus manga |
| POST | `/api/manga/{id}` | Cek update satu manga |
| POST | `/api/manga/check-all` | Cek semua update |
| GET | `/api/manga/search?source=...&query=...` | Cari manga |
| GET | `/api/sources` | Daftar sumber tersedia |

### 6. Environment Variables

| Variable | Deskripsi |
|----------|-----------|
| `TELEGRAM_TOKEN` | Token bot Telegram (wajib jika telegram aktif) |
| `TELEGRAM_CHAT_ID` | Chat ID Telegram (wajib jika telegram aktif) |
| `MANGA_TELEGRAM_TOKEN` | Alternatif token Telegram |
| `MANGA_TELEGRAM_CHAT_ID` | Alternatif chat ID Telegram |
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
│   │   ├── main.go      # Root command
│   │   ├── app.go       # App initialization
│   │   ├── add.go       # Tambah manga
│   │   ├── list.go      # Daftar manga
│   │   ├── remove.go    # Hapus manga
│   │   ├── search.go    # Cari manga
│   │   ├── check.go     # Cek update
│   │   └── run.go       # Jalankan scheduler
│   └── web/             # Web server (API + frontend)
├── web/
│   └── index.html       # Frontend single-page app
├── internal/
│   ├── config/          # Load YAML + env
│   ├── storage/         # SQLite repository
│   ├── source/          # Adapter per sumber (kiryuu, mangaplus)
│   ├── checker/         # Logika deteksi chapter baru
│   ├── notifier/        # Telegram + email SMTP
│   └── scheduler/       # Cron job
├── docs/                # Dokumentasi proyek
├── config.yaml.example  # Contoh konfigurasi
├── .env.example         # Contoh environment variables
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

## Dependencies

```
github.com/PuerkitoBio/goquery      # HTML parsing
github.com/luevano/mangoplus         # Manga Plus API
github.com/robfig/cron/v3            # Cron scheduler
modernc.org/sqlite                   # SQLite pure Go
github.com/spf13/cobra               # CLI framework
gopkg.in/yaml.v3                     # YAML parsing
golang.org/x/sync                    # Concurrent utilities
github.com/joho/godotenv             # Environment variables
```

## Konfigurasi

### Telegram (Direkomendasikan)

```yaml
telegram:
  enabled: true
```

Token dan chat ID disimpan di file `.env`:
```env
TELEGRAM_TOKEN=123456:ABC-DEF
TELEGRAM_CHAT_ID=-1001234567890
```

### Email (Alternatif)

```yaml
email:
  enabled: true
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: "your@gmail.com"
  from: "your@gmail.com"
  to:
    - "your@gmail.com"
```

## Lisensi

Private / personal use. Scraping hanya untuk keperluan pribadi; patuhi ToS masing-masing situs.