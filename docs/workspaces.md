# Workspaces (Devcontainer Superset for WSL)

`wslb` uses a convention-first `devcontainer` superset:

- `.devcontainer/devcontainer.json`
- `.devcontainer/devcontainer.jsonc`
- `devcontainer.json`
- `devcontainer.jsonc`

## Schema Association

Use this at the top of your file:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json"
}
```

Local schema path in this repo:

- `schemas/wslb-workspace.schema.json`

## Minimal Example

```json
{
  "$schema": "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
  "name": "Ubuntu Dev",
  "image": "ubuntu:24.04",
  "features": {
    "wslb:feature/wsl-prereqs": {},
    "ghcr.io/devcontainers/features/common-utils:2": {}
  },
  "remoteUser": "dev",
  "wslb": {
    "distroName": "UbuntuDev",
    "managed": true,
    "oobe": {
      "mode": "auto",
      "strategy": "hybrid",
      "promptForPassword": true
    },
    "state": {
      "mode": "windows-dir"
    },
    "wslconf": {
      "boot": {
        "systemd": true
      }
    },
    "distribution": {
      "shortcut": {
        "enabled": true,
        "icon": {
          "simpleIcon": "ubuntu",
          "color": "E95420",
          "style": "flat"
        }
      },
      "windowsTerminal": {
        "enabled": true,
        "template": {
          "profiles": [
            {
              "font": { "face": "Cascadia Mono" }
            }
          ]
        }
      }
    }
  }
}
```

`wslb:feature/wsl-prereqs` is recommended for generic base images (non-WSL-specialized images) to bootstrap user-management prerequisites before OOBE/user provisioning.

## Convention Defaults

You do not need to repeat many WSL-only values:

- `wslb.target` defaults to `wsl`
- image id defaults from `wslb.id` / `wslb.imageId` / `wslb.distroName` / `name`
- `wslb.workspaceName` defaults from `name`
- `wslb.outputDir` defaults to `./.wslb-out`
- `wslb.installDir` defaults to `<outputDir>/distros/<distroName>`
- user defaults from `remoteUser` / `containerUser` / `dev`
- `wslconf.user.default` defaults from resolved user
- `state.mountPoint` defaults to `/home`
- `state.fsLabel` defaults to `WSLB_STATE`

## Advanced WSL Settings

`wslb.wslconf` maps to `/etc/wsl.conf` and supports the full sections currently implemented:

- `[user]`: `default`
- `[boot]`: `systemd`, `command`, `protectBinfmt`
- `[automount]`: `enabled`, `root`, `options`, `mountFsTab`, `crossDistro`
- `[network]`: `hostname`, `generateHosts`, `generateResolvConf`
- `[interop]`: `enabled`, `appendWindowsPath`
- `[gpu]`: `enabled`
- `[time]`: `useWindowsTimezone`

`wslb.oobe` controls first-launch behavior when `wslb` manages OOBE:

- `mode`: `auto` | `predefined` | `interactive`
- `strategy`: `hybrid` | `wslb-only` | `native-only`
- `promptForPassword`: `true|false`

Legacy compatibility shortcuts:

- `wslconf.systemd` -> `wslconf.boot.systemd`
- `wslconf.automount: true|false` -> `wslconf.automount.enabled`

Forward-compat passthrough:

- any unknown object under `wslb.wslconf` is emitted as an extra INI section
- values inside extra sections must be scalar (`string`, `number`, `boolean`)

## Custom Distro Metadata

`wslb.distribution` maps to `/etc/wsl-distribution.conf`:

- `oobe.defaultName`, `oobe.defaultUid`, `oobe.command`
- `shortcut.enabled`
- `shortcut.icon`
  - string path/URL to `.ico/.svg/.png/.jpg`
  - object `{ "path": "..." }`
  - object `{ "simpleIcon": "ubuntu", "color": "E95420", "style": "flat|badge" }`
- `windowsterminal.enabled`
- `windowsterminal.profileTemplate` (compat alias `windowsterminal.ProfileTemplate` is still accepted)
- `windowsterminal.template` / `windowsterminal.templateJson`
  - inline JSON object for the Windows Terminal profile template
  - or string path to a JSON file (resolved relative to the manifest)
- `windowsTerminal` (camelCase alias for `windowsterminal`; use only one form)

Forward-compat passthrough:

- any unknown object under `wslb.distribution` is emitted as an extra section in `/etc/wsl-distribution.conf`
- values inside extra sections must be scalar (`string`, `number`, `boolean`)

If an icon is set and no profile template is provided, `wslb` auto-generates a template.

## Global VM Settings (.wslconfig)

`wslb.wslconfig` manages `%USERPROFILE%/.wslconfig`:

```json
{
  "wslb": {
    "wslconfig": {
      "apply": true,
      "wsl2": {
        "memory": "4GB",
        "processors": 4
      },
      "experimental": {
        "sparseVhd": true
      }
    }
  }
}
```

Notes:

- `apply` defaults to `true` when `wslconfig` is present.
- Values under `wsl2` and `experimental` must be scalar (`string`/`number`/`boolean`).
- When changed, `wslb` writes `%USERPROFILE%/.wslconfig` and runs `wsl --shutdown`.

## Commands

- `wslb workspace init`
- `wslb workspace validate`
- `wslb workspace plan [--all|<imageId>]`
- `wslb workspace build [--all|<imageId>] [--engine docker|podman]`
- `wslb workspace publish [--all|<imageId>]`

## References

- Microsoft custom distro packaging:
  - `https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro`
- Microsoft advanced WSL config (`wsl.conf` and `.wslconfig`):
  - `https://learn.microsoft.com/en-us/windows/wsl/wsl-config`
