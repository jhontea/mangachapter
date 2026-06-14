package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// Kiryuu implements Source for the Kiryuu manga site using WordPress REST API.
type Kiryuu struct {
	baseURL string
	client  *HTTPClient
}

// NewKiryuu creates a new Kiryuu source adapter.
func NewKiryuu(baseURL, userAgent string, client *HTTPClient) *Kiryuu {
	return &Kiryuu{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

// kiryuuManga represents a manga from the WordPress REST API.
type kiryuuManga struct {
	ID       int              `json:"id"`
	Title    kiryuuTitle      `json:"title"`
	Link     string           `json:"link"`
	Slug     string           `json:"slug"`
	Modified string           `json:"modified"`
}

type kiryuuTitle struct {
	Rendered string `json:"rendered"`
}

// Search searches for manga on Kiryuu using the WordPress REST API.
func (k *Kiryuu) Search(ctx context.Context, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("%s/wp-json/wp/v2/manga?search=%s&per_page=20", k.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", k.client.userAgent)

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: HTTP %d", resp.StatusCode)
	}

	var mangas []kiryuuManga
	if err := json.NewDecoder(resp.Body).Decode(&mangas); err != nil {
		return nil, fmt.Errorf("decode search results: %w", err)
	}

	var results []SearchResult
	for _, m := range mangas {
		if m.Title.Rendered == "" || m.Link == "" {
			continue
		}
		results = append(results, SearchResult{
			Title: m.Title.Rendered,
			URL:   m.Link,
			ID:    m.Slug,
		})
	}

	slog.Debug("kiryuu search", "query", query, "results", len(results))
	return results, nil
}

// GetLatestChapter fetches the latest chapter for a manga by parsing the manga page HTML.
func (k *Kiryuu) GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mangaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create chapter request: %w", err)
	}
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en;q=0.8")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chapter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chapter page: HTTP %d", resp.StatusCode)
	}

	return k.parseLatestChapter(resp.Body, mangaURL)
}

// parseLatestChapter extracts the manga slug from the URL and fetches the latest chapter via REST API.
// The new Kiryuu theme only renders "First Chapter" in the HTML; all other chapters are loaded via HTMX AJAX.
// We skip HTML parsing entirely and use the WordPress REST API with slug-based filtering.
func (k *Kiryuu) parseLatestChapter(body io.Reader, mangaURL string) (*ChapterInfo, error) {
	mangaSlug := extractSlugFromURL(mangaURL)
	if mangaSlug != "" {
		return k.getLatestChapterFromRESTAPI(context.Background(), mangaSlug, mangaURL)
	}
	return nil, fmt.Errorf("no chapters found on page %s", mangaURL)
}

// extractSlugFromURL extracts the manga slug from a Kiryuu manga URL.
func extractSlugFromURL(mangaURL string) string {
	parsed, err := url.Parse(mangaURL)
	if err != nil {
		return ""
	}
	// URL format: /manga/{slug}/
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, part := range parts {
		if part == "manga" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// getLatestChapterFromRESTAPI fetches the latest chapter using the WordPress REST API.
// The REST API doesn't support parent filtering, so we search broadly and filter by slug pattern.
func (k *Kiryuu) getLatestChapterFromRESTAPI(ctx context.Context, mangaSlug, mangaURL string) (*ChapterInfo, error) {
	// Extract a short search term from the slug (first 2-3 words)
	searchTerm := extractSearchTerm(mangaSlug)

	// Fetch more results to filter from
	searchURL := fmt.Sprintf("%s/wp-json/wp/v2/chapter?search=%s&per_page=50&orderby=date&order=desc",
		k.baseURL, url.QueryEscape(searchTerm))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create REST chapter request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("REST chapter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REST chapter: HTTP %d", resp.StatusCode)
	}

	var chapters []struct {
		Title struct {
			Rendered string `json:"rendered"`
		} `json:"title"`
		Link string `json:"link"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chapters); err != nil {
		return nil, fmt.Errorf("decode chapters: %w", err)
	}

	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found via REST API for manga %s", mangaSlug)
	}

	// Filter chapters belonging to this manga by checking if slug starts with manga slug
	mangaSlugPrefix := strings.TrimSuffix(mangaSlug, "-") + "-"
	var bestChapter *ChapterInfo
	for _, ch := range chapters {
		if !strings.HasPrefix(ch.Slug, mangaSlugPrefix) {
			continue
		}
		info, err := parseChapterFromLink(ch.Link, ch.Title.Rendered)
		if err != nil || info.NumValue == 0 {
			continue
		}
		if bestChapter == nil || info.NumValue > bestChapter.NumValue {
			bestChapter = info
		}
	}

	if bestChapter == nil {
		return nil, fmt.Errorf("no matching chapters found for manga %s", mangaSlug)
	}

	return bestChapter, nil
}

// extractSearchTerm extracts a short search term from a manga slug.
func extractSearchTerm(slug string) string {
	// Take first 2-3 meaningful words from the slug
	parts := strings.Split(slug, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], " ")
	}
	return strings.Join(parts, " ")
}

// parseChapterFromLink extracts chapter info from a chapter link and text.
func parseChapterFromLink(href, text string) (*ChapterInfo, error) {
	if href == "" && text == "" {
		return nil, fmt.Errorf("empty chapter link or text")
	}

	// Clean up href
	href = strings.TrimSpace(href)

	// Try to extract chapter number from text first
	numValue, cleanTitle := ParseChapterNumber(text)

	// If parsing from text failed, try from URL slug
	// Kiryuu URL format: /?chapter=manga-slug-chapter-N or /manga/slug/chapter-N/
	if numValue == 0 {
		// Try extracting from URL path like /manga/slug/chapter-446/
		if idx := strings.Index(href, "/chapter-"); idx >= 0 {
			numValue, cleanTitle = ParseChapterNumber(href[idx:])
		}
		// Try extracting from query param like ?chapter=slug-chapter-446
		if numValue == 0 {
			parsed, err := url.Parse(href)
			if err == nil {
				slug := parsed.Query().Get("chapter")
				if slug == "" {
					slug = parsed.Path
				}
				// Look for -chapter-N pattern in slug
				if idx := strings.Index(slug, "-chapter-"); idx >= 0 {
					numValue, cleanTitle = ParseChapterNumber(slug[idx+1:])
				}
			}
		}
	}

	info := &ChapterInfo{
		Number:   cleanTitle,
		URL:      href,
		NumValue: numValue,
	}

	slog.Debug("kiryuu latest chapter", "chapter", info.Number, "num", info.NumValue, "url", info.URL)
	return info, nil
}
