# Panduan Deploy ke Cloudflare

## Arsitektur

```
Cloudflare Pages (mangachapterweb.pages.dev)
├── public/index.html          # Web UI (static)
└── functions/                 # API endpoints (Pages Functions)
    └── api/
        ├── sources.js         # GET /api/sources
        └── manga/
            ├── index.js       # GET/POST /api/manga
            ├── [id].js        # GET/POST/DELETE /api/manga/:id
            ├── search.js      # GET /api/manga/search
            └── check-all.js   # POST /api/manga/check-all

Cloudflare Worker (mangachapter-scheduler)
└── Cron: setiap jam, cek update + kirim Telegram

Cloudflare D1 (mangachapter-db)
└── tracked_manga, notifications tables
```

## Struktur Folder

```
cloudflare/
├── wrangler.toml              # Config untuk Pages
├── scheduler-wrangler.toml    # Config untuk Scheduler Worker
├── package.json
├── .gitignore
├── DEPLOY.md                  # File ini
├── public/                    # Web UI (deploy ke Pages)
│   └── index.html
├── functions/                 # API endpoints
│   └── api/
│       ├── sources.js
│       └── manga/
│           ├── index.js
│           ├── [id].js
│           ├── search.js
│           └── check-all.js
├── src/                       # Shared modules
│   ├── scheduler.js           # Cron Worker
│   ├── telegram.js
│   ├── kiryuu.js
│   └── mangaplus.js
└── migrations/
    └── 0001_init.sql
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

Output akan memberikan `database_id`. Copy ID tersebut ke:
- `wrangler.toml` (untuk Pages)
- `scheduler-wrangler.toml` (untuk Scheduler Worker)

### 3. Jalankan Migrasi Database

**PENTING:** Gunakan `--remote` flag untuk migrasi ke D1 production:

```bash
cd cloudflare
npx wrangler d1 execute mangachapter-db --remote --file=./migrations/0001_init.sql
```

Tanpa `--remote`, migrasi hanya berjalan di database lokal (miniflare) dan tidak akan mempengaruhi D1 production.

### 4. Deploy Web UI + API (Pages)

```bash
cd cloudflare
npx wrangler pages deploy ./public
```

Atau connect ke GitHub repo untuk auto-deploy:
1. Buka Cloudflare Dashboard → Pages
2. Create project → Connect to Git
3. Pilih repo `mangachapter`
4. Set:
   - Build output directory: `cloudflare/public`
   - Root directory: `cloudflare/`

### 5. Set Secrets untuk Pages

```bash
cd cloudflare
npx wrangler pages secret put TELEGRAM_TOKEN
npx wrangler pages secret put TELEGRAM_CHAT_ID
```

### 6. Deploy Scheduler Worker (Cron)

```bash
cd cloudflare
npx wrangler deploy src/scheduler.js --name mangachapter-scheduler --config scheduler-wrangler.toml
```

### 7. Set Secrets untuk Scheduler Worker

```bash
npx wrangler secret put TELEGRAM_TOKEN --name mangachapter-scheduler --config scheduler-wrangler.toml
npx wrangler secret put TELEGRAM_CHAT_ID --name mangachapter-scheduler --config scheduler-wrangler.toml
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

### Error: "D1_ERROR: no such table: tracked_manga"
Migrasi belum dijalankan ke D1 remote. Jalankan:
```bash
npx wrangler d1 execute mangachapter-db --remote --file=./migrations/0001_init.sql
```

### Error: "ENOENT: no such file or directory, scandir 'public'"
Pastikan folder `cloudflare/public/` ada dan berisi `index.html`.

### Error: "Unexpected token '<', not valid JSON"
API request mengembalikan HTML alih-alih JSON. Pastikan:
1. File functions ada di folder `cloudflare/functions/`
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
- Worker: 100,000 request/hari
- D1: 5GB storage