const path = require("node:path");
const { defineConfig } = require("@playwright/test");

const repoRoot = path.resolve(__dirname, "..", "..");
const host = process.env.PS_WEB_RUNNER_HOST || "127.0.0.1";
const port = Number(process.env.PS_WEB_RUNNER_PORT || "4891");
const baseURL = `http://${host}:${port}`;

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: /.*\.spec\.js/,
  timeout: 45 * 60 * 1000,
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL,
  },
  webServer: {
    command: "node tools/ps-web-runner/server.js",
    cwd: repoRoot,
    url: `${baseURL}/healthz`,
    timeout: 120 * 1000,
    reuseExistingServer: !process.env.CI,
    stdout: "pipe",
    stderr: "pipe",
  },
});

