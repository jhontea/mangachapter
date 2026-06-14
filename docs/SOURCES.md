# Source Adapters — Technical Reference

Detail teknis untuk mengintegrasikan Kiryuu dan Manga Plus.

---

## Kiryuu (`v6.kiryuu.to`)

### Overview

| Property | Value |
|----------|-------|
| Base URL | `https://v6.kiryuu.to` |
| Method | WordPress REST API |
| Library | Standard library only (`net/http`, `encoding/json`) |
| Auth | None |
| Rate limit | 1-2 detik antar request (configurable) |

> **Catatan:** Domain Kiryuu sering berganti (kiryuu.org, v6.kiryuu.to, dll.). Simpan full URL di DB; base URL dari config.

### URL Patterns

| Action | URL Pattern | Contoh |
|--------|-------------|--------|
| Home | `/` | |
| Manga detail | `/manga/{slug}/` | `/manga/one-piece/` |
| Chapter (new) | `/?chapter={slug}-chapter-{N}` | `/?chapter=one-piece-chapter-1120` |
| Chapter (old) | `/manga/{slug}/chapter-{N}/` | `/manga/one-piece/chapter-123/` |
| Search API | `/wp-json/wp/v2/manga?search={query}` | |
| Chapter API | `/wp-json/wp/v2/chapter?search={term}&per_page=50` | |

### Implementasi Search

Menggunakan WordPress REST API:

```
GET /wp-json/wp/v2/manga?search={url.QueryEscape(query)}&per_page=20
```

Response: JSON array of manga objects with `title.rendered`, `link`, `slug`.

### Implementasi GetLatestChapter

Tema baru Kiryuu hanya render "First Chapter" di HTML; sisanya dimuat via HTMX AJAX.
Kita skip HTML parsing dan gunakan REST API secara langsung.

**Desain URL-agnostic:**
- Saat user menambah manga, full URL disimpan di DB (misal `https://v6.kiryuu.to/manga/one-piece/`)
- Saat `GetLatestChapter`, kita **extract slug** dari URL tersebut
- Semua request REST API menggunakan **`baseURL` dari config**, bukan dari URL di DB
- Jika domain Kiryuu berubah, user cukup update `sources.kiryuu.base_url` di `config.yaml`

**Flow:**
1. Extract slug dari manga URL di DB (`/manga/{slug}/` → `one-piece`)
2. Validasi manga exists: `GET {baseURL}/manga/{slug}/` (HTTP 200 = OK)
3. Search chapters: `GET {baseURL}/wp-json/wp/v2/chapter?search={slug_words}&per_page=50&orderby=date&order=desc`
4. Filter hasil by slug prefix (hanya ambil chapter yang slug-nya dimulai dengan `{manga-slug}-`)
5. Ambil chapter dengan `NumValue` tertinggi

**Expected output:**

```go
ChapterInfo{
    Number:   "Chapter 446",
    Title:    "",
    URL:      "https://v6.kiryuu.to/?chapter=mairimashita-iruma-kun-chapter-446",
    NumValue: 446,
}
```

### Chapter Number Parsing

```go
// Input examples → NumValue
"Chapter 123"              → 123
"Ch. 123"                  → 123
"123"                      → 123
"Chapter 123.5"            → 123.5
"Manga Name Chapter 446"   → 446
"chapter-446"              → 446
```

Regex:

```go
re := regexp.MustCompile(`(?i)(?:chapter|ch\.?)\s*[#]?(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$`)
```

### Error Cases

| Kondisi | Handling |
|---------|----------|
| 404 | Return error "manga not found" |
| Cloudflare challenge | Log error; mungkin perlu cookie/header tambahan |
| Empty chapter list | Return error "no chapters found" |
| REST API unavailable | Return error dengan pesan jelas |
| Slug not found in URL | Return error "no chapters found on page" |

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
  kiryuu_search.json
  kiryuu_chapter.json
  mangaplus_title_detail.json