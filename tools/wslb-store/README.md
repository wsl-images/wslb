# WSLB Store (MVP)

Local web UI for managing multiple WSL distro profiles backed by `wslb` commands.

## What it does

- Manages multiple profile records (`.wslb-store/profiles/*.json`)
- Edits per-profile `devcontainer.json` superset manifests
- Searches Dev Container Features from `https://containers.dev/features`
- Searches Simple Icons from `https://raw.githubusercontent.com/simple-icons/simple-icons/develop/slugs.md`
- Configures Windows Terminal profile defaults (`colorScheme`, font, opacity, acrylic, cursor) written to `wslb.distribution.windowsterminal.template`
- Runs WSLB lifecycle actions through JSON APIs:
  - `workspace validate|plan|build`
  - `wsl state create|install|upgrade|rollback|status`
  - `doctor`

## Run

From repo root:

```powershell
node tools/wslb-store/server.js
```

Open:

`http://127.0.0.1:4893`

Optional env vars:

- `WSLB_STORE_HOST` (default `127.0.0.1`)
- `WSLB_STORE_PORT` (default `4893`)
- `WSLB_BINARY` (override `wslb` executable path)

## Notes

- The UI is inspired by app-store style flows (profile catalog + details + actions), while using native web tech for easy local execution on Windows.
- `ubuntu/app-center` is Flutter-based; this MVP does not embed that codebase.
