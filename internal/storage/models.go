package storage

import "time"

// TrackedManga merepresentasikan manga yang dilacak.
type TrackedManga struct {
	ID             int64
	Source         string
	SourceID       string
	Title          string
	URL            string
	LastChapter    string
	LastChapterNum float64
	LastChecked    *time.Time
	CreatedAt      time.Time
}

// Notification merepresentasikan notifikasi yang telah dikirim.
type Notification struct {
	ID         int64
	MangaID    int64
	Chapter    string
	ChapterURL string
	SentAt     time.Time
}

// ChapterUpdate merepresentasikan pembaruan chapter.
type ChapterUpdate struct {
	Number   string
	NumValue float64
}