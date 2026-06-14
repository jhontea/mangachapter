package checker

import (
	"testing"

	"project/mangachapter/internal/source"
)

func TestHasNewChapter(t *testing.T) {
	tests := []struct {
		name         string
		storedNum    float64
		fetched      *source.ChapterInfo
		wantNew      bool
	}{
		{
			name:      "new chapter is higher",
			storedNum: 100,
			fetched:   &source.ChapterInfo{Number: "Chapter 101", NumValue: 101},
			wantNew:   true,
		},
		{
			name:      "same chapter",
			storedNum: 100,
			fetched:   &source.ChapterInfo{Number: "Chapter 100", NumValue: 100},
			wantNew:   false,
		},
		{
			name:      "older chapter",
			storedNum: 100,
			fetched:   &source.ChapterInfo{Number: "Chapter 99", NumValue: 99},
			wantNew:   false,
		},
		{
			name:      "no baseline stored",
			storedNum: 0,
			fetched:   &source.ChapterInfo{Number: "Chapter 1", NumValue: 1},
			wantNew:   true,
		},
		{
			name:      "nil fetched",
			storedNum: 100,
			fetched:   nil,
			wantNew:   false,
		},
		{
			name:      "decimal chapter update",
			storedNum: 123.5,
			fetched:   &source.ChapterInfo{Number: "Chapter 124", NumValue: 124},
			wantNew:   true,
		},
		{
			name:      "decimal to decimal",
			storedNum: 123,
			fetched:   &source.ChapterInfo{Number: "Chapter 123.5", NumValue: 123.5},
			wantNew:   true,
		},
		{
			name:      "decimal no change",
			storedNum: 123.5,
			fetched:   &source.ChapterInfo{Number: "Chapter 123.5", NumValue: 123.5},
			wantNew:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasNewChapter(tt.storedNum, tt.fetched)
			if got != tt.wantNew {
				t.Errorf("HasNewChapter(%v, %v) = %v, want %v",
					tt.storedNum, tt.fetched, got, tt.wantNew)
			}
		})
	}
}