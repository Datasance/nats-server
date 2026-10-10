# How the NATS wrapper works

`iofog-nats` is the container entrypoint. It starts upstream `nats-server` v2.15.1 as a child process, watches the files the platform mounts into the container, and applies those changes with `SIGHUP`, a restart, or a JWT sync. The same binary runs on a Kubernetes control-plane hub and on an edge node (Eclipse ioFog Agent or Datasance PoT Edgelet). There is no PoT, ioFog, or Kubernetes branch in the wrapper. The platforms differ only in who writes the files and how they appear in the container.

This page describes the wrapper in this repository (`cmd/iofog-nats` and `internal/*`). Controller and operator behavior is included only where it explains which files and environment variables the wrapper actually sees.

## What runs in the container

| Piece | Path | Role |
| --- | --- | --- |
| Wrapper | `/home/runner/bin/iofog-nats` | Entrypoint. Watches files, signals the child, syncs JWTs, purges removed JetStream accounts, pushes claims. |
| Server | `/home/runner/bin/nats-server` | Upstream binary. Listens for clients, cluster, leaf, and MQTT. Owns config reload on `SIGHUP`. |
| User | uid `10000` (`runner`) | Every image runs as this user. The Kubernetes operator sets the same uid, gid, and `fsGroup`. |

The child is a separate process. The wrapper does not link `nats-server` in-process. It starts the binary with:

```text
nats-server -c <NATS_CONF> -m <NATS_MONITOR_PORT>
```

`-m` is omitted when `NATS_MONITOR_PORT` is `0`. The working directory is the directory that contains the server config, so a relative `include` resolves next to that file. The child inherits the full container environment, so any `$NAME` left in the config file is expanded by `nats-server` from that environment.

Stdout and stderr from `nats-server` are copied into the wrapper log. If the child exits and the wrapper did not ask for a restart, the wrapper exits with it (`1` on error, `0` on a clean exit).

## Where the same image is used

```text
Kubernetes control plane                         Edge node
(ioFog operator StatefulSet "nats")              (ioFog Agent or PoT Edgelet)
        |                                                |
        | ConfigMaps and Secrets                         | Controller system microservice "nats"
        | mounted by kubelet                             | catalog item name "NATS"
        v                                                v
   /etc/nats/config/server.conf                 /etc/nats/config/server.conf
   /tmp/nats/jwt/*.jwt                          /tmp/nats/jwt/*.jwt
   /etc/nats/certs/...                          /etc/nats/certs/...
   /etc/nats/creds/...                          /etc/nats/creds/...
   /home/runner/data  (PVC js-data)             /home/runner/data  (agent volume)
        |                                                |
        +------------------+-----------------------------+
                           v
                    iofog-nats
                           |
                    nats-server (child)
```

Both sides use image name `nats` (`ghcr.io/eclipse-iofog/nats` or `ghcr.io/datasance/nats`). The workload identity on an agent stays `iofog-nats`. Edgelet records that role as `edgelet.iofog.org/role=nats`.

### Kubernetes hub

The operator creates StatefulSet `nats` (at least two replicas when NATS is enabled). That pod is always a **server**, never a leaf:

| Env | Value the operator sets |
| --- | --- |
| `NATS_SERVER_MODE` | `server` |
| `NATS_CONF` | `/etc/nats/config/server.conf` |
| `NATS_JWT_DIR` | `/home/runner/nats/jwt` |
| `NATS_JWT_MOUNT_DIR` | `/tmp/nats/jwt` |
| `NATS_TLS_DIR` | `/etc/nats/certs` |
| `NATS_CREDS_DIR` | `/etc/nats/creds` |
| `NATS_SYS_USER_CRED_PATH` | `/etc/nats/creds/admin-hub.creds` |
| `NATS_JETSTREAM_STORE_DIR` | `/home/runner/data` |
| `NATS_MONITOR_PORT` | `8222` |
| `SELFNAME` | Pod name (`metadata.name`) |

Mounts:

| Kubernetes object | Mount path |
| --- | --- |
| ConfigMap `iofog-nats-config` (key `server.conf`) | `/etc/nats/config` |
| ConfigMap `iofog-nats-jwt-bundle` | `/tmp/nats/jwt` |
| Secret `nats-site-server` | `/etc/nats/certs/nats-site-server` |
| Secret `nats-mqtt-server` | `/etc/nats/certs/nats-mqtt-server` |
| Secret `nats-creds-sys-admin-hub` (key projected as `admin-hub.creds`) | `/etc/nats/creds` |
| Secret `nats-jetstream-key-<controlplane>` | `/etc/nats/jetstream` (read-only) |
| PVC `js-data` | `/home/runner/data` |

