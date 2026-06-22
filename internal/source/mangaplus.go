package source

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	mangaplusBaseAPI   = "https://jumpg-api.tokyo-cdn.com/api"
	mangaplusAppVer    = "300"
	mangaplusOSVer     = "30"
	mangaplusSecretKey = "4Kin9vGg"
)

// MangaPlus mengimplementasikan Source untuk Manga Plus API.
type MangaPlus struct {
	httpClient *HTTPClient
	secret     *string
}

// mangaPlusResponse adalah response API level atas.
type mangaPlusResponse struct {
	Success *mangaPlusSuccess `json:"success"`
	Error   *mangaPlusError   `json:"error"`
}

type mangaPlusSuccess struct {
	TitleDetailView   *mangaPlusTitleDetailView  `json:"titleDetailView"`
	AllTitlesViewV2   *mangaPlusAllTitlesViewV2  `json:"allTitlesViewV2"`
	RegisterationData *mangaPlusRegistrationData `json:"registerationData"`
}

type mangaPlusRegistrationData struct {
	DeviceSecret string `json:"deviceSecret"`
}

type mangaPlusError struct {
	EnglishPopup *mangaPlusPopup `json:"englishPopup"`
}

type mangaPlusPopup struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type mangaPlusAllTitlesViewV2 struct {
	AllTitlesGroup []mangaPlusAllTitlesGroup `json:"allTitlesGroup"`
}

type mangaPlusAllTitlesGroup struct {
	Titles []mangaPlusTitle `json:"titles"`
}

type mangaPlusTitle struct {
	TitleID int    `json:"titleId"`
	Name    string `json:"name"`
}

type mangaPlusTitleDetailView struct {
	ChapterListV2    []mangaPlusChapter          `json:"chapterListV2"`
	ChapterListGroup []mangaPlusChapterListGroup `json:"chapterListGroup"`
}

type mangaPlusChapter struct {
	ChapterID      int     `json:"chapterId"`
	Name           string  `json:"name"`
	SubTitle       *string `json:"subTitle"`
	StartTimeStamp int     `json:"startTimeStamp"`
}

type mangaPlusChapterListGroup struct {
	FirstChapterList []mangaPlusChapter `json:"firstChapterList"`
	MidChapterList   []mangaPlusChapter `json:"midChapterList"`
	LastChapterList  []mangaPlusChapter `json:"lastChapterList"`
}

// NewMangaPlus membuat adapter sumber Manga Plus baru.
func NewMangaPlus(_ string) *MangaPlus {
	mp := &MangaPlus{
		httpClient: NewHTTPClient("MangaPlusShonenJump/"+mangaplusAppVer, 0),
	}
	// Coba registrasi device
	if err := mp.register(); err != nil {
		slog.Warn("registrasi device mangaplus gagal, akan mencoba tanpa autentikasi", "error", err)
	}
	return mp
}

// register mendaftarkan device ke Manga Plus API untuk mendapatkan token secret.
func (m *MangaPlus) register() error {
	deviceToken := md5Hex(fmt.Sprintf("manga-notifier-%d", time.Now().UnixNano()))
	securityKey := md5Hex(deviceToken + mangaplusSecretKey)

	params := map[string]string{
		"device_token": deviceToken,
		"security_key": securityKey,
	}

	var resp mangaPlusResponse
	if err := m.doAPI(context.Background(), "register", params, &resp); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	if resp.Success == nil || resp.Success.RegisterationData == nil {
		return fmt.Errorf("register: data registrasi kosong")
	}

	m.secret = &resp.Success.RegisterationData.DeviceSecret
	slog.Debug("device mangaplus terdaftar")
	return nil
}

