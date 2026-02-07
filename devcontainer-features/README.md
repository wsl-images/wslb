# wsl-images Dev Container Features

Official Dev Container Features published by `wsl-images`.

## Source Layout

- `src/<feature-id>/devcontainer-feature.json`
- `src/<feature-id>/install.sh`
- `src/<feature-id>/NOTES.md`
- `test/<feature-id>/scenarios.json`
- `test/<feature-id>/test.sh`

## Features

- `wsl-prereqs`
- `first-boot-user`
- `persist-home`

## Publish

GitHub Actions publishes to GHCR on push to `main`:

- `ghcr.io/wsl-images/devcontainer-features/wsl-prereqs:1`
- `ghcr.io/wsl-images/devcontainer-features/first-boot-user:1`
- `ghcr.io/wsl-images/devcontainer-features/persist-home:1`

## Local Test

Run from this folder:

```bash
devcontainer features test --features src/wsl-prereqs --base-image ubuntu:24.04
devcontainer features test --features src/first-boot-user --base-image ubuntu:24.04
devcontainer features test --features src/persist-home --base-image ubuntu:24.04
```
