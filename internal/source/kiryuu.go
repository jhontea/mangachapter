package source

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// Kiryuu mengimplementasikan Source untuk situs manga Kiryuu menggunakan WordPress REST API.
type Kiryuu struct {
	baseURL string
	client  *HTTPClient
}

// NewKiryuu membuat adapter sumber Kiryuu baru.
func NewKiryuu(baseURL, userAgent string, client *HTTPClient) *Kiryuu {
	return &Kiryuu{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

// kiryuuManga merepresentasikan manga dari WordPress REST API.
type kiryuuManga struct {
	ID       int         `json:"id"`
	Title    kiryuuTitle `json:"title"`
	Link     string      `json:"link"`
	Slug     string      `json:"slug"`
	Modified string      `json:"modified"`
}

type kiryuuTitle struct {
	Rendered string `json:"rendered"`
}

// Search mencari manga di Kiryuu menggunakan WordPress REST API.
func (k *Kiryuu) Search(ctx context.Context, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("%s/wp-json/wp/v2/manga?search=%s&per_page=20", k.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("buat request pencarian: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", k.client.userAgent)

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request pencarian: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pencarian: HTTP %d", resp.StatusCode)
	}

	var mangas []kiryuuManga
	if err := json.NewDecoder(resp.Body).Decode(&mangas); err != nil {
		return nil, fmt.Errorf("decode hasil pencarian: %w", err)
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

	slog.Debug("pencarian kiryuu", "query", query, "hasil", len(results))
	return results, nil
}

// GetLatestChapter mengambil chapter terbaru untuk manga.
// mangaURL adalah URL lengkap yang tersimpan di DB (misal https://v6.kiryuu.to/manga/one-piece/).
// Kita mengekstrak slug dari URL tersebut dan menggunakan baseURL yang dikonfigurasi untuk semua panggilan API.
// Dengan demikian, jika domain berubah, hanya config yang perlu diperbarui.
func (k *Kiryuu) GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error) {
	// Ekstrak slug dari URL yang tersimpan
	slug := extractSlugFromURL(mangaURL)
	if slug == "" {
		return nil, fmt.Errorf("tidak bisa mengekstrak slug manga dari URL: %s", mangaURL)
	}

	// Verifikasi manga ada dengan mengakses halaman manga di baseURL saat ini
	mangaPageURL := fmt.Sprintf("%s/manga/%s/", k.baseURL, slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mangaPageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("buat request halaman manga: %w", err)
	}
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en;q=0.8")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request halaman manga: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("manga tidak ditemukan: %s", slug)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("halaman manga: HTTP %d", resp.StatusCode)
	}

	// Ambil chapter terbaru via REST API menggunakan baseURL saat ini
	return k.getLatestChapterFromRESTAPI(ctx, slug)
}

// extractSlugFromURL mengekstrak slug manga dari URL Kiryuu.
// Mendukung format:
//   - https://v6.kiryuu.to/manga/one-piece/
//   - https://v6.kiryuu.to/manga/one-piece
//   - /manga/one-piece/
func extractSlugFromURL(mangaURL string) string {
	parsed, err := url.Parse(mangaURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, part := range parts {
		if part == "manga" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// getLatestChapterFromRESTAPI mengambil chapter terbaru menggunakan WordPress REST API.
// REST API tidak mendukung filter parent, jadi kita mencari secara luas dan filter berdasarkan pola slug.
func (k *Kiryuu) getLatestChapterFromRESTAPI(ctx context.Context, mangaSlug string) (*ChapterInfo, error) {
	// Ekstrak istilah pencarian pendek dari slug (2-3 kata pertama)
	searchTerm := extractSearchTerm(mangaSlug)

	// Ambil lebih banyak hasil untuk difilter
	searchURL := fmt.Sprintf("%s/wp-json/wp/v2/chapter?search=%s&per_page=50&orderby=date&order=desc",
		k.baseURL, url.QueryEscape(searchTerm))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("buat request chapter REST: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request chapter REST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chapter REST: HTTP %d", resp.StatusCode)
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
		return nil, fmt.Errorf("tidak ditemukan chapter via REST API untuk manga %s", mangaSlug)
	}

	// Filter chapter yang milik manga ini dengan memeriksa apakah slug dimulai dengan slug manga
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
		return nil, fmt.Errorf("tidak ditemukan chapter yang cocok untuk manga %s", mangaSlug)
	}

	return bestChapter, nil
}

// extractSearchTerm mengekstrak istilah pencarian pendek dari slug manga.
func extractSearchTerm(slug string) string {
	parts := strings.Split(slug, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], " ")
	}
	return strings.Join(parts, " ")
}

// parseChapterFromLink mengekstrak info chapter dari link dan teks chapter.
func parseChapterFromLink(href, text string) (*ChapterInfo, error) {
	if href == "" && text == "" {
		return nil, fmt.Errorf("link atau teks chapter kosong")
	}

	href = strings.TrimSpace(href)

	// Coba ekstrak nomor chapter dari teks terlebih dahulu
	numValue, cleanTitle := ParseChapterNumber(text)

	// Jika parsing dari teks gagal, coba dari slug URL
	// Format URL Kiryuu: /?chapter=manga-slug-chapter-N atau /manga/slug/chapter-N/
	if numValue == 0 {
		// Coba ekstrak dari path URL seperti /manga/slug/chapter-446/
		if idx := strings.Index(href, "/chapter-"); idx >= 0 {
			numValue, cleanTitle = ParseChapterNumber(href[idx:])
		}
		// Coba ekstrak dari query param seperti ?chapter=slug-chapter-446
		if numValue == 0 {
			parsed, err := url.Parse(href)
			if err == nil {
				slug := parsed.Query().Get("chapter")
				if slug == "" {
					slug = parsed.Path
				}
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

	slog.Debug("chapter terbaru kiryuu", "chapter", info.Number, "num", info.NumValue, "url", info.URL)
	return info, nil
}