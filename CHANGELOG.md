# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.15.1] - 2026-10-10

### Added

- `docs/nats-wrapper.md` — how `iofog-nats` starts `nats-server`, watches mounts, and applies config, TLS, and JWT changes on a Kubernetes hub and on an edge node. Linked from `README.md`.

### Changed

- Embedded **nats-server v2.15.1** (from v2.15.0) in `Dockerfile`, `Dockerfile.dev`, and `Dockerfile.edge`; see [nats-server compare v2.15.0...v2.15.1](https://github.com/nats-io/nats-server/compare/v2.15.0...v2.15.1) and the upstream [v2.15.1 release notes](https://github.com/nats-io/nats-server/releases/tag/v2.15.1).
- Go toolchain **1.26.9** (from 1.26.6) in `go.mod`, CI workflows (`ci.yml`, `govulncheck.yml`, `release.yml`), and Docker builder stages.
- **`golang:1.26.9-alpine`** builder digest pin in `Dockerfile` and `Dockerfile.edge`. `Dockerfile.dev` tracks the same tag.
- **`Dockerfile`** — refresh the **UBI 9 minimal** digest pin (`user-stage`, `runtime-staging`). UBI 9 micro and Alpine 3.22 pins were already current.
- Docs (`README.md`, `CONTRIBUTING.md`) — version references updated to **nats-server v2.15.1** and **Go 1.26.9**.

### Security

- Rebuild the wrapper and the embedded `nats-server` with **Go 1.26.9**. That clears the reachable `net/http` findings reported by [govulncheck #20](https://github.com/eclipse-iofog/nats-server/actions/runs/38041695913/job/114183055503) (`jspurge.PurgeAccount` calling `nats.Connect`). Go 1.26.9 is the 8 October 2026 security release; details are in the [release announcement](https://www.openwall.com/lists/oss-security/2026/10/08/9).

### Notes

- On top of the 2.15.0 upgrade guide: `max_ha_assets` is now enforced during stream and consumer placement, scale, and move; the legacy `$GR.` gateway reply prefix used by servers before v2.1.2 is gone; auth-callout `auth_users` bypass issues are fixed. No wrapper code or `NATS_*` env contract changes.

## [2.15.0] - 2026-09-26

### Changed

- Embedded **nats-server v2.15.0** (from v2.14.7) in `Dockerfile`, `Dockerfile.dev`, and `Dockerfile.edge`; see [nats-server compare v2.14.7...v2.15.0](https://github.com/nats-io/nats-server/compare/v2.14.7...v2.15.0) and the upstream [v2.15.0 release notes](https://github.com/nats-io/nats-server/releases/tag/v2.15.0).
- Wrapper dependency **`github.com/nats-io/nats.go` v1.54.0** (from v1.53.1) for claims push and JetStream account purge.
- Docs (`README.md`, `CONTRIBUTING.md`) — version references updated to **v2.15.0**.

### Notes

- Cluster operators should read the upstream [Upgrade to NATS 2.15](https://docs.nats.io/release-notes/upgrade-to-2.15) guide (desired-state metalayer, default per-stream consumer limit of 1000, leaf/sync behavior). No wrapper code or env contract changes are required.


## [2.14.7] - 2026-09-26

### Changed

- Embedded **nats-server v2.14.7** (from v2.14.5) in `Dockerfile`, `Dockerfile.dev`, and `Dockerfile.edge`; includes upstream [v2.14.6](https://github.com/nats-io/nats-server/releases/tag/v2.14.6) and [v2.14.7](https://github.com/nats-io/nats-server/releases/tag/v2.14.7) fixes.
- **`Dockerfile`** (linux/amd64, linux/arm64) — refresh digest pins for **UBI 9 minimal** (`user-stage`, `runtime-staging`) and **UBI 9 micro** (final runtime image).
- **`Dockerfile.edge`** (linux/armv7, linux/riscv64) — refresh digest pins for **alpine 3.22** (`runtime-staging`).
- Docs (`README.md`, `CONTRIBUTING.md`) — version references updated to **v2.14.7**.

### Fixed

- **`scripts/check-dockerfile-digests.sh`** — parse digest-pinned `FROM` lines that use BuildKit flags (e.g. `--platform=$BUILDPLATFORM`) so builder-stage pins are checked and update suggestions preserve the flag.

### Security

- Upstream JetStream stability and correctness fixes in v2.14.6 and v2.14.7, including filestore sync/compaction, Raft catchup signalling, consumer state races, and metalayer stall fixes; see [nats-server compare v2.14.5...v2.14.7](https://github.com/nats-io/nats-server/compare/v2.14.5...v2.14.7) for the full upstream changelog.

## [2.14.5] - 2026-08-20

### Changed

- Embedded **nats-server v2.14.5** (from v2.14.3); includes upstream [v2.14.4](https://github.com/nats-io/nats-server/releases/tag/v2.14.4) and [v2.14.5](https://github.com/nats-io/nats-server/releases/tag/v2.14.5) fixes.
- Go toolchain **1.26.6** (from 1.26.5) in `go.mod`, CI workflows, and Docker builder stages.
- **`golang:1.26.6-alpine`** builder image digest pin in `Dockerfile`, `Dockerfile.dev`, and `Dockerfile.edge`.
- **`Dockerfile`** (linux/amd64, linux/arm64) — refresh digest pins for **UBI 9 minimal** and **UBI 9 micro**.
- Wrapper dependency **`github.com/nats-io/nats.go` v1.53.1** (from v1.51.0).

### Security

- Upstream [v2.14.4](https://github.com/nats-io/nats-server/releases/tag/v2.14.4) auth and permission fixes, including TLS `verify_and_map` blank-password bypass, `no_auth_user` with auth callouts, and MQTT `$MQTT.>` subscription bypass; see [nats-server compare v2.14.3...v2.14.5](https://github.com/nats-io/nats-server/compare/v2.14.3...v2.14.5) for the full upstream changelog (JetStream stability, monitoring, and leafnode `dial_timeout` in v2.14.5).

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
