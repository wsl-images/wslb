# E2E Testing

Two layers are provided:

1. PowerShell deterministic E2E runner (`scripts/e2e.ps1`)
2. Browser-channel harness (Playwright + `tools/ps-web-runner`)

## 1) PowerShell Runner

Run:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/e2e.ps1 -Json -Out .\e2e-report.json
```

Optional:

- `-Strict` fail if prerequisites are missing
- `-CleanupStateDisk` remove state storage at end

Output:

- report JSON written to `-Out` path
- includes pass/fail/skipped per step and timestamps

## 2) Playwright Browser-Channel Harness

Install Node deps:

```bash
npm install
npx playwright install
```

Run:

```bash
npm run test:playwright
```

Playwright starts the local PowerShell web runner and invokes:

- `POST /api/run`
- running `scripts/e2e.ps1`
- validating generated report contents

## Runner API

`POST /api/run`

```json
{
  "cwd": "d:/code/wslb",
  "command": "pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/e2e.ps1 -Json -Out d:/tmp/e2e-report.json",
  "timeoutMs": 600000
}
```

Response:

```json
{
  "ok": true,
  "requestId": "req-...",
  "exitCode": 0,
  "stdout": "...",
  "stderr": "",
  "durationMs": 12345
}
```

## 3) WSL Images OOBE Matrix

Run the docker-backed OOBE smoke matrix for the official `wsl-images` distro set:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/oobe-matrix.ps1 -Out .\wsl-images-oobe-matrix-report.json
```

This executes `TestWSLImagesOOBEMatrix` and writes pass/fail per image to the report JSON.

## 4) Base Distro Matrix (25 Images)

Run the base-image compatibility matrix (Docker official + vendor-maintained distro images):

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/base-image-matrix.ps1 -Out .\base-image-matrix-report.json
```

This executes `TestBaseImageMatrixWSLPrereqsAndOOBE` and writes pass/fail/skip per base image.
