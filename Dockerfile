# Build iofog-nats wrapper and install nats-server
# golang:1.26.4-alpine — sha256:9169234cc43b396435c64e45538fe6d4ffa237e7f988b9ab32abdfa0c3141979
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine@sha256:9169234cc43b396435c64e45538fe6d4ffa237e7f988b9ab32abdfa0c3141979 AS go-builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILDPLATFORM

WORKDIR /build
COPY . .

ENV CGO_ENABLED=0

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o iofog-nats ./cmd/iofog-nats

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install github.com/nats-io/nats-server/v2@v2.14.2

RUN mkdir -p /out && \
    find /go/bin -name "nats-server" -type f -exec cp {} /out/nats-server \;

# Create non-root user and writable dirs for pid file and JetStream store
# ubi9/ubi-minimal — sha256:1ae81b51dcbb0a1cd6a59b3d42b52f98201778c7cd6e9765194574258d3f29de
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:1ae81b51dcbb0a1cd6a59b3d42b52f98201778c7cd6e9765194574258d3f29de AS user-stage
RUN microdnf install -y ca-certificates shadow-utils && microdnf install -y tzdata && microdnf reinstall -y tzdata && microdnf clean all -y
RUN useradd --uid 10000 --create-home runner
RUN mkdir -p /home/runner/run /home/runner/data /home/runner/bin /home/runner/nats/jwt && chown -R runner:runner /home/runner

# Stage runtime files so final image can use a single COPY layer
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:1ae81b51dcbb0a1cd6a59b3d42b52f98201778c7cd6e9765194574258d3f29de AS runtime-staging
COPY --from=user-stage /etc/passwd /staging/etc/passwd
COPY --from=user-stage /etc/group /staging/etc/group
COPY --from=user-stage /home/runner /staging/home/runner
COPY --from=user-stage /etc/ssl/certs/ca-bundle.crt /staging/etc/ssl/certs/ca-bundle.crt
COPY --from=user-stage /etc/pki/tls/certs/ca-bundle.crt /staging/etc/pki/tls/certs/ca-bundle.crt
COPY --from=user-stage /usr/share/zoneinfo /staging/usr/share/zoneinfo

# Final image: UBI 9 micro
# ubi9/ubi-micro — sha256:59daac603227814ee7fe5cde69ac5ec2815b361c7cb1b2bc9ed3f55673499d38
FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:59daac603227814ee7fe5cde69ac5ec2815b361c7cb1b2bc9ed3f55673499d38

ARG OCI_SOURCE_REPO
ARG OCI_VERSION
ARG OCI_REVISION
ARG NATS_DISTRIBUTION

LABEL org.opencontainers.image.source="${OCI_SOURCE_REPO}" \
      org.opencontainers.image.version="${OCI_VERSION}" \
      org.opencontainers.image.revision="${OCI_REVISION}" \
      distribution="${NATS_DISTRIBUTION}"

COPY --from=runtime-staging /staging/ /

COPY --from=go-builder /build/iofog-nats /home/runner/bin/iofog-nats
COPY --from=go-builder /out/nats-server /home/runner/bin/nats-server

COPY LICENSE /licenses/LICENSE

USER 10000
WORKDIR /home/runner

CMD ["/home/runner/bin/iofog-nats"]
