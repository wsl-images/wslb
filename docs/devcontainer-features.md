# Dev Container Feature Setup (wsl-images)

`wslb` supports any OCI Dev Container Feature reference.  
For first-party features, `wsl-images` now has a dedicated feature collection repository layout under `devcontainer-features/`.

## Included first-party features

- `wsl-prereqs`
- `first-boot-user`
- `persist-home`

## Publishing to `wsl-images` organization

Use the publish script from this repo root:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/publish-devcontainer-features.ps1
```

Notes:

- The script resolves GitHub CLI directly via:
  - `WSLB_GH_EXE` env var, or
  - `C:\Program Files\GitHub CLI\gh.exe`
- It creates `wsl-images/devcontainer-features` if missing.
- It syncs local `devcontainer-features/` content, commits, and pushes to `main`.

## Consuming published features in `devcontainer.json`

```json
{
  "features": {
    "ghcr.io/wsl-images/devcontainer-features/wsl-prereqs:1": {},
    "ghcr.io/wsl-images/devcontainer-features/first-boot-user:1": {
      "username": "dev"
    },
    "ghcr.io/wsl-images/devcontainer-features/persist-home:1": {
      "mountPoint": "/home"
    }
  }
}
```

## Local feature testing

From `devcontainer-features/`:

```bash
devcontainer features test --project-folder . --features ./src/wsl-prereqs --base-image ubuntu:24.04
devcontainer features test --project-folder . --features ./src/first-boot-user --base-image ubuntu:24.04
devcontainer features test --project-folder . --features ./src/persist-home --base-image ubuntu:24.04
```
