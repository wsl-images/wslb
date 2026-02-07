const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");

const STORE_ROOT = path.resolve(process.cwd(), ".wslb-store");
const PROFILES_DIR = path.join(STORE_ROOT, "profiles");
const DEFAULTS_PATH = path.join(STORE_ROOT, "defaults.json");

const BASE_DEFAULTS = {
  baseImage: "ubuntu:24.04",
  remoteUser: "dev",
  managed: true,
  stateMode: "windows-dir",
  systemd: true,
  iconSimpleIcon: "ubuntu",
  iconColor: "E95420",
  iconStyle: "flat",
  terminalTheme: "Campbell",
  terminalFont: "Cascadia Mono",
  terminalOpacity: 100,
  terminalUseAcrylic: false,
  terminalCursorShape: "bar",
};

function normalizeOpacity(value, fallback) {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  if (n < 30) return 30;
  if (n > 100) return 100;
  return Math.round(n);
}

function buildTerminalTemplate(defaults) {
  const theme = String(defaults.terminalTheme || BASE_DEFAULTS.terminalTheme).trim();
  const font = String(defaults.terminalFont || BASE_DEFAULTS.terminalFont).trim();
  const opacity = normalizeOpacity(defaults.terminalOpacity, BASE_DEFAULTS.terminalOpacity);
  const useAcrylic = defaults.terminalUseAcrylic === true;
  const cursorShape = String(defaults.terminalCursorShape || BASE_DEFAULTS.terminalCursorShape).trim();
  return {
    profiles: [
      {
        colorScheme: theme,
        font: { face: font },
        opacity,
        useAcrylic,
        cursorShape,
      },
    ],
  };
}

function ensureStore() {
  fs.mkdirSync(PROFILES_DIR, { recursive: true });
}

function readJsonFile(filePath, fallback) {
  try {
    const content = fs.readFileSync(filePath, "utf8");
    return JSON.parse(content);
  } catch {
    return fallback;
  }
}

function writeJsonFile(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(value, null, 2) + "\n", "utf8");
}

function slug(inValue) {
  const out = String(inValue || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return out || "devcontainer-wsl";
}

function deriveImageId(manifest) {
  const w = manifest?.wslb || {};
  if (String(w.id || "").trim()) return String(w.id).trim();
  if (String(w.imageId || "").trim()) return String(w.imageId).trim();
  if (String(w.distroName || "").trim()) return slug(String(w.distroName));
  if (String(manifest?.name || "").trim()) return slug(String(manifest.name));
  return "devcontainer-wsl";
}

function buildDefaultManifest(profileName, defaults) {
  const name = profileName || "WSLB Profile";
  const imageId = slug(name);
  const distroName = (name || "WSLB").replace(/[^A-Za-z0-9]/g, "") || "WSLB";
  return {
    $schema: "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
    name,
    image: defaults.baseImage || BASE_DEFAULTS.baseImage,
    remoteUser: defaults.remoteUser || BASE_DEFAULTS.remoteUser,
    features: {
      "wslb:feature/wsl-prereqs": {},
      "ghcr.io/devcontainers/features/common-utils:2": {},
    },
    wslb: {
      version: 1,
      id: imageId,
      distroName,
      managed: defaults.managed !== false,
      state: {
        mode: defaults.stateMode || BASE_DEFAULTS.stateMode,
      },
      wslconf: {
        boot: {
          systemd: defaults.systemd !== false,
        },
      },
      distribution: {
        shortcut: {
          enabled: true,
          icon: {
            simpleIcon: defaults.iconSimpleIcon || BASE_DEFAULTS.iconSimpleIcon,
            color: defaults.iconColor || BASE_DEFAULTS.iconColor,
            style: defaults.iconStyle || BASE_DEFAULTS.iconStyle,
          },
        },
        windowsterminal: {
          enabled: true,
          template: buildTerminalTemplate(defaults),
        },
      },
    },
  };
}

function listProfiles() {
  ensureStore();
  const entries = fs.readdirSync(PROFILES_DIR, { withFileTypes: true });
  const profiles = [];
  for (const ent of entries) {
    if (!ent.isFile() || !ent.name.endsWith(".json")) continue;
    const filePath = path.join(PROFILES_DIR, ent.name);
    const record = readJsonFile(filePath, null);
    if (!record || !record.id) continue;
    profiles.push({
      id: record.id,
      name: record.name,
      manifestPath: record.manifestPath,
      updatedAt: record.updatedAt,
      imageId: deriveImageId(record.manifest || {}),
      distroName: record?.manifest?.wslb?.distroName || "",
      baseImage: record?.manifest?.image || "",
    });
  }
  profiles.sort((a, b) => String(b.updatedAt || "").localeCompare(String(a.updatedAt || "")));
  return profiles;
}

function getProfile(id) {
  ensureStore();
  const filePath = path.join(PROFILES_DIR, `${id}.json`);
  const profile = readJsonFile(filePath, null);
  if (!profile || !profile.id) return null;
  return profile;
}

function saveProfile(input) {
  ensureStore();
  const now = new Date().toISOString();
  const id = slug(input.id || input.name || crypto.randomUUID());
  const existing = getProfile(id);
  const merged = {
    id,
    name: input.name || existing?.name || id,
    manifestPath: path.resolve(input.manifestPath || existing?.manifestPath || path.join(".devcontainer", "devcontainer.json")),
    createdAt: existing?.createdAt || now,
    updatedAt: now,
    manifest: input.manifest || existing?.manifest || {},
  };
  writeJsonFile(path.join(PROFILES_DIR, `${id}.json`), merged);
  return merged;
}

function deleteProfile(id) {
  ensureStore();
  const filePath = path.join(PROFILES_DIR, `${id}.json`);
  if (fs.existsSync(filePath)) {
    fs.unlinkSync(filePath);
  }
}

function getDefaults() {
  ensureStore();
  const custom = readJsonFile(DEFAULTS_PATH, {});
  return { ...BASE_DEFAULTS, ...custom };
}

function saveDefaults(nextDefaults) {
  ensureStore();
  const value = { ...BASE_DEFAULTS, ...(nextDefaults || {}) };
  writeJsonFile(DEFAULTS_PATH, value);
  return value;
}

module.exports = {
  STORE_ROOT,
  PROFILES_DIR,
  DEFAULTS_PATH,
  BASE_DEFAULTS,
  slug,
  deriveImageId,
  buildDefaultManifest,
  ensureStore,
  listProfiles,
  getProfile,
  saveProfile,
  deleteProfile,
  getDefaults,
  saveDefaults,
};
