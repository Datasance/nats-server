# pot-nats – NATS Server for Datasance PoT

NATS server image for use on **Datasance PoT** (Kubernetes or edge with PoT-agent). Config, account config, and SSL certs are provided via **volume mounts**; the wrapper starts [nats-server](https://github.com/nats-io/nats-server) v2.12.4 and watches for file changes, triggering a config reload (SIGHUP) without restart.

Both **Kubernetes** (PoT-controller mounting ConfigMaps/Secrets) and **PoT edge** (PoT-agent binding config) use the same contract: mount the server config, account config, and SSL directory at the paths below (or override with env vars).

## Environment variables

| Variable            | Default                   | Description                                                                 |
| ------------------- | ------------------------- | --------------------------------------------------------------------------- |
| `NATS_CONF`         | `/etc/nats/config/server.conf`   | Server config file path (passed to nats-server as `-c`).                    |
| `NATS_ACCOUNTS`     | `/etc/nats/config/accounts.conf` | Account config file; watched for changes and triggers reload.               |
| `NATS_SSL_DIR`      | `/etc/nats/certs`           | Directory for TLS material; watched for changes and triggers reload.        |
| `NATS_JWT_DIR`      | `/etc/nats/jwt`             | Directory for JWT assets; watched for changes and triggers reload.          |
| `NATS_CREDS_DIR`    | `/etc/nats/creds/`          | Directory for creds files; watched for changes and triggers reload.         |
| `NATS_SERVER_BIN`   | `/home/runner/bin/nats-server` | Path to the nats-server binary (override for local dev, e.g. `nats-server`). |
| `NATS_MONITOR_PORT` | `8222`                    | HTTP monitoring port (nats-server `-m`). Set to `0` to disable.             |
| `NATS_SYS_USER_CRED_PATH` | (none)              | Path to system account user credentials file. If not absolute, resolved relative to `NATS_CREDS_DIR`. When set, the wrapper calls the JetStream Account Purge API for accounts removed from the resolver (see below). |
| `NATS_CLIENT_URL`   | `nats://127.0.0.1:4222` | URL used by the wrapper to connect to NATS for the JetStream purge API.   |
| `NATS_JETSTREAM_STORE_DIR` | (none)            | JetStream store directory (same as `jetstream.store_dir` in server config). If unset, the wrapper tries to parse it from the server config file. |

The server config file may use **environment variable placeholders** (e.g. `$SERVER_NAME`, `$HUB_NAME`). NATS resolves these from the process environment; the wrapper preserves the container environment when starting nats-server so K8s/PoT-injected vars are available.

## Volume mounts

- **Server config**: Mount the NATS server config file at `NATS_CONF`. It may `include` the account file and reference cert paths under `NATS_SSL_DIR`.
- **Account config**: Mount at `NATS_ACCOUNTS` (or include it from the server config via a relative path).
- **SSL certs**: Mount TLS material (e.g. `ca.crt`, `tls.crt`, `tls.key`) under `NATS_SSL_DIR` (or subdirs). Paths in the server config should match the mount location.

## Reload behaviour

The wrapper watches `NATS_CONF`, `NATS_ACCOUNTS` (if present), `NATS_SSL_DIR`, `NATS_JWT_DIR`, and `NATS_CREDS_DIR` (directory watchers start only if paths exist). On any change (after a short debounce), it sends **SIGHUP** to the running nats-server process. NATS reloads config and certs without restart.

## JetStream account purge (reconcile on account removal)

When an account is removed from the JWT resolver directory, NATS no longer accepts that account but JetStream may still hold its data. The wrapper reconciles accounts that have JetStream data on disk (subdirectories under the JetStream store directory) with the current resolver accounts (`NATS_JWT_DIR`). Any account that has a JetStream directory but is no longer in the resolver is purged via the JetStream Account Purge API (`$JS.API.ACCOUNT.PURGE.{account}`) using system account credentials. This runs once after startup (after a short delay) and again after each JWT directory change (after reload). No snapshot file is used; behaviour is consistent across reboots. Set `NATS_SYS_USER_CRED_PATH` (and optionally `NATS_JETSTREAM_STORE_DIR` or rely on parsing from server config) to enable purge; if unset, reconciliation still runs but purge API calls are skipped.

## Image

- **Base**: Red Hat UBI 9 micro, non-root user `runner` (uid 10000).
- **Binaries**: `pot-nats` (entrypoint), nats-server v2.12.4, and **nats-cli** at `/home/runner/bin/nats` for debugging.

## Build

```bash
make build
# or
go build -o bin/pot-nats ./cmd/pot-nats
```

Other targets: `make test`, `make lint`, `make clean`, `make docker-build` (image name via `IMAGE=...`).

## License

See [LICENSE](LICENSE).
