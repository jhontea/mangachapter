package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"project/mangachapter/internal/checker"
	"project/mangachapter/internal/config"
	"project/mangachapter/internal/notifier"
	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"
)

type app struct {
	configPath string
	debug      bool
	cfg        *config.Config
	repo       storage.Repository
	checker    *checker.Checker
	notifier   notifier.Notifier
}

func (a *app) init() error {
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return fmt.Errorf("muat config: %w", err)
	}
	a.cfg = cfg

	level := slog.LevelInfo
	if a.debug {
		level = slog.LevelDebug
	} else {
		switch cfg.Log.Level {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	// Prioritaskan DSN (PostgreSQL) jika tersedia, fallback ke Path (SQLite)
	dbDSN := cfg.Storage.DSN
	if dbDSN == "" {
		dbDSN = cfg.Storage.Path
	}
	repo, err := storage.Open(dbDSN)
	if err != nil {
		return fmt.Errorf("buka storage: %w", err)
	}
	a.repo = repo

	// Inisialisasi adapter sumber
	a.initSources()

	// Inisialisasi notifier
	a.initNotifier()

	// Inisialisasi checker
	a.checker = checker.New(a.repo, source.AvailableMap(), a.notifier)

	return nil
}

func (a *app) close() {
	if a.repo != nil {
		_ = a.repo.Close()
	}
}

func (a *app) context() context.Context {
	return context.Background()
}

func (a *app) initSources() {
	// Kiryuu
	kiryuuClient := source.NewHTTPClient(
		a.cfg.Sources.Kiryuu.UserAgent,
		a.cfg.KiryuuRateLimit(),
	)
	kiryuu := source.NewKiryuu(
		a.cfg.Sources.Kiryuu.BaseURL,
		kiryuuClient,
	)
	source.Register("kiryuu", kiryuu)

	// Manga Plus
	mangaplus := source.NewMangaPlus(a.cfg.Sources.MangaPlus.Language)
	source.Register("mangaplus", mangaplus)
}

func (a *app) initNotifier() {
	// Utamakan Telegram jika aktif, fallback ke email
	if a.cfg.Telegram.Enabled {
		a.notifier = notifier.NewTelegram(a.cfg.Telegram.Token, a.cfg.Telegram.ChatID)
		slog.Info("notifier: telegram aktif")
		return
	}
	if a.cfg.Email.Enabled {
		a.notifier = notifier.NewEmail(
			a.cfg.Email.SMTPHost,
			a.cfg.Email.SMTPPort,
			a.cfg.Email.Username,
			a.cfg.Email.Password,
			a.cfg.Email.From,
			a.cfg.Email.To,
		)
		slog.Info("notifier: email aktif")
		return
	}
	a.notifier = nil
	slog.Warn("notifier: tidak ada notifier yang dikonfigurasi")
}
