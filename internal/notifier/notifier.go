package notifier

import "context"

// Notifier mendefinisikan interface untuk mengirim notifikasi chapter baru.
type Notifier interface {
	SendNewChapter(ctx context.Context, n NewChapterNotification) error
}

// NewChapterNotification berisi data yang dibutuhkan untuk mengirim notifikasi chapter baru.
type NewChapterNotification struct {
	MangaTitle      string
	Source          string
	Chapter         string
	ChapterURL      string
	PreviousChapter string
}