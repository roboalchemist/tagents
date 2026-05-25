BINARY=tagents
VERSION=$(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GOFLAGS=

.PHONY: build clean test test-unit test-integration install dev-install deps fmt lint man docs-gen release-snapshot release check help

build:
	go build $(LDFLAGS) -o $(BINARY) .

clean:
	rm -f $(BINARY)
	rm -rf dist/

test: build
	@echo "=== Smoke tests ==="
	@./$(BINARY) --help > /dev/null && echo "PASS: --help"
	@./$(BINARY) --version > /dev/null && echo "PASS: --version"
	@./$(BINARY) docs > /dev/null && echo "PASS: docs"
	@./$(BINARY) completion bash > /dev/null && echo "PASS: completion bash"
	@./$(BINARY) skill print > /dev/null && echo "PASS: skill print"
	@echo "All smoke tests passed."

test-unit:
	go test -race -coverprofile=coverage.out ./pkg/...
	go tool cover -func=coverage.out | grep total

test-integration:
	go test -v -short -timeout 120s -run TestIntegration ./...

test-integration-full:
	go test -v -timeout 600s -run TestIntegration ./...

install:
	go build $(LDFLAGS) -o $(BINARY) . && sudo install -m 755 $(BINARY) /usr/local/bin/

dev-install:
	go build $(LDFLAGS) -o $(BINARY) . && ln -sf $(PWD)/$(BINARY) /usr/local/bin/$(BINARY)

deps:
	go mod download && go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

man: build
	mkdir -p man/man1
	go run ./cmd/gendocs/ man/man1/

docs-gen: build
	mkdir -p docs/cli
	cobra-docs completion markdown --dir docs/cli 2>/dev/null || go run ./cmd/gendocs/ docs/cli/

release-snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

check: fmt lint test test-unit

help:
	@grep -E '^[a-zA-Z_-]+:' Makefile | awk -F: '{print $$1}' | sort | column
