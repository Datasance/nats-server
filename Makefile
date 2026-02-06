# pot-nats – NATS server wrapper for Datasance PoT

BINARY      := pot-nats
CMD_PATH    := ./cmd/pot-nats
BINARY_PATH := bin/$(BINARY)
LDFLAGS     := -trimpath -ldflags="-s -w"
IMAGE       ?= pot-nats:latest

.PHONY: all build test lint fmt fmt-check clean install docker-build

all: build test

build:
	@mkdir -p bin
	go build $(LDFLAGS) -o $(BINARY_PATH) $(CMD_PATH)

test:
	go test ./...

lint: fmt-check
	go vet ./...

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' or 'gofmt -w .'"; exit 1)

clean:
	rm -rf bin/

install: build
	go install $(LDFLAGS) $(CMD_PATH)

docker-build:
	docker build -t $(IMAGE) .
