/**
 * Manga Chapter Notifier - Cloudflare Worker
 * REST API + Cron Scheduler
 */

import { search as kiryuuSearch, getLatestChapter as kiryuuGetLatest } from "./kiryuu.js";
import { search as mangaplusSearch, getLatestChapter as mangaplusGetLatest, register as mangaplusRegister } from "./mangaplus.js";
import { sendNewChapter } from "./telegram.js";

// Source adapters
const SOURCES = {
  kiryuu: { search: kiryuuSearch, getLatest: kiryuuGetLatest },
  mangaplus: { search: mangaplusSearch, getLatest: mangaplusGetLatest },
};

/**
 * Main Worker entry point
 */
export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const path = url.pathname;
    const method = request.method;

    // CORS headers
    const corsHeaders = {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, DELETE, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    };

    // Handle CORS preflight
    if (method === "OPTIONS") {
      return new Response(null, { headers: corsHeaders });
    }

    try {
      // GET /api/sources
      if (path === "/api/sources" && method === "GET") {
        return jsonResponse(Object.keys(SOURCES), corsHeaders);
      }

      // GET /api/manga - List all manga
      if (path === "/api/manga" && method === "GET") {
        const results = await env.DB.prepare(
          "SELECT * FROM tracked_manga ORDER BY title ASC"
        ).all();
        return jsonResponse(results.results, corsHeaders);
      }

      // POST /api/manga - Add new manga
      if (path === "/api/manga" && method === "POST") {
        const body = await request.json();
        return addManga(env, body, corsHeaders);
      }

      // POST /api/manga/check-all - Check all manga for updates
      if (path === "/api/manga/check-all" && method === "POST") {
        return checkAllManga(env, corsHeaders);
      }

      // GET /api/manga/search?source=...&query=... - Search manga
      if (path === "/api/manga/search" && method === "GET") {
        const source = url.searchParams.get("source");
        const query = url.searchParams.get("query");
        return searchManga(source, query, corsHeaders);
      }

      // Routes with ID: /api/manga/{id}
      const mangaMatch = path.match(/^\/api\/manga\/(\d+)$/);
      if (mangaMatch) {
        const mangaId = parseInt(mangaMatch[1], 10);

        // GET /api/manga/{id} - Get manga detail
        if (method === "GET") {
          return getManga(env, mangaId, corsHeaders);
        }

        // DELETE /api/manga/{id} - Remove manga
        if (method === "DELETE") {
          return removeManga(env, mangaId, corsHeaders);
        }

        // POST /api/manga/{id} - Check single manga
        if (method === "POST") {
          return checkOneManga(env, mangaId, corsHeaders);
        }
      }

      // 404
      return new Response(JSON.stringify({ error: "Not found" }), {
        status: 404,
        headers: { "Content-Type": "application/json", ...corsHeaders },
      });
    } catch (error) {
      console.error("API error: " + error.message);
      return new Response(JSON.stringify({ error: error.message }), {
        status: 500,
        headers: { "Content-Type": "application/json", ...corsHeaders },
      });
    }
  },

  /**
   * Cron trigger - Check all manga for updates
   */
  async scheduled(event, env, ctx) {
    console.log("Cron: checking all manga for updates");
    await checkAllMangaInternal(env);
    console.log("Cron: check complete");
  },
};

/**
 * Add new manga to tracking list
 */
