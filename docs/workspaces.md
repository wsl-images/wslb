# Workspaces

`wslb` supports a `devcontainer.json` superset manifest at `.devcontainer/devcontainer.json`.

## Superset Shape

```json
{
  "name": "Ubuntu Dev",
  "image": "ubuntu:24.04",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {},
    "wslb:feature/first-boot-user": {
      "USERNAME": "dev"
    }
  },
  "remoteUser": "dev",
  "wslb": {
    "version": 1,
    "workspaceName": "demo",
    "imageId": "dev-ubuntu",
    "target": "wsl",
    "distroName": "DevUbuntu",
    "managed": true,
    "outputDir": "./.wslb-out",
    "installDir": "./distros/dev-ubuntu",
    "state": {
      "mode": "vhdx",
      "path": "./state/dev-ubuntu.vhdx",
      "mountPoint": "/home",
      "fsLabel": "WSLB_STATE"
    },
    "wslconf": {
      "user": { "default": "dev" },
      "boot": { "systemd": true, "command": "echo booted" },
      "automount": {
        "enabled": true,
        "mountFsTab": true,
        "options": "metadata"
      },
      "network": {
        "hostname": "dev-ubuntu",
        "generateHosts": true,
        "generateResolvConf": true
      },
      "interop": { "enabled": true, "appendWindowsPath": true }
    },
    "distribution": {
      "oobe": { "command": "usermod --shell /bin/bash dev" },
      "shortcut": {
        "enabled": true,
        "icon": {
          "simpleIcon": "ubuntu",
          "color": "E95420"
        }
      }
    }
  }
}
```

## Devcontainer Superset (WSL)

`wslb` loads `.devcontainer/devcontainer.json` (or any `*.json` path you pass via `--workspace-file`) as a WSL superset.

- Standard Dev Container fields are reused (`name`, `image`, `features`, `remoteUser`).
- A top-level `wslb` block adds WSL lifecycle settings (`distroName`, `managed`, `state`, `wslconf`, etc.).
- `wslb.wslconf` supports full `wsl.conf` sections:
  - `user.default`
  - `boot.systemd`, `boot.command`
  - `automount.enabled`, `automount.root`, `automount.options`, `automount.mountFsTab`
  - `network.hostname`, `network.generateHosts`, `network.generateResolvConf`
  - `interop.enabled`, `interop.appendWindowsPath`
- `wslb.distribution` writes `/etc/wsl-distribution.conf`:
  - `oobe.command`
  - `shortcut.enabled`
  - `shortcut.icon` can be:
    - string path/URL to `.ico`
    - object `{ "path": "...ico" }`
    - object `{ "simpleIcon": "<slug>", "color": "E95420" }` (fetched from Simple Icons and converted to `.ico`)
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
- `wslb.target` must be `wsl`
- managed WSL requires `wslb.state`
- feature refs must parse as OCI/devcontainer or `wslb:feature/<id>`

## Plan Output

`workspace plan` generates deterministic:

- prerequisites
- ordered steps
- artifact paths
- candidate/backup naming for managed WSL
