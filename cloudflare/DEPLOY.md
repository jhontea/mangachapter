# Panduan Deploy ke Cloudflare

## Prasyarat

1. **Akun Cloudflare** (gratis): https://dash.cloudflare.com/sign-up
2. **Node.js** (v18+): https://nodejs.org/
3. **Wrangler CLI**: `npm install -g wrangler`
4. **GitHub repo** (opsional, untuk CI/CD)

## Langkah Deploy

### 1. Login ke Cloudflare

```bash
cd cloudflare
npx wrangler login
```

### 2. Buat Database D1

```bash
npx wrangler d1 create mangachapter-db
```

Output akan memberikan `database_id`. Copy ID tersebut ke `wrangler.toml`:

```toml
[[d1_databases]]
binding = "DB"
database_name = "mangachapter-db"
database_id = "PASTE_DATABASE_ID_DISINI"
```

### 3. Jalankan Migrasi Database

```bash
# Untuk remote (production)
npx wrangler d1 execute mangachapter-db --file=./migrations/0001_init.sql

# Untuk local development
npx wrangler d1 execute mangachapter-db --local --file=./migrations/0001_init.sql
```

### 4. Set Secrets (Telegram)

```bash
npx wrangler secret put TELEGRAM_TOKEN
# Masukkan token bot Telegram kamu

npx wrangler secret put TELEGRAM_CHAT_ID
# Masukkan chat ID Telegram kamu
```

### 5. Test Lokal

```bash
npx wrangler dev
```

Buka browser: http://localhost:8787

### 6. Deploy Worker (API + Scheduler)

```bash
npx wrangler deploy
```

Worker akan di-deploy ke: `mangachapter.<subdomain>.workers.dev`

### 7. Deploy Web UI (Pages)

```bash
npx wrangler pages deploy ./public
```

Atau connect ke GitHub repo untuk auto-deploy:
1. Buka Cloudflare Dashboard → Pages
2. Create project → Connect to Git
3. Pilih repo `mangachapter`
4. Set build output directory: `cloudflare/public`

## Struktur File

```
cloudflare/
├── wrangler.toml          # Konfigurasi Cloudflare
├── package.json           # Dependencies
├── DEPLOY.md              # Panduan ini
├── migrations/
│   └── 0001_init.sql      # Schema database D1
├── src/
│   ├── api.js             # REST API + Cron handler
│   ├── telegram.js        # Telegram notifier
│   ├── kiryuu.js          # Kiryuu source adapter
│   └── mangaplus.js       # MangaPlus source adapter
└── public/
    └── index.html         # Web UI (copy dari web/)
```

## API Endpoints

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/sources` | Daftar sumber tersedia |
| GET | `/api/manga` | List semua manga |
| POST | `/api/manga` | Tambah manga baru |
| GET | `/api/manga/{id}` | Detail manga |
| DELETE | `/api/manga/{id}` | Hapus manga |
| POST | `/api/manga/{id}` | Cek update satu manga |
| POST | `/api/manga/check-all` | Cek semua update |
| GET | `/api/manga/search?source=...&query=...` | Cari manga |

## Cron Scheduler

Scheduler berjalan otomatis setiap jam (sesuai `wrangler.toml`):

```toml
[triggers]
crons = ["0 * * * *"]  # Setiap jam
```

Untuk mengubah interval, edit cron expression:
- `*/30 * * * *` — setiap 30 menit
- `0 */2 * * *` — setiap 2 jam
- `0 8,20 * * *` — jam 8 pagi dan 8 malam

## Troubleshooting

### Error: "database_id not found"
Pastikan `database_id` di `wrangler.toml` sudah benar.

### Error: "TELEGRAM_TOKEN not found"
Pastikan sudah menjalankan `wrangler secret put TELEGRAM_TOKEN`.

### Error: "D1 database not bound"
Pastikan `[[d1_databases]]` di `wrangler.toml` sudah benar.

### Test API secara manual
```bash
# List manga
curl https://mangachapter.<subdomain>.workers.dev/api/manga

# Tambah manga
curl -X POST https://mangachapter.<subdomain>.workers.dev/api/manga \
  -H "Content-Type: application/json" \
  -d '{"source":"kiryuu","title":"One Piece","url":"https://v6.kiryuu.to/manga/one-piece/"}'

# Cek update
curl -X POST https://mangachapter.<subdomain>.workers.dev/api/manga/check-all
```

## Biaya

Semua layanan yang digunakan **gratis**:

| Layanan | Limit Gratis |
|---------|--------------|
| Workers | 100,000 request/hari |
| D1 | 5GB storage, 5M baris baca/hari |
| Pages | Unlimited bandwidth |
| Cron Triggers | 3 schedules per worker |

## Update Deployment

Setelah membuat perubahan:

```bash
# Update Worker (API + Scheduler)
npx wrangler deploy

# Update Pages (Web UI)
npx wrangler pages deploy ./public