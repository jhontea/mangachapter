# Deploy ke VPS menggunakan Docker

## Prasyarat

- VPS dengan OS Ubuntu 22.04 / Debian 12
- Docker & Docker Compose terinstall
- Nginx terinstall
- Domain yang sudah diarahkan ke IP VPS

## 1. Instalasi Docker (jika belum)

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker
```

## 2. Clone repo ke VPS

```bash
git clone https://github.com/jhontea/mangachapter.git
cd mangachapter
```

## 3. Setup konfigurasi

```bash
# Copy dan edit env
cp .env.example .env
nano .env   # isi TELEGRAM_TOKEN dan TELEGRAM_CHAT_ID

# Copy dan edit config
cp config.yaml.example config.yaml
nano config.yaml   # sesuaikan jika perlu

# Buat direktori data
mkdir -p data
```

## 4. Build dan jalankan container

```bash
docker compose up -d --build
```

Cek status:
```bash
docker compose ps
docker compose logs -f
```

## 5. Setup Nginx

```bash
# Install nginx jika belum
sudo apt install -y nginx

# Copy config
sudo cp deploy/nginx.conf /etc/nginx/sites-available/mangachapter

# Edit domain di config
sudo nano /etc/nginx/sites-available/mangachapter
# Ganti: your-domain.com -> domain kamu

# Aktifkan site
sudo ln -s /etc/nginx/sites-available/mangachapter /etc/nginx/sites-enabled/

# Test konfigurasi
sudo nginx -t

# Aktifkan tanpa SSL dulu (HTTP only) untuk verifikasi certbot
# Edit config, comment blok SSL dan hapus redirect 301 sementara
sudo systemctl reload nginx
```

## 6. Setup SSL dengan Certbot

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d manga.navisha.cloud

# Certbot akan otomatis update nginx config
sudo systemctl reload nginx
```

## 7. Verifikasi

```bash
# Cek app berjalan
curl http://localhost:8070/

# Cek via domain
curl https://manga.navisha.cloud/

# Cek health container
docker inspect --format='{{.State.Health.Status}}' mangachapter
```

## Update aplikasi

```bash
git pull
docker compose up -d --build
```

## Perintah berguna

```bash
# Lihat log
docker compose logs -f manga-web

# Restart
docker compose restart manga-web

# Stop
docker compose down

# Masuk ke container
docker compose exec manga-web sh
```

## Struktur file di VPS

```
~/mangachapter/
├── .env              # secrets (jangan di-commit)
├── config.yaml       # konfigurasi app
├── data/
│   └── manga.db      # database SQLite (persistent via volume)
├── docker-compose.yml
└── deploy/
    └── nginx.conf
```
