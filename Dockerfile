# Build iofog-nats wrapper and install nats-server
# golang:1.26.4-alpine — sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS go-builder
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
# ubi9/ubi-minimal — sha256:850143255ee0d1915f09aaa09f6ed31f24086ba605c323badfbefa95b8c52b0e
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:850143255ee0d1915f09aaa09f6ed31f24086ba605c323badfbefa95b8c52b0e AS user-stage
RUN microdnf install -y ca-certificates shadow-utils && microdnf install -y tzdata && microdnf reinstall -y tzdata && microdnf clean all -y
RUN useradd --uid 10000 --create-home runner
RUN mkdir -p /home/runner/run /home/runner/data /home/runner/bin /home/runner/nats/jwt && chown -R runner:runner /home/runner

# Stage runtime files so final image can use a single COPY layer
FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:850143255ee0d1915f09aaa09f6ed31f24086ba605c323badfbefa95b8c52b0e AS runtime-staging
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
# ubi9/ubi-micro — sha256:b498b3ea26111ab4b81d65139f2ebd2ef9a2abb7a4588b7fdcc54889f95e9caa
FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:b498b3ea26111ab4b81d65139f2ebd2ef9a2abb7a4588b7fdcc54889f95e9caa

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
