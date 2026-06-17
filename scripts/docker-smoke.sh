#!/usr/bin/env bash
# Build a single-platform image, start iofog-nats with a minimal config, and verify
# monitoring /healthz from the host and from in-container curl (fleet healthcheck pattern).
set -euo pipefail

PLATFORM="${1:?platform required, e.g. linux/amd64}"
DOCKERFILE="${2:?dockerfile required, e.g. Dockerfile}"
IMAGE_TAG="${3:?image tag required}"
CONFIG="${4:-test/fixtures/ci/server.conf}"
MONITOR_PORT="${5:-18222}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

if [ ! -f "${CONFIG}" ]; then
  echo "config not found: ${CONFIG}" >&2
  exit 1
fi

echo "Building ${IMAGE_TAG} (${PLATFORM}) from ${DOCKERFILE}..."
docker buildx build \
  --platform "${PLATFORM}" \
  -f "${DOCKERFILE}" \
  --load \
  -t "${IMAGE_TAG}" \
  .

cid=""
cleanup() {
  if [ -n "${cid}" ]; then
    docker rm -f "${cid}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

cid="$(docker run -d \
  -v "${ROOT}/${CONFIG}:/etc/nats/config/server.conf:ro" \
  -p "${MONITOR_PORT}:8222" \
  "${IMAGE_TAG}")"

echo "Started container ${cid}; waiting for /healthz on port ${MONITOR_PORT}..."
for i in $(seq 1 60); do
  if curl -sf "http://127.0.0.1:${MONITOR_PORT}/healthz" >/dev/null; then
    break
  fi
  if [ "${i}" -eq 60 ]; then
    echo "host health check timed out" >&2
    docker logs "${cid}" >&2 || true
    exit 1
  fi
  sleep 1
done

echo "Host health check OK"
docker exec "${cid}" /usr/bin/curl -sf http://127.0.0.1:8222/healthz >/dev/null
echo "In-container curl health check OK"

grep_bin=""
for candidate in /usr/bin/grep /bin/grep; do
  if docker exec "${cid}" test -x "${candidate}" 2>/dev/null; then
    grep_bin="${candidate}"
    break
  fi
done
if [ -z "${grep_bin}" ]; then
  echo "grep not found in container" >&2
  exit 1
fi
docker exec "${cid}" "${grep_bin}" --version >/dev/null 2>&1 || docker exec "${cid}" "${grep_bin}" -V >/dev/null 2>&1
echo "In-container grep OK (${grep_bin})"

echo "Smoke test passed: ${PLATFORM} ${DOCKERFILE}"
