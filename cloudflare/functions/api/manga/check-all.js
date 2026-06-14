/**
 * API Handler untuk /api/manga/check-all
 * POST /api/manga/check-all
 */

import { getLatestChapter as kiryuuGetLatest } from "../../../src/kiryuu.js";
import { getLatestChapter as mangaplusGetLatest } from "../../../src/mangaplus.js";
import { sendNewChapter } from "../../../src/telegram.js";

const SOURCES = {
  kiryuu: { getLatest: kiryuuGetLatest },
  mangaplus: { getLatest: mangaplusGetLatest },
};

export async function onRequest(context) {
  const { request, env } = context;

  const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "POST, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
  };

  if (request.method === "OPTIONS") {
    return new Response(null, { headers: corsHeaders });
  }

  if (request.method !== "POST") {
    return new Response(JSON.stringify({ error: "Method not allowed" }), {
      status: 405,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  try {
    const { results: mangaList } = await env.DB.prepare(
      "SELECT * FROM tracked_manga ORDER BY title ASC"
    ).all();

    const checkResults = [];

    for (const manga of mangaList) {
      try {
        const result = await checkManga(env, manga);
        checkResults.push(result);
      } catch (error) {
        console.error("Check failed for manga " + manga.id + ": " + error.message);
        checkResults.push({
          manga_id: manga.id,
          title: manga.title,
          source: manga.source,
          error: error.message,
        });
      }
    }

    return new Response(JSON.stringify(checkResults), {
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  } catch (error) {
    console.error("Check-all error: " + error.message);
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }
}

async function checkManga(env, manga) {
  const adapter = SOURCES[manga.source];
  if (!adapter) {
    return {
      manga_id: manga.id,
      title: manga.title,
      source: manga.source,
      error: "Sumber tidak dikenal: " + manga.source,
    };
  }

  let ch;
  try {
    ch = await adapter.getLatest(manga.url);
  } catch (error) {
    return {
      manga_id: manga.id,
      title: manga.title,
      source: manga.source,
      error: "Gagal ambil chapter: " + error.message,
    };
  }

  const hasNew = !manga.last_chapter_num || ch.numValue > manga.last_chapter_num;

  if (!hasNew) {
    await env.DB.prepare(
      "UPDATE tracked_manga SET last_checked = datetime('now') WHERE id = ?"
    ).bind(manga.id).run();

    return {
      manga_id: manga.id,
      title: manga.title,
      source: manga.source,
      checked: true,
      new_chapter: null,
    };
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

  return {
    manga_id: manga.id,
    title: manga.title,
    source: manga.source,
    checked: true,
    new_chapter: ch.number,
    chapter_url: ch.url,
  };
}