package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestKiryuu_Search(t *testing.T) {
	// Load fixture
	fixture, err := os.ReadFile("../../testdata/kiryuu_search.html")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, "test-agent", client)

	results, err := k.Search(context.Background(), "one piece")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Search() got %d results, want 3", len(results))
	}

	// Check first result
	if results[0].Title != "One Piece" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "One Piece")
	}
	if results[0].ID != "one-piece" {
		t.Errorf("results[0].ID = %q, want %q", results[0].ID, "one-piece")
	}
	if results[0].URL != "https://v6.kiryuu.to/manga/one-piece/" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://v6.kiryuu.to/manga/one-piece/")
	}

	// Check second result
	if results[1].Title != "One Punch Man" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "One Punch Man")
	}
	if results[1].ID != "one-punch-man" {
		t.Errorf("results[1].ID = %q, want %q", results[1].ID, "one-punch-man")
	}
}

func TestKiryuu_GetLatestChapter(t *testing.T) {
	// Load fixture
	fixture, err := os.ReadFile("../../testdata/kiryuu_manga_detail.html")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, "test-agent", client)

	ch, err := k.GetLatestChapter(context.Background(), srv.URL+"/manga/one-piece/")
	if err != nil {
		t.Fatalf("GetLatestChapter() error: %v", err)
	}

	if ch.Number != "Chapter 1130" {
		t.Errorf("ch.Number = %q, want %q", ch.Number, "Chapter 1130")
	}
	if ch.NumValue != 1130 {
		t.Errorf("ch.NumValue = %v, want %v", ch.NumValue, 1130)
	}
	if ch.URL != "https://v6.kiryuu.to/manga/one-piece/chapter-1130/" {
		t.Errorf("ch.URL = %q, want %q", ch.URL, "https://v6.kiryuu.to/manga/one-piece/chapter-1130/")
	}
}

func TestKiryuu_SearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, "test-agent", client)

	_, err := k.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("Search() expected error, got nil")
	}
}

func TestKiryuu_GetLatestChapterHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, "test-agent", client)

	_, err := k.GetLatestChapter(context.Background(), srv.URL+"/manga/nonexistent/")
	if err == nil {
		t.Fatal("GetLatestChapter() expected error, got nil")
	}
}

func TestKiryuu_GetLatestChapterNoChapters(t *testing.T) {
	// HTML with no chapter list
	html := `<!DOCTYPE html><html><body><div class="eplister"><ul></ul></div></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, "test-agent", client)

	_, err := k.GetLatestChapter(context.Background(), srv.URL+"/manga/test/")
	if err == nil {
		t.Fatal("GetLatestChapter() expected error for empty chapter list, got nil")
	}
}

func TestHTTPClient_RateLimit(t *testing.T) {
	client := NewHTTPClient("test-agent", 100*time.Millisecond)

	// First request should be fast
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	start := time.Now()
	client.Do(req)
	first := time.Since(start)

	// Second request should be rate-limited
	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	start = time.Now()
	client.Do(req2)
	second := time.Since(start)

	// Second request should take at least ~100ms due to rate limiting
	if second < 50*time.Millisecond {
		t.Errorf("rate limit not enforced: second request took %v", second)
	}
	_ = first // suppress unused
}