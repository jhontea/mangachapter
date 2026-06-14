package source

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Kiryuu implements Source for the Kiryuu manga site.
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

// Search searches for manga on Kiryuu by query.
func (k *Kiryuu) Search(ctx context.Context, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("%s/?s=%s", k.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en;q=0.8")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search: HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	return k.parseSearchResults(resp.Body)
}

// parseSearchResults parses the search results page HTML.
func (k *Kiryuu) parseSearchResults(body io.Reader) ([]SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parse search HTML: %w", err)
	}

	var results []SearchResult
	doc.Find(".bsx").Each(func(i int, s *goquery.Selection) {
		a := s.Find("a")
		if a.Length() == 0 {
			return
		}
		href, _ := a.Attr("href")
		title := strings.TrimSpace(s.Find(".tt").Text())
		if title == "" {
			title = strings.TrimSpace(a.AttrOr("title", ""))
		}

		// Extract slug from URL: https://v6.kiryuu.to/manga/slug/ → slug
		slug := extractSlug(href)

		if title != "" && href != "" {
			results = append(results, SearchResult{
				Title: title,
				URL:   href,
				ID:    slug,
			})
		}
	})

	slog.Debug("kiryuu search", "query_results", len(results))
	return results, nil
}

// GetLatestChapter fetches the latest chapter for a manga from its page.
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

// parseLatestChapter parses the manga detail page to find the latest chapter.
func (k *Kiryuu) parseLatestChapter(body io.Reader, mangaURL string) (*ChapterInfo, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parse chapter HTML: %w", err)
	}

	// Try multiple selectors for chapter list (WordPress manga themes vary)
	// Common patterns: .eplister, .chapterlist, ul.chapterlist, #chapterlist
	var chapterLink *goquery.Selection
	chapterSelectors := []string{
		".eplister ul li:first-child a",
		".eplister li:first-child a",
		".chapterlist ul li:first-child a",
		"#chapterlist ul li:first-child a",
		"ul.chapterlist li:first-child a",
	}

	for _, sel := range chapterSelectors {
		chapterLink = doc.Find(sel)
		if chapterLink.Length() > 0 {
			break
		}
	}

	if chapterLink == nil || chapterLink.Length() == 0 {
		return nil, fmt.Errorf("no chapters found on page %s", mangaURL)
	}

	chapterTitle := strings.TrimSpace(chapterLink.Text())
	chapterHref, _ := chapterLink.Attr("href")

	numValue, cleanTitle := ParseChapterNumber(chapterTitle)

	info := &ChapterInfo{
		Number:   cleanTitle,
		URL:      chapterHref,
		NumValue: numValue,
	}

	// Try to extract chapter subtitle (e.g., the manga title part after chapter number)
	subtitleSel := chapterLink.Find(".chapternum")
	if subtitleSel.Length() > 0 {
		info.Title = strings.TrimSpace(subtitleSel.Text())
	}

	slog.Debug("kiryuu latest chapter", "chapter", info.Number, "num", info.NumValue)
	return info, nil
}

// slugRe extracts the manga slug from a URL path.
var slugRe = regexp.MustCompile(`/manga/([^/]+)/`)

// extractSlug extracts the manga slug from a Kiryuu URL.
func extractSlug(href string) string {
	matches := slugRe.FindStringSubmatch(href)
	if len(matches) >= 2 {
		return matches[1]
	}
	// Fallback: use last path segment
	parts := strings.Split(strings.TrimRight(href, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return href
}