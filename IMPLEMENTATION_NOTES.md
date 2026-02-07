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
  - `[windowsterminal] enabled=...` and `ProfileTemplate=<path>`
  - https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro
- Disk/VHD mount behavior and admin requirement for `wsl --mount` are documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/wsl2-mount-disk
- `wsl.conf` behavior (`[user] default=...`, systemd on `[boot]`) and restart/shutdown requirements are documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/wsl-config
- Advanced per-distro `wsl.conf` sections and keys (`[automount]`, `[network]`, `[interop]`, `[user]`, `[boot]`, `[gpu]`, `[time]`) are documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/wsl-config
- Custom distro `windowsterminal.ProfileTemplate` in `/etc/wsl-distribution.conf` is documented by Microsoft.
  - https://learn.microsoft.com/en-us/windows/wsl/build-custom-distro

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
  - `https://simpleicons.org/icons/<slug>.svg`
  - https://simpleicons.org
- Reliable machine-readable slug index used by Store icon search:
  - `https://raw.githubusercontent.com/simple-icons/simple-icons/develop/slugs.md`
  - https://github.com/simple-icons/simple-icons

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
- `distribution.windowsterminal` supports both section-style and devcontainer-friendly input:
  - canonical `windowsterminal` key
  - alias `windowsTerminal` key
  - `profileTemplate` path or inline/file-backed `template` JSON payload
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

## 7) OOBE and Distro Compatibility Expansion

- Added first-class `wslb.oobe` config:
  - `mode`: `auto | predefined | interactive`
  - `strategy`: `hybrid | wslb-only | native-only`
  - `promptForPassword`: bool
- `interactive` mode no longer auto-writes `wslconf.user.default` and no longer forces default-user verification.
- Managed mount ownership now resolves runtime UID/GID from the actual created user in the distro, preventing UID drift issues when base images already occupy UID 1000.
- `ensureDistroDefaultUser` now hard-fails if user creation does not succeed (instead of silently continuing).

## 8) Forward-Compatible Config Coverage

- Added pass-through section support so users can control additional config keys without waiting for code changes:
  - unknown object sections under `wslb.wslconf` are emitted into `/etc/wsl.conf`
  - unknown object sections under `wslb.distribution` are emitted into `/etc/wsl-distribution.conf`
- Validation enforces scalar values (`string|number|boolean`) for pass-through section keys.

## 9) Verification Added

- Added docker-backed official distro matrix test:
  - `internal/targets/wsl/oobe_matrix_test.go`
  - gated by `WSLB_OOBE_MATRIX=1`
  - runs OOBE smoke checks across latest official `wsl-images` images.
- Added wrapper script and report generator:
  - `scripts/oobe-matrix.ps1`
  - writes `wsl-images-oobe-matrix-report.json`
- Added OS-agnostic internal feature for generic distro bases:
  - `wslb:feature/wsl-prereqs`
  - installs/bootstrap user-management prerequisites across apt/dnf/yum/microdnf/apk/zypper/pacman/xbps/tdnf ecosystems
  - provides `getent` fallback shim when missing.
- Added second matrix for broad distro base-image compatibility:
  - `internal/targets/wsl/base_image_matrix_test.go`
  - `scripts/base-image-matrix.ps1`
  - writes `base-image-matrix-report.json`
- Verified current run locally:
  - `go test ./...` passes
  - `scripts/e2e.ps1` pass (install/upgrade/rollback sentinel preservation)
  - OOBE matrix pass against 19 official `wsl-images` entries.

Additional references used for this pass:
- Official WSL images package catalog:
  - https://github.com/orgs/wsl-images/packages?repo_name=images

## 10) Store UI (Local App) for Multi-Distro Profile Management

- Added `tools/wslb-store` as a local Store-style web app backed by `wslb` CLI APIs.
- The Store persists multiple profiles (`.wslb-store/profiles/*.json`) and user defaults (`.wslb-store/defaults.json`).
- Each profile maps to a devcontainer superset manifest and can write its own manifest path (for example `.devcontainer/ubuntu-dev.json`), enabling multiple distro configurations side-by-side.
- The Store executes lifecycle actions through CLI JSON contracts:
  - `workspace validate|plan|build`
  - `wsl state create|install|upgrade|rollback|status`
  - `doctor`
- Added feature catalog search sourced from the official containers.dev feature registry page.
- Added parser tests for feature index extraction.

### Source-backed facts for Store decisions

- Ubuntu App Center repository is Flutter-based, so it is a UX reference rather than a drop-in web codebase for this Node/Windows local tooling context.
  - https://github.com/ubuntu/app-center
- `containers.dev/features` exposes the authoritative, continuously-updated feature listing and references for devcontainer feature discovery.
  - https://containers.dev/features
- Dev Container feature semantics used by UI assumptions (feature references, options, dependency ordering) come from the Dev Container Features specification.
  - https://containers.dev/implementors/features/
