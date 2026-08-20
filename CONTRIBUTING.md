# Contributing to NATS Server wrapper

Thank you for contributing to the **ioFog NATS wrapper image** (`github.com/eclipse-iofog/nats-server`). The wrapper binary is **`iofog-nats`**; it execs upstream **nats-server v2.14.5** as a child process with config watch, JWT sync, JetStream reconcile, and claims push.

## Dual-mirror workflow

The same codebase is maintained on two GitHub remotes. CI variables select the registry and OCI labels per repo; wrapper code does **not** branch on product name.

| Variable | [eclipse-iofog/nats-server](https://github.com/eclipse-iofog/nats-server) | [Datasance/nats-server](https://github.com/Datasance/nats-server) |
|----------|---------------------------------------------------------------------------|-------------------------------------------------------------------|
| `IMAGE_REGISTRY` | `ghcr.io/eclipse-iofog` | `ghcr.io/datasance` |
| `NATS_CONTAINER_IMAGE` | `ghcr.io/eclipse-iofog/nats` | `ghcr.io/datasance/nats` |
| `OCI_SOURCE_REPO` | `https://github.com/eclipse-iofog/nats-server` | `https://github.com/Datasance/nats-server` |
| `NATS_DISTRIBUTION` | `iofog` | `datasance` |
| `NATS_GITHUB_REPO` | `eclipse-iofog/nats-server` | `Datasance/nats-server` |

Set these under **Settings → Actions → Variables** on each repo (Plan 8 bootstrap). When unset, [`.github/actions/set-build-env`](.github/actions/set-build-env/action.yml) derives defaults from `github.repository`.

**Development flow:** implement on **`Datasance/nats-server` `develop`**, open a PR to **`eclipse-iofog/nats-server` `develop`**. Integration branch is **`develop`** (not `main` for daily work).

### Remotes (developer clone)

```bash
git remote add upstream https://github.com/eclipse-iofog/nats-server.git
git remote add datasance https://github.com/Datasance/nats-server.git
```

Push feature branches to **datasance**; merge target upstream is **upstream develop**.

## Branch naming

Feature branches: **`nats-server/<plan>-<slug>`** (e.g. `nats-server/03-ci`, `nats-server/07-docs`).

## Required gates

Run from the repository root before opening a pull request:

```bash
make fmt-check
make lint              # golangci-lint — zero issues required
make test
make security-code     # gosec on ./cmd/... ./internal/...
make vulncheck         # govulncheck + go mod verify
```

CI runs the same gates on PR and push to **`develop`**, plus a Docker runtime smoke test on all four platforms (build, start with `test/fixtures/ci/server.conf`, verify `/healthz` from host and in-container `curl`).

## Pull requests

1. Target **`develop`** with a focused description and test notes.
2. Update **CHANGELOG.md** under `[Unreleased]` for user-facing changes that ship before the next tagged release.
3. Preserve wrapper integration contracts documented in [README.md](README.md): child exec of nats-server, SIGHUP reload (leaf: SIGHUP for TLS only, SIGINT+restart otherwise), JWT sync, JetStream purge, claims push, and the **`NATS_*`** env contract.
4. Do **not** add product-specific runtime env vars (`IOFOG_*`, `POT_*`, `NATS_DISTRIBUTION` in Go code).
5. Do **not** rename the workload label **`iofog-nats`** or the wrapper binary name.

## Updating digest-pinned dependencies

### Docker base images

Production Dockerfiles pin base images by digest:

- **`Dockerfile`** — `golang:1.26.6-alpine`, UBI 9 minimal/micro (amd64/arm64).
- **`Dockerfile.edge`** — `golang:1.26.6-alpine`, `alpine:3.22` (arm/v7, riscv64).

When bumping a base image:

1. Pull the new tag and note its **`sha256:`** digest.
2. Update the **`FROM ...@sha256:...`** line in the relevant Dockerfile(s).
3. Update the matching comment line above each `FROM` (edgelet pattern — digest documented in comment and pin).
4. Rebuild locally: `make docker-build` and/or `docker buildx build -f Dockerfile.edge ...`.
5. Confirm CI docker-build-smoke job passes on your PR.

### GitHub Actions

Workflows SHA-pin third-party actions with a version comment (e.g. `# v4`). When upgrading an action:

1. Find the release tag’s full commit SHA on GitHub.
2. Replace `uses: org/action@<sha> # vX.Y.Z` in `.github/workflows/*.yml`.
3. Keep **`permissions: read-all`** on CI workflows; release publish job alone uses **`packages: write`**.

### Go module and embedded nats-server

- Bump **`go`** in `go.mod` and **`GO_VERSION`** in workflows together.
- Pin embedded server with `go install github.com/nats-io/nats-server/v2@vX.Y.Z` in Dockerfiles; align git tag and CHANGELOG with that version.

## Release procedure (maintainers)

Container images publish **only** on **`v*`** git tags (not ordinary `develop` pushes). For dual-mirror releases:

1. Merge to **`develop`** on both remotes at the **same SHA**.
2. Create identical annotated tags (e.g. **`v2.14.5`**) on both remotes and push.
3. Verify both **release.yml** workflows succeed and tags `:semver`, `:latest`, `:main` exist on each registry.

Record published image digests in CHANGELOG release notes when tagging.

## Security

Report vulnerabilities privately — do not file public issues for exploitable findings.

## Questions

Open a GitHub issue for bugs and feature discussion.
