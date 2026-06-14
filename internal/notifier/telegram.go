package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	telegramAPIBase = "https://api.telegram.org/bot%s/sendMessage"
	httpTimeout     = 10 * time.Second
)

// TelegramNotifier mengirim notifikasi via Telegram Bot API.
type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

// NewTelegram membuat TelegramNotifier baru.
func NewTelegram(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: httpTimeout},
	}
}

// SendNewChapter mengirim pesan Telegram tentang chapter baru.
func (t *TelegramNotifier) SendNewChapter(ctx context.Context, n NewChapterNotification) error {
	text := t.buildMessage(n)

	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload telegram: %w", err)
	}

	url := fmt.Sprintf(telegramAPIBase, t.botToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buat request telegram: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("kirim pesan telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error API telegram (status %d): %s", resp.StatusCode, string(respBody))
	}

	slog.Info("notifikasi telegram terkirim",
		"chat_id", t.chatID,
		"manga", n.MangaTitle,
		"chapter", n.Chapter,
	)
	return nil
}

// buildMessage membuat pesan Telegram dengan format HTML.
func (t *TelegramNotifier) buildMessage(n NewChapterNotification) string {
	var sb bytes.Buffer
	sb.WriteString("📚 <b>Chapter Baru!</b>\n\n")
	sb.WriteString(fmt.Sprintf("📖 <b>%s</b>\n", n.MangaTitle))
	sb.WriteString(fmt.Sprintf("🔗 Sumber: %s\n", n.Source))
	sb.WriteString(fmt.Sprintf("📄 Chapter: <b>%s</b>\n", n.Chapter))
	if n.PreviousChapter != "" {
		sb.WriteString(fmt.Sprintf("⬅️ Sebelumnya: %s\n", n.PreviousChapter))
	}
	if n.ChapterURL != "" {
		sb.WriteString(fmt.Sprintf("\n🔗 <a href=\"%s\">Baca Chapter</a>\n", n.ChapterURL))
	}
	return sb.String()
}