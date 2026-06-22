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
	fixture, err := os.ReadFile("../../testdata/kiryuu_search.json")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, client)

	results, err := k.Search(context.Background(), "one piece")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Search() got %d results, want 3", len(results))
	}

	if results[0].Title != "One Piece" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "One Piece")
	}
	if results[0].ID != "one-piece" {
		t.Errorf("results[0].ID = %q, want %q", results[0].ID, "one-piece")
	}
	if results[1].Title != "One Punch Man" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "One Punch Man")
	}
}

func TestKiryuu_GetLatestChapter(t *testing.T) {
	chapterFixture, err := os.ReadFile("../../testdata/kiryuu_chapter.json")
	if err != nil {
		t.Fatalf("load chapter fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/manga/mairimashita-iruma-kun/":
			// Manga page — return 200 to confirm manga exists
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<!DOCTYPE html><html><body></body></html>`))
		case r.URL.Path == "/wp-json/wp/v2/chapter":
			w.Header().Set("Content-Type", "application/json")
			w.Write(chapterFixture)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, client)

	// mangaURL from DB can have a different domain — we extract slug and use baseURL
	oldDomainURL := "https://old-domain.example.com/manga/mairimashita-iruma-kun/"
	ch, err := k.GetLatestChapter(context.Background(), oldDomainURL)
	if err != nil {
		t.Fatalf("GetLatestChapter() error: %v", err)
	}

	if ch.NumValue != 446 {
		t.Errorf("ch.NumValue = %v, want %v", ch.NumValue, 446)
	}
}

func TestKiryuu_SearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, client)

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
	k := NewKiryuu(srv.URL, client)

	_, err := k.GetLatestChapter(context.Background(), "https://old-domain.example.com/manga/nonexistent/")
	if err == nil {
		t.Fatal("GetLatestChapter() expected error, got nil")
	}
}

func TestKiryuu_GetLatestChapterNoChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/manga/test/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<!DOCTYPE html><html><body></body></html>`))
		case r.URL.Path == "/wp-json/wp/v2/chapter":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, client)

	_, err := k.GetLatestChapter(context.Background(), srv.URL+"/manga/test/")
	if err == nil {
		t.Fatal("GetLatestChapter() expected error for empty chapter list, got nil")
	}
}

func TestKiryuu_GetLatestChapterInvalidURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewHTTPClient("test-agent", 0)
	k := NewKiryuu(srv.URL, client)

	// URL without /manga/ path — should fail to extract slug
	_, err := k.GetLatestChapter(context.Background(), "https://example.com/invalid-path/")
	if err == nil {
		t.Fatal("GetLatestChapter() expected error for invalid URL, got nil")
	}
}

func TestHTTPClient_RateLimit(t *testing.T) {
	client := NewHTTPClient("test-agent", 100*time.Millisecond)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	start := time.Now()
	client.Do(req)
	first := time.Since(start)

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	start = time.Now()
	client.Do(req2)
	second := time.Since(start)

	if second < 50*time.Millisecond {
		t.Errorf("rate limit not enforced: second request took %v", second)
	}
	_ = first
}
