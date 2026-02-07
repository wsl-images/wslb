# WSL Managed Upgrades

Managed mode separates immutable system artifacts from external state so upgrades/rollback can preserve user files.

## Commands

- `wslb wsl state create <imageId>`
- `wslb wsl install <imageId>`
- `wslb wsl upgrade <imageId>`
- `wslb wsl rollback <imageId> [--to <versionOrBackupPath>]`
- `wslb wsl status <imageId>`

## Safety Model

- Stable distro is never unregistered until candidate verification succeeds.
- Upgrade flow:
  1. Build artifact
  2. Import candidate distro
  3. Verify candidate
  4. Export stable backup
  5. Swap stable import
  6. Persist version metadata
- Rollback imports from recorded backup tar and re-applies WSL settings.

## State Modes

- `vhdx` (primary)
- `windows-dir` (fallback)

For non-interactive runs, fallback is only used when explicitly enabled with `--fallback-windows-dir`.

## Admin and Mount Requirements

Microsoft documents `wsl --mount` as requiring admin privileges for disk attach:
- https://learn.microsoft.com/en-us/windows/wsl/wsl2-mount-disk

If VHD attach fails due to privilege:

- rerun elevated:
  - `wsl --mount --vhd <path> --bare`
- or rerun with fallback:
  - `wslb wsl state create <imageId> --fallback-windows-dir`

## Recovery Commands

- list distros: `wsl -l -v`
- terminate stable: `wsl --terminate <distro>`
- import from backup: `wsl --import <distro> <installDir> <backup.tar> --version 2`
- full shutdown: `wsl --shutdown`

## References

- WSL basic commands:
  - https://learn.microsoft.com/en-us/windows/wsl/basic-commands
- WSL custom distro packaging (`/etc/wsl-distribution.conf`, OOBE, shortcut icon):
  - https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro
- WSL config (`/etc/wsl.conf`):
  - https://learn.microsoft.com/en-us/windows/wsl/wsl-config
- Simple Icons catalog (used for `wslb.distribution.shortcut.icon.simpleIcon`):
  - https://simpleicons.org
