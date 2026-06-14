# Project Context — Manga Chapter Notifier

> **Dokumen ini** adalah context utama untuk melanjutkan development. Baca file ini terlebih dahulu sebelum menulis kode.

## Ringkasan

Aplikasi **Go CLI + scheduler daemon** yang:

1. Menyimpan daftar manga yang ingin dipantau
2. Mengecek chapter terbaru dari 2 sumber web
3. Membandingkan dengan chapter terakhir yang tersimpan
4. Mengirim email jika ada chapter baru

## Keputusan Desain (sudah disepakati)

| Keputusan | Pilihan | Alasan |
|-----------|---------|--------|
| Bahasa | Go 1.22+ | Static binary, concurrency, HTTP stdlib solid |
| Storage | SQLite (file lokal) | Single-user, tanpa dependency DB server |
| UI | CLI only (fase awal) | Simple; Web UI opsional nanti |
| Scheduler | `robfig/cron/v3` | Cron expression fleksibel |
| Config | YAML + env override | Mudah edit, secrets via env |
| Email | SMTP (stdlib / jordan-wright/email) | Universal, Gmail/Outlook/custom |
| Baseline saat add | Simpan chapter saat ini **tanpa** email | Hindari spam notifikasi chapter lama |

## Sumber Data

### Kiryuu (`v6.kiryuu.to`)

- **Metode:** HTML scraping (`goquery`)
- **Identifikasi:** slug dari URL, contoh `one-piece` dari `/manga/one-piece/`
- **Search:** `GET /?s={query}`
- **Latest chapter:** scrape halaman detail manga
- **Risiko:** HTML structure bisa berubah; perlu inspect saat implementasi

### Manga Plus (`mangaplus.shueisha.co.jp`)

- **Metode:** Unofficial API
- **Library Go:** `github.com/luevano/mangoplus` (opsional; bisa HTTP manual)
- **Identifikasi:** Title ID numerik, contoh `100026`
- **Endpoint updates:** halaman `/updates` memuat data via API internal
- **Limitation penting:** Manga Plus hanya menampilkan chapter **pertama** dan **terakhir** — chapter tengah tidak muncul di API/site

## Alur Bisnis

### Menambah manga

```
User: manga add kiryuu "One Piece" --url https://v6.kiryuu.to/manga/one-piece/
  → Fetch latest chapter dari source
  → Simpan ke DB (title, url, source_id, last_chapter, last_chapter_num)
  → TIDAK kirim email (baseline)
```

### Pengecekan scheduler

```
Setiap interval (default 1 jam):
  → Ambil semua tracked_manga dari DB
  → Untuk setiap manga:
      → Fetch latest chapter dari source adapter
      → Jika latest.num > stored.last_chapter_num:
          → Kirim email notifikasi
          → Update last_chapter + last_chapter_num + last_checked
      → Else:
          → Update last_checked saja
```

### Perbandingan chapter

- Parse nomor ke `float64` untuk compare numerik
- Handle format: `"123"`, `"Chapter 123"`, `"123.5"`
- Chapter non-numerik ("Special", "Side Story") — bandingkan string equality; kirim notif jika berbeda

## Interface Source (kontrak)

Semua adapter harus implement:

```go
type Source interface {
    Name() string
    Search(ctx context.Context, query string) ([]MangaInfo, error)
    GetLatestChapter(ctx context.Context, ref MangaRef) (ChapterInfo, error)
}

type MangaRef struct {
    SourceID string // slug atau title ID
    URL      string // full URL halaman manga
    Title    string // untuk logging
}

type MangaInfo struct {
    Title    string
    URL      string
    SourceID string
}

type ChapterInfo struct {
    Number   string  // display: "Chapter 123"
    Title    string  // judul chapter jika ada
    URL      string  // link baca
    NumValue float64 // untuk perbandingan
}
```

## CLI Commands (target)

| Command | Args | Deskripsi |
|---------|------|-----------|
| `add` | `<source> <title> [--url URL \| --id ID]` | Tambah manga |
| `list` | | Tampilkan watchlist |
| `remove` | `<id>` | Hapus dari watchlist |
| `search` | `<source> <query>` | Cari manga di source |
| `check` | `[--id ID]` | Cek manual |
| `run` | | Jalankan scheduler daemon |

## Dependencies (rencana)

```
github.com/PuerkitoBio/goquery      # Kiryuu scraping
github.com/luevano/mangoplus        # Manga Plus API (opsional)
github.com/robfig/cron/v3           # Scheduler
modernc.org/sqlite                  # SQLite pure Go (no CGO)
github.com/spf13/cobra              # CLI
gopkg.in/yaml.v3                    # Config
```

Alternatif SQLite: `github.com/mattn/go-sqlite3` (butuh CGO).

## Yang Belum Dibuat

Workspace saat ini hanya berisi dokumentasi. Belum ada:

- [ ] `go.mod`
- [ ] Kode Go apapun
- [ ] `config.yaml` (hanya `.example`)
- [ ] Database / migrations

**Langkah pertama implementasi:** Fase 1 di [IMPLEMENTATION.md](IMPLEMENTATION.md).

## Pertanyaan Terbuka (opsional, bisa default)

| Pertanyaan | Default yang dipakai |
|------------|---------------------|
| SMTP provider? | Gmail (App Password) — user edit config |
| Deployment? | `manga run` manual / Windows Task Scheduler |
| Prioritas source? | Kiryuu dulu, lalu Manga Plus |
| Logging? | `log/slog` structured |

## Referensi Eksternal

- Manga Plus Go lib: https://github.com/luevano/mangoplus
- Kiryuu scraper (Node.js, referensi selector): npm `@boboiboyturuu_nih/kiryuu-scraper`
- Proyek serupa: Mantium (multi-source manga tracker)

## File Terkait

- [ARCHITECTURE.md](ARCHITECTURE.md) — diagram & modul
- [IMPLEMENTATION.md](IMPLEMENTATION.md) — checklist task per fase
- [SOURCES.md](SOURCES.md) — detail scraping/API
- [DATABASE.md](DATABASE.md) — schema
- [CONFIG.md](CONFIG.md) — config reference
- [../AGENTS.md](../AGENTS.md) — panduan untuk AI agent
