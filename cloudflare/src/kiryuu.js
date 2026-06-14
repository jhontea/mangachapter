/**
 * Kiryuu Source Adapter
 * Mengambil data manga dari Kiryuu menggunakan WordPress REST API
 */

const KIRYUU_BASE_URL = "https://v6.kiryuu.to";
const USER_AGENT = "MangaChapterNotifier/1.0 (+personal use)";

// Regex untuk parse nomor chapter
const CHAPTER_RE = /(?:chapter|ch\.?)\s*[#]?(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$|^#(\d+(?:\.\d+)?)/i;

/**
 * Cari manga di Kiryuu
 */
export async function search(query) {
  const searchUrl = KIRYUU_BASE_URL + "/wp-json/wp/v2/manga?search=" + encodeURIComponent(query) + "&per_page=20";

  try {
    const response = await fetch(searchUrl, {
      headers: {
        "Accept": "application/json",
        "User-Agent": USER_AGENT,
      },
    });

    if (!response.ok) {
      throw new Error("Search HTTP " + response.status);
    }

    const mangas = await response.json();
    const results = [];

    for (const m of mangas) {
      if (m.title && m.title.rendered && m.link) {
        results.push({
          title: m.title.rendered,
          url: m.link,
          id: m.slug,
        });
      }
    }

    return results;
  } catch (error) {
    console.error("Kiryuu search error: " + error.message);
    throw error;
  }
}

/**
 * Ambil chapter terbaru untuk manga tertentu
 */
export async function getLatestChapter(mangaUrl) {
  const slug = extractSlugFromURL(mangaUrl);
  if (!slug) {
    throw new Error("Tidak bisa mengekstrak slug dari URL: " + mangaUrl);
  }

  // Verifikasi manga ada
  const mangaPageUrl = KIRYUU_BASE_URL + "/manga/" + slug + "/";
  const pageResponse = await fetch(mangaPageUrl, {
    headers: { "Accept-Language": "id-ID,id;q=0.9,en;q=0.8" },
  });

  if (pageResponse.status === 404) {
    throw new Error("Manga tidak ditemukan: " + slug);
  }
  if (!pageResponse.ok) {
    throw new Error("Manga page HTTP " + pageResponse.status);
  }

  // Ambil chapter terbaru via REST API
  return getLatestChapterFromAPI(slug);
}

/**
 * Ambil chapter terbaru via WordPress REST API
 */
async function getLatestChapterFromAPI(mangaSlug) {
  const searchTerm = extractSearchTerm(mangaSlug);
  const apiUrl = KIRYUU_BASE_URL + "/wp-json/wp/v2/chapter?search=" + encodeURIComponent(searchTerm) + "&per_page=50&orderby=date&order=desc";

  const response = await fetch(apiUrl, {
    headers: { "Accept": "application/json" },
  });

  if (!response.ok) {
    throw new Error("Chapter API HTTP " + response.status);
  }

  const chapters = await response.json();
  if (!chapters || chapters.length === 0) {
    throw new Error("Tidak ditemukan chapter untuk manga: " + mangaSlug);
  }

  // Filter chapter yang milik manga ini
  const mangaSlugPrefix = mangaSlug.replace(/-$/, "") + "-";
  let bestChapter = null;

  for (const ch of chapters) {
    if (!ch.slug || !ch.slug.startsWith(mangaSlugPrefix)) {
      continue;
    }
    const info = parseChapterFromLink(ch.link, ch.title ? ch.title.rendered : "");
    if (!info || info.numValue === 0) {
      continue;
    }
    if (!bestChapter || info.numValue > bestChapter.numValue) {
      bestChapter = info;
    }
  }

  if (!bestChapter) {
    throw new Error("Tidak ditemukan chapter yang cocok untuk: " + mangaSlug);
  }

  return bestChapter;
}

/**
 * Parse info chapter dari link dan teks
 */
function parseChapterFromLink(href, text) {
  if (!href && !text) {
    return null;
  }

  href = (href || "").trim();

  // Coba parse nomor chapter dari teks
  let numValue = 0;
  let cleanTitle = text || "";

  const textMatch = text ? text.match(CHAPTER_RE) : null;
  if (textMatch) {
    const numStr = textMatch[1] || textMatch[2] || textMatch[3];
    if (numStr) {
      numValue = parseFloat(numStr);
      cleanTitle = "Chapter " + numStr;
    }
  }

  // Jika gagal, coba dari URL
  if (numValue === 0 && href) {
    const chapterIdx = href.indexOf("/chapter-");
    if (chapterIdx >= 0) {
      const urlMatch = href.substring(chapterIdx).match(CHAPTER_RE);
      if (urlMatch) {
        const numStr = urlMatch[1] || urlMatch[2] || urlMatch[3];
        if (numStr) {
          numValue = parseFloat(numStr);
          cleanTitle = "Chapter " + numStr;
        }
      }
    }
  }

  return {
    number: cleanTitle,
    url: href,
    numValue: numValue,
  };
}

/**
 * Ekstrak slug dari URL Kiryuu
 */
function extractSlugFromURL(mangaUrl) {
  try {
    const url = new URL(mangaUrl);
    const parts = url.pathname.replace(/^\/|\/$/g, "").split("/");
    for (let i = 0; i < parts.length; i++) {
      if (parts[i] === "manga" && i + 1 < parts.length) {
        return parts[i + 1];
      }
    }
  } catch (e) {
    // URL tidak valid
  }
  return "";
}

/**
 * Ekstrak istilah pencarian pendek dari slug
 */
function extractSearchTerm(slug) {
  const parts = slug.split("-");
  if (parts.length >= 3) {
    return parts.slice(0, 3).join(" ");
  }
  return parts.join(" ");
}