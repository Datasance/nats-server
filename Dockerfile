FROM golang:1.20.14-alpine AS go-builder

ARG TARGETOS
ARG TARGETARCH

RUN mkdir -p /go/src/github.com/datasance/nats-server
WORKDIR /go/src/github.com/datasance/nats-server
COPY . /go/src/github.com/datasance/nats-server
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/iofog-nats



FROM nats:alpine
COPY LICENSE /licenses/LICENSE
COPY --from=go-builder /go/src/github.com/datasance/nats-server/bin/iofog-nats /bin/iofog-nats

CMD ["/bin/iofog-nats"]
