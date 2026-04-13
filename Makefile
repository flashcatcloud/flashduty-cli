# Build configuration
BINARY_NAME := flashduty
BUILD_DIR := bin
GOLANGCI_LINT_VERSION := v2.2.1
GOLANGCI_LINT := $(BUILD_DIR)/golangci-lint
GCI_VERSION := v0.13.5
GCI := $(BUILD_DIR)/gci

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOFMT := gofmt
MODULE := $(shell go list -m)

# Build metadata
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
.PHONY: all
all: check

# ============================================================================
# Development targets
# ============================================================================

.PHONY: build
build: ## Build the binary
	$(GOBUILD) -v $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/flashduty

.PHONY: run
run: build ## Build and run the CLI
	./$(BUILD_DIR)/$(BINARY_NAME)

# ============================================================================
# Quality assurance targets
# ============================================================================

FMT_DIRS := cmd internal

.PHONY: fmt
fmt: $(GCI) ## Format Go source code and sort imports
	$(GOFMT) -s -w $(FMT_DIRS)
	$(GCI) write --skip-generated -s standard -s default -s "prefix($(MODULE))" $(FMT_DIRS)

.PHONY: gci
gci: $(GCI) ## Sort imports using gci
	$(GCI) write --skip-generated -s standard -s default -s "prefix($(MODULE))" $(FMT_DIRS)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Run golangci-lint with auto-fix
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: ## Run unit tests
	$(GOTEST) -race ./...

.PHONY: test-v
test-v: ## Run unit tests with verbose output
	$(GOTEST) -race -v ./...

.PHONY: test-cover
test-cover: ## Run unit tests with coverage
	$(GOTEST) -race -cover ./...

# ============================================================================
# Pre-push check (recommended before pushing)
# ============================================================================

.PHONY: check
check: fmt lint test build ## Run all checks (fmt, lint, test, build) - recommended before pushing

.PHONY: ci
ci: check ## Alias for check

# ============================================================================
# Dependency management
# ============================================================================

.PHONY: deps
deps: ## Download Go dependencies
	$(GOCMD) mod download

.PHONY: deps-tidy
deps-tidy: ## Tidy Go modules
	$(GOCMD) mod tidy

.PHONY: deps-verify
deps-verify: ## Verify Go dependencies
	$(GOCMD) mod verify

# ============================================================================
# Tools installation
# ============================================================================

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(GOLANGCI_LINT): $(BUILD_DIR)
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION); \
	fi

$(GCI): $(BUILD_DIR)
	@if [ ! -f "$(GCI)" ]; then \
		echo "Installing gci $(GCI_VERSION)..."; \
		GOBIN=$(CURDIR)/$(BUILD_DIR) $(GOCMD) install github.com/daixiang0/gci@$(GCI_VERSION); \
	fi

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GCI) ## Install required tools

# ============================================================================
# Cleanup
# ============================================================================

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)

# ============================================================================
# Help
# ============================================================================

.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Quick start:"
	@echo "  make check    - Run all pre-push checks (recommended before pushing)"
	@echo "  make lint     - Run linter only"
	@echo "  make test     - Run unit tests only"
