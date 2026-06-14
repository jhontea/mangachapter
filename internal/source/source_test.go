package source

import (
	"testing"
)

func TestParseChapterNumber(t *testing.T) {
	tests := []struct {
		input     string
		wantNum   float64
		wantTitle string
	}{
		{"Chapter 123", 123, "Chapter 123"},
		{"Ch. 123", 123, "Chapter 123"},
		{"Ch 123", 123, "Chapter 123"},
		{"chapter 123.5", 123.5, "Chapter 123.5"},
		{"123", 123, "Chapter 123"},
		{"Chapter 123 - Special", 123, "Chapter 123"},
		{"Chapter 1130", 1130, "Chapter 1130"},
		{"Chapter 0.5", 0.5, "Chapter 0.5"},
		{"  Chapter 42  ", 42, "Chapter 42"},
		{"No chapter here", 0, "No chapter here"},
		{"", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotNum, gotTitle := ParseChapterNumber(tt.input)
			if gotNum != tt.wantNum {
				t.Errorf("ParseChapterNumber(%q) num = %v, want %v", tt.input, gotNum, tt.wantNum)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("ParseChapterNumber(%q) title = %q, want %q", tt.input, gotTitle, tt.wantTitle)
			}
		})
	}
}

func TestExtractSlugFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Standard Kiryuu URLs
		{"https://v6.kiryuu.to/manga/one-piece/", "one-piece"},
		{"https://v6.kiryuu.to/manga/one-piece", "one-piece"},
		{"https://v6.kiryuu.to/manga/jujutsu-kaisen/", "jujutsu-kaisen"},
		// Old domain — slug extraction still works
		{"https://old-domain.example.com/manga/mairimashita-iruma-kun/", "mairimashita-iruma-kun"},
		// Any domain with /manga/ path
		{"https://example.com/manga/just-a-slug/", "just-a-slug"},
		// No /manga/ path — returns empty
		{"https://example.com/no-manga-path/", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractSlugFromURL(tt.input)
			if got != tt.want {
				t.Errorf("extractSlugFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTitleID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"100020", "100020"},
		{"https://mangaplus.shueisha.co.jp/titles/100020", "100020"},
		{"https://mangaplus.shueisha.co.jp/titles/100020/", "100020"},
		{"https://mangaplus.shueisha.co.jp/titles/100026/overview", "100026"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractTitleID(tt.input)
			if got != tt.want {
				t.Errorf("extractTitleID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}