The operator rewrites `server.conf` on each NATS reconcile so cluster routes stay current. It leaves `$SELFNAME` in the file. `nats-server` expands that from `SELFNAME`, so `nats-0` and `nats-1` get different server names from the same ConfigMap. The resolver block points at `/home/runner/nats/jwt` with `type: full`. The JWT ConfigMap is the read-only source. The wrapper copies those files into the writable resolver directory before the child starts.

Readiness and liveness are HTTP GET `http://<pod>:8222/healthz?js-enabled-only=true`. That port is the `nats-server` monitor port the wrapper passes as `-m`. The wrapper itself does not serve HTTP.

On a Kubernetes control plane, agent NATS instances are not the hub. The hub is this StatefulSet. The Controller still creates a NATS microservice on each agent.

### Edge node (ioFog Agent and PoT Edgelet)

The Controller creates a system microservice named `nats` from catalog item `NATS` (`category: SYSTEM`, publisher `Eclipse ioFog`). The agent runtime (ioFog Agent, or PoT Edgelet on `edgelet`, Docker, or Podman) starts the same image and bind-mounts Controller volume mounts into the container.

Default mode when the fog has no `natsConfig.mode` is **leaf**. `mode: none` means the Controller does not deploy NATS on that node. `mode: server` deploys a clustered server. On a non-Kubernetes control plane, the first system fog in server mode becomes the hub. On Kubernetes, `isHub` is never set on an agent. The hub is the operator StatefulSet.

The Controller writes these env vars onto the microservice (values shown are the contract, not a sample fog):

| Env | Value |
| --- | --- |
| `NATS_SERVER_MODE` | `server` or `leaf` |
| `NATS_CONF` | `/etc/nats/config/server.conf` |
| `NATS_JWT_DIR` | `/home/runner/nats/jwt` |
| `NATS_JWT_MOUNT_DIR` | `/tmp/nats/jwt` |
| `NATS_TLS_DIR` | `/etc/nats/certs` |
| `NATS_CREDS_DIR` | `/etc/nats/creds` |
| `NATS_SYS_USER_CRED_PATH` | `/etc/nats/creds/<account>/<user>.creds` |
| `NATS_JETSTREAM_STORE_DIR` | `/home/runner/data` |
| `NATS_MONITOR_PORT` | The instance HTTP port (default `8222`) |
| `NATS_SERVER_PORT`, `NATS_LEAF_PORT`, `NATS_CLUSTER_PORT`, `NATS_MQTT_PORT`, `NATS_HTTP_PORT` | Listener ports. The wrapper does not read these. They exist so a config file can still reference them, and so the rendered template has a source. |
| `NATS_CERT_NAME`, `NATS_MQTT_CERT_NAME` | Certificate directory names under `NATS_TLS_DIR`. Used while the Controller renders `server.conf`. The wrapper does not read them. |
| `JETSTREAM_KEY`, `JETSTREAM_PREV_KEY` | JetStream encryption key material. Consumed by `nats-server` from the config, not by the wrapper. |

Volume mappings:

| Source | Container path | Access | Type |
| --- | --- | --- | --- |
| ConfigMap `nats-server-conf-<fog>` (key `server.conf`) | `/etc/nats/config` | read-only | volume mount |
| JWT bundle ConfigMap | `/tmp/nats/jwt` | read-write mapping | volume mount |
| Server cert secret | `/etc/nats/certs/<cert-name>` | read-only | volume mount |
| MQTT cert secret | `/etc/nats/certs/<mqtt-cert-name>` | read-only | volume mount |
| System-user creds secret (server mode) | `/etc/nats/creds` | read-only | volume mount |
| `<fog-name>-nats-jetstream` | `/home/runner/data` | read-write | volume (local disk, not a ConfigMap) |

Leaf nodes do not mount the server system-user secret at `/etc/nats/creds`. Leaf credentials used to dial upstream servers are mounted from the leaf creds ConfigMap at the same `/etc/nats/creds` path. `NATS_SYS_USER_CRED_PATH` still points at a file under that directory. If that file is absent, purge and claims push are skipped.

