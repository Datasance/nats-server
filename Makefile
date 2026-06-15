# iofog-nats – NATS server wrapper for Eclipse ioFog

BINARY      := iofog-nats
CMD_PATH    := ./cmd/iofog-nats
BINARY_PATH := bin/$(BINARY)
LDFLAGS     := -trimpath -ldflags="-s -w"
IMAGE       ?= iofog-nats:latest

GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

export PATH := $(GOBIN):$(PATH)

# golangci-lint — pinned version; override with GOLANGCI_LINT_VERSION=vX.Y.Z
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT         := $(GOBIN)/golangci-lint

GOVULNCHECK_VERSION ?= v1.1.4
GOSEC_SCOPE           := ./cmd/... ./internal/...

.PHONY: all build test lint lint-fix install-lint fmt fmt-check vet vulncheck security-code clean install docker-build

all: build test

build:
	@mkdir -p bin
	go build $(LDFLAGS) -o $(BINARY_PATH) $(CMD_PATH)

test:
	go test ./...

$(GOLANGCI_LINT):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) -> $(GOBIN)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
	@echo "golangci-lint $(GOLANGCI_LINT_VERSION) installed"

install-lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) version

lint: $(GOLANGCI_LINT) fmt-check
	@echo "Running golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@$(GOLANGCI_LINT) run --config .golangci.yaml

lint-fix: $(GOLANGCI_LINT)
	@echo "Running golangci-lint $(GOLANGCI_LINT_VERSION) with --fix..."
	@$(GOLANGCI_LINT) run --config .golangci.yaml --fix

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' or 'gofmt -w .'"; exit 1)

vet:
	go vet ./...

vulncheck:
	@echo "Running govulncheck..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
	fi
	@chmod +x scripts/vulncheck.sh
	@scripts/vulncheck.sh
	@echo "Verifying module integrity..."
	@go mod verify

security-code:
	@echo "Running Go static security analysis..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -exclude-dir=build $(GOSEC_SCOPE)

clean:
	rm -rf bin/

install: build
	go install $(LDFLAGS) $(CMD_PATH)

docker-build:
	docker build -t $(IMAGE) .
