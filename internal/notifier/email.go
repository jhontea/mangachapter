package notifier

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// EmailNotifier sends notifications via SMTP email.
type EmailNotifier struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       []string
}

// NewEmail creates a new EmailNotifier with the given SMTP settings.
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

// SendNewChapter sends an email notification about a new chapter.
func (e *EmailNotifier) SendNewChapter(ctx context.Context, n NewChapterNotification) error {
	subject := fmt.Sprintf("New chapter: %s — %s", n.MangaTitle, n.Chapter)
	body := e.buildBody(n)
	msg := e.buildMessage(subject, body)

	addr := net.JoinHostPort(e.host, fmt.Sprintf("%d", e.port))

	var auth smtp.Auth
	if e.username != "" || e.password != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	// Send to all recipients
	err := smtp.SendMail(addr, auth, e.from, e.to, []byte(msg))
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	slog.Info("email sent",
		"recipients", len(e.to),
		"subject", subject,
	)
	return nil
}

// buildBody creates the plain text email body.
func (e *EmailNotifier) buildBody(n NewChapterNotification) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Manga chapter baru terdeteksi!\n\n"))
	sb.WriteString(fmt.Sprintf("Manga   : %s\n", n.MangaTitle))
	sb.WriteString(fmt.Sprintf("Source  : %s\n", n.Source))
	sb.WriteString(fmt.Sprintf("Chapter : %s\n", n.Chapter))
	if n.PreviousChapter != "" {
		sb.WriteString(fmt.Sprintf("Sebelum : %s\n", n.PreviousChapter))
	}
	if n.ChapterURL != "" {
		sb.WriteString(fmt.Sprintf("URL     : %s\n", n.ChapterURL))
	}
	sb.WriteString("\n---\nManga Chapter Notifier\n")
	return sb.String()
}

// buildMessage creates the full email message with headers.
func (e *EmailNotifier) buildMessage(subject, body string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", e.from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.to, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String()
}