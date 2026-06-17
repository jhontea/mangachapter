# Deploy Manga Chapter Notifier ke Fly.io + Supabase

Panduan lengkap deploy backend Go ke **Fly.io** dan database PostgreSQL ke **Supabase**.

## Prasyarat

- [ ] Akun [Fly.io](https://fly.io) (gratis, credit card untuk verifikasi)
- [ ] Akun [Supabase](https://supabase.com) (gratis)
- [ ] [Fly CLI](https://fly.io/docs/hands-on/install-flyctl/) terinstall
- [ ] [Docker](https://docs.docker.com/get-docker/) terinstall
- [ ] Git terinstall
- [ ] Telegram Bot Token + Chat ID (opsional)

---

## Langkah 1: Setup Supabase (Database)

### 1.1 Buat Project Supabase

1. Buka https://supabase.com → Login
2. Klik **"New Project"**
3. Isi:
   - **Organization**: pilih atau buat baru
   - **Project Name**: `mangachapter`
   - **Database Password**: buat password yang kuat (simpan!)
   - **Region**: pilih terdekat (misal: `Southeast Asia (Singapore)`)
4. Klik **"Create new project"**
5. Tunggu ~2 menit sampai project selesai dibuat

### 1.2 Buat Tabel Database

1. Buka **SQL Editor** di dashboard Supabase (menu kiri)
2. Klik **"New query"**
3. Paste SQL berikut:

```sql
-- Manga Chapter Notifier - Database Schema

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
```

4. Klik **"Run"** (tombol biru di pojok kanan bawah)
5. Pastikan tidak ada error

### 1.3 Ambil Connection String

1. Buka **Settings** → **Database** (menu kiri)
2. Scroll ke bagian **Connection string**
3. Pilih tab **URI**
4. Copy connection string, format seperti:
   ```
   postgresql://postgres.[ref]:[YOUR-PASSWORD]@aws-0-[region].pooler.supabase.com:6543/postgres
   ```
5. **Simpan connection string ini** — Anda akan butuh nanti

> **Penting**: Gunakan **port 6543** (Transaction mode) untuk production, bukan 5432.

---

## Langkah 2: Setup Fly.io (Backend Go)

### 2.1 Install Fly CLI

**Windows:**
```powershell
# Buka PowerShell sebagai Administrator
iwr https://fly.io/install.ps1 -useb | iex
```

**Mac/Linux:**
```bash
curl -L https://fly.io/install.sh | sh
```

### 2.2 Login Fly.io

```bash
fly auth login
```

Browser akan terbuka → login dengan akun Fly.io Anda.

### 2.3 Inisialisasi Project

```bash
# Pastikan Anda di direktori project
cd C:\Users\PC\go\src\project\mangachapter

# Init project (pilih nama app, region, dan Dockerfile)
fly launch
```

Fly akan menanyakan:
- **App Name**: `mangachapter` (atau nama lain)
- **Region**: pilih `sin` (Singapore) atau terdekat
- **Detected Dockerfile**: `Yes`
- **Setup PostgreSQL**: `No` (kita pakai Supabase)
- **Setup Redis**: `No`
- **Deploy now**: `No` (kita set env vars dulu)

### 2.4 Set Environment Variables

```bash
# Database (dari Supabase)
fly secrets set DATABASE_URL="postgresql://postgres.[ref]:[PASSWORD]@aws-0-singapore.pooler.supabase.com:6543/postgres"

# Telegram (opsional)
fly secrets set TELEGRAM_TOKEN="your_bot_token"
fly secrets set TELEGRAM_CHAT_ID="your_chat_id"

# Port (opsional, default 8080)
fly secrets set PORT="8080"
```

### 2.5 Deploy

```bash
fly deploy
```

Tunggu sampai selesai (~2-5 menit). Fly akan:
1. Build Docker image
2. Push ke registry
3. Deploy ke edge server

### 2.6 Cek Status

```bash
# Lihat status
fly status

# Lihat logs
fly logs

# Buka di browser
fly open
```

### 2.7 Cek Aplikasi

Buka browser: `https://mangachapter.fly.dev`

Anda harus melihat Web UI Manga Chapter Notifier.

---

## Langkah 3: Verifikasi

### 3.1 Test API

```bash
# List manga (harusnya kosong)
curl https://mangachapter.fly.dev/api/manga

# List sources
curl https://mangachapter.fly.dev/api/sources

# Cari manga
curl "https://mangachapter.fly.dev/api/manga/search?source=kiryuu&query=one+piece"

# Tambah manga
curl -X POST https://mangachapter.fly.dev/api/manga \
  -H "Content-Type: application/json" \
  -d '{"source":"kiryuu","title":"One Piece","url":"https://v6.kiryuu.to/manga/one-piece/","source_id":"one-piece"}'

# Cek update
curl -X POST https://mangachapter.fly.dev/api/manga/check-all
```

### 3.2 Cek Database di Supabase

1. Buka dashboard Supabase
2. Buka **Table Editor** (menu kiri)
3. Pilih tabel `tracked_manga`
4. Anda harus melihat data manga yang baru ditambahkan

---

## Langkah 4: Konfigurasi Tambahan

### 4.1 Custom Domain (Opsional)

```bash
# Tambah custom domain
fly certs add yourdomain.com

# Cek status sertifikat
fly certs list
```

### 4.2 Scaling (Opsional)

```bash
# Tambah instance
fly scale count 2

# Tambah memory
fly scale memory 512
```

### 4.3 Update Aplikasi

```bash
# Deploy ulang setelah perubahan kode
fly deploy
```

---

## Troubleshooting

### Error: "connection refused"
- Pastikan `DATABASE_URL` benar
- Pastikan Supabase project aktif
- Cek logs: `fly logs`

### Error: "authentication failed"
- Pastikan password database benar
- Cek connection string di Supabase Settings → Database

### Error: "port already in use"
- Pastikan tidak ada proses lain di port 8080
- Atau set PORT berbeda: `fly secrets set PORT="3000"`

### Error: "build failed"
- Pastikan Docker terinstall
- Cek Dockerfile: `fly logs`

### Error: "no such host"
- Pastikan koneksi internet aktif
- Cek DNS: `nslookup aws-0-singapore.pooler.supabase.com`

---

## Biaya

### Supabase (Gratis)
- 500 MB database
- 1 GB bandwidth
- 50,000 rows
- Cukup untuk personal use

### Fly.io (Gratis)
- 3 shared-cpu-1x (256 MB RAM)
- 160 GB bandwidth/bulan
- 3 GB persistent storage
- Cukup untuk personal use

**Total: $0/bulan** untuk personal use.

---

## Arsitektur Akhir

```
┌─────────────────────────────────────────────────┐
│              Fly.io (Edge Server)                │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │         Go Binary (cmd/web)                │  │
│  │                                            │  │
│  │  ┌──────────┐  ┌──────────────────────┐   │  │
│  │  │ REST API │  │   Web UI (index.html)│   │  │
│  │  │ Handlers │  │                      │   │  │
│  │  └─────┬────┘  └──────────────────────┘   │  │
│  │        │                                   │  │
│  │  ┌─────▼──────────────────────────────┐   │  │
│  │  │     Checker + Scheduler            │   │  │
│  │  │     (cron setiap 1 jam)            │   │  │
│  │  └─────┬──────────┬───────────────────┘   │  │
│  │        │          │                       │  │
│  │  ┌─────▼───┐  ┌───▼────────────────┐     │  │
│  │  │ Sources  │  │   Notifier         │     │  │
│  │  │ Kiryuu   │  │  (Telegram)        │     │  │
│  │  │ MangaPlus│  │                    │     │  │
│  │  └─────────┘  └────────────────────┘     │  │
│  └────────────────────────────────────────────┘  │
│                   │                              │
│            pgx driver (Go)                       │
└───────────────────┼──────────────────────────────┘
                    │
┌───────────────────▼──────────────────────────────┐
│         Supabase (PostgreSQL)                     │
│    - mangachapter DB                              │
│    - Table: tracked_manga                         │
│    - Table: notifications                         │
│    - Auto-backup, dashboard, etc.                 │
└──────────────────────────────────────────────────┘
```

---

## Checklist Deploy

- [ ] Buat akun Supabase
- [ ] Buat project Supabase
- [ ] Jalankan SQL schema di Supabase SQL Editor
- [ ] Copy connection string
- [ ] Install Fly CLI
- [ ] Login Fly.io
- [ ] `fly launch` (init project)
- [ ] `fly secrets set DATABASE_URL="..."`
- [ ] `fly secrets set TELEGRAM_TOKEN="..."` (opsional)
- [ ] `fly deploy`
- [ ] `fly open` (buka di browser)
- [ ] Test API endpoints
- [ ] Cek data di Supabase Table Editor