Which config text is mounted:

| Instance | Template | Resolver |
| --- | --- | --- |
| Server, and the cluster has more than one route | `server.conf` (client, cluster, leaf listener, MQTT, full resolver) | Hub JWT bundle `iofog-nats-jwt-bundle` |
| Server, and the cluster route list has exactly one entry | `server-no-cluster.conf` (no `cluster` block) | Same hub bundle |
| Leaf | `leaf.conf` (leaf `remotes`, no cluster block, full resolver) | Per-fog bundle `nats-jwt-bundle-<fog>` |

The Controller renders those templates before they are stored in the ConfigMap. Placeholders such as `$NATS_SERVER_PORT` and `$OPERATOR_JWT` are replaced in the Controller. Anything still written as `$NAME` is expanded later by `nats-server` from the container environment.

JetStream domain in the rendered file is the Controller namespace for a server, and the fog name for a leaf. Store directory is `/home/runner/data` in both cases.

The agent health check is:

```text
curl -f http://localhost:8222/healthz | grep -q 'status.*ok'
```

Interval 30s, timeout 10s, start period 60s, 3 retries. That command needs a shell. The Alpine edge image has `/bin/sh`. The UBI micro image used for amd64 and arm64 does not, which is why the Kubernetes probes are HTTP GET and do not call `curl`.

A new volume mapping sets `rebuild` on the microservice, so the agent recreates the container. A config or JWT change that only updates an existing mount does not require a new container. The wrapper is what notices the file change.

## Startup sequence

