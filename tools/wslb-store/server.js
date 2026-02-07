const http = require("node:http");
const path = require("node:path");
const fs = require("node:fs");
const crypto = require("node:crypto");
const { spawn } = require("node:child_process");
const { URL } = require("node:url");

const {
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
} = require("./lib/profiles");
const { fetchFeatures, searchFeatures } = require("./lib/feature-index");
const { fetchSimpleIcons, searchSimpleIcons } = require("./lib/simple-icons-index");

const HOST = process.env.WSLB_STORE_HOST || "127.0.0.1";
const PORT = Number(process.env.WSLB_STORE_PORT || "4893");
const ROOT_DIR = path.resolve(__dirname, "..", "..");
const PUBLIC_DIR = path.join(__dirname, "public");

const DEFAULT_TIMEOUT_MS = 30 * 60 * 1000;
const MAX_BODY_BYTES = 2 * 1024 * 1024;

function nowIso() {
  return new Date().toISOString();
}

function resolveWSLBBinary() {
  const explicit = process.env.WSLB_BINARY;
  if (explicit) return explicit;
  const localExe = path.join(ROOT_DIR, "wslb.exe");
  if (fs.existsSync(localExe)) return localExe;
  const binExe = path.join(ROOT_DIR, "bin", "wslb.exe");
  if (fs.existsSync(binExe)) return binExe;
  return "wslb";
}

function contentTypeFor(filePath) {
  const ext = path.extname(filePath).toLowerCase();
  switch (ext) {
    case ".html":
      return "text/html; charset=utf-8";
    case ".js":
      return "application/javascript; charset=utf-8";
    case ".css":
      return "text/css; charset=utf-8";
    case ".json":
      return "application/json; charset=utf-8";
    case ".svg":
      return "image/svg+xml";
    case ".png":
      return "image/png";
    case ".ico":
      return "image/x-icon";
    default:
      return "application/octet-stream";
  }
}

function sendJson(res, statusCode, payload) {
  const body = JSON.stringify(payload);
  res.statusCode = statusCode;
  res.setHeader("Content-Type", "application/json; charset=utf-8");
  res.setHeader("Cache-Control", "no-store");
  res.end(body);
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    let total = 0;
    req.on("data", (chunk) => {
      total += chunk.length;
      if (total > MAX_BODY_BYTES) {
        reject(new Error("request body too large"));
        req.destroy();
        return;
      }
      chunks.push(chunk);
    });
    req.on("end", () => {
      resolve(Buffer.concat(chunks).toString("utf8"));
    });
    req.on("error", reject);
  });
}

function parseBodyJson(raw) {
  if (!raw || !String(raw).trim()) return {};
  return JSON.parse(raw);
}

