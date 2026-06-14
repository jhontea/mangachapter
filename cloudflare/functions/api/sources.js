/**
 * API Handler untuk /api/sources
 * GET /api/sources
 */

export async function onRequest(context) {
  const { request } = context;

  const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
  };

  if (request.method === "OPTIONS") {
    return new Response(null, { headers: corsHeaders });
  }

  const sources = ["kiryuu", "mangaplus"];

  return new Response(JSON.stringify(sources), {
    headers: { "Content-Type": "application/json", ...corsHeaders },
  });
}