1. Read every `NATS_*` setting once from the environment (see [Environment](#environment)).
2. Wait until `NATS_CONF` exists. Poll every 1 second, up to 30 times. This covers a volume that is not visible at the first instant the process starts. If the file is still missing, the process exits.
3. Hash `NATS_CONF` with SHA-256. That hash is the baseline for later config comparisons.
4. If `NATS_JWT_MOUNT_DIR` exists and is a directory, copy `*.jwt` into `NATS_JWT_DIR` (see [JWT sync](#jwt-sync)). A sync error is logged. Startup continues.
5. Start `nats-server`.
6. After 3 seconds, run JetStream account reconciliation once (see [JetStream purge](#jetstream-account-purge)).
7. Start the watchers that apply to paths that exist now.

Watchers are not started later if a path appears after this point. `NATS_ACCOUNTS`, `NATS_TLS_DIR`, `NATS_JWT_MOUNT_DIR`, and `NATS_CREDS_DIR` are watched only when they exist at startup. The config file is the exception: the process waits for it, then always watches it.

## What is watched

Watching uses `fsnotify`. Each watcher debounces for 500ms. The main loop then debounces the combined causes for another 500ms. A burst of writes is handled about one second after the last event.

| Cause name | Path | Watcher | Events that count |
| --- | --- | --- | --- |
| `config` | `NATS_CONF` (default `/etc/nats/config/server.conf`) | Parent directory. Events whose path is not exactly this file are ignored. | Create, Write |
| `accounts` | `NATS_ACCOUNTS` (default `/etc/nats/config/accounts.conf`) | Same file watcher, only if the file exists at startup. | Create, Write |
| `tls` | `NATS_TLS_DIR` (default `/etc/nats/certs`) | Directory plus immediate subdirectories. | Create, Write, Remove |
| `jwt` | `NATS_JWT_MOUNT_DIR` (default `/tmp/nats/jwt`) | Same directory watcher. | Create, Write, Remove |
| `creds` | `NATS_CREDS_DIR` (default `/etc/nats/creds/`) | Same directory watcher. | Create, Write, Remove |

Directory watches do not follow trees deeper than one subdirectory. A new immediate subdirectory created while the process is running is added to the watch. Rename and chmod events do not schedule work.

The Controller and the operator do not mount `accounts.conf`. Account state is the JWT resolver directory. The accounts watcher exists for a deployment that places that file beside the server config. If the file is absent at startup, account changes arrive only through the JWT mount.

### Config content hash

When the debounced callback runs and `config` is one of the causes, the wrapper hashes `NATS_CONF` again.

- If the hash matches the last applied hash, `config` is cleared for this round.
- If the hash differs, the new hash is stored and `config` stays set.
- If the file cannot be read, `config` stays set.

That check changes leaf behavior. In server mode the wrapper still sends `SIGHUP` after the timer, including when the only cause was an unchanged config file. The hash exists so a ConfigMap rewrite with identical bytes does not restart a leaf.

## How a change is applied

All watchers call one coalescer. Causes that arrive inside the 500ms window are merged. One callback runs for the whole set.

Order inside that callback:

1. Apply the config hash rule above.
2. If `jwt` is set, sync the mount directory into `NATS_JWT_DIR` before any signal.
3. Choose `SIGHUP` or restart from the table below.
4. If `jwt` is set, wait 3 seconds, then reconcile JetStream accounts and push claims. This runs in the background and does not block the next watch event.

`NATS_SERVER_MODE` is read again at decision time. The value is trimmed and lowercased. Anything other than `leaf` is treated as server mode. The default is `server`.

| Mode | Causes still set | What the wrapper does |
| --- | --- | --- |
| server | any (including a round whose only cause was an unchanged config file) | `SIGHUP` to `nats-server`. The server reloads config, TLS material, and the resolver directory. |
| leaf | `tls` is set (other causes may be set too) | `SIGHUP` only. The restart branch is not taken. Leaf reload in `nats-server` is limited to TLS certificate changes. |
| leaf | `config`, `accounts`, or `creds`, and `tls` is not set | `SIGINT` so the child exits, then the wrapper starts `nats-server` again with the same config path. |
| leaf | `jwt` only | No signal. JWTs are already copied. After 3 seconds the wrapper reconciles and pushes claims. The running server keeps its process. The resolver directory on disk has the new files, and claims push tells the server about them. |

If a leaf sees TLS and config changes in the same window, only `SIGHUP` is sent. The config change is not applied by a restart until a later window that does not include TLS.

`SIGHUP` and `SIGINT` are sent to the child process, not to the wrapper. A failed signal is logged. The wrapper keeps running.

On the restart path the wrapper sets a flag, sends `SIGINT`, and waits on the child exit channel. When that exit arrives with the flag set, it starts `nats-server` again and does not exit. A child exit without that flag ends the wrapper.

## JWT sync

`NATS_JWT_MOUNT_DIR` is the platform mount. On Kubernetes that is a ConfigMap volume and is effectively read-only. On an agent it is a Controller volume mount of a ConfigMap. `nats-server` is configured with `resolver.dir` = `NATS_JWT_DIR` (`/home/runner/nats/jwt`), which is a normal directory created in the image and owned by `runner`.

`SyncMountToJWT` makes the writable directory match the mount:

1. If the two paths are the same, do nothing.
2. If the mount directory is missing, do nothing.
3. List top-level `*.jwt` names. Skip directories. Skip names that end in `.delete` (the `*.jwt.delete` marker used when a resolver has `allow_delete`).
4. If that list is empty, do nothing. Files already in `NATS_JWT_DIR` stay. This avoids deleting every account during a ConfigMap rotation that briefly shows an empty directory.
5. Otherwise create `NATS_JWT_DIR` (`0755`) if needed, copy each `*.jwt` over the destination (overwrite, `fsync`), then delete `*.jwt` files in the destination that are not in the mount. `*.jwt.delete` files in the destination are left in place.

Copy is file content only. The directory is never renamed, so the sync works when `NATS_JWT_DIR` is itself a volume.

Startup sync runs before the child starts, so the resolver directory is populated on the first listen. A later sync runs only when the JWT watcher fired, and it runs before `SIGHUP` or restart.

## JetStream account purge

Removing an account JWT stops new traffic for that account. JetStream data can remain under the store directory. The wrapper deletes that data through the JetStream API. It does not keep a snapshot of the previous account list. After a reboot it compares disk to the current resolver.

Store directory, in order:

1. `NATS_JETSTREAM_STORE_DIR` if set. A relative path is joined to the server config's directory.
2. Otherwise the `store_dir` value inside the first `jetstream { ... }` block of the server config. A relative value is joined the same way. Comments are skipped. The parser is line-oriented. It expects `store_dir:` inside that block.

Accounts that have data are the immediate subdirectory names of `<store_dir>/jetstream`. If that directory is missing, the store list is empty and reconciliation still logs, with nothing to purge.

Resolver accounts are the `*.jwt` file names in `NATS_JWT_DIR`, without the suffix. `*.jwt.delete` is ignored.

An account is purged when it has a JetStream subdirectory and is not in the resolver list. For each such account the wrapper connects to `NATS_CLIENT_URL` (default `nats://127.0.0.1:4222`) with `NATS_SYS_USER_CRED_PATH` and requests:

```text
$JS.API.ACCOUNT.PURGE.<account>
```

Body is `{}`. Timeout is 10 seconds. A response with no `error` object is success. The wrapper opens a new connection per account.

If `NATS_SYS_USER_CRED_PATH` is unset, reconciliation still lists accounts and logs `to_purge`, then skips the API. A relative creds path is joined to `NATS_CREDS_DIR`. An absolute path is used as given.

This runs 3 seconds after process start, and again 3 seconds after any round whose causes included `jwt`.

## Claims push

After that same post-JWT delay, and only when system creds are configured, the wrapper sends each account JWT to the local server:

```text
$SYS.REQ.CLAIMS.UPDATE
```

The payload is the raw JWT bytes from `<NATS_JWT_DIR>/<account>.jwt`. One request per account, 10 second timeout each, same client URL and creds file as purge. Failures are logged per account and do not stop the process. An empty creds path returns immediately.

Server and leaf both do this. Leaf configs use `resolver.type: full`, same as server configs.

## Environment

The wrapper reads only the variables below. Unset means the default. There is no `IOFOG_*`, `POT_*`, or `DATASANCE_*` runtime switch.

| Variable | Default | Effect |
| --- | --- | --- |
| `NATS_CONF` | `/etc/nats/config/server.conf` | Passed to `nats-server` as `-c`. Watched. Hashed. |
| `NATS_ACCOUNTS` | `/etc/nats/config/accounts.conf` | Watched when the file exists at startup. |
| `NATS_TLS_DIR` | `/etc/nats/certs` | Watched when the directory exists at startup. `NATS_TLS_DIR` wins over `NATS_SSL_DIR`. |
| `NATS_SSL_DIR` | none | Deprecated. Used only when `NATS_TLS_DIR` is unset. One warning is logged. |
| `NATS_JWT_DIR` | `/home/runner/nats/jwt` | Writable resolver directory. Sync destination. Not watched. The server config must point `resolver.dir` here. |
| `NATS_JWT_MOUNT_DIR` | `/tmp/nats/jwt` | Read-only source of `*.jwt` files. Watched when it exists at startup. |
| `NATS_SERVER_MODE` | `server` | `leaf` selects the leaf signal rules. Any other value, after trim and lowercasing, selects server rules. |
| `NATS_CREDS_DIR` | `/etc/nats/creds/` | Watched when it exists at startup. Also the base for a relative `NATS_SYS_USER_CRED_PATH`. |
| `NATS_SERVER_BIN` | `/home/runner/bin/nats-server` | Child binary. Override for local runs. |
| `NATS_MONITOR_PORT` | `8222` | Passed as `-m` when in `0..65535`. `0` disables the monitor flag. Invalid values fall back to `8222`. |
| `NATS_SYS_USER_CRED_PATH` | empty | System-account user credentials. Required for purge and claims push. Empty skips both API calls. |
| `NATS_CLIENT_URL` | `nats://127.0.0.1:4222` | Where the wrapper connects for purge and claims push. |
| `NATS_JETSTREAM_STORE_DIR` | parsed from config | Overrides `jetstream.store_dir` for reconciliation only. It does not change the server's store path. |

Other variables in the container (`SELFNAME`, `SERVER_NAME`, `JETSTREAM_KEY`, port variables, and anything an operator injects) are forwarded to `nats-server` and are not interpreted by the wrapper.

## Filesystem the image creates

These directories exist before any volume is mounted. A mount hides the image directory at that path.

| Path | Purpose |
| --- | --- |
| `/home/runner/bin` | `iofog-nats` and `nats-server` |
| `/home/runner/nats/jwt` | Default writable JWT directory |
| `/home/runner/data` | Default JetStream store when the platform mounts it here |
| `/home/runner/run` | `pid_file` in the Controller templates (`/home/runner/run/nats.pid`) |
| `/etc/nats/config` | Created by the mount, not by the image |
| `/etc/nats/certs` | Created by the mount |
| `/etc/nats/creds` | Created by the mount |
| `/tmp/nats/jwt` | Created by the mount |

Published images include `curl` and `grep` for health checks and do not include `nats-cli`. `Dockerfile.dev` adds `nats-cli` and is not published.

| Platform | Dockerfile | Base |
| --- | --- | --- |
| linux/amd64, linux/arm64 | `Dockerfile` | UBI 9 micro |
| linux/arm/v7, linux/riscv64 | `Dockerfile.edge` | Alpine 3.22, copied into a scratch image |

## End-to-end examples

### Hub certificate rotation (Kubernetes or an edge server)

1. The operator or Controller updates the TLS secret.
2. kubelet or the agent updates files under `/etc/nats/certs/<cert-name>/`.
3. The directory watcher reports `tls`.
4. After the debounce, server mode sends `SIGHUP`. Leaf mode also sends `SIGHUP` for TLS.
5. `nats-server` reloads certificates. The process is not replaced.

### Hub or edge server config edit

1. The Controller or operator writes a new `server.conf` into the ConfigMap.
2. The file watcher sees Create or Write on that exact path.
3. The content hash differs, so `config` stays set.
4. Server mode sends `SIGHUP`.
5. Leaf mode sends `SIGINT` and the wrapper starts the child again, unless this same window also included a TLS change.

### Account added or removed

1. The Controller updates the JWT bundle ConfigMap (`iofog-nats-jwt-bundle` on the hub and on edge servers, `nats-jwt-bundle-<fog>` on a leaf).
2. The JWT directory watcher reports `jwt`.
3. The wrapper copies `*.jwt` into `/home/runner/nats/jwt` and deletes orphans that are no longer in the mount.
4. Server mode sends `SIGHUP`. A leaf with no other cause does not.
5. Three seconds later the wrapper purges JetStream data for accounts that remain on disk but are gone from the resolver, then pushes every remaining account JWT on `$SYS.REQ.CLAIMS.UPDATE`.

An empty bundle during a rotation does not delete resolver files.

### Leaf upstream change

1. The Controller rewrites `leaf.conf` with a new `remotes` list and updates the ConfigMap.
2. The config watcher fires.
3. Leaf mode stops the child with `SIGINT` and starts it again so the new remotes are loaded.
4. TLS-only edits on that same leaf still use `SIGHUP` and keep the process.

## Logs that show each step

| Log text | Meaning |
| --- | --- |
| `Waiting for NATS config at ...` | Volume not visible yet. Fatal after 30 attempts. |
| `JWT sync at startup: copied=N removed=M` | Resolver directory filled before the child started. |
| `NATS server started with config ...` | Child is running. |
| `Sent SIGHUP to nats-server for config reload` | Server mode, or leaf TLS. |
| `Sent SIGINT to nats-server for graceful stop (restart)` | Leaf non-TLS restart. |
| `NATS server stopped for restart, starting again` | Child exited and the wrapper is starting it again. |
| `JWT sync after change: copied=N removed=M` | Mount was copied after a JWT watch event. |
| `JetStream account reconciliation: store_dir=... to_purge=N` | Purge pass finished listing. |
| `NATS_SYS_USER_CRED_PATH unset, skipping purge API calls` | Reconcile ran. No API calls. |
| `JetStream account purge initiated for <account>` | Purge API returned success. |
| `Claims update: pushed N account JWTs` | Claims push finished. |
| `NATS_SSL_DIR is deprecated; use NATS_TLS_DIR instead` | Legacy TLS directory env var. |

## What this process does not do

- It does not render `server.conf` or sign JWTs. The Controller and the operator do that.
- It does not watch the Kubernetes API or the Controller API. It only watches files inside the container.
- It does not create the agent microservice, the StatefulSet, or the volume mounts.
- It does not change `nats-server` listener ports except by passing the config file through. Port env vars are for the config text and for `-m`.
- It does not reload itself when `NATS_*` variables change. Those are read at startup, except `NATS_SERVER_MODE`, creds path, client URL, and the JetStream store directory, which are read again when a change is handled.

## Source map

| Behavior | File |
| --- | --- |
| Startup, coalescer, signal choice, reconcile trigger | `cmd/iofog-nats/main.go` |
| Environment defaults | `internal/config/env.go` |
| File watcher | `internal/watch/config.go` |
| Directory watcher | `internal/watch/dir.go` |
| Child process, `SIGHUP`, `SIGINT` | `internal/nats/nats.go`, `internal/exec/exec.go` |
| JWT copy and orphan delete | `internal/jwtcopy/jwtcopy.go` |
| JetStream account purge | `internal/jspurge/jspurge.go` |
| `$SYS.REQ.CLAIMS.UPDATE` | `internal/claimspush/claimspush.go` |