// Search mencari manga di Manga Plus berdasarkan query.
func (m *MangaPlus) Search(ctx context.Context, query string) ([]SearchResult, error) {
	params := map[string]string{}
	var resp mangaPlusResponse
	if err := m.doAPI(ctx, "title_list/allV2", params, &resp); err != nil {
		return nil, fmt.Errorf("mangaplus semua judul: %w", err)
	}

	if resp.Success == nil || resp.Success.AllTitlesViewV2 == nil {
		return nil, fmt.Errorf("tidak ditemukan judul")
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult

	for _, group := range resp.Success.AllTitlesViewV2.AllTitlesGroup {
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

	slog.Debug("pencarian mangaplus", "query", query, "hasil", len(results))
	return results, nil
}

// GetLatestChapter mengambil chapter terbaru untuk manga berdasarkan title ID.
func (m *MangaPlus) GetLatestChapter(ctx context.Context, mangaURL string) (*ChapterInfo, error) {
	titleID := extractTitleID(mangaURL)
	if titleID == "" {
		return nil, fmt.Errorf("URL atau ID manga tidak valid: %s", mangaURL)
	}

	params := map[string]string{
		"title_id": titleID,
	}

	var resp mangaPlusResponse
	if err := m.doAPI(ctx, "title_detailV3", params, &resp); err != nil {
		return nil, fmt.Errorf("mangaplus ambil judul %s: %w", titleID, err)
	}

	if resp.Success == nil || resp.Success.TitleDetailView == nil {
		return nil, fmt.Errorf("tidak ada detail untuk manga id %s", titleID)
	}

	return m.findLatestChapter(*resp.Success.TitleDetailView)
}

// doAPI melakukan request API ke Manga Plus API.
func (m *MangaPlus) doAPI(ctx context.Context, apiPath string, params map[string]string, result *mangaPlusResponse) error {
	method := http.MethodGet
	if apiPath == "register" {
		method = http.MethodPut
	}

	u, _ := url.Parse(mangaplusBaseAPI)
	u = u.JoinPath(apiPath)

	q := u.Query()
	q.Set("os", "android")
	q.Set("os_ver", mangaplusOSVer)
	q.Set("app_ver", mangaplusAppVer)
	q.Set("format", "json")
	if m.secret != nil {
		q.Set("secret", *m.secret)
	}
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return fmt.Errorf("buat request: %w", err)
	}
	req.Header.Set("User-Agent", "MangaPlusShonenJump/"+mangaplusAppVer)
	req.Header.Set("Accept", "*/*")

	resp, err := m.httpClient.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("baca response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil && result.Error.EnglishPopup != nil {
		return fmt.Errorf("error API: %s (%s)", result.Error.EnglishPopup.Subject, result.Error.EnglishPopup.Body)
	}

	return nil
}

// findLatestChapter mencari chapter terbaru dari detail judul.
func (m *MangaPlus) findLatestChapter(detail mangaPlusTitleDetailView) (*ChapterInfo, error) {
	if len(detail.ChapterListV2) > 0 {
		var latest mangaPlusChapter
		found := false
		for _, ch := range detail.ChapterListV2 {
			if !found || ch.StartTimeStamp > latest.StartTimeStamp {
				latest = ch
				found = true
			}
		}
		if found {
			return chapterFromMangaPlusV2(latest), nil
		}
	}

	for _, group := range detail.ChapterListGroup {
		if ch := findLatestFromGroup(group); ch != nil {
			return ch, nil
		}
	}

	return nil, fmt.Errorf("tidak ditemukan chapter untuk manga")
}

func findLatestFromGroup(group mangaPlusChapterListGroup) *ChapterInfo {
	lists := [][]mangaPlusChapter{
		group.LastChapterList,
		group.MidChapterList,
		group.FirstChapterList,
	}
	for _, list := range lists {
		if len(list) == 0 {
			continue
		}
		var latest mangaPlusChapter
		found := false
		for _, ch := range list {
			if !found || ch.StartTimeStamp > latest.StartTimeStamp {
				latest = ch
				found = true
			}
		}
		if found {
			return chapterFromMangaPlusV2(latest)
		}
	}
	return nil
}

func chapterFromMangaPlusV2(ch mangaPlusChapter) *ChapterInfo {
	numValue, cleanTitle := ParseChapterNumber(ch.Name)
	info := &ChapterInfo{
		Number:   cleanTitle,
		URL:      fmt.Sprintf("https://mangaplus.shueisha.co.jp/viewer/%d", ch.ChapterID),
		NumValue: numValue,
	}
	if ch.SubTitle != nil {
		info.Title = *ch.SubTitle
	}
	slog.Debug("chapter terbaru mangaplus", "chapter", info.Number, "num", info.NumValue)
	return info
}

func extractTitleID(input string) string {
	input = strings.TrimSpace(input)
	if isNumeric(input) {
		return input
	}
	parts := strings.Split(input, "/")
	for i, part := range parts {
		if part == "titles" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return input
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
