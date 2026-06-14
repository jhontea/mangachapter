package notifier

import "context"

// Notifier defines the interface for sending new chapter notifications.
type Notifier interface {
	SendNewChapter(ctx context.Context, n NewChapterNotification) error
}

// NewChapterNotification represents the data needed to send a new chapter notification.
type NewChapterNotification struct {
	MangaTitle      string
	Source          string
	Chapter         string
	ChapterURL      string
	PreviousChapter string
}