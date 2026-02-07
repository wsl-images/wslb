# WSLB

`wslb` builds and manages WSL images from a `devcontainer.json/jsonc` superset.

It keeps legacy commands (`build`, `install`, `ls/list`, `status`, `stop`, `shutdown`, `rm`, `version`) and adds workspace + managed lifecycle APIs.

## Quick Start

```bash
go build -o wslb.exe .
./wslb.exe workspace init
./wslb.exe workspace validate --json
./wslb.exe workspace plan --all --json
```

You can also use a `devcontainer` superset with a top-level `wslb` block (`.json` or `.jsonc`):

```bash
./wslb.exe workspace validate --workspace-file .devcontainer/devcontainer.json --json
./wslb.exe wsl --workspace-file .devcontainer/devcontainer.json install <imageId>
```

The `wslb` superset supports full `wsl.conf` sections and `/etc/wsl-distribution.conf` settings (`oobe`, shortcut icon), including `simpleIcon` slugs from Simple Icons.
Schema URL for editor association:
- `https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json`

## New Commands

- `wslb workspace init|validate|plan|build|publish`
- `wslb doctor`
- `wslb catalog feature describe <ref>`
- `wslb wsl state create|install|upgrade|rollback|status <imageId>`

All major new commands support `--json`; long-running commands optionally emit `--ndjson`.

## Docs

- `docs/workspaces.md`
- `docs/wsl-managed-upgrades.md`
- `docs/json-contract.md`
- `docs/e2e-testing.md`

## Examples

- `examples/devcontainer/devcontainer.json`
- `examples/devcontainer/minimal.json`
- `examples/devcontainer/managed-vhdx.json`
- `examples/devcontainer/ubuntu-pro.json`

## Unattended Verification

- PowerShell runner:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/e2e.ps1 -Json -Out .\e2e-report.json
```

- Browser-channel harness:

```bash
npm install
npx playwright install
npm run test:playwright
```

## WSL Stuck Recovery

If WSL commands hang, run:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/wsl-recover.ps1
```

If that is not enough, run elevated:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/wsl-recover.ps1 -Deep
```

Or auto-elevate:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/wsl-recover.ps1 -Deep -AutoElevate
```
