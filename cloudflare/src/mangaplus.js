/**
 * Manga Plus Source Adapter
 * Mengambil data manga dari Manga Plus API (unofficial)
 * 
 * CATATAN: API MangaPlus tidak stabil dan sering berubah.
 * Search mungkin tidak berfungsi jika API berubah.
 */

const MANGAPLUS_API_BASE = "https://jumpg-api.tokyo-cdn.com/api";
const APP_VER = "300";
const OS_VER = "30";
const SECRET_KEY = "4Kin9vGg";

let deviceSecret = null;

/**
 * Generate hash untuk device token
 */
function simpleHash(input) {
  const encoder = new TextEncoder();
  const data = encoder.encode(input);
  return crypto.subtle.digest("SHA-256", data).then((hash) => {
    const hashArray = Array.from(new Uint8Array(hash));
    return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("").substring(0, 32);
  });
}

/**
 * Register device untuk mendapatkan secret token
 */
export async function register() {
  try {
    const timestamp = Date.now().toString();
    const deviceToken = await simpleHash("manga-notifier-" + timestamp);
    const securityKey = await simpleHash(deviceToken + SECRET_KEY);

    const params = new URLSearchParams({
      device_token: deviceToken,
      security_key: securityKey,
    });

    const url = MANGAPLUS_API_BASE + "/register?" + params.toString() + "&os=android&os_ver=" + OS_VER + "&app_ver=" + APP_VER + "&format=json";

    const response = await fetch(url, {
      method: "PUT",
      headers: { "User-Agent": "MangaPlusShonenJump/" + APP_VER },
    });

    if (!response.ok) {
      throw new Error("Register HTTP " + response.status);
    }

    const data = await response.json();
    if (data.success && data.success.registerationData) {
      deviceSecret = data.success.registerationData.deviceSecret;
      console.log("MangaPlus device registered, secret:", deviceSecret);
    } else {
      console.warn("MangaPlus register: no secret in response");
    }
  } catch (error) {
    console.warn("MangaPlus register failed: " + error.message);
  }
}

/**
 * Cari manga di Manga Plus
 */
export async function search(query) {
  console.log("MangaPlus search: query=" + query);
  
  const url = buildUrl("title_list/allV2", {});
  console.log("MangaPlus search URL: " + url);

  try {
    const response = await fetch(url, {
      headers: { "User-Agent": "MangaPlusShonenJump/" + APP_VER },
    });

    console.log("MangaPlus search response status: " + response.status);

    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }

    const data = await response.json();
    console.log("MangaPlus search response:", JSON.stringify(data).substring(0, 300));

    // Check for API error
    if (data.error) {
      const errorMsg = data.error.englishPopup ? data.error.englishPopup.body : JSON.stringify(data.error);
      console.error("MangaPlus API error: " + errorMsg);
      throw new Error("MangaPlus API error: " + errorMsg);
    }

    if (!data.success || !data.success.allTitlesViewV2) {
      throw new Error("Tidak ada judul ditemukan (API response tidak valid)");
    }

    const queryLower = query.toLowerCase();
    const results = [];

    for (const group of data.success.allTitlesViewV2.allTitlesGroup) {
      for (const title of group.titles) {
        if (title.name && title.name.toLowerCase().includes(queryLower)) {
          results.push({
            title: title.name,
            url: "https://mangaplus.shueisha.co.jp/titles/" + title.titleId,
            id: title.titleId.toString(),
          });
        }
      }
    }

    console.log("MangaPlus search results: " + results.length);
    return results;
  } catch (error) {
    console.error("MangaPlus search error: " + error.message);
    throw error;
  }
}

/**
 * Ambil chapter terbaru untuk manga tertentu
 */
export async function getLatestChapter(mangaUrl) {
  const titleId = extractTitleID(mangaUrl);
  if (!titleId) {
    throw new Error("URL atau ID manga tidak valid: " + mangaUrl);
  }

  console.log("MangaPlus getLatestChapter: titleId=" + titleId);

  const url = buildUrl("title_detailV3", { title_id: titleId });

  try {
    const response = await fetch(url, {
      headers: { "User-Agent": "MangaPlusShonenJump/" + APP_VER },
    });

    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }

    const data = await response.json();
    
    if (data.error) {
      const errorMsg = data.error.englishPopup ? data.error.englishPopup.body : JSON.stringify(data.error);
      throw new Error("MangaPlus API error: " + errorMsg);
    }

    if (!data.success || !data.success.titleDetailView) {
      throw new Error("Tidak ada detail untuk manga id: " + titleId);
    }

    return findLatestChapter(data.success.titleDetailView);
  } catch (error) {
    console.error("MangaPlus getLatestChapter error: " + error.message);
    throw error;
  }
}

/**
 * Build URL dengan parameter
 */
function buildUrl(apiPath, params) {
  const url = new URL(MANGAPLUS_API_BASE + "/" + apiPath);
  url.searchParams.set("os", "android");
  url.searchParams.set("os_ver", OS_VER);
  url.searchParams.set("app_ver", APP_VER);
  url.searchParams.set("format", "json");
  if (deviceSecret) {
    url.searchParams.set("secret", deviceSecret);
  }
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value);
  }
  return url.toString();
}

/**
 * Cari chapter terbaru dari detail judul
 */
function findLatestChapter(detail) {
  if (detail.chapterListV2 && detail.chapterListV2.length > 0) {
    let latest = detail.chapterListV2[0];
    for (const ch of detail.chapterListV2) {
      if (ch.startTimeStamp > latest.startTimeStamp) {
        latest = ch;
      }
    }
    return chapterFromData(latest);
  }

  if (detail.chapterListGroup) {
    for (const group of detail.chapterListGroup) {
      const lists = [group.lastChapterList, group.midChapterList, group.firstChapterList];
      for (const list of lists) {
        if (list && list.length > 0) {
          let latest = list[0];
          for (const ch of list) {
            if (ch.startTimeStamp > latest.startTimeStamp) {
              latest = ch;
            }
          }
          return chapterFromData(latest);
        }
      }
    }
  }

  throw new Error("Tidak ditemukan chapter untuk manga");
}

/**
 * Buat objek chapter dari data API
 */
function chapterFromData(ch) {
  const numValue = parseChapterNumber(ch.name);
  const result = {
    number: ch.name || "Chapter " + numValue,
    url: "https://mangaplus.shueisha.co.jp/viewer/" + ch.chapterId,
    numValue: numValue,
  };
  if (ch.subTitle) {
    result.title = ch.subTitle;
  }
  return result;
}

/**
 * Parse nomor chapter dari teks
 */
function parseChapterNumber(text) {
  if (!text) return 0;
  const re = /(?:chapter|ch\.?)\s*[#]?(\d+(?:\.\d+)?)|^(\d+(?:\.\d+)?)$|^#(\d+(?:\.\d+)?)/i;
  const match = text.match(re);
  if (!match) return 0;
  const numStr = match[1] || match[2] || match[3];
  if (!numStr) return 0;
  return parseFloat(numStr);
}

/**
 * Ekstrak title ID dari URL atau input
 */
function extractTitleID(input) {
  input = input.trim();
  if (/^\d+$/.test(input)) {
    return input;
  }
  const parts = input.split("/");
  for (let i = 0; i < parts.length; i++) {
    if (parts[i] === "titles" && i + 1 < parts.length) {
      return parts[i + 1];
    }
  }
  return input;
}