# pot-nats – NATS Server for Datasance PoT

NATS server image for use on **Datasance PoT** (Kubernetes or edge with PoT-agent). Config, account config, and SSL certs are provided via **volume mounts**; the wrapper starts [nats-server](https://github.com/nats-io/nats-server) v2.12.4 and watches for file changes, triggering a config reload (SIGHUP) without restart.

Both **Kubernetes** (PoT-controller mounting ConfigMaps/Secrets) and **PoT edge** (PoT-agent binding config) use the same contract: mount the server config, account config, and SSL directory at the paths below (or override with env vars).

## Environment variables

| Variable            | Default                   | Description                                                                 |
| ------------------- | ------------------------- | --------------------------------------------------------------------------- |
| `NATS_CONF`         | `/etc/nats/config/server.conf`   | Server config file path (passed to nats-server as `-c`).                    |
| `NATS_ACCOUNTS`     | `/etc/nats/config/accounts.conf` | Account config file; watched for changes and triggers reload.               |
| `NATS_SSL_DIR`      | `/etc/nats/certs`           | Directory for TLS material; watched for changes and triggers reload.        |
| `NATS_SERVER_BIN`   | `/home/runner/bin/nats-server` | Path to the nats-server binary (override for local dev, e.g. `nats-server`). |
| `NATS_MONITOR_PORT` | `8222`                    | HTTP monitoring port (nats-server `-m`). Set to `0` to disable.             |

The server config file may use **environment variable placeholders** (e.g. `$SERVER_NAME`, `$HUB_NAME`). NATS resolves these from the process environment; the wrapper preserves the container environment when starting nats-server so K8s/PoT-injected vars are available.

## Volume mounts

- **Server config**: Mount the NATS server config file at `NATS_CONF`. It may `include` the account file and reference cert paths under `NATS_SSL_DIR`.
- **Account config**: Mount at `NATS_ACCOUNTS` (or include it from the server config via a relative path).
- **SSL certs**: Mount TLS material (e.g. `ca.crt`, `tls.crt`, `tls.key`) under `NATS_SSL_DIR` (or subdirs). Paths in the server config should match the mount location.

## Reload behaviour

The wrapper watches `NATS_CONF`, `NATS_ACCOUNTS` (if present), and `NATS_SSL_DIR`. On any change (after a short debounce), it sends **SIGHUP** to the running nats-server process. NATS reloads config and certs without restart.

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
