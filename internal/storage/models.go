package storage

import "time"

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

type Notification struct {
	ID         int64
	MangaID    int64
	Chapter    string
	ChapterURL string
	SentAt     time.Time
}

type ChapterUpdate struct {
	Number   string
	NumValue float64
}
