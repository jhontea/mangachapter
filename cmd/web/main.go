package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"project/mangachapter/internal/checker"
	"project/mangachapter/internal/config"
	"project/mangachapter/internal/notifier"
	"project/mangachapter/internal/scheduler"
	"project/mangachapter/internal/source"
	"project/mangachapter/internal/storage"
)

type server struct {
	cfg      *config.Config
	repo     storage.Repository
	checker  *checker.Checker
	notifier notifier.Notifier
}

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	s := &server{}
	if err := s.init(); err != nil {
		slog.Error("gagal menginisialisasi", "error", err)
		os.Exit(1)
	}
	defer s.repo.Close()

	mux := http.NewServeMux()

	// File statis
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/static/", s.handleStatic)

	// Endpoint API
	mux.HandleFunc("/api/manga", s.handleManga)
	mux.HandleFunc("/api/manga/", s.handleMangaByID)
	mux.HandleFunc("/api/manga/check-all", s.handleCheckAll)
	mux.HandleFunc("/api/manga/search", s.handleSearch)
	mux.HandleFunc("/api/sources", s.handleSources)

	slog.Info("server web dimulai", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("error server", "error", err)
		os.Exit(1)
	}
}

func (s *server) init() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("muat config: %w", err)
	}
	s.cfg = cfg

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Prioritaskan DSN (PostgreSQL) jika tersedia, fallback ke Path (SQLite)
	dbDSN := cfg.Storage.DSN
	if dbDSN == "" {
		dbDSN = cfg.Storage.Path
	}
	repo, err := storage.Open(dbDSN)
	if err != nil {
		return fmt.Errorf("buka storage: %w", err)
	}
	s.repo = repo

	// Inisialisasi sumber
	kiryuuClient := source.NewHTTPClient(
		cfg.Sources.Kiryuu.UserAgent,
		cfg.KiryuuRateLimit(),
	)
	kiryuu := source.NewKiryuu(
		cfg.Sources.Kiryuu.BaseURL,
		kiryuuClient,
	)
	source.Register("kiryuu", kiryuu)

	mangaplus := source.NewMangaPlus(cfg.Sources.MangaPlus.Language)
	source.Register("mangaplus", mangaplus)

	// Inisialisasi notifier — utamakan Telegram jika aktif, fallback ke email
	if cfg.Telegram.Enabled {
		s.notifier = notifier.NewTelegram(cfg.Telegram.Token, cfg.Telegram.ChatID)
		slog.Info("notifier: telegram aktif")
	} else if cfg.Email.Enabled {
		s.notifier = notifier.NewEmail(
			cfg.Email.SMTPHost,
			cfg.Email.SMTPPort,
			cfg.Email.Username,
			cfg.Email.Password,
			cfg.Email.From,
			cfg.Email.To,
		)
		slog.Info("notifier: email aktif")
	} else {
		slog.Warn("notifier: tidak ada notifier yang dikonfigurasi")
	}

	s.checker = checker.New(s.repo, source.AvailableMap(), s.notifier)

	// Jalankan scheduler sebagai background goroutine
	checkFn := scheduler.CheckAllFunc(func(ctx context.Context) error {
		_, err := s.checker.CheckAll(ctx)
		return err
	})
	var sched *scheduler.Scheduler
	if s.cfg.Scheduler.Cron != "" {
		sched = scheduler.NewWithCron(checkFn, s.cfg.Scheduler.Cron)
		slog.Info("scheduler dimulai", "cron", s.cfg.Scheduler.Cron)
	} else if s.cfg.Scheduler.Interval != "" {
		sched = scheduler.New(checkFn, s.cfg.SchedulerInterval())
		slog.Info("scheduler dimulai", "interval", s.cfg.Scheduler.Interval)
	}
	if sched != nil {
		go sched.Run(context.Background())
	}

	return nil
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func (s *server) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/static/", http.FileServer(http.Dir("web"))).ServeHTTP(w, r)
}

func (s *server) handleSources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source.Available())
}

func (s *server) handleManga(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		items, err := s.repo.ListManga(ctx)
		if err != nil {
			httpError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var req struct {
			Source   string `json:"source"`
			Title    string `json:"title"`
			URL      string `json:"url"`
			SourceID string `json:"source_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Errorf("body request tidak valid: %w", err))
			return
		}

		src, ok := source.Get(req.Source)
		if !ok {
			httpError(w, fmt.Errorf("sumber tidak dikenal %q", req.Source))
			return
		}

		// Ambil chapter terbaru
		ch, err := src.GetLatestChapter(ctx, req.URL)
		if err != nil {
			httpError(w, fmt.Errorf("ambil chapter terbaru: %w", err))
			return
		}

		m := &storage.TrackedManga{
			Source:         req.Source,
			SourceID:       req.SourceID,
			Title:          req.Title,
			URL:            req.URL,
			LastChapter:    ch.Number,
			LastChapterNum: ch.NumValue,
		}

		if err := s.repo.AddManga(ctx, m); err != nil {
			httpError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(m)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleMangaByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Ekstrak ID dari path: /api/manga/{id}
	suffix := strings.TrimPrefix(r.URL.Path, "/api/manga/")
	if suffix == "" || suffix == "check-all" || suffix == "search" {
		return
	}

	idStr := strings.SplitN(suffix, "/", 2)[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpError(w, fmt.Errorf("ID manga tidak valid: %w", err))
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, err := s.repo.GetManga(ctx, id)
		if err != nil {
			httpError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)

	case http.MethodDelete:
		if err := s.repo.RemoveManga(ctx, id); err != nil {
			httpError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodPost:
		// Periksa satu manga
		result, err := s.checker.CheckOne(ctx, id)
		if err != nil {
			httpError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleCheckAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	results, err := s.checker.CheckAll(ctx)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sourceName := r.URL.Query().Get("source")
	query := r.URL.Query().Get("query")

	if sourceName == "" || query == "" {
		httpError(w, fmt.Errorf("parameter source dan query wajib diisi"))
		return
	}

	src, ok := source.Get(sourceName)
	if !ok {
		httpError(w, fmt.Errorf("sumber tidak dikenal %q", sourceName))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := src.Search(ctx, query)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func httpError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
