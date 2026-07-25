# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.14.3-2] - 2026-07-25

Wrapper-only release; embedded upstream server unchanged.

### Changed

- **`Dockerfile`** (linux/amd64, linux/arm64) — refresh digest pins for **UBI 9 minimal** (`user-stage`, `runtime-staging`) and **UBI 9 micro** (final runtime image).

### Embedded server

- **nats-server v2.14.3** (unchanged).

## [2.14.3-1] - 2026-07-10

Wrapper-only release; embedded upstream server unchanged.

### Changed

- Go toolchain **1.26.5** (from 1.26.4) in `go.mod`, CI workflows, and Docker builder stages.
- **`golang:1.26.5-alpine`** builder image digest pin in `Dockerfile`, `Dockerfile.dev`, and `Dockerfile.edge`.
- Release workflow (`.github/workflows/release.yml`) — disable setup-go module cache and Docker Buildx GHA layer cache for reproducible cold release builds.

### Embedded server

- **nats-server v2.14.3** (unchanged).

## [2.14.3] - 2026-07-04

### Changed

- Embedded **nats-server v2.14.3** (from v2.14.2).

### Security

- Upstream patch for [GHSA-hmmp-q8cx-v964](https://github.com/nats-io/nats-server/security/advisories/GHSA-hmmp-q8cx-v964) (`no_auth_user` connection restriction bypass); see [nats-server v2.14.3](https://github.com/nats-io/nats-server/releases/tag/v2.14.3) for full upstream fixes (JWT claim updates, JetStream/Raft stability, leaf/service import fixes).

## [2.14.2] - 2026-06-15

### Added

- Root `NOTICE` (EPL-2.0); per-file copyright blocks removed from Go sources.
- golangci-lint v2, unit tests, and Makefile quality targets (`lint`, `fmt-check`, `vulncheck`, `security-code`).
- CI workflow (`.github/workflows/ci.yml`) — lint, test, and Docker runtime smoke on `develop` (all four platforms).
- Weekly `govulncheck` workflow (`.github/workflows/govulncheck.yml`).
- `.github/actions/set-build-env` composite action for dual-mirror registry and OCI label variables.
- Release workflow (`.github/workflows/release.yml`) — multi-arch GHCR publish on `v*` tags only.
- `Dockerfile.edge` for **linux/arm/v7** and **linux/riscv64** (Alpine 3.22 runtime).
- Multi-arch manifest merge (`:2.14.2`, `:latest`, `:main` on each release).
- **`curl`** and **`grep`** in production images on all platforms for in-container healthchecks.
- CI runtime smoke test: `test/fixtures/ci/server.conf` and `scripts/docker-smoke.sh` verify `/healthz` from the runner and in-container `curl`.
- Public docs: README badges, `CONTRIBUTING.md` dual-mirror workflow.

### Changed

- Embedded **nats-server v2.14.2** (from v2.12.x lineage).
- Go toolchain **1.26.4** (`go.mod`, builder stages, CI).
- **`NATS_TLS_DIR`** is the primary TLS directory env var; **`NATS_SSL_DIR`** remains as a deprecated fallback when `NATS_TLS_DIR` is unset.
- UBI production image: static `CGO_ENABLED=0` binaries; runtime adds `ca-certificates`, `tzdata`, `curl-minimal`, and `grep`.
- Renamed `Dockerfile-dev` → `Dockerfile.dev` (local debug image with nats-cli; not published).
- Replaced legacy push-on-every-branch CI with develop checks + tag-only release publish.

### Removed

- **nats-cli** from production container images.
- Legacy `.github/workflows/push.yaml`.

### Security

- SHA-pinned GitHub Actions and Docker base images.
- `govulncheck` and `gosec` integrated into CI and Makefile.
