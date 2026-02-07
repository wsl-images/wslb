const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { test, expect } = require("@playwright/test");

test("runs unattended e2e.ps1 through ps-web-runner API", async ({ request }) => {
  const repoRoot = path.resolve(__dirname, "..", "..");
  const reportPath = path.join(os.tmpdir(), `wslb-e2e-report-${Date.now()}.json`);
  const command = `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/e2e.ps1 -Json -Out "${reportPath}"`;

  const res = await request.post("/api/run", {
    data: {
      cwd: repoRoot,
      command,
      timeoutMs: 45 * 60 * 1000,
    },
  });

  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  expect(typeof body.exitCode).toBe("number");
  expect(body.exitCode).toBe(0);

  expect(fs.existsSync(reportPath)).toBeTruthy();
  const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
  expect(report).toHaveProperty("ok");
  expect(report).toHaveProperty("steps");
  expect(Array.isArray(report.steps)).toBeTruthy();
  expect(report.steps.length).toBeGreaterThan(0);

  const failedCritical = report.steps.filter((s) => s.critical && s.status === "failed");
  expect(failedCritical.length).toBe(0);
});

