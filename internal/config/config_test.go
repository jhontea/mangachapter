package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
scheduler:
  interval: "30m"
email:
  enabled: false
storage:
  path: "./data/test.db"
sources:
  kiryuu:
    base_url: "https://v6.kiryuu.to"
    user_agent: "TestAgent/1.0"
    rate_limit: "1s"
  mangaplus:
    base_url: "https://mangaplus.shueisha.co.jp"
    language: "ind"
log:
  level: "debug"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Scheduler.Interval != "30m" {
		t.Errorf("Scheduler.Interval = %q, want 30m", cfg.Scheduler.Interval)
	}
	if cfg.Email.Enabled {
		t.Error("Email.Enabled should be false")
	}
	if cfg.Storage.Path != "./data/test.db" {
		t.Errorf("Storage.Path = %q", cfg.Storage.Path)
	}
	if cfg.Sources.MangaPlus.Language != "ind" {
		t.Errorf("Sources.MangaPlus.Language = %q, want ind", cfg.Sources.MangaPlus.Language)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
email:
  enabled: false
storage:
  path: "./data/manga.db"
sources:
  kiryuu:
    rate_limit: "1s"
log:
  level: "info"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MANGA_DB_PATH", "/tmp/custom.db")
	t.Setenv("MANGA_LOG_LEVEL", "warn")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Storage.Path != "/tmp/custom.db" {
		t.Errorf("Storage.Path = %q, want /tmp/custom.db", cfg.Storage.Path)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("Log.Level = %q, want warn", cfg.Log.Level)
	}
}

func TestValidateEmailEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
email:
  enabled: true
storage:
  path: "./data/manga.db"
sources:
  kiryuu:
    rate_limit: "1s"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected validation error when email enabled without SMTP config")
	}
}
