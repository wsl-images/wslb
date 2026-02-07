# JSON Contract

Commands supporting machine-readable output accept `--json`. Long-running commands may also emit progress events with `--ndjson`.

## Result Envelope

```json
{
  "ok": true,
  "command": "workspace plan",
  "imageId": "dev-ubuntu",
  "startedAt": "2026-02-07T00:00:00Z",
  "endedAt": "2026-02-07T00:00:01Z",
  "durationMs": 1000,
  "steps": [],
  "artifacts": [],
  "errors": []
}
```

## Step

```json
{
  "id": "build",
  "name": "build images",
  "status": "completed",
  "startedAt": "2026-02-07T00:00:00Z",
  "endedAt": "2026-02-07T00:00:01Z",
  "detail": "",
  "remediation": ""
}
```

## Error

```json
{
  "code": "WSLB_COMMAND_FAILED",
  "message": "human readable message",
  "remediation": "what to do next"
}
```

## NDJSON Event

```json
{
  "ts": "2026-02-07T00:00:00Z",
  "level": "info",
  "command": "wsl upgrade",
  "imageId": "dev-ubuntu",
  "event": "start",
  "stepId": "upgrade",
  "message": "",
  "data": {}
}
```

## Command Coverage

- `workspace validate|plan|build|publish`
- `wsl state create|install|upgrade|rollback|status`
- `doctor`
- `catalog feature describe`
