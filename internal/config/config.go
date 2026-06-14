package config

import (
	"fmt"
	"os"
	"time"

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

func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv("MANGA_CONFIG_PATH")
	}
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
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
		Email:     EmailConfig{Enabled: true, SMTPPort: 587},
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
	if v := os.Getenv("MANGA_DB_PATH"); v != "" {
		c.Storage.Path = v
	}
	if v := os.Getenv("MANGA_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
}

func (c *Config) validate() error {
	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path is required")
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
			return fmt.Errorf("email.smtp_host is required when email is enabled")
		}
		if c.Email.SMTPPort == 0 {
			return fmt.Errorf("email.smtp_port is required when email is enabled")
		}
		if len(c.Email.To) == 0 {
			return fmt.Errorf("email.to is required when email is enabled")
		}
	}

	return nil
}

func (c *Config) KiryuuRateLimit() time.Duration {
	d, _ := time.ParseDuration(c.Sources.Kiryuu.RateLimit)
	return d
}

func (c *Config) SchedulerInterval() time.Duration {
	d, _ := time.ParseDuration(c.Scheduler.Interval)
	return d
}
