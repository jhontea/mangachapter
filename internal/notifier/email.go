package notifier

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// EmailNotifier mengirim notifikasi via SMTP email.
type EmailNotifier struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       []string
}

// NewEmail membuat EmailNotifier baru dengan pengaturan SMTP yang diberikan.
func NewEmail(host string, port int, username, password, from string, to []string) *EmailNotifier {
	return &EmailNotifier{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}
}

// SendNewChapter mengirim notifikasi email tentang chapter baru.
func (e *EmailNotifier) SendNewChapter(ctx context.Context, n NewChapterNotification) error {
	subject := fmt.Sprintf("Chapter baru: %s — %s", n.MangaTitle, n.Chapter)
	body := e.buildBody(n)
	msg := e.buildMessage(subject, body)

	addr := net.JoinHostPort(e.host, fmt.Sprintf("%d", e.port))

	var auth smtp.Auth
	if e.username != "" || e.password != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	// Kirim ke semua penerima
	err := smtp.SendMail(addr, auth, e.from, e.to, []byte(msg))
	if err != nil {
		return fmt.Errorf("kirim email: %w", err)
	}

	slog.Info("email terkirim",
		"penerima", len(e.to),
		"subjek", subject,
	)
	return nil
}

// buildBody membuat isi email teks biasa.
func (e *EmailNotifier) buildBody(n NewChapterNotification) string {
	var sb strings.Builder
	sb.WriteString("Manga chapter baru terdeteksi!\n\n")
	fmt.Fprintf(&sb, "Manga   : %s\n", n.MangaTitle)
	fmt.Fprintf(&sb, "Sumber  : %s\n", n.Source)
	fmt.Fprintf(&sb, "Chapter : %s\n", n.Chapter)
	if n.PreviousChapter != "" {
		fmt.Fprintf(&sb, "Sebelum : %s\n", n.PreviousChapter)
	}
	if n.ChapterURL != "" {
		fmt.Fprintf(&sb, "URL     : %s\n", n.ChapterURL)
	}
	sb.WriteString("\n---\nManga Chapter Notifier\n")
	return sb.String()
}

// buildMessage membuat pesan email lengkap dengan header.
func (e *EmailNotifier) buildMessage(subject, body string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "From: %s\r\n", e.from)
	fmt.Fprintf(&sb, "To: %s\r\n", strings.Join(e.to, ", "))
	fmt.Fprintf(&sb, "Subject: %s\r\n", subject)
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String()
}
