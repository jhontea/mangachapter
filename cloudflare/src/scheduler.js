/**
 * Scheduler Worker - Cron Trigger
 * Berjalan setiap jam untuk cek update manga + kirim notifikasi Telegram
 * 
 * Deploy terpisah: npx wrangler deploy src/scheduler.js --name mangachapter-scheduler
 */

import { getLatestChapter as kiryuuGetLatest } from "./kiryuu.js";
import { getLatestChapter as mangaplusGetLatest } from "./mangaplus.js";
import { sendNewChapter } from "./telegram.js";

const SOURCES = {
  kiryuu: { getLatest: kiryuuGetLatest },
  mangaplus: { getLatest: mangaplusGetLatest },
};

export default {
  async scheduled(event, env, ctx) {
    console.log("Cron: checking all manga for updates");

    try {
      const { results: mangaList } = await env.DB.prepare(
        "SELECT * FROM tracked_manga ORDER BY title ASC"
      ).all();

      let checked = 0;
      let newChapters = 0;
      let errors = 0;

      for (const manga of mangaList) {
        try {
          const result = await checkManga(env, manga);
          checked++;
          if (result.new_chapter) {
            newChapters++;
          }
        } catch (error) {
          errors++;
          console.error("Check failed for manga " + manga.id + ": " + error.message);
        }
      }

      console.log("Cron: complete - checked=" + checked + " new=" + newChapters + " errors=" + errors);
    } catch (error) {
      console.error("Cron: fatal error - " + error.message);
    }
  },
};

async function checkManga(env, manga) {
  const adapter = SOURCES[manga.source];
  if (!adapter) {
    return { manga_id: manga.id, error: "Unknown source: " + manga.source };
  }

  let ch;
  try {
    ch = await adapter.getLatest(manga.url);
  } catch (error) {
    return { manga_id: manga.id, error: "Fetch failed: " + error.message };
  }

  const hasNew = !manga.last_chapter_num || ch.numValue > manga.last_chapter_num;

  if (!hasNew) {
    await env.DB.prepare(
      "UPDATE tracked_manga SET last_checked = datetime('now') WHERE id = ?"
    ).bind(manga.id).run();
    return { manga_id: manga.id, new_chapter: null };
  }

  const notification = {
    mangaTitle: manga.title,
    source: manga.source,
    chapter: ch.number,
    chapterUrl: ch.url,
    previousChapter: manga.last_chapter,
  };

  if (env.TELEGRAM_TOKEN && env.TELEGRAM_CHAT_ID) {
    await sendNewChapter(env.TELEGRAM_TOKEN, env.TELEGRAM_CHAT_ID, notification);
  }

  await env.DB.prepare(
    "UPDATE tracked_manga SET last_chapter = ?, last_chapter_num = ?, last_checked = datetime('now') WHERE id = ?"
  ).bind(ch.number, ch.numValue, manga.id).run();

  await env.DB.prepare(
    "INSERT INTO notifications (manga_id, chapter, chapter_url) VALUES (?, ?, ?)"
  ).bind(manga.id, ch.number, ch.url).run();

  return { manga_id: manga.id, new_chapter: ch.number };
}