function parseStdoutJson(raw) {
  const text = String(raw || "").trim();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

function runCommand(executable, args, options = {}) {
  return new Promise((resolve) => {
    const cwd = options.cwd ? path.resolve(options.cwd) : ROOT_DIR;
    const timeoutMs = Number(options.timeoutMs || DEFAULT_TIMEOUT_MS);
    const id = `run-${crypto.randomUUID()}`;
    const started = Date.now();
    const child = spawn(executable, args, {
      cwd,
      env: process.env,
      windowsHide: true,
      stdio: ["ignore", "pipe", "pipe"],
    });

    const stdout = [];
    const stderr = [];
    let timedOut = false;
    const timer = setTimeout(() => {
      timedOut = true;
      try {
        child.kill("SIGTERM");
      } catch {}
    }, timeoutMs);

    child.stdout.on("data", (d) => stdout.push(Buffer.from(d)));
    child.stderr.on("data", (d) => stderr.push(Buffer.from(d)));
    child.on("close", (code) => {
      clearTimeout(timer);
      const stdoutText = Buffer.concat(stdout).toString("utf8");
      const stderrText = Buffer.concat(stderr).toString("utf8");
      resolve({
        id,
        ok: !timedOut && code === 0,
        exitCode: timedOut ? 124 : (code ?? 1),
        cwd,
        executable,
        args,
        timedOut,
        durationMs: Date.now() - started,
        stdout: stdoutText,
        stderr: timedOut ? `${stderrText}\nTimed out after ${timeoutMs}ms` : stderrText,
        json: parseStdoutJson(stdoutText),
      });
    });
    child.on("error", (err) => {
      clearTimeout(timer);
      resolve({
        id,
        ok: false,
        exitCode: 1,
        cwd,
        executable,
        args,
        timedOut: false,
        durationMs: Date.now() - started,
        stdout: "",
        stderr: String(err?.message || err),
        json: null,
      });
    });
  });
}

function profileActionArgs(profile, body) {
  const imageId = deriveImageId(profile.manifest || {});
  const manifestPath = profile.manifestPath;
  const action = String(body.action || "").trim();
  const commonFlags = ["--workspace-file", manifestPath];
  const maybeEngine = body.engine ? ["--engine", String(body.engine)] : [];
  const maybeFallback = body.fallbackWindowsDir ? ["--fallback-windows-dir"] : [];
  const maybeNonInteractive = body.nonInteractive !== false ? ["--non-interactive"] : [];

  switch (action) {
    case "validate":
      return ["workspace", "validate", ...commonFlags, "--json"];
    case "plan":
      return ["workspace", "plan", imageId, ...commonFlags, "--json"];
    case "build":
      return ["workspace", "build", imageId, ...commonFlags, ...maybeEngine, "--json"];
    case "state-create":
      return ["wsl", "state", "create", imageId, ...commonFlags, ...maybeFallback, ...maybeNonInteractive, "--json"];
    case "install":
      return ["wsl", "install", imageId, ...commonFlags, ...maybeEngine, ...maybeFallback, ...maybeNonInteractive, "--json"];
    case "upgrade":
      return ["wsl", "upgrade", imageId, ...commonFlags, ...maybeEngine, ...maybeFallback, ...maybeNonInteractive, "--json"];
    case "rollback": {
      const to = String(body.rollbackTo || "").trim();
      const toFlag = to ? ["--to", to] : [];
      return ["wsl", "rollback", imageId, ...commonFlags, ...toFlag, ...maybeFallback, ...maybeNonInteractive, "--json"];
    }
    case "status":
      return ["wsl", "status", imageId, ...commonFlags, "--json"];
    default:
      throw new Error(`unsupported action: ${action}`);
  }
}

function writeManifest(profile) {
  fs.mkdirSync(path.dirname(profile.manifestPath), { recursive: true });
  fs.writeFileSync(profile.manifestPath, JSON.stringify(profile.manifest, null, 2) + "\n", "utf8");
}

function normalizeProfileInput(body) {
  const defaults = getDefaults();
  const name = String(body.name || "").trim() || "New Profile";
  const manifestPath = body.manifestPath || path.join(".devcontainer", `${slug(name)}.json`);
  const manifest = body.manifest || buildDefaultManifest(name, defaults);
  return {
    id: body.id || slug(name),
    name,
    manifestPath,
    manifest,
  };
}

async function handleApi(req, res, urlObj) {
  const method = req.method || "GET";
  const pathName = urlObj.pathname || "/";

  if (method === "GET" && pathName === "/api/health") {
    return sendJson(res, 200, {
      ok: true,
      ts: nowIso(),
      rootDir: ROOT_DIR,
      wslbBinary: resolveWSLBBinary(),
    });
  }

  if (method === "GET" && pathName === "/api/defaults") {
    return sendJson(res, 200, { ok: true, defaults: getDefaults() });
  }

  if (method === "PUT" && pathName === "/api/defaults") {
    const raw = await readBody(req);
    const body = parseBodyJson(raw);
    const next = saveDefaults(body.defaults || body);
    return sendJson(res, 200, { ok: true, defaults: next });
  }

  if (method === "GET" && pathName === "/api/profiles") {
    return sendJson(res, 200, { ok: true, profiles: listProfiles() });
  }

  if (method === "POST" && pathName === "/api/profiles") {
    const raw = await readBody(req);
    const body = parseBodyJson(raw);
    const normalized = normalizeProfileInput(body);
    const profile = saveProfile(normalized);
    return sendJson(res, 201, { ok: true, profile });
  }

  if (method === "POST" && pathName === "/api/doctor") {
    const result = await runCommand(resolveWSLBBinary(), ["doctor", "--json"]);
    return sendJson(res, result.ok ? 200 : 500, { ok: result.ok, result });
  }

  if (method === "GET" && pathName === "/api/features/search") {
    const q = urlObj.searchParams.get("q") || "";
    const limit = Number(urlObj.searchParams.get("limit") || "120");
    const refresh = urlObj.searchParams.get("refresh") === "1";
    const all = await fetchFeatures(refresh);
    const filtered = searchFeatures(all, q, limit);
    return sendJson(res, 200, {
      ok: true,
      total: all.length,
      count: filtered.length,
      items: filtered,
    });
  }

  if (method === "GET" && pathName === "/api/features/describe") {
    const ref = String(urlObj.searchParams.get("ref") || "").trim();
    if (!ref) {
      return sendJson(res, 400, { ok: false, error: "ref is required" });
    }
    const result = await runCommand(resolveWSLBBinary(), ["catalog", "feature", "describe", ref, "--json"]);
    return sendJson(res, result.ok ? 200 : 500, { ok: result.ok, result });
  }

  if (method === "GET" && pathName === "/api/icons/search") {
    const q = urlObj.searchParams.get("q") || "";
    const limit = Number(urlObj.searchParams.get("limit") || "200");
    const refresh = urlObj.searchParams.get("refresh") === "1";
    const all = await fetchSimpleIcons(refresh);
    const filtered = searchSimpleIcons(all, q, limit);
    return sendJson(res, 200, {
      ok: true,
      total: all.length,
      count: filtered.length,
      items: filtered,
    });
  }

  const profileMatch = pathName.match(/^\/api\/profiles\/([a-z0-9-]+)$/);
  if (profileMatch) {
    const profileId = profileMatch[1];
    const profile = getProfile(profileId);
    if (!profile) return sendJson(res, 404, { ok: false, error: "profile not found" });

    if (method === "GET") {
      return sendJson(res, 200, { ok: true, profile });
    }
    if (method === "DELETE") {
      deleteProfile(profileId);
      return sendJson(res, 200, { ok: true });
    }
    if (method === "PUT") {
      const raw = await readBody(req);
      const body = parseBodyJson(raw);
      const saved = saveProfile({
        id: profileId,
        name: body.name || profile.name,
        manifestPath: body.manifestPath || profile.manifestPath,
        manifest: body.manifest || profile.manifest,
      });
      return sendJson(res, 200, { ok: true, profile: saved });
    }
  }

  const applyMatch = pathName.match(/^\/api\/profiles\/([a-z0-9-]+)\/apply$/);
  if (applyMatch && method === "POST") {
    const profileId = applyMatch[1];
    const profile = getProfile(profileId);
    if (!profile) return sendJson(res, 404, { ok: false, error: "profile not found" });
    writeManifest(profile);
    return sendJson(res, 200, { ok: true, manifestPath: profile.manifestPath });
  }

  const actionMatch = pathName.match(/^\/api\/profiles\/([a-z0-9-]+)\/action$/);
  if (actionMatch && method === "POST") {
    const profileId = actionMatch[1];
    const profile = getProfile(profileId);
    if (!profile) return sendJson(res, 404, { ok: false, error: "profile not found" });
    const raw = await readBody(req);
    const body = parseBodyJson(raw);
    if (body.writeManifest !== false) {
      writeManifest(profile);
    }
    let args;
    try {
      args = profileActionArgs(profile, body);
    } catch (err) {
      return sendJson(res, 400, { ok: false, error: String(err.message || err) });
    }
    const result = await runCommand(resolveWSLBBinary(), args, {
      timeoutMs: body.timeoutMs || DEFAULT_TIMEOUT_MS,
      cwd: body.cwd || ROOT_DIR,
    });
    return sendJson(res, result.ok ? 200 : 500, {
      ok: result.ok,
      profileId,
      imageId: deriveImageId(profile.manifest || {}),
      result,
    });
  }

  if (method === "POST" && pathName === "/api/seed-profile") {
    const defaults = getDefaults();
    const manifest = buildDefaultManifest("Ubuntu Developer", defaults);
    const saved = saveProfile({
      id: "ubuntu-developer",
      name: "Ubuntu Developer",
      manifestPath: path.join(".devcontainer", "store-ubuntu.json"),
      manifest,
    });
    return sendJson(res, 201, { ok: true, profile: saved });
  }

  return sendJson(res, 404, { ok: false, error: "not found" });
}

function serveStatic(req, res, urlObj) {
  let reqPath = urlObj.pathname || "/";
  if (reqPath === "/") reqPath = "/index.html";
  const safePath = path.normalize(reqPath).replace(/^(\.\.[/\\])+/, "");
  const fullPath = path.join(PUBLIC_DIR, safePath);
  if (!fullPath.startsWith(PUBLIC_DIR)) {
    res.statusCode = 403;
    res.end("forbidden");
    return;
  }
  if (!fs.existsSync(fullPath) || fs.statSync(fullPath).isDirectory()) {
    res.statusCode = 404;
    res.end("not found");
    return;
  }
  res.statusCode = 200;
  res.setHeader("Content-Type", contentTypeFor(fullPath));
  fs.createReadStream(fullPath).pipe(res);
}

ensureStore();

const server = http.createServer(async (req, res) => {
  try {
    const urlObj = new URL(req.url || "/", `http://${req.headers.host || "localhost"}`);
    if (urlObj.pathname.startsWith("/api/")) {
      await handleApi(req, res, urlObj);
      return;
    }
    serveStatic(req, res, urlObj);
  } catch (err) {
    sendJson(res, 500, {
      ok: false,
      error: String(err?.message || err),
    });
  }
});

server.listen(PORT, HOST, () => {
  process.stdout.write(`[${nowIso()}] [wslb-store] listening on http://${HOST}:${PORT}\n`);
});
