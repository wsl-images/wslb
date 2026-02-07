# WSLB MVP Implementation Notes

## 1) Existing Repo Baseline (Before This Change)

- CLI was a thin Cobra surface for one-off WSL actions (`build`, `install`, `ls/list`, `status`, `stop`, `shutdown`, `rm`, `version`).
- No multi-image workspace manifest, no typed application service layer, no JSON/NDJSON contract, no managed upgrade planner, and almost no tests.
- Command handlers were mostly direct command-to-package calls with minimal orchestration.

## 2) Architecture Added

- New library-first packages:
  - `internal/app` for orchestration interfaces/services
  - `internal/workspace` for manifest load/default/validate/plan
  - `internal/features` + providers (`devcontainer`, `internalfeatures`)
  - `internal/targets/wsl` for managed lifecycle logic
  - `internal/engine` for Docker/Podman selection
  - `internal/output` for JSON/NDJSON envelopes
  - `internal/platform/windows` for command execution and UTF-16 decoding
- CLI now includes:
  - `workspace init|validate|plan|build|publish`
  - `doctor`
  - `catalog feature describe <ref>`
  - `wsl state create|install|upgrade|rollback|status`

## 3) Source-Backed External Facts Used

### WSL command behavior / packaging / mount

- WSL import/export surfaces and `--import-in-place` behavior are documented in Microsoft WSL basic commands.
  - https://learn.microsoft.com/en-us/windows/wsl/basic-commands
- WSL custom distro packaging guidance confirms `.wsl` as tar-rootfs packaging flow and `wsl --install --from-file`.
  - https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro
- Microsoft custom distro docs also define `/etc/wsl-distribution.conf` for first-launch behavior and shortcut metadata:
  - `[oobe] command=...`
  - `[shortcut] enabled=true|false` and `icon=<path-to-ico>`
  - https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro
- Disk/VHD mount behavior and admin requirement for `wsl --mount` are documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/wsl2-mount-disk
- `wsl.conf` behavior (`[user] default=...`, systemd on `[boot]`) and restart/shutdown requirements are documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/wsl-config

### Dev Container Features

- Feature metadata/required files (`devcontainer-feature.json`, `install.sh`), options-to-env behavior, and ordering semantics (`dependsOn`, `installsAfter`) come from the Dev Containers spec.
  - https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/devcontainer-features.md
- Distribution details (OCI artifact naming and media types `application/vnd.devcontainers` + `application/vnd.devcontainers.layer.v1+tar`) are from the Dev Containers distribution spec.
  - https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/devcontainer-features-distribution.md

### Playwright and local harness pattern

- Playwright `webServer` usage and lifecycle handling are documented here:
  - https://playwright.dev/docs/test-webserver
- API-driven verification via request context:
  - https://playwright.dev/docs/api/class-apirequestcontext
- Node child process implementation details for robust command runner behavior:
  - https://nodejs.org/api/child_process.html

### Icon source (Simple Icons)

- Simple Icons publishes icon slugs and SVG assets:
  - https://simpleicons.org
- Reliable machine-readable SVG fetch endpoint used in implementation:
  - `https://cdn.jsdelivr.net/npm/simple-icons@v15/icons/<slug>.svg`
  - https://www.jsdelivr.com/package/npm/simple-icons

## 4) Key Design Decisions

- Managed WSL upgrade uses candidate verification before any stable unregister.
- Non-interactive state creation is deterministic: fail unless explicit fallback flag is set.
- `doctor` emits actionable remediation text and machine-readable check IDs.
- Workspace planning returns deterministic names/paths for UI preview.
- JSON envelope is stable and command-agnostic; NDJSON events are optional for progress streams.

## 5) Caveats

- VHDX creation/mount behavior is host-policy dependent (admin/UAC/service environment).
- Full E2E WSL install/upgrade/rollback requires host prerequisites (WSL + available container engine daemon).
- This build is intentionally WSL-only; non-WSL targets are out of scope.

## 6) Devcontainer Superset Notes

- `workspace.Load` now supports `.devcontainer/devcontainer.json` as a first-class manifest source.
- Standard Dev Container fields are mapped into the WSL workspace model:
  - `name`, `image`, `features`, `remoteUser` / `containerUser`.
- WSL-specific controls are provided via a top-level `wslb` extension object:
  - `workspaceName`, `outputDir`, `imageId`, `distroName`, `installDir`, `managed`, `state`, `wslconf`, `distribution`.
- `wslconf` now supports the sectioned model documented by Microsoft:
  - `user`, `boot`, `automount`, `network`, `interop`
  - legacy shorthand (`systemd`, `automount` booleans) remains accepted.
- `distribution.shortcut.icon` supports:
  - direct `.ico` path/URL
  - Simple Icons slug (`simpleIcon`) with optional `color` and `style` (`flat` default, `badge` optional); `wslb` fetches SVG and converts to `.ico`.
- Icon/start-menu reliability updates:
  - distro metadata (`/etc/wsl-distribution.conf`) and icon are now embedded into the built rootfs before install.
  - install prefers `wsl --install --from-file` so WSL applies custom-distro metadata at install time.
- Config simplification updates:
  - convention-over-configuration defaults remove repeated `target/workspaceName/outputDir/installDir/imageId` requirements for typical WSL usage.
  - `wslb.user` is the preferred single user override (legacy `defaultUser` still supported).
  - managed user creation + managed mount wiring are automatic, so baseline manifests do not need to repeat internal `wslb:feature/*` entries.
- Manifest format updates:
  - both JSON and JSONC are accepted.
  - schema file provided at `schemas/wslb-workspace.schema.json` with raw GitHub URL for `$schema` association.
- Feature tool availability is now verified during install/upgrade for selected feature refs (`node`, `go`, `python`, `git`, `github-cli`).
