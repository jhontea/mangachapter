package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath    = "config.yaml"
	defaultInterval      = "1h"
	defaultDBPath        = "./data/manga.db"
	defaultLogLevel      = "info"
	defaultKiryuuBase    = "https://v6.kiryuu.to"
	defaultKiryuuUA      = "MangaChapterNotifier/1.0 (+personal use)"
	defaultKiryuuRate    = "2s"
	defaultMangaPlusBase = "https://mangaplus.shueisha.co.jp"
	defaultMangaPlusLang = "eng"
)

// Config menyimpan semua konfigurasi aplikasi.
type Config struct {
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Email     EmailConfig     `yaml:"email"`
	Telegram  TelegramConfig  `yaml:"telegram"`
	Storage   StorageConfig   `yaml:"storage"`
	Sources   SourcesConfig   `yaml:"sources"`
	Log       LogConfig       `yaml:"log"`
}

// SchedulerConfig untuk pengaturan interval pengecekan.
type SchedulerConfig struct {
	Interval string `yaml:"interval"`
	Cron     string `yaml:"cron"`
}

// EmailConfig untuk pengaturan notifikasi email.
type EmailConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

// TelegramConfig untuk pengaturan notifikasi Telegram.
type TelegramConfig struct {
	Enabled bool   `yaml:"enabled"`
	Token   string `yaml:"token"`
	ChatID  string `yaml:"chat_id"`
}

// StorageConfig untuk pengaturan penyimpanan data.
type StorageConfig struct {
	Path string `yaml:"path"` // SQLite path (lokal)
	DSN  string `yaml:"dsn"`  // PostgreSQL DSN (Supabase)
}

// SourcesConfig untuk pengaturan sumber manga.
type SourcesConfig struct {
	Kiryuu    KiryuuConfig    `yaml:"kiryuu"`
	MangaPlus MangaPlusConfig `yaml:"mangaplus"`
}

// KiryuuConfig untuk pengaturan sumber Kiryuu.
type KiryuuConfig struct {
	BaseURL   string `yaml:"base_url"`
	UserAgent string `yaml:"user_agent"`
	RateLimit string `yaml:"rate_limit"`
}

// MangaPlusConfig untuk pengaturan sumber Manga Plus.
type MangaPlusConfig struct {
	BaseURL  string `yaml:"base_url"`
	Language string `yaml:"language"`
}

// LogConfig untuk pengaturan level log.
type LogConfig struct {
	Level string `yaml:"level"`
}

// Load membaca konfigurasi dari file YAML dan environment variables.
func Load(path string) (*Config, error) {
	// Muat file .env jika ada (abaikan jika tidak ditemukan)
	if err := godotenv.Load(); err != nil {
		slog.Debug("file .env tidak ditemukan, menggunakan env sistem", "error", err)
	}

	if path == "" {
		path = os.Getenv("MANGA_CONFIG_PATH")
	}
	if path == "" {
		path = defaultConfigPath
	}

	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		// Config file tidak wajib jika DATABASE_URL sudah di-set
		if os.Getenv("DATABASE_URL") == "" {
			return nil, fmt.Errorf("baca config %q: %w", path, err)
		}
		slog.Warn("config file tidak ditemukan, menggunakan env variables", "path", path)
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	cfg.applyEnvOverrides()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Scheduler: SchedulerConfig{Interval: defaultInterval},
		Email:     EmailConfig{Enabled: false, SMTPPort: 587},
		Telegram:  TelegramConfig{Enabled: false},
		Storage:   StorageConfig{Path: defaultDBPath},
		Sources: SourcesConfig{
			Kiryuu: KiryuuConfig{
				BaseURL:   defaultKiryuuBase,
				UserAgent: defaultKiryuuUA,
				RateLimit: defaultKiryuuRate,
			},
			MangaPlus: MangaPlusConfig{
				BaseURL:  defaultMangaPlusBase,
				Language: defaultMangaPlusLang,
			},
		},
		Log: LogConfig{Level: defaultLogLevel},
	}
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("MANGA_SMTP_PASSWORD"); v != "" {
		c.Email.Password = v
	}
	if v := os.Getenv("MANGA_SMTP_USERNAME"); v != "" {
		c.Email.Username = v
	}
	if v := os.Getenv("TELEGRAM_TOKEN"); v != "" {
		c.Telegram.Token = v
	}
	if v := os.Getenv("TELEGRAM_CHAT_ID"); v != "" {
		c.Telegram.ChatID = v
	}
	if v := os.Getenv("MANGA_TELEGRAM_TOKEN"); v != "" {
		c.Telegram.Token = v
	}
	if v := os.Getenv("MANGA_TELEGRAM_CHAT_ID"); v != "" {
		c.Telegram.ChatID = v
	}
	// Auto-enable Telegram jika token dan chat ID sudah di-set
	if c.Telegram.Token != "" && c.Telegram.ChatID != "" {
		c.Telegram.Enabled = true
	}
	if v := os.Getenv("MANGA_DB_PATH"); v != "" {
		c.Storage.Path = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.Storage.DSN = v
	}
	if v := os.Getenv("MANGA_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
}

func (c *Config) validate() error {
	// Storage: butuh DSN (PostgreSQL) atau path (SQLite)
	if c.Storage.DSN == "" && c.Storage.Path == "" {
		return fmt.Errorf("storage.dsn atau storage.path wajib diisi (set DATABASE_URL atau MANGA_DB_PATH)")
	}

	if c.Scheduler.Cron == "" {
		if c.Scheduler.Interval == "" {
			c.Scheduler.Interval = defaultInterval
		}
		if _, err := time.ParseDuration(c.Scheduler.Interval); err != nil {
			return fmt.Errorf("scheduler.interval: %w", err)
		}
	}

	if _, err := time.ParseDuration(c.Sources.Kiryuu.RateLimit); err != nil {
		return fmt.Errorf("sources.kiryuu.rate_limit: %w", err)
	}

	if c.Email.Enabled {
		if c.Email.SMTPHost == "" {
			return fmt.Errorf("email.smtp_host wajib diisi jika email aktif")
		}
		if c.Email.SMTPPort == 0 {
			return fmt.Errorf("email.smtp_port wajib diisi jika email aktif")
		}
		if len(c.Email.To) == 0 {
			return fmt.Errorf("email.to wajib diisi jika email aktif")
		}
	}

	if c.Telegram.Enabled {
		if c.Telegram.Token == "" {
			return fmt.Errorf("telegram.token wajib diisi jika telegram aktif")
		}
		if c.Telegram.ChatID == "" {
			return fmt.Errorf("telegram.chat_id wajib diisi jika telegram aktif")
		}
	}

	return nil
}

// KiryuuRateLimit mengembalikan rate limit sebagai time.Duration.
func (c *Config) KiryuuRateLimit() time.Duration {
	d, _ := time.ParseDuration(c.Sources.Kiryuu.RateLimit)
	return d
}

// SchedulerInterval mengembalikan interval scheduler sebagai time.Duration.
func (c *Config) SchedulerInterval() time.Duration {
	d, _ := time.ParseDuration(c.Scheduler.Interval)
	return d
}