async function addManga(env, body, corsHeaders) {
  const { source, title, url, source_id } = body;

  if (!source || !title || !url) {
    return new Response(JSON.stringify({ error: "source, title, url wajib diisi" }), {
      status: 400,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  const adapter = SOURCES[source];
  if (!adapter) {
    return new Response(JSON.stringify({ error: "Sumber tidak dikenal: " + source }), {
      status: 400,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  // Get latest chapter as baseline
  let ch;
  try {
    ch = await adapter.getLatest(url);
  } catch (error) {
    return new Response(JSON.stringify({ error: "Gagal ambil chapter: " + error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  // Insert into database
  try {
    const result = await env.DB.prepare(
      "INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num, last_checked) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))"
    ).bind(source, source_id || extractSourceId(source, url), title, url, ch.number, ch.numValue).run();

    return new Response(JSON.stringify({
      id: result.meta.last_row_id,
      source,
      title,
      url,
      last_chapter: ch.number,
      last_chapter_num: ch.numValue,
    }), {
      status: 201,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  } catch (error) {
    if (error.message.includes("UNIQUE constraint")) {
      return new Response(JSON.stringify({ error: "Manga sudah dilacak" }), {
        status: 409,
        headers: { "Content-Type": "application/json", ...corsHeaders },
      });
    }
    throw error;
  }
}

/**
 * Get manga by ID
 */
async function getManga(env, id, corsHeaders) {
  const result = await env.DB.prepare(
    "SELECT * FROM tracked_manga WHERE id = ?"
  ).bind(id).first();

  if (!result) {
    return new Response(JSON.stringify({ error: "Manga tidak ditemukan" }), {
      status: 404,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  return jsonResponse(result, corsHeaders);
}

/**
 * Remove manga from tracking list
 */
async function removeManga(env, id, corsHeaders) {
  const result = await env.DB.prepare(
    "DELETE FROM tracked_manga WHERE id = ?"
  ).bind(id).run();

  if (result.meta.changes === 0) {
    return new Response(JSON.stringify({ error: "Manga tidak ditemukan" }), {
      status: 404,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  return new Response(null, { status: 204, headers: corsHeaders });
}

/**
 * Search manga from source
 */
async function searchManga(source, query, corsHeaders) {
  if (!source || !query) {
    return new Response(JSON.stringify({ error: "Parameter source dan query wajib diisi" }), {
      status: 400,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  const adapter = SOURCES[source];
  if (!adapter) {
    return new Response(JSON.stringify({ error: "Sumber tidak dikenal: " + source }), {
      status: 400,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  try {
    const results = await adapter.search(query);
    return jsonResponse(results, corsHeaders);
  } catch (error) {
    return new Response(JSON.stringify({ error: "Pencarian gagal: " + error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }
}

/**
 * Check single manga for updates
 */
async function checkOneManga(env, id, corsHeaders) {
  const manga = await env.DB.prepare(
    "SELECT * FROM tracked_manga WHERE id = ?"
  ).bind(id).first();

  if (!manga) {
    return new Response(JSON.stringify({ error: "Manga tidak ditemukan" }), {
      status: 404,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  const result = await checkMangaInternal(env, manga);
  return jsonResponse(result, corsHeaders);
}

/**
 * Check all manga for updates
 */
async function checkAllManga(env, corsHeaders) {
  const results = await checkAllMangaInternal(env);
  return jsonResponse(results, corsHeaders);
}

/**
 * Internal: Check all manga
 */
async function checkAllMangaInternal(env) {
  const { results: mangaList } = await env.DB.prepare(
    "SELECT * FROM tracked_manga ORDER BY title ASC"
  ).all();

  const checkResults = [];

  for (const manga of mangaList) {
    try {
      const result = await checkMangaInternal(env, manga);
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

  return checkResults;
}

/**
 * Internal: Check single manga
 */
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

  // Check if new chapter
  const hasNew = !manga.last_chapter_num || ch.numValue > manga.last_chapter_num;

  if (!hasNew) {
    // Update last_checked
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

  // New chapter found - send notification
  const notification = {
    mangaTitle: manga.title,
    source: manga.source,
    chapter: ch.number,
    chapterUrl: ch.url,
    previousChapter: manga.last_chapter,
  };

  // Send Telegram notification
  if (env.TELEGRAM_TOKEN && env.TELEGRAM_CHAT_ID) {
    await sendNewChapter(env.TELEGRAM_TOKEN, env.TELEGRAM_CHAT_ID, notification);
  }

  // Update database
  await env.DB.prepare(
    "UPDATE tracked_manga SET last_chapter = ?, last_chapter_num = ?, last_checked = datetime('now') WHERE id = ?"
  ).bind(ch.number, ch.numValue, manga.id).run();

  // Log notification
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

/**
 * Extract source ID from URL
 */
function extractSourceId(source, url) {
  if (source === "kiryuu") {
    try {
      const parsed = new URL(url);
      const parts = parsed.pathname.replace(/^\/|\/$/g, "").split("/");
      for (let i = 0; i < parts.length; i++) {
        if (parts[i] === "manga" && i + 1 < parts.length) {
          return parts[i + 1];
        }
      }
    } catch (e) {
      // ignore
    }
  }
  return url;
}

/**
 * Create JSON response
 */
function jsonResponse(data, extraHeaders) {
  return new Response(JSON.stringify(data), {
    headers: {
      "Content-Type": "application/json",
      ...extraHeaders,
    },
  });
}