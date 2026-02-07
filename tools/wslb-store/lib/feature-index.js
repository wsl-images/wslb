const FEATURE_INDEX_URL = "https://containers.dev/features";
const CACHE_TTL_MS = 10 * 60 * 1000;

let cache = {
  fetchedAt: 0,
  items: [],
};

function decodeHtml(input) {
  if (!input) return "";
  return input
    .replace(/&amp;/g, "&")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&nbsp;/g, " ")
    .replace(/<[^>]+>/g, "")
    .trim();
}

function parseFeatureRows(html) {
  const out = [];
  const rowRegex = /<tr>\s*<td[^>]*>\s*<a[^>]*>([\s\S]*?)<\/a>\s*<\/td>\s*<td[^>]*>([\s\S]*?)<\/td>\s*<td[^>]*>\s*<code>([\s\S]*?)<\/code>\s*<\/td>\s*<td[^>]*>\s*<code>([\s\S]*?)<\/code>\s*<\/td>\s*<\/tr>/gi;
  let match = rowRegex.exec(html);
  while (match) {
    const name = decodeHtml(match[1]);
    const maintainer = decodeHtml(match[2]);
    const ref = decodeHtml(match[3]);
    const version = decodeHtml(match[4]);
    if (name && ref) {
      out.push({ name, maintainer, ref, version });
    }
    match = rowRegex.exec(html);
  }
  return out;
}

async function fetchFeatures(forceRefresh) {
  const now = Date.now();
  if (!forceRefresh && cache.items.length > 0 && now-cache.fetchedAt < CACHE_TTL_MS) {
    return cache.items;
  }
  const res = await fetch(FEATURE_INDEX_URL, {
    headers: {
      "User-Agent": "wslb-store/0.1",
      "Accept": "text/html,application/xhtml+xml",
    },
  });
  if (!res.ok) {
    throw new Error(`failed to fetch feature index: HTTP ${res.status}`);
  }
  const html = await res.text();
  const items = parseFeatureRows(html);
  if (items.length === 0) {
    throw new Error("failed to parse features from containers.dev");
  }
  cache = {
    fetchedAt: now,
    items,
  };
  return items;
}

function searchFeatures(items, query, limit) {
  const q = String(query || "").trim().toLowerCase();
  const max = Number(limit || 100);
  if (!q) return items.slice(0, max);
  return items
    .filter((item) => {
      const hay = `${item.name} ${item.maintainer} ${item.ref} ${item.version}`.toLowerCase();
      return hay.includes(q);
    })
    .slice(0, max);
}

module.exports = {
  FEATURE_INDEX_URL,
  parseFeatureRows,
  fetchFeatures,
  searchFeatures,
};
