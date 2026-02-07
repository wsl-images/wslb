# WSLB Store UI

`WSLB Store` is a local web app for managing multiple distro profiles and their `devcontainer.json` superset files, backed by `wslb` CLI commands.

## Run

```powershell
npm run store
```

Open:

- `http://127.0.0.1:4893`

## What the UI manages

- Profile list (multiple distro configurations)
- Default values:
  - base image, user, managed mode
  - icon defaults (`simpleIcon`, color, style)
  - terminal defaults (theme, font, opacity, acrylic, cursor)
- Manifest authoring:
  - top-level `name`, `image`, `remoteUser`
  - `wslb` fields (`id`, `distroName`, `managed`, `state.mode`, icon settings)
  - `wslb.distribution.windowsterminal.template` presets for terminal appearance
  - feature map (`features`)
- Feature discovery from `https://containers.dev/features`
- Icon discovery from Simple Icons (`simpleIcon` slug search + color preview)
- One-click actions through `wslb --json`:
  - `workspace validate|plan|build`
  - `wsl state create|install|upgrade|rollback|status`
  - `doctor`

## Data location

The UI stores profile metadata in:

- `.wslb-store/profiles/*.json`
- `.wslb-store/defaults.json`

Manifest files are written to each profile's configured `manifestPath` (for example `.devcontainer/my-distro.json`).

## Notes

- This is a local-only admin surface intended for Windows + WSL workflows.
- It uses the existing CLI as the source of truth for operations rather than duplicating lifecycle logic in JavaScript.
