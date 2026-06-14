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

// Source defines the interface that all manga source adapters must implement.
type Source interface {
	// Search queries the source for manga matching the given query string.
	Search(ctx context.Context, query string) ([]SearchResult, error)

	// GetLatestChapter fetches the latest chapter info for a manga identified by its URL or ID.
	GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error)
}

// SearchResult represents a single search result from a source.
type SearchResult struct {
	Title string // Display title
	URL   string // Full URL to the manga page
	ID    string // Slug or numeric ID used by the source
}

// ChapterInfo represents information about a specific chapter.
type ChapterInfo struct {
	Number   string  // "Chapter 123" or "Ch. 123.5"
	Title    string  // Optional subtitle (e.g., "Sabaody Archipelago")
	URL      string  // Full URL to the chapter page
	NumValue float64 // Numeric value for comparison (e.g., 123.0, 123.5)
}

// Registry holds all registered source implementations.
var registry = map[string]Source{}

// Register adds a source implementation to the registry.
func Register(name string, s Source) {
	registry[name] = s
}

// Get retrieves a source by name. Returns false if not found.
func Get(name string) (Source, bool) {
	s, ok := registry[name]
	return s, ok
}

// Available returns a list of registered source names.
func Available() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// AvailableMap returns a copy of the registry map.
func AvailableMap() map[string]Source {
	m := make(map[string]Source, len(registry))
	for k, v := range registry {
		m[k] = v
	}
	return m
}

// HTTPClient is a shared HTTP client with rate limiting and custom User-Agent.
type HTTPClient struct {
	client    *http.Client
	userAgent string
	rateLimit time.Duration
	lastReq   time.Time
	mu        sync.Mutex
}

// NewHTTPClient creates a new shared HTTP client with the given settings.
func NewHTTPClient(userAgent string, rateLimit time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
		rateLimit: rateLimit,
	}
}

// Do executes an HTTP request with rate limiting and custom User-Agent.
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

// chapterRe matches chapter numbers in various formats.
// Examples: "Chapter 123", "Ch. 123", "Ch 123.5", "123"
var chapterRe = regexp.MustCompile(`(?i)(?:chapter|ch\.?)\s*(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$`)

// ParseChapterNumber extracts a numeric chapter value from a string.
// Returns 0 and the original string if no number is found.
func ParseChapterNumber(s string) (float64, string) {
	s = strings.TrimSpace(s)
	matches := chapterRe.FindStringSubmatch(s)
	if matches == nil {
		return 0, s
	}

	// matches[1] is from the capture group, matches[2] is from the alternative
	numStr := matches[1]
	if numStr == "" {
		numStr = matches[2]
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, s
	}
	return val, fmt.Sprintf("Chapter %s", numStr)
}

// ToChapterUpdate converts a ChapterInfo to the storage ChapterUpdate type
// by returning the number string and numeric value.
func (c *ChapterInfo) ToChapterUpdate() (string, float64) {
	if c == nil {
		return "", 0
	}
	return c.Number, c.NumValue
}