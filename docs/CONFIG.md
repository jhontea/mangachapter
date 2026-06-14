# Configuration Reference

## File Location

Priority (highest first):

1. Path dari flag `--config /path/to/config.yaml`
2. Env `MANGA_CONFIG_PATH`
3. `./config.yaml` (working directory)

Salin dari `config.yaml.example`:

```bash
cp config.yaml.example config.yaml
```

**Jangan commit `config.yaml`** — berisi credentials.

---

## Full Schema

```yaml
# Scheduler settings
scheduler:
  # Option A: simple interval (Go duration string)
  interval: "1h"
  # Option B: cron expression (if set, overrides interval)
  # cron: "0 * * * *"

# Email notification (SMTP)
email:
  enabled: true
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: "your@gmail.com"
  password: ""          # prefer env MANGA_SMTP_PASSWORD
  from: "your@gmail.com"
  to:
    - "your@gmail.com"
  # Optional: TLS
  # insecure_skip_verify: false

# Database
storage:
  path: "./data/manga.db"

# Source-specific settings
sources:
  kiryuu:
    base_url: "https://v6.kiryuu.to"
    user_agent: "MangaChapterNotifier/1.0 (+personal use)"
    rate_limit: "2s"      # min delay between requests

  mangaplus:
    base_url: "https://mangaplus.shueisha.co.jp"
    language: "eng"       # eng | ind | spa | por | etc.

# Logging
log:
  level: "info"           # debug | info | warn | error
```

---

## Environment Variables

| Variable | Overrides | Description |
|----------|-----------|-------------|
| `MANGA_CONFIG_PATH` | — | Path to config file |
| `MANGA_SMTP_PASSWORD` | `email.password` | SMTP password / app password |
| `MANGA_SMTP_USERNAME` | `email.username` | SMTP username |
| `MANGA_DB_PATH` | `storage.path` | SQLite file path |
| `MANGA_LOG_LEVEL` | `log.level` | Log level |

---

## Gmail Setup

1. Enable 2-Factor Authentication on Google account
2. Generate App Password: Google Account → Security → App passwords
3. Use App Password as `MANGA_SMTP_PASSWORD`

```yaml
email:
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: "you@gmail.com"
  from: "you@gmail.com"
  to: ["you@gmail.com"]
```

```bash
export MANGA_SMTP_PASSWORD="xxxx xxxx xxxx xxxx"
```

---

## Outlook / Microsoft 365

```yaml
email:
  smtp_host: "smtp.office365.com"
  smtp_port: 587
  username: "you@outlook.com"
```

---

## Scheduler Options

| Config | Behavior |
|--------|----------|
| `interval: "30m"` | Every 30 minutes |
| `interval: "1h"` | Every hour (default) |
| `interval: "6h"` | Every 6 hours |
| `cron: "0 */2 * * *"` | Every 2 hours (cron syntax) |
| `cron: "0 8,20 * * *"` | At 08:00 and 20:00 daily |

If both `cron` and `interval` set, **`cron` takes precedence**.

---

## Disabling Email (dev/test)

```yaml
email:
  enabled: false
```

Checker tetap jalan; log saja chapter baru tanpa kirim email.

---

## Go Config Struct (reference)

```go
type Config struct {
    Scheduler SchedulerConfig `yaml:"scheduler"`
    Email     EmailConfig     `yaml:"email"`
    Storage   StorageConfig   `yaml:"storage"`
    Sources   SourcesConfig   `yaml:"sources"`
    Log       LogConfig       `yaml:"log"`
}

type SchedulerConfig struct {
    Interval string `yaml:"interval"`
    Cron     string `yaml:"cron"`
}

type EmailConfig struct {
    Enabled  bool     `yaml:"enabled"`
    SMTPHost string   `yaml:"smtp_host"`
    SMTPPort int      `yaml:"smtp_port"`
    Username string   `yaml:"username"`
    Password string   `yaml:"password"`
    From     string   `yaml:"from"`
    To       []string `yaml:"to"`
}

type StorageConfig struct {
    Path string `yaml:"path"`
}

type SourcesConfig struct {
    Kiryuu    KiryuuConfig    `yaml:"kiryuu"`
    MangaPlus MangaPlusConfig `yaml:"mangaplus"`
}

type KiryuuConfig struct {
    BaseURL   string `yaml:"base_url"`
    UserAgent string `yaml:"user_agent"`
    RateLimit string `yaml:"rate_limit"`
}

type MangaPlusConfig struct {
    BaseURL  string `yaml:"base_url"`
    Language string `yaml:"language"`
}

type LogConfig struct {
    Level string `yaml:"level"`
}
```
