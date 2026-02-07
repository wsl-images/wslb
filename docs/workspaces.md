# Workspaces

`wslb` supports a convention-first `devcontainer` superset:

- `.devcontainer/devcontainer.json`
- `.devcontainer/devcontainer.jsonc`
- `devcontainer.json`
- `devcontainer.jsonc`

## Superset Shape

```json
{
  "$schema": "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
  "name": "Ubuntu Dev",
  "image": "ubuntu:24.04",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {}
  },
  "remoteUser": "dev",
  "wslb": {
    "distroName": "DevUbuntu",
    "managed": true,
    "state": {
      "mode": "windows-dir"
    },
    "wslconf": {
      "boot": { "systemd": true }
    },
    "distribution": {
      "oobe": { "defaultName": "dev" },
      "shortcut": {
        "enabled": true,
        "icon": { "simpleIcon": "ubuntu", "color": "E95420", "style": "flat" }
      }
    }
  }
}
```

## Convention Over Configuration

You no longer need to repeat most WSL-only values:

- `target` is implicit (`wsl`); if provided it must still be `wsl`
- `workspaceName` defaults from `name`
- `outputDir` defaults to `./.wslb-out`
- `installDir` defaults to `<outputDir>/distros/<distroName>`
- `imageId` defaults from `wslb.id` / `wslb.imageId` / `wslb.distroName` / `name`
- `wslb.user` defaults from `remoteUser` (or `containerUser`, then `dev`)
- `wslconf.user.default` defaults from resolved user name
- `state.path`, `state.mountPoint`, `state.fsLabel` are defaulted when omitted
- managed user creation + managed state mount wiring are handled automatically; you do not need to repeat internal features for baseline behavior

## JSON, JSONC, and Schema

- `wslb` accepts both JSON and JSONC (comments and trailing commas).
- Schema URL for editor IntelliSense:
  - `https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json`
- Local schema file in repo:
  - `schemas/wslb-workspace.schema.json`

- Standard Dev Container fields are reused (`name`, `image`, `features`, `remoteUser`).
- A top-level `wslb` block adds WSL lifecycle settings (`distroName`, `managed`, `state`, `wslconf`, `distribution`).
- `wslb.wslconf` supports full `wsl.conf` sections:
  - `user.default`
  - `boot.systemd`, `boot.command`
  - `automount.enabled`, `automount.root`, `automount.options`, `automount.mountFsTab`
  - `network.hostname`, `network.generateHosts`, `network.generateResolvConf`
  - `interop.enabled`, `interop.appendWindowsPath`
- `wslb.distribution` writes `/etc/wsl-distribution.conf`:
  - `oobe.defaultName`, `oobe.defaultUid`, `oobe.command`
  - `shortcut.enabled`
  - `windowsterminal.ProfileTemplate` is auto-emitted when an icon is configured
  - `shortcut.icon` can be:
    - string path/URL to `.ico`
    - object `{ "path": "...ico" }`
    - object `{ "simpleIcon": "<slug>", "color": "E95420", "style": "flat|badge" }` (fetched from Simple Icons and converted to `.ico`)
- `features` map entries are converted to ordered feature refs/options for the WSL build pipeline.

Examples:

- `examples/devcontainer/devcontainer.json`
- `examples/devcontainer/minimal.json`
- `examples/devcontainer/managed-vhdx.json`
- `examples/devcontainer/ubuntu-pro.json`

## Commands

- `wslb workspace init`
- `wslb workspace validate`
- `wslb workspace plan [--all|<imageId>]`
- `wslb workspace build [--all|<imageId>] [--engine docker|podman]`
- `wslb workspace publish [--all|<imageId>]`

## Validation Rules

- `image` is required
- `wslb.target`, if provided, must be `wsl`
- managed WSL requires `wslb.state`
- feature refs must parse as OCI/devcontainer or `wslb:feature/<id>`

## Plan Output

`workspace plan` generates deterministic:

- prerequisites
- ordered steps
- artifact paths
- candidate/backup naming for managed WSL


