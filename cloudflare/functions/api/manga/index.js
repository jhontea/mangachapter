/**
 * API Handler untuk /api/manga
 * Mendukung: GET (list), POST (add)
 */

import { search as kiryuuSearch, getLatestChapter as kiryuuGetLatest } from "../../../src/kiryuu.js";
import { search as mangaplusSearch, getLatestChapter as mangaplusGetLatest } from "../../../src/mangaplus.js";

const SOURCES = {
  kiryuu: { search: kiryuuSearch, getLatest: kiryuuGetLatest },
  mangaplus: { search: mangaplusSearch, getLatest: mangaplusGetLatest },
};

export async function onRequest(context) {
  const { request, env } = context;
  const url = new URL(request.url);
  const method = request.method;

  const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, POST, DELETE, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
  };

  if (method === "OPTIONS") {
    return new Response(null, { headers: corsHeaders });
  }

  try {
    // GET /api/manga - List all manga
    if (method === "GET") {
      const results = await env.DB.prepare(
        "SELECT * FROM tracked_manga ORDER BY title ASC"
      ).all();
      return jsonResponse(results.results, corsHeaders);
    }

    // POST /api/manga - Add new manga
    if (method === "POST") {
      const body = await request.json();
      return addManga(env, body, corsHeaders);
    }

    return notFound(corsHeaders);
  } catch (error) {
    return serverError(error, corsHeaders);
  }
}

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

  let ch;
  try {
    ch = await adapter.getLatest(url);
  } catch (error) {
    return new Response(JSON.stringify({ error: "Gagal ambil chapter: " + error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }

  try {
    const result = await env.DB.prepare(
      "INSERT INTO tracked_manga (source, source_id, title, url, last_chapter, last_chapter_num, last_checked) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))"
    ).bind(source, source_id || url, title, url, ch.number, ch.numValue).run();

    return new Response(JSON.stringify({
      id: result.meta.last_row_id,
      source, title, url,
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

function jsonResponse(data, extraHeaders) {
  return new Response(JSON.stringify(data), {
    headers: { "Content-Type": "application/json", ...extraHeaders },
  });
}

function notFound(corsHeaders) {
  return new Response(JSON.stringify({ error: "Not found" }), {
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