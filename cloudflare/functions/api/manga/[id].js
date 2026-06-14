/**
 * API Handler untuk /api/manga/:id
 * Mendukung: GET (detail), DELETE (remove), POST (check)
 */

import { search as kiryuuSearch, getLatestChapter as kiryuuGetLatest } from "../../../src/kiryuu.js";
import { search as mangaplusSearch, getLatestChapter as mangaplusGetLatest } from "../../../src/mangaplus.js";
import { sendNewChapter } from "../../../src/telegram.js";

const SOURCES = {
  kiryuu: { search: kiryuuSearch, getLatest: kiryuuGetLatest },
  mangaplus: { search: mangaplusSearch, getLatest: mangaplusGetLatest },
};

export async function onRequest(context) {
  const { request, env, params } = context;
  const method = request.method;
  const mangaId = parseInt(params.id, 10);

  const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, POST, DELETE, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
  };

  if (method === "OPTIONS") {
    return new Response(null, { headers: corsHeaders });
  }

  try {
    // GET /api/manga/:id - Get manga detail
    if (method === "GET") {
      const result = await env.DB.prepare(
        "SELECT * FROM tracked_manga WHERE id = ?"
      ).bind(mangaId).first();

      if (!result) {
        return notFound(corsHeaders);
      }
      return jsonResponse(result, corsHeaders);
    }

    // DELETE /api/manga/:id - Remove manga
    if (method === "DELETE") {
      const result = await env.DB.prepare(
        "DELETE FROM tracked_manga WHERE id = ?"
      ).bind(mangaId).run();

      if (result.meta.changes === 0) {
        return notFound(corsHeaders);
      }
      return new Response(null, { status: 204, headers: corsHeaders });
    }

    // POST /api/manga/:id - Check single manga
    if (method === "POST") {
      const manga = await env.DB.prepare(
        "SELECT * FROM tracked_manga WHERE id = ?"
      ).bind(mangaId).first();

      if (!manga) {
        return notFound(corsHeaders);
      }

      const checkResult = await checkMangaInternal(env, manga);
      return jsonResponse(checkResult, corsHeaders);
    }

    return notFound(corsHeaders);
  } catch (error) {
    return serverError(error, corsHeaders);
  }
}

async function checkMangaInternal(env, manga) {
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

  // New chapter found
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

function jsonResponse(data, extraHeaders) {
  return new Response(JSON.stringify(data), {
    headers: { "Content-Type": "application/json", ...extraHeaders },
  });
}

function notFound(corsHeaders) {
  return new Response(JSON.stringify({ error: "Manga tidak ditemukan" }), {
    status: 404,
    headers: { "Content-Type": "application/json", ...corsHeaders },
  });
}

function serverError(error, corsHeaders) {
  console.error("API error: " + error.message);
  return new Response(JSON.stringify({ error: error.message }), {
    status: 500,
    headers: { "Content-Type": "application/json", ...corsHeaders },
  });
}