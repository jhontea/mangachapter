# Manga Chapter Notifier

Aplikasi Go untuk memantau chapter manga baru dari beberapa sumber web, lalu mengirim notifikasi email saat ada update.

## Fitur

- Menambah manga ke daftar pantauan (watchlist)
- Menyimpan chapter terakhir yang diketahui per manga
- Pengecekan otomatis berkala (default: setiap 1 jam)
- Notifikasi email saat chapter baru terdeteksi
- Dua sumber awal:
  - [Kiryuu](https://v6.kiryuu.to/) — HTML scraping
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

## Quick Start (setelah implementasi)

```bash
# Clone / masuk ke direktori proyek
cd mangachapter

# Salin dan edit konfigurasi
cp config.yaml.example config.yaml
# Edit SMTP, interval scheduler, dll.

# Build
go build -o manga ./cmd/manga

# Tambah manga
./manga add kiryuu "One Piece" --url https://v6.kiryuu.to/manga/one-piece/
./manga add mangaplus "Jujutsu Kaisen" --id 100026

# Lihat daftar
./manga list

# Cek manual
./manga check

# Jalankan scheduler
./manga run
```

## Struktur Proyek (target)

```
mangachapter/
├── cmd/manga/           # Entry point CLI
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
