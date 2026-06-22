package source

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"
)

// Source mendefinisikan interface yang harus diimplementasikan oleh semua adapter sumber manga.
type Source interface {
	// Search mencari manga yang cocok dengan query yang diberikan.
	Search(ctx context.Context, query string) ([]SearchResult, error)

	// GetLatestChapter mengambil info chapter terbaru untuk manga yang diidentifikasi oleh URL atau ID.
	GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error)
}

// SearchResult merepresentasikan satu hasil pencarian dari sumber.
type SearchResult struct {
	Title string // Judul tampilan
	URL   string // URL lengkap ke halaman manga
	ID    string // Slug atau ID numerik yang digunakan oleh sumber
}

// ChapterInfo merepresentasikan informasi tentang chapter tertentu.
type ChapterInfo struct {
	Number   string  // "Chapter 123" atau "Ch. 123.5"
	Title    string  // Subtitle opsional (misal "Sabaody Archipelago")
	URL      string  // URL lengkap ke halaman chapter
	NumValue float64 // Nilai numerik untuk perbandingan (misal 123.0, 123.5)
}

// Registry menyimpan semua implementasi sumber yang terdaftar.
var registry = map[string]Source{}

// Register menambahkan implementasi sumber ke registry.
func Register(name string, s Source) {
	registry[name] = s
}

// Get mengambil sumber berdasarkan nama. Mengembalikan false jika tidak ditemukan.
func Get(name string) (Source, bool) {
	s, ok := registry[name]
	return s, ok
}

// Available mengembalikan daftar nama sumber yang terdaftar.
func Available() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// AvailableMap mengembalikan salinan map registry.
func AvailableMap() map[string]Source {
	m := make(map[string]Source, len(registry))
	for k, v := range registry {
		m[k] = v
	}
	return m
}

// HTTPClient adalah HTTP client bersama dengan rate limiting dan User-Agent kustom.
type HTTPClient struct {
	client    *http.Client
	userAgent string
	rateLimit time.Duration
	lastReq   time.Time
	mu        sync.Mutex
}

// NewHTTPClient membuat HTTP client baru dengan pengaturan yang diberikan.
func NewHTTPClient(userAgent string, rateLimit time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
		rateLimit: rateLimit,
	}
}

// Do menjalankan request HTTP dengan rate limiting dan User-Agent kustom.
func (h *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	h.mu.Lock()
	elapsed := time.Since(h.lastReq)
	if elapsed < h.rateLimit {
		time.Sleep(h.rateLimit - elapsed)
	}
	h.lastReq = time.Now()
	h.mu.Unlock()

	req.Header.Set("User-Agent", h.userAgent)
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	return h.client.Do(req)
}

// chapterRe mencocokkan nomor chapter dalam berbagai format.
// Contoh: "Chapter 123", "Ch. 123", "Ch 123.5", "123", "#236", "Manga Name Chapter 446", "chapter-446"
var chapterRe = regexp.MustCompile(`(?i)(?:chapter|ch\.?)\s*[#]?(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$|^#(\d+(?:\.\d+)?)`)

// ParseChapterNumber mengekstrak nilai numerik chapter dari string.
// Mengembalikan 0 dan string asli jika tidak ditemukan angka.
func ParseChapterNumber(s string) (float64, string) {
	s = strings.TrimSpace(s)
	matches := chapterRe.FindStringSubmatch(s)
	if matches == nil {
		return 0, s
	}

	// matches[1] dari grup (chapter|ch), matches[2] dari angka standalone, matches[3] dari #angka
	numStr := matches[1]
	if numStr == "" {
		numStr = matches[2]
	}
	if numStr == "" {
		numStr = matches[3]
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, s
	}
	return val, fmt.Sprintf("Chapter %s", numStr)
}
