const http = require("node:http");
const path = require("node:path");
const crypto = require("node:crypto");
const { spawn, spawnSync } = require("node:child_process");

const PORT = Number(process.env.PS_WEB_RUNNER_PORT || "4891");
const HOST = process.env.PS_WEB_RUNNER_HOST || "127.0.0.1";

function nowIso() {
  return new Date().toISOString();
}

function log(id, msg, data) {
  const suffix = data ? ` ${JSON.stringify(data)}` : "";
  process.stdout.write(`[${nowIso()}] [${id}] ${msg}${suffix}\n`);
}

function resolveShell() {
  const pwshProbe = spawnSync("pwsh", ["-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"], {
    stdio: "ignore",
    windowsHide: true,
  });
  if (pwshProbe.status === 0) {
    return "pwsh";
  }
  return "powershell.exe";
}

const shell = resolveShell();
const rootDir = path.resolve(__dirname, "..", "..");

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (d) => chunks.push(d));
    req.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
    req.on("error", reject);
  });
}

function runPowerShellCommand({ requestId, cwd, command, timeoutMs }) {
  return new Promise((resolve) => {
    const started = Date.now();
    const psPrelude =
      "$ErrorActionPreference='Stop'; " +
      "[Console]::InputEncoding=[System.Text.UTF8Encoding]::new(); " +
      "[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); ";
    const wrappedCommand = `${psPrelude}${command}`;

    const args =
      shell === "pwsh"
        ? ["-NoLogo", "-NoProfile", "-NonInteractive", "-Command", wrappedCommand]
        : ["-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", wrappedCommand];

    const child = spawn(shell, args, {
      cwd: cwd || rootDir,
      windowsHide: true,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let done = false;
    const stdoutChunks = [];
    const stderrChunks = [];
    let timedOut = false;

    const timer = setTimeout(() => {
      timedOut = true;
      try {
        child.kill("SIGTERM");
      } catch {}
      try {
        spawnSync("taskkill", ["/PID", String(child.pid), "/T", "/F"], { windowsHide: true, stdio: "ignore" });
      } catch {}
    }, timeoutMs);

    child.stdout.on("data", (d) => stdoutChunks.push(Buffer.from(d)));
    child.stderr.on("data", (d) => stderrChunks.push(Buffer.from(d)));

    child.on("close", (code) => {
      if (done) return;
      done = true;
      clearTimeout(timer);
      const durationMs = Date.now() - started;
      const stdout = Buffer.concat(stdoutChunks).toString("utf8");
      const stderr = Buffer.concat(stderrChunks).toString("utf8");
      resolve({
        ok: code === 0 && !timedOut,
        requestId,
        shell,
        exitCode: timedOut ? 124 : code ?? 1,
        stdout,
        stderr: timedOut ? `${stderr}\nCommand timed out after ${timeoutMs}ms.` : stderr,
        durationMs,
        timedOut,
      });
    });

    child.on("error", (err) => {
      if (done) return;
      done = true;
      clearTimeout(timer);
      const durationMs = Date.now() - started;
      resolve({
        ok: false,
        requestId,
        shell,
        exitCode: 1,
        stdout: "",
        stderr: String(err && err.message ? err.message : err),
        durationMs,
      });
    });
  });
}

function sendJson(res, status, payload) {
  res.statusCode = status;
  res.setHeader("Content-Type", "application/json; charset=utf-8");
  res.end(JSON.stringify(payload));
}

const server = http.createServer(async (req, res) => {
  if (req.url === "/healthz" && req.method === "GET") {
    return sendJson(res, 200, { ok: true, shell });
  }

  if (req.url === "/" && req.method === "GET") {
    res.statusCode = 200;
    res.setHeader("Content-Type", "text/html; charset=utf-8");
    res.end(`<!doctype html>
<html>
  <head><meta charset="utf-8"><title>ps-web-runner</title></head>
  <body>
    <h1>ps-web-runner</h1>
    <p>POST /api/run with {"cwd","command","timeoutMs"}.</p>
  </body>
</html>`);
    return;
  }

  if (req.url === "/api/run" && req.method === "POST") {
    const requestId = `req-${crypto.randomUUID()}`;
    try {
      const raw = await readBody(req);
      const body = raw ? JSON.parse(raw) : {};
      const command = String(body.command || "");
      const cwd = body.cwd ? String(body.cwd) : rootDir;
      const timeoutMs = Number(body.timeoutMs || 600000);

      if (!command.trim()) {
        return sendJson(res, 400, { ok: false, requestId, error: "command is required" });
      }

      log(requestId, "run start", { cwd, timeoutMs, command });
      const result = await runPowerShellCommand({ requestId, cwd, command, timeoutMs });
      log(requestId, "run end", { exitCode: result.exitCode, durationMs: result.durationMs });
      return sendJson(res, 200, result);
    } catch (err) {
      const msg = String(err && err.message ? err.message : err);
      log(requestId, "run error", { error: msg });
      return sendJson(res, 500, { ok: false, requestId, error: msg });
    }
  }

  sendJson(res, 404, { ok: false, error: "not found" });
});

server.listen(PORT, HOST, () => {
  process.stdout.write(`[${nowIso()}] [server] listening on http://${HOST}:${PORT} (shell=${shell})\n`);
});

