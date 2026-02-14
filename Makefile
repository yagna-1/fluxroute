BINARY := bin/agent-router
GO ?= $(shell command -v go 2>/dev/null || echo $(HOME)/.local/go1.26.0/bin/go)
MANIFEST_PATH ?=

.PHONY: build test test-unit test-integ test-replay lint run clean

build:
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BINARY) ./cmd/router

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

clean:
	rm -rf bin
