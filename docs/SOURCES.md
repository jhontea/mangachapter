# Source Adapters — Technical Reference

Detail teknis untuk mengintegrasikan Kiryuu dan Manga Plus.

---

## Kiryuu (`v6.kiryuu.to`)

### Overview

| Property | Value |
|----------|-------|
| Base URL | `https://v6.kiryuu.to` |
| Method | HTML scraping |
| Library | `github.com/PuerkitoBio/goquery` |
| Auth | None |
| Rate limit | 1-2 detik antar request (configurable) |

### URL Patterns

| Action | URL Pattern | Contoh |
|--------|-------------|--------|
| Home | `/` | |
| Search | `/?s={query}` | `/?s=one+piece` |
| Manga detail | `/manga/{slug}/` | `/manga/one-piece/` |
| Chapter | `/manga/{slug}/{chapter-slug}/` | `/manga/one-piece/chapter-123/` |

> **Catatan:** Domain Kiryuu sering berganti (kiryuu.org, v6.kiryuu.to, dll.). Simpan full URL di DB; base URL dari config.

### Implementasi Search

```
GET {base_url}/?s={url.QueryEscape(query)}
Headers:
  User-Agent: {config.sources.kiryuu.user_agent}
  Accept-Language: id-ID,id;q=0.9,en;q=0.8
```

**Parsing (placeholder — INSPECT SAAT IMPLEMENTASI):**

Selector perlu diverifikasi dengan inspect HTML aktual. Pola umum situs manga WordPress theme:

```go
// Contoh selector — GANTI setelah inspect
doc.Find(".bsx a").Each(func(i int, s *goquery.Selection) {
    title := strings.TrimSpace(s.Find(".tt").Text())
    href, _ := s.Attr("href")
    // extract slug from href
})
```

**TODO saat Fase 2:**
1. Fetch `/?s=one+piece`
2. Simpan HTML sample ke `testdata/kiryuu_search.html`
3. Tentukan selector final
4. Update section ini

### Implementasi GetLatestChapter

```
GET {manga_url}
Parse chapter list → ambil entry pertama (terbaru)
```

**Expected output:**

```go
ChapterInfo{
    Number:   "Chapter 123",
    Title:    "", // jika ada subtitle
    URL:      "https://v6.kiryuu.to/manga/one-piece/chapter-123/",
    NumValue: 123,
}
```

### Chapter Number Parsing

```go
// Input examples → NumValue
"Chapter 123"     → 123
"Ch. 123"         → 123
"123"             → 123
"Chapter 123.5"   → 123.5
"Chapter 123 - Special" → 123 (strip suffix) atau 0 + string compare
```

Regex suggestion:

```go
re := regexp.MustCompile(`(?i)(?:chapter|ch\.?)\s*(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$`)
```

### Error Cases

| Kondisi | Handling |
|---------|----------|
| 404 | Return error "manga not found" |
| Cloudflare challenge | Log error; mungkin perlu cookie/header tambahan |
| Empty chapter list | Return error "no chapters found" |
| HTML structure changed | Log + return parse error |

### Referensi

- Node.js scraper: `@boboiboyturuu_nih/kiryuu-scraper` (npm) — lihat selector yang dipakai
- Domain lama: `kiryuu.org` — struktur mungkin mirip

---

## Manga Plus (`mangaplus.shueisha.co.jp`)

### Overview

| Property | Value |
|----------|-------|
| Base URL | `https://mangaplus.shueisha.co.jp` |
| Method | Unofficial REST API |
| Library | `github.com/luevano/mangoplus` (Go) |
| Auth | None for public titles |
| Stability | **Tidak stabil** — API tidak documented officially |

### API Base

Manga Plus web app memanggil API internal, typically:

```
https://mangaplus.shueisha.co.jp/api/...
```

Library `luevano/mangoplus` sudah wrap endpoint-endpoint ini.

### Title ID

Setiap manga punya numeric ID. Cara mendapatkan ID:

1. Buka halaman manga di browser
2. Inspect network tab → cari request `title_detail` atau similar
3. Atau gunakan search API

Contoh known IDs (verify saat implementasi):

| Manga | Title ID (approx) |
|-------|-------------------|
| One Piece | 100020 |
| Jujutsu Kaisen | 100026 |
| Chainsaw Man | 100037 |

### GetLatestChapter via luevano/mangoplus

```go
import "github.com/luevano/mangoplus"

c := mangoplus.NewPlusClient(mangoplus.DefaultOptions())
detail, err := c.Manga.Get(titleID)

// Chapter lists grouped — ambil yang terbaru
for _, group := range detail.ChapterListGroup {
    for _, ch := range group.LatestChapterList {
        // ch.Name, ch.ChapterId, dll.
    }
    for _, ch := range group.FirstChapterList {
        // first chapters
    }
}
```

> **Limitation:** Hanya `FirstChapterList` dan `LatestChapterList` — tidak ada semua chapter di antaranya.

### Updates Page

URL: `https://mangaplus.shueisha.co.jp/updates`

Halaman ini menampilkan manga dengan update terbaru. Bisa dipakai untuk:

- Discovery (manga apa saja yang baru update)
- **Bukan** untuk track manga spesifik — tetap pakai `title_detail` per manga

### Search

Library mungkin expose search; jika tidak, reverse-engineer dari network tab saat search di web UI.

**TODO saat Fase 2:**
1. Test `luevano/mangoplus` dengan Go 1.22+
2. Document method untuk search
3. Fallback HTTP manual jika library broken

### Language

Manga Plus support multi-language. Set via client options:

- `eng` — English
- `ind` — Indonesian (jika tersedia)
- `spa`, `por`, dll.

Config: `sources.mangaplus.language`

### Error Cases

| Kondisi | Handling |
|---------|----------|
| Invalid title ID | API error → "manga not found" |
| API structure changed | Log + return error; mungkin perlu update library |
| No latest chapter | Title completed atau one-shot — handle gracefully |
| Rate limit | Jarang; tetap respect delay |

### Legal

Manga Plus ToS — API unofficial. Gunakan untuk personal notification only; jangan redistribute content.

---

## Shared: HTTP Client

```go
type HTTPClient struct {
    client    *http.Client
    userAgent string
    rateLimit time.Duration
    lastReq   time.Time
    mu        sync.Mutex
}

func (h *HTTPClient) Do(req *http.Request) (*http.Response, error) {
    h.mu.Lock()
    elapsed := time.Since(h.lastReq)
    if elapsed < h.rateLimit {
        time.Sleep(h.rateLimit - elapsed)
    }
    h.lastReq = time.Now()
    h.mu.Unlock()

    req.Header.Set("User-Agent", h.userAgent)
    return h.client.Do(req)
}
```

Default timeout: 30s per request.

---

## Testing Strategy

1. **Mock HTML/API** — simpan fixture di `testdata/`
2. **httptest.Server** — serve fixture untuk unit test
3. **Integration test** — tag `//go:build integration`, skip di CI default
4. Jangan hit live site di unit test

```
internal/source/
  kiryuu_test.go
  mangaplus_test.go
testdata/
  kiryuu_search.html
  kiryuu_manga_detail.html
  mangaplus_title_detail.json
```
