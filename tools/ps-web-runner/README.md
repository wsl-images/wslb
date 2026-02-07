# ps-web-runner

Local HTTP runner that executes PowerShell commands for unattended browser-channel tests.

## Start

```bash
node tools/ps-web-runner/server.js
```

Health:

- `GET /healthz`

Run command:

- `POST /api/run`

Request:

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
  "shell": "pwsh",
  "exitCode": 0,
  "stdout": "",
  "stderr": "",
  "durationMs": 1234
}
```
