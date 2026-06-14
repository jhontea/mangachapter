/**
 * Telegram Notifier
 * Mengirim notifikasi chapter baru via Telegram Bot API
 */

const TELEGRAM_API_BASE = "https://api.telegram.org/bot";

export async function sendNewChapter(botToken, chatId, notification) {
  const text = buildMessage(notification);
  const payload = {
    chat_id: chatId,
    text: text,
    parse_mode: "HTML",
  };
  const url = TELEGRAM_API_BASE + botToken + "/sendMessage";

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      const errorText = await response.text();
      console.error("Telegram API error: " + response.status + " - " + errorText);
      return false;
    }
    console.log("Notifikasi terkirim: " + notification.mangaTitle + " - " + notification.chapter);
    return true;
  } catch (error) {
    console.error("Gagal kirim Telegram: " + error.message);
    return false;
  }
}

function buildMessage(n) {
  var lines = [];
  lines.push("<b>Chapter Baru!</b>");
  lines.push("");
  lines.push("<b>" + escapeHtml(n.mangaTitle) + "</b>");
  lines.push("Sumber: " + escapeHtml(n.source));
  lines.push("Chapter: <b>" + escapeHtml(n.chapter) + "</b>");
  if (n.previousChapter) {
    lines.push("Sebelumnya: " + escapeHtml(n.previousChapter));
  }
  if (n.chapterUrl) {
    lines.push("");
    lines.push('<a href="' + n.chapterUrl + '">Baca Chapter</a>');
  }
  return lines.join("\n");
}

function escapeHtml(text) {
  if (!text) return "";
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}