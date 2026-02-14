BINARY := bin/agent-router
CLI_BINARY := bin/agent-router-cli
CONTROLPLANE_BINARY := bin/agent-router-controlplane
GO ?= $(shell command -v go 2>/dev/null || echo $(HOME)/.local/go1.26.0/bin/go)
MANIFEST_PATH ?=

.PHONY: build build-cli build-controlplane test test-unit test-integ test-replay lint run cli-run validate replay bench run-controlplane clean

build:
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BINARY) ./cmd/router

build-cli:
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(CLI_BINARY) ./cmd/cli

build-controlplane:
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(CONTROLPLANE_BINARY) ./cmd/controlplane

test: test-unit test-integ test-replay

test-unit:
	$(GO) test ./tests/unit/...

test-integ:
	$(GO) test ./tests/integration/...

test-replay:
	$(GO) test ./tests/replay/...

lint:
	$(GO) test ./...

run:
	$(GO) run ./cmd/router $(MANIFEST_PATH)

cli-run:
	$(GO) run ./cmd/cli run $(MANIFEST_PATH)

validate:
	$(GO) run ./cmd/cli validate $(MANIFEST_PATH)

replay:
	$(GO) run ./cmd/cli replay $(MANIFEST_PATH)

bench:
	$(GO) test ./tests/bench/... -bench . -benchmem

run-controlplane:
	$(GO) run ./cmd/controlplane

clean:
	rm -rf bin
