# Google Play Console CLI Makefile

# Variables
BINARY_NAME := gplay
MODULE := github.com/tamtom/play-console-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go variables
GO := go
GOBIN := $(shell $(GO) env GOPATH)/bin
INSTALL_PREFIX ?= /usr/local/bin

# Build configuration
GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(DATE)'

# Directories
BUILD_DIR := ./build
DIST_DIR  := ./dist

# Platforms for cross-compilation
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

# Colors
GREEN  := \033[0;32m
YELLOW := \033[0;33m
BLUE   := \033[0;34m
RED    := \033[0;31m
NC     := \033[0m

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build: $(BINARY_NAME)
	@echo "$(GREEN)✓ Build complete: $(BINARY_NAME)$(NC)"

$(BINARY_NAME): go.mod
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "$(BLUE)Building $(BINARY_NAME) $(VERSION) for all platforms...$(NC)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o $(DIST_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$([ "$${platform%/*}" = "windows" ] && echo ".exe") . ; \
		echo "  $(GREEN)✓$(NC) Built: $(BINARY_NAME)-$${platform%/*}-$${platform#*/}" ; \
	done

# Build with debug symbols
.PHONY: build-debug
build-debug:
	@echo "$(BLUE)Building with debug symbols...$(NC)"
	$(GO) build -gcflags="all=-N -l" -o $(BINARY_NAME)-debug .

# Generate checksums for release
.PHONY: checksums
checksums:
	@echo "$(BLUE)Generating checksums...$(NC)"
	@cd $(DIST_DIR) && sha256sum * > checksums.txt
	@echo "$(GREEN)✓ Checksums written to $(DIST_DIR)/checksums.txt$(NC)"

# Run tests
.PHONY: test
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(NC)"

# Run integration tests (opt-in, requires credentials)
.PHONY: test-integration
test-integration:
	@echo "$(BLUE)Running integration tests (requires GPLAY_* env vars)...$(NC)"
	$(GO) test -tags=integration -v ./internal/playclient -run Integration

# Lint the code
.PHONY: lint
lint:
	@echo "$(BLUE)Linting code...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not found; falling back to 'go vet ./...'.$(NC)"; \
		echo "$(YELLOW)Install with: make tools$(NC)"; \
		$(GO) vet ./...; \
	fi

# Format code
.PHONY: format
fmt: format
format:
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GO) fmt ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "$(YELLOW)gofumpt not found; using go fmt only. Install with: make tools$(NC)"; \
	fi

# Install dev tools
.PHONY: tools
tools:
	@echo "$(BLUE)Installing dev tools...$(NC)"
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"
	@echo "$(YELLOW)Make sure '$(GOBIN)' is on your PATH$(NC)"

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	$(GO) mod tidy

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy

# Install to system
.PHONY: install
install: build
	@echo "$(BLUE)Installing to $(INSTALL_PREFIX)...$(NC)"
	install -d $(INSTALL_PREFIX)
	install -m 755 $(BINARY_NAME) $(INSTALL_PREFIX)/$(BINARY_NAME)
	@echo "$(GREEN)✓ Installed $(BINARY_NAME) to $(INSTALL_PREFIX)$(NC)"

# Uninstall from system
.PHONY: uninstall
uninstall:
	@echo "$(BLUE)Uninstalling...$(NC)"
	rm -f $(INSTALL_PREFIX)/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled $(BINARY_NAME)$(NC)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -f $(BINARY_NAME) $(BINARY_NAME)-debug
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html

# Show version info
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# Development: build and run
.PHONY: run
run: build
	./$(BINARY_NAME) $(ARGS)

# Development shortcut: format, lint, test, build
.PHONY: dev
dev: format lint test build
	@echo "$(GREEN)✓ Ready for development!$(NC)"

# Generate shell completions
.PHONY: completions
completions: build
	@mkdir -p $(BUILD_DIR)/completions
	./$(BINARY_NAME) completion bash > $(BUILD_DIR)/completions/$(BINARY_NAME).bash
	./$(BINARY_NAME) completion zsh > $(BUILD_DIR)/completions/_$(BINARY_NAME)
	./$(BINARY_NAME) completion fish > $(BUILD_DIR)/completions/$(BINARY_NAME).fish
	@echo "$(GREEN)✓ Completions generated in $(BUILD_DIR)/completions/$(NC)"

# Check for security vulnerabilities
.PHONY: security
security:
	@echo "$(BLUE)Checking for security vulnerabilities...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not found. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "$(YELLOW)govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest$(NC)"; \
	fi

# Help
.PHONY: help
help:
	@echo ""
	@echo "$(GREEN)Google Play Console CLI$(NC) - Build System"
	@echo ""
	@echo "$(BLUE)Build:$(NC)"
	@echo "  build           Build for current platform"
	@echo "  build-all       Build for all platforms (darwin, linux, windows)"
	@echo "  build-debug     Build with debug symbols"
	@echo "  checksums       Generate SHA256 checksums for releases"
	@echo ""
	@echo "$(BLUE)Test:$(NC)"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  test-integration Run integration tests (requires credentials)"
	@echo ""
	@echo "$(BLUE)Quality:$(NC)"
	@echo "  lint            Lint the code"
	@echo "  format          Format code (go fmt + gofumpt)"
	@echo "  security        Check for security vulnerabilities"
	@echo ""
	@echo "$(BLUE)Dependencies:$(NC)"
	@echo "  tidy            Tidy go.mod dependencies"
	@echo "  update-deps     Update all dependencies"
	@echo "  tools           Install dev tools (gofumpt, golangci-lint)"
	@echo ""
	@echo "$(BLUE)Install:$(NC)"
	@echo "  install         Install binary to $(INSTALL_PREFIX)"
	@echo "  uninstall       Uninstall binary"
	@echo ""
	@echo "$(BLUE)Other:$(NC)"
	@echo "  clean           Clean build artifacts"
	@echo "  version         Show version info"
	@echo "  run             Build and run (use ARGS= for arguments)"
	@echo "  completions     Generate shell completion scripts"
	@echo "  dev             format + lint + test + build"
	@echo "  update-api-spec Update Google Play API discovery document"
	@echo "  help            Show this help"
	@echo ""
	@echo "$(BLUE)Environment:$(NC)"
	@echo "  VERSION         Override version (default: git tag)"
	@echo "  INSTALL_PREFIX  Install location (default: /usr/local/bin)"
	@echo ""

# Update Google Play API discovery document and endpoints index
.PHONY: update-api-spec
update-api-spec:
	@echo "$(BLUE)Downloading Google Play Android Publisher API v3 discovery document...$(NC)"
	@curl -s 'https://androidpublisher.googleapis.com/$$discovery/rest?version=v3' | \
		python3 -m json.tool > docs/api/discovery.json
	@echo "$(BLUE)Generating endpoints index...$(NC)"
	@python3 scripts/gen-endpoints.py
	@echo "$(GREEN)✓ API spec updated. Review changes with: git diff docs/api/$(NC)"
