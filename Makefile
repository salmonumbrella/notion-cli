SHELL := /bin/bash

.PHONY: build test lint clean fmt fmt-check tools setup

setup:
	@command -v lefthook >/dev/null || (echo "Install lefthook: brew install lefthook" && exit 1)
	lefthook install

BINARY_NAME := ntn
BUILD_DIR := bin
TOOLS_DIR := $(CURDIR)/.tools
GOFUMPT := $(TOOLS_DIR)/gofumpt
GOIMPORTS := $(TOOLS_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -ldflags="-s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)"

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ntn

# Run tests
test:
	@go test ./...

# Install development tools
tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install mvdan.cc/gofumpt@v0.9.2
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/goimports@v0.40.0
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# Format code
fmt: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .

# Check formatting
fmt-check: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .
	@git diff --exit-code -- '*.go' go.mod go.sum

# Run linter
lint: tools
	@$(GOLANGCI_LINT) run

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -rf $(TOOLS_DIR)

# CI target
ci: fmt-check lint test
