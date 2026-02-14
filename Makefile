BINARY := bin/agent-router
CLI_BINARY := bin/agent-router-cli
CONTROLPLANE_BINARY := bin/agent-router-controlplane
GO ?= $(shell command -v go 2>/dev/null || echo $(HOME)/.local/go1.26.0/bin/go)
MANIFEST_PATH ?=
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/your-org/agent-router/internal/version.Version=$(VERSION) -X github.com/your-org/agent-router/internal/version.Commit=$(COMMIT) -X github.com/your-org/agent-router/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build build-cli build-controlplane docker-router docker-controlplane test test-unit test-integ test-replay lint run cli-run validate replay audit-export bench run-controlplane clean

build:
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/router

build-cli:
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS)" -o $(CLI_BINARY) ./cmd/cli

build-controlplane:
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS)" -o $(CONTROLPLANE_BINARY) ./cmd/controlplane

docker-router:
	docker build -f deploy/Dockerfile.router --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t agent-router:$(VERSION) .

docker-controlplane:
	docker build -f deploy/Dockerfile.controlplane --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t agent-router-controlplane:$(VERSION) .

test: test-unit test-integ test-replay

test-unit:
	$(GO) test ./tests/unit/...

test-integ:
	$(GO) test ./tests/integration/...

test-replay:
	$(GO) test ./tests/replay/...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else $(GO) test ./...; fi

run:
	$(GO) run ./cmd/router $(MANIFEST_PATH)

cli-run:
	$(GO) run ./cmd/cli run $(MANIFEST_PATH)

validate:
	$(GO) run ./cmd/cli validate $(MANIFEST_PATH)

replay:
	$(GO) run ./cmd/cli replay $(MANIFEST_PATH)

audit-export:
	$(GO) run ./cmd/cli audit-export $(MANIFEST_PATH)

bench:
	$(GO) test ./tests/bench/... -bench . -benchmem

run-controlplane:
	$(GO) run ./cmd/controlplane

clean:
	rm -rf bin
