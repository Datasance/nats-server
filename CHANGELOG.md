# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.14.2] - 2026-06-15

### Added

- Root `NOTICE` (EPL-2.0); per-file copyright blocks removed from Go sources.
- golangci-lint v2, unit tests, and Makefile quality targets (`lint`, `fmt-check`, `vulncheck`, `security-code`).
- CI workflow (`.github/workflows/ci.yml`) — lint, test, and Docker build smoke on `develop` (4 platforms, no registry push).
- Weekly `govulncheck` workflow (`.github/workflows/govulncheck.yml`).
- `.github/actions/set-build-env` composite action for dual-mirror registry and OCI label variables.
- Release workflow (`.github/workflows/release.yml`) — multi-arch GHCR publish on `v*` tags only.
- `Dockerfile.edge` for **linux/arm/v7** and **linux/riscv64** (Alpine 3.22 runtime).
- Multi-arch manifest merge (`:2.14.2`, `:latest`, `:main` on each release).

### Changed

- Embedded **nats-server v2.14.2** (from v2.12.x lineage).
- Go toolchain **1.26.4** (`go.mod`, builder stages, CI).
- **`NATS_TLS_DIR`** is the primary TLS directory env var; **`NATS_SSL_DIR`** remains as a deprecated fallback when `NATS_TLS_DIR` is unset.
- UBI production image slimmed: static `CGO_ENABLED=0` binaries; runtime adds only `ca-certificates` and `tzdata`.
- Renamed `Dockerfile-dev` → `Dockerfile.dev` (local debug image with nats-cli; not published).
- Replaced legacy push-on-every-branch CI with develop checks + tag-only release publish.

### Removed

- **nats-cli** from production container images.
- curl, grep, and copied OpenSSL `.so` libraries from UBI runtime image.
- Legacy `.github/workflows/push.yaml`.

### Security

- SHA-pinned GitHub Actions and Docker base images.
- `govulncheck` and `gosec` integrated into CI and Makefile.
