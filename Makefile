BINARY := bin/agent-router
CLI_BINARY := bin/agent-router-cli
CONTROLPLANE_BINARY := bin/agent-router-controlplane
GO ?= $(shell command -v go 2>/dev/null || echo $(HOME)/.local/go1.26.0/bin/go)
MANIFEST_PATH ?=
TARGET_DIR ?= ./scaffold-output
PIPELINE_NAME ?= sample
EXPECTED_TRACE ?=
ACTUAL_TRACE ?=
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/your-org/agent-router/internal/version.Version=$(VERSION) -X github.com/your-org/agent-router/internal/version.Commit=$(COMMIT) -X github.com/your-org/agent-router/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build build-cli build-controlplane docker-router docker-controlplane test test-unit test-integ test-replay lint lint-docker run serve cli-run validate replay audit-export scaffold debug bench trace-view trace-down run-controlplane k8s-apply k8s-delete k8s-validate clean

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
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		$(MAKE) lint-docker; \
	fi

lint-docker:
	@if command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/app" -w /app golangci/golangci-lint:v2.4.0 golangci-lint run ./...; \
	else \
		echo "golangci-lint not found and docker is unavailable"; \
		exit 1; \
	fi

run:
	$(GO) run ./cmd/router $(MANIFEST_PATH)

serve:
	$(GO) run ./cmd/router serve

cli-run:
	$(GO) run ./cmd/cli run $(MANIFEST_PATH)

validate:
	$(GO) run ./cmd/cli validate $(MANIFEST_PATH)

replay:
	$(GO) run ./cmd/cli replay $(MANIFEST_PATH)

audit-export:
	$(GO) run ./cmd/cli audit-export $(MANIFEST_PATH)

scaffold:
	$(GO) run ./cmd/cli scaffold $(TARGET_DIR) $(PIPELINE_NAME)

debug:
	$(GO) run ./cmd/cli debug $(EXPECTED_TRACE) $(ACTUAL_TRACE)

bench:
	$(GO) test ./tests/bench/... -bench . -benchmem

trace-view:
	docker compose -f deploy/observability/docker-compose.yml up -d

trace-down:
	docker compose -f deploy/observability/docker-compose.yml down

run-controlplane:
	$(GO) run ./cmd/controlplane

k8s-apply:
	kubectl apply -k deploy/k8s

k8s-delete:
	kubectl delete -k deploy/k8s

k8s-validate:
	kubectl kustomize deploy/k8s >/dev/null

clean:
	rm -rf bin
