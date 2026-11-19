# Build iofog-nats wrapper and install nats-server
# golang:1.26.11-alpine
FROM  --platform=$BUILDPLATFORM golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab AS go-builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILDPLATFORM

WORKDIR /build
COPY . .

ENV CGO_ENABLED=0

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o iofog-nats ./cmd/iofog-nats

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install -trimpath -ldflags="-s -w" github.com/nats-io/nats-server/v2@v2.15.3

RUN mkdir -p /out && \
    find /go/bin -name "nats-server" -type f -exec cp {} /out/nats-server \;

# Create non-root user and writable dirs for pid file and JetStream store
# ubi9/ubi-minimal
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:d8162789d7913ef8941481687b92ab4bbec311870948f66cf52b9cc7e0c57c06 AS user-stage
RUN microdnf install -y ca-certificates shadow-utils && microdnf install -y tzdata && microdnf reinstall -y tzdata && microdnf clean all -y
RUN useradd --uid 10000 --create-home runner
RUN mkdir -p /home/runner/run /home/runner/data /home/runner/bin /home/runner/nats/jwt && chown -R runner:runner /home/runner

# Stage runtime files so final image can use a single COPY layer
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:d8162789d7913ef8941481687b92ab4bbec311870948f66cf52b9cc7e0c57c06 AS runtime-staging
COPY --from=user-stage /etc/passwd /staging/etc/passwd
COPY --from=user-stage /etc/group /staging/etc/group
COPY --from=user-stage /home/runner /staging/home/runner
COPY --from=user-stage /etc/ssl/certs/ca-bundle.crt /staging/etc/ssl/certs/ca-bundle.crt
COPY --from=user-stage /etc/pki/tls/certs/ca-bundle.crt /staging/etc/pki/tls/certs/ca-bundle.crt
COPY --from=user-stage /usr/share/zoneinfo /staging/usr/share/zoneinfo
# curl-minimal and grep from ubi-minimal base (do not install full curl — conflicts with curl-minimal).
COPY --from=user-stage /usr/bin/curl /staging/usr/bin/curl
COPY --from=user-stage /usr/bin/grep /staging/usr/bin/grep
COPY --from=user-stage /usr/lib64/libcurl.so.4 /staging/usr/lib64/libcurl.so.4
COPY --from=user-stage /usr/lib64/libc.so.6 /staging/usr/lib64/libc.so.6
COPY --from=user-stage /usr/lib64/libnghttp2.so.14 /staging/usr/lib64/libnghttp2.so.14
COPY --from=user-stage /usr/lib64/libssl.so.3 /staging/usr/lib64/libssl.so.3
COPY --from=user-stage /usr/lib64/libcrypto.so.3 /staging/usr/lib64/libcrypto.so.3
COPY --from=user-stage /usr/lib64/libgssapi_krb5.so.2 /staging/usr/lib64/libgssapi_krb5.so.2
COPY --from=user-stage /usr/lib64/libkrb5.so.3 /staging/usr/lib64/libkrb5.so.3
COPY --from=user-stage /usr/lib64/libk5crypto.so.3 /staging/usr/lib64/libk5crypto.so.3
COPY --from=user-stage /usr/lib64/libcom_err.so.2 /staging/usr/lib64/libcom_err.so.2
COPY --from=user-stage /usr/lib64/libz.so.1 /staging/usr/lib64/libz.so.1
COPY --from=user-stage /usr/lib64/libkrb5support.so.0 /staging/usr/lib64/libkrb5support.so.0
COPY --from=user-stage /usr/lib64/libkeyutils.so.1 /staging/usr/lib64/libkeyutils.so.1
COPY --from=user-stage /usr/lib64/libresolv.so.2 /staging/usr/lib64/libresolv.so.2
COPY --from=user-stage /usr/lib64/libselinux.so.1 /staging/usr/lib64/libselinux.so.1
COPY --from=user-stage /usr/lib64/libpcre2-8.so.0 /staging/usr/lib64/libpcre2-8.so.0
COPY --from=user-stage /usr/lib64/libpcre.so.1 /staging/usr/lib64/libpcre.so.1
COPY --from=user-stage /usr/lib64/libsigsegv.so.2 /staging/usr/lib64/libsigsegv.so.2

# Final image: UBI 9 micro
# ubi9/ubi-micro
FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:dbd325b96bcd432dd42816d31412bdf1dc0fb973b79ed310344bc23c76c068db

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
