const SIMPLE_ICONS_SLUGS_URL = "https://raw.githubusercontent.com/simple-icons/simple-icons/develop/slugs.md";
const CACHE_TTL_MS = 24 * 60 * 60 * 1000;

let cache = {
  fetchedAt: 0,
  items: [],
};

function stripTicks(value) {
  return String(value || "")
    .trim()
    .replace(/^`+/, "")
    .replace(/`+$/, "")
    .trim();
}

function parseSimpleIconSlugs(markdown) {
  const lines = String(markdown || "").split(/\r?\n/);
  const bySlug = new Map();

  for (const line of lines) {
    if (!line.startsWith("|")) continue;
    const cols = line.split("|");
    if (cols.length < 4) continue;

    const rawName = stripTicks(cols[1]);
    const rawSlug = stripTicks(cols[2]);
    if (!rawName || !rawSlug) continue;
    if (rawName.toLowerCase() === "brand name" || rawSlug.toLowerCase() === "brand slug") continue;
    if (rawName.startsWith(":") || rawSlug.startsWith(":")) continue;

    const slug = rawSlug.toLowerCase().replace(/[^a-z0-9]+/g, "");
    if (!slug) continue;
    if (bySlug.has(slug)) continue;

    bySlug.set(slug, {
      name: rawName,
      slug,
      svgUrl: `https://simpleicons.org/icons/${slug}.svg`,
      siteUrl: `https://simpleicons.org/?q=${encodeURIComponent(slug)}`,
    });
  }

  return Array.from(bySlug.values());
}

async function fetchSimpleIcons(forceRefresh) {
  const now = Date.now();
  if (!forceRefresh && cache.items.length > 0 && now-cache.fetchedAt < CACHE_TTL_MS) {
    return cache.items;
  }

  const res = await fetch(SIMPLE_ICONS_SLUGS_URL, {
    headers: {
      "User-Agent": "wslb-store/0.1",
      "Accept": "text/plain; charset=utf-8",
    },
  });
  if (!res.ok) {
    throw new Error(`failed to fetch simple icons index: HTTP ${res.status}`);
  }
  const markdown = await res.text();
  const items = parseSimpleIconSlugs(markdown);
  if (items.length === 0) {
    throw new Error("failed to parse simple icons index");
  }

  cache = {
    fetchedAt: now,
    items,
  };
  return items;
}

function searchSimpleIcons(items, query, limit) {
  const q = String(query || "").trim().toLowerCase();
  const max = Number(limit || 160);
  if (!q) return items.slice(0, max);

  return items
    .filter((item) => {
      const hay = `${item.name} ${item.slug}`.toLowerCase();
      return hay.includes(q);
    })
    .slice(0, max);
}

module.exports = {
  SIMPLE_ICONS_SLUGS_URL,
  parseSimpleIconSlugs,
  fetchSimpleIcons,
  searchSimpleIcons,
};
