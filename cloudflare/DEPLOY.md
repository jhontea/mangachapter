# Panduan Deploy ke Cloudflare

## Arsitektur

```
Cloudflare Pages (mangachapterweb.pages.dev)
├── public/index.html          # Web UI (static)
└── functions/                 # API endpoints (Pages Functions)
    ├── api/
    │   ├── sources.js         # GET /api/sources
    │   └── manga/
    │       ├── index.js       # GET/POST /api/manga
    │       ├── [id].js        # GET/POST/DELETE /api/manga/:id
    │       ├── search.js      # GET /api/manga/search
    │       └── check-all.js   # POST /api/manga/check-all

Cloudflare D1 (mangachapter-db)
└── tracked_manga, notifications tables

Cloudflare Worker Terpisah (Scheduler)
└── Cron: setiap jam, cek update + kirim Telegram
```

## Prasyarat

1. **Akun Cloudflare** (gratis): https://dash.cloudflare.com/sign-up
2. **Node.js** (v18+): https://nodejs.org/
3. **Wrangler CLI**: `npm install -g wrangler`

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
npx wrangler d1 execute mangachapter-db --file=./migrations/0001_init.sql
```

### 4. Set Secrets (Telegram)

```bash
# Untuk Pages Functions
npx wrangler pages secret put TELEGRAM_TOKEN
npx wrangler pages secret put TELEGRAM_CHAT_ID
```

### 5. Deploy Web UI + API (Pages)

```bash
npx wrangler pages deploy ./public
```

Atau connect ke GitHub repo untuk auto-deploy:
1. Buka Cloudflare Dashboard → Pages
2. Create project → Connect to Git
3. Pilih repo `mangachapter`
4. Set:
   - Build output directory: `cloudflare/public`
   - Root directory: `cloudflare/`

### 6. Deploy Scheduler (Worker Terpisah)

Scheduler berjalan sebagai Worker terpisah dengan cron trigger:

```bash
# Buat file scheduler-worker.js terpisah
npx wrangler deploy src/scheduler.js --name mangachapter-scheduler
```

## API Endpoints

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/sources` | Daftar sumber tersedia |
| GET | `/api/manga` | List semua manga |
| POST | `/api/manga` | Tambah manga baru |
| GET | `/api/manga/:id` | Detail manga |
| DELETE | `/api/manga/:id` | Hapus manga |
| POST | `/api/manga/:id` | Cek update satu manga |
| POST | `/api/manga/check-all` | Cek semua update |
| GET | `/api/manga/search?source=...&query=...` | Cari manga |

## Test API

```bash
# List manga
curl https://mangachapterweb.pages.dev/api/manga

# Tambah manga
curl -X POST https://mangachapterweb.pages.dev/api/manga \
  -H "Content-Type: application/json" \
  -d '{"source":"kiryuu","title":"One Piece","url":"https://v6.kiryuu.to/manga/one-piece/"}'

# Cek update
curl -X POST https://mangachapterweb.pages.dev/api/manga/check-all
```

## Troubleshooting

### Error: "Unexpected token '<', not valid JSON"
API request mengembalikan HTML alih-alih JSON. Pastikan:
1. File functions ada di folder yang benar
2. Deploy ulang: `npx wrangler pages deploy ./public`

### Error: "TELEGRAM_TOKEN not found"
Pastikan sudah set secrets:
```bash
npx wrangler pages secret put TELEGRAM_TOKEN
```

### Error: "D1 database not bound"
Pastikan `database_id` di `wrangler.toml` sudah benar.

## Biaya

Semua layanan **gratis**:
- Pages: Unlimited bandwidth
- Functions: 100,000 request/hari
- D1: 5GB storage