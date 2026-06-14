/**
 * API Handler untuk /api/manga/search
 * GET /api/manga/search?source=...&query=...
 */

import { search as kiryuuSearch } from "../../../src/kiryuu.js";
import { search as mangaplusSearch } from "../../../src/mangaplus.js";

const SOURCES = {
  kiryuu: { search: kiryuuSearch },
  mangaplus: { search: mangaplusSearch },
};

export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
  };

  if (request.method === "OPTIONS") {
    return new Response(null, { headers: corsHeaders });
  }

  const source = url.searchParams.get("source");
  const query = url.searchParams.get("query");

  console.log("Search request: source=" + source + " query=" + query);

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
    console.log("Search results: " + results.length + " items");
    return new Response(JSON.stringify(results), {
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  } catch (error) {
    console.error("Search error: " + error.message);
    return new Response(JSON.stringify({ error: "Pencarian gagal: " + error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json", ...corsHeaders },
    });
  }
}