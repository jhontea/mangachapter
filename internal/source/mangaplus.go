package source

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luevano/mangoplus"
)

// MangaPlus implements Source for the Manga Plus API.
type MangaPlus struct {
	client   *mangoplus.PlusClient
	language mangoplus.Language
}

// NewMangaPlus creates a new Manga Plus source adapter.
func NewMangaPlus(language string) *MangaPlus {
	lang := mangoplus.LanguageEnglish
	switch strings.ToLower(language) {
	case "indonesian", "ind":
		lang = mangoplus.LanguageIndonesian
	case "spanish", "spa":
		lang = mangoplus.LanguageSpanish
	case "french", "fre":
		lang = mangoplus.LanguageFrench
	case "portuguese", "por":
		lang = mangoplus.LanguagePortugueseBR
	case "russian", "rus":
		lang = mangoplus.LanguageRussian
	case "thai":
		lang = mangoplus.LanguageThai
	case "vietnamese", "vie":
		lang = mangoplus.LanguageVietnamese
	case "german", "ger":
		lang = mangoplus.LanguageGerman
	}

	return &MangaPlus{
		client:   mangoplus.NewPlusClient(mangoplus.DefaultOptions()),
		language: lang,
	}
}

// Search searches for manga on Manga Plus by query.
// Uses the All() endpoint and filters by title name.
func (m *MangaPlus) Search(ctx context.Context, query string) ([]SearchResult, error) {
	allTitles, err := m.client.Manga.All()
	if err != nil {
		return nil, fmt.Errorf("mangaplus all titles: %w", err)
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult

	for _, group := range allTitles {
		for _, title := range group.Titles {
			if strings.Contains(strings.ToLower(title.Name), queryLower) {
				idStr := fmt.Sprintf("%d", title.TitleID)
				results = append(results, SearchResult{
					Title: title.Name,
					URL:   fmt.Sprintf("https://mangaplus.shueisha.co.jp/titles/%d", title.TitleID),
					ID:    idStr,
				})
			}
		}
	}

	slog.Debug("mangaplus search", "query", query, "results", len(results))
	return results, nil
}

// GetLatestChapter fetches the latest chapter for a manga by its title ID.
// The mangaURL parameter should be the title ID (numeric string) or a URL containing the ID.
func (m *MangaPlus) GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error) {
	titleID := extractTitleID(mangaURL)
	if titleID == "" {
		return nil, fmt.Errorf("invalid manga URL or ID: %s", mangaURL)
	}

	detail, err := m.client.Manga.Get(titleID)
	if err != nil {
		return nil, fmt.Errorf("mangaplus get title %s: %w", titleID, err)
	}

	return m.findLatestChapter(detail)
}

// findLatestChapter finds the latest chapter from the title detail.
func (m *MangaPlus) findLatestChapter(detail mangoplus.TitleDetailView) (*ChapterInfo, error) {
	// Try ChapterListV2 first (newer API)
	if len(detail.ChapterListV2) > 0 {
		// Find the chapter with the highest start timestamp
		var latest mangoplus.Chapter
		found := false
		for _, ch := range detail.ChapterListV2 {
			if !found || ch.StartTimeStamp > latest.StartTimeStamp {
				latest = ch
				found = true
			}
		}
		if found {
			return chapterFromMangaPlus(latest), nil
		}
	}

	// Fallback to ChapterListGroup (older API)
	for _, group := range detail.ChapterListGroup {
		// Check LastChapterList first (latest chapters)
		if len(group.LastChapterList) > 0 {
			// Find the one with highest timestamp
			var latest mangoplus.Chapter
			found := false
			for _, ch := range group.LastChapterList {
				if !found || ch.StartTimeStamp > latest.StartTimeStamp {
					latest = ch
					found = true
				}
			}
			if found {
				return chapterFromMangaPlus(latest), nil
			}
		}
		// Check MidChapterList
		if len(group.MidChapterList) > 0 {
			var latest mangoplus.Chapter
			found := false
			for _, ch := range group.MidChapterList {
				if !found || ch.StartTimeStamp > latest.StartTimeStamp {
					latest = ch
					found = true
				}
			}
			if found {
				return chapterFromMangaPlus(latest), nil
			}
		}
		// Check FirstChapterList
		if len(group.FirstChapterList) > 0 {
			var latest mangoplus.Chapter
			found := false
			for _, ch := range group.FirstChapterList {
				if !found || ch.StartTimeStamp > latest.StartTimeStamp {
					latest = ch
					found = true
				}
			}
			if found {
				return chapterFromMangaPlus(latest), nil
			}
		}
	}

	return nil, fmt.Errorf("no chapters found for manga")
}

// chapterFromMangaPlus converts a mangoplus.Chapter to our ChapterInfo.
func chapterFromMangaPlus(ch mangoplus.Chapter) *ChapterInfo {
	chapterName := ch.Name
	numValue, cleanTitle := ParseChapterNumber(chapterName)

	info := &ChapterInfo{
		Number:   cleanTitle,
		URL:      fmt.Sprintf("https://mangaplus.shueisha.co.jp/chapters/%d", ch.ChapterId),
		NumValue: numValue,
	}

	if ch.SubTitle != nil {
		info.Title = *ch.SubTitle
	}

	slog.Debug("mangaplus latest chapter", "chapter", info.Number, "num", info.NumValue)
	return info
}

// extractTitleID extracts the numeric title ID from a URL or returns the ID directly.
func extractTitleID(input string) string {
	input = strings.TrimSpace(input)

	// If it's just a number, return it
	if isNumeric(input) {
		return input
	}

	// Try to extract from URL: https://mangaplus.shueisha.co.jp/titles/100020
	parts := strings.Split(input, "/")
	for i, part := range parts {
		if part == "titles" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return input
}

// isNumeric checks if a string is a valid integer.
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}