.PHONY: help build clean install test fmt vet

BINARY_NAME=gh-worktree
GO=go
GOFLAGS=-ldflags="-s -w"
INSTALL_PATH=$(HOME)/.local/share/gh/extensions/gh-worktree

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/gh-worktree

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)

install: build ## Install as gh extension
	@mkdir -p $(INSTALL_PATH)
	@cp $(BINARY_NAME) $(INSTALL_PATH)/
	@chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "gh-worktree installed as gh extension"
	@echo "You can now use: gh worktree"
	@echo ""
	@echo "To enable tab completion, run:"
	@echo "  ./scripts/setup-completion.sh"

test: ## Run tests
	$(GO) test -v ./internal/...
	cd test/e2e && $(GO) test -v ./...

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...
