# ====================================================================================
# Setup Project
PROJECT_NAME := k8s-app-plugin
PROJECT_REPO := github.com/odnielgonzalez/$(PROJECT_NAME)

# ====================================================================================
# Setup Go
GO := go
GOBIN := $(shell $(GO) env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell $(GO) env GOPATH)/bin
endif

# Binary names
KUBECTL_PLUGIN := kubectl-scanfail
KSCAN := kscan
KSCAN_GUI := kscan-gui

# Build directories
BUILD_DIR := bin
DIST_DIR := dist

# Go build flags
GO_BUILD_FLAGS := -v
GO_LDFLAGS := -w -s

# ====================================================================================
# Setup Tools
KIND_VERSION ?= v0.20.0
KUBECTL_VERSION ?= v1.29.0
GOLANGCI_LINT_VERSION ?= v1.55.2

TOOLS_DIR := $(shell pwd)/tools
KIND := $(TOOLS_DIR)/kind
KUBECTL := $(TOOLS_DIR)/kubectl
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

# ====================================================================================
# Setup KIND Cluster
CLUSTER_NAME ?= scanfail-test
KIND_CONFIG ?= kind-config.yaml

# ====================================================================================
# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

define INFO
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)INFO:$(COLOR_RESET) $(1)"
endef

define OK
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)OK:$(COLOR_RESET) $(1)"
endef

define WARN
	@echo "$(COLOR_BOLD)$(COLOR_YELLOW)WARN:$(COLOR_RESET) $(1)"
endef

# ====================================================================================
# Build Targets

.PHONY: all
all: build

.PHONY: help
help:
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Build targets:$(COLOR_RESET)"
	@echo "  make build              - Build all binaries"
	@echo "  make build-plugin       - Build kubectl plugin"
	@echo "  make build-cli          - Build standalone CLI"
	@echo "  make build-gui          - Build GUI application"
	@echo "  make install-plugin     - Install kubectl plugin to GOBIN"
	@echo ""
	@echo "$(COLOR_BOLD)Development targets:$(COLOR_RESET)"
	@echo "  make run-cli            - Run standalone CLI"
	@echo "  make run-gui            - Run GUI application"
	@echo "  make test               - Run tests"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make lint               - Run linters"
	@echo "  make fmt                - Format code"
	@echo ""
	@echo "$(COLOR_BOLD)KIND cluster targets:$(COLOR_RESET)"
	@echo "  make cluster-up         - Create KIND cluster"
	@echo "  make cluster-down       - Delete KIND cluster"
	@echo "  make cluster-restart    - Restart KIND cluster"
	@echo "  make cluster-info       - Show cluster info"
	@echo ""
	@echo "$(COLOR_BOLD)Example resource targets:$(COLOR_RESET)"
	@echo "  make deploy-examples    - Deploy all example resources"
	@echo "  make deploy-healthy     - Deploy healthy resources"
	@echo "  make deploy-failing     - Deploy failing resources"
	@echo "  make undeploy-examples  - Remove all example resources"
	@echo ""
	@echo "$(COLOR_BOLD)Utility targets:$(COLOR_RESET)"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make deps               - Download dependencies"
	@echo "  make tools              - Install development tools"
	@echo "  make scan               - Run scanner against current cluster (interactive)"
	@echo "  make scan-strict        - Run scanner with strict exit codes (for CI/CD)"

.PHONY: build
build: build-plugin build-cli build-gui
	$(call OK,All binaries built successfully)

.PHONY: build-plugin
build-plugin:
	$(call INFO,Building kubectl plugin...)
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(KUBECTL_PLUGIN) ./cmd/kubectl-scanfail
	$(call OK,kubectl plugin built: $(BUILD_DIR)/$(KUBECTL_PLUGIN))

.PHONY: build-cli
build-cli:
	$(call INFO,Building standalone CLI...)
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(KSCAN) ./cmd/kscan
	$(call OK,Standalone CLI built: $(BUILD_DIR)/$(KSCAN))

.PHONY: build-gui
build-gui:
	$(call INFO,Building GUI application...)
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -ldflags "$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(KSCAN_GUI) ./cmd/kscan-gui
	$(call OK,GUI application built: $(BUILD_DIR)/$(KSCAN_GUI))

.PHONY: install-plugin
install-plugin: build-plugin
	$(call INFO,Installing kubectl plugin to $(GOBIN)...)
	@cp $(BUILD_DIR)/$(KUBECTL_PLUGIN) $(GOBIN)/$(KUBECTL_PLUGIN)
	@chmod +x $(GOBIN)/$(KUBECTL_PLUGIN)
	$(call OK,kubectl plugin installed: $(GOBIN)/$(KUBECTL_PLUGIN))
	@echo ""
	@echo "You can now use: kubectl scanfail --help"

# ====================================================================================
# Development Targets

.PHONY: run-cli
run-cli: build-cli
	$(call INFO,Running standalone CLI...)
	./$(BUILD_DIR)/$(KSCAN) --help

.PHONY: run-gui
run-gui: build-gui
	$(call INFO,Running GUI application...)
	./$(BUILD_DIR)/$(KSCAN_GUI)

.PHONY: test
test:
	$(call INFO,Running tests...)
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(call OK,Tests completed)

.PHONY: test-verbose
test-verbose:
	$(call INFO,Running tests with verbose output...)
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(call OK,Tests completed)

.PHONY: test-coverage
test-coverage: test
	$(call INFO,Generating coverage report...)
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(call OK,Coverage report generated: coverage.html)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(call INFO,Running linters...)
	$(GOLANGCI_LINT) run --timeout=5m ./...
	$(call OK,Linting completed)

.PHONY: fmt
fmt:
	$(call INFO,Formatting code...)
	$(GO) fmt ./...
	$(call OK,Code formatted)

.PHONY: vet
vet:
	$(call INFO,Running go vet...)
	$(GO) vet ./...
	$(call OK,go vet completed)

# ====================================================================================
# KIND Cluster Targets

.PHONY: cluster-up
cluster-up: $(KIND) $(KUBECTL)
	$(call INFO,Creating KIND cluster: $(CLUSTER_NAME))
	@$(KIND) get clusters | grep -q "^$(CLUSTER_NAME)$$" && \
		$(call WARN,Cluster $(CLUSTER_NAME) already exists) || \
		$(KIND) create cluster --name=$(CLUSTER_NAME)
	@$(KUBECTL) cluster-info --context kind-$(CLUSTER_NAME)
	$(call OK,KIND cluster created: $(CLUSTER_NAME))

.PHONY: cluster-down
cluster-down: $(KIND)
	$(call INFO,Deleting KIND cluster: $(CLUSTER_NAME))
	$(KIND) delete cluster --name=$(CLUSTER_NAME) || true
	$(call OK,KIND cluster deleted)

.PHONY: cluster-restart
cluster-restart: cluster-down cluster-up
	$(call OK,KIND cluster restarted)

.PHONY: cluster-info
cluster-info: $(KUBECTL)
	$(call INFO,Cluster information:)
	@$(KUBECTL) cluster-info --context kind-$(CLUSTER_NAME)
	@echo ""
	@$(KUBECTL) get nodes
	@echo ""
	@$(KUBECTL) get pods --all-namespaces

# ====================================================================================
# Example Resource Targets

.PHONY: deploy-examples
deploy-examples: deploy-healthy deploy-failing
	$(call OK,All example resources deployed)

.PHONY: deploy-healthy
deploy-healthy: $(KUBECTL)
	$(call INFO,Deploying healthy resources...)
	$(KUBECTL) apply -f examples/healthy-resources.yaml
	$(call OK,Healthy resources deployed)

.PHONY: deploy-failing
deploy-failing: $(KUBECTL)
	$(call INFO,Deploying failing resources...)
	$(KUBECTL) apply -f examples/failing-resources.yaml
	$(call OK,Failing resources deployed)
	$(call WARN,Note: It may take a minute for resources to start failing)

.PHONY: undeploy-examples
undeploy-examples: $(KUBECTL)
	$(call INFO,Removing example resources...)
	-$(KUBECTL) delete -f examples/healthy-resources.yaml
	-$(KUBECTL) delete -f examples/failing-resources.yaml
	$(call OK,Example resources removed)

.PHONY: scan
scan: build-cli
	$(call INFO,Scanning cluster for failing resources...)
	@./$(BUILD_DIR)/$(KSCAN) --all-namespaces || true
	@echo ""
	$(call INFO,Scan complete - check results above)

.PHONY: scan-strict
scan-strict: build-cli
	$(call INFO,Scanning cluster for failing resources [STRICT MODE]...)
	./$(BUILD_DIR)/$(KSCAN) --all-namespaces
	$(call OK,No failures found)

# ====================================================================================
# Tool Installation Targets

.PHONY: tools
tools: $(KIND) $(KUBECTL) $(GOLANGCI_LINT)
	$(call OK,All tools installed)

$(KIND):
	$(call INFO,Installing KIND $(KIND_VERSION)...)
	@mkdir -p $(TOOLS_DIR)
	@curl -Lo $(KIND) https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$$(uname -s | tr '[:upper:]' '[:lower:]')-$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
	@chmod +x $(KIND)
	$(call OK,KIND installed: $(KIND))

$(KUBECTL):
	$(call INFO,Installing kubectl $(KUBECTL_VERSION)...)
	@mkdir -p $(TOOLS_DIR)
	@curl -Lo $(KUBECTL) "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$$(uname -s | tr '[:upper:]' '[:lower:]')/$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')/kubectl"
	@chmod +x $(KUBECTL)
	$(call OK,kubectl installed: $(KUBECTL))

$(GOLANGCI_LINT):
	$(call INFO,Installing golangci-lint $(GOLANGCI_LINT_VERSION)...)
	@mkdir -p $(TOOLS_DIR)
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_DIR) $(GOLANGCI_LINT_VERSION)
	$(call OK,golangci-lint installed: $(GOLANGCI_LINT))

# ====================================================================================
# Utility Targets

.PHONY: deps
deps:
	$(call INFO,Downloading dependencies...)
	$(GO) mod download
	$(GO) mod tidy
	$(call OK,Dependencies downloaded)

.PHONY: clean
clean:
	$(call INFO,Cleaning build artifacts...)
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	$(call OK,Build artifacts cleaned)

.PHONY: clean-tools
clean-tools:
	$(call INFO,Removing tools...)
	@rm -rf $(TOOLS_DIR)
	$(call OK,Tools removed)

.PHONY: distclean
distclean: clean clean-tools cluster-down
	$(call INFO,Performing deep clean...)
	@rm -rf vendor/
	$(call OK,Deep clean completed)

# ====================================================================================
# CI/CD Targets

.PHONY: ci
ci: deps fmt vet lint test build
	$(call OK,CI checks passed)

.PHONY: pre-commit
pre-commit: fmt vet lint test
	$(call OK,Pre-commit checks passed)

# ====================================================================================
# Advanced Targets

.PHONY: demo
demo: cluster-up deploy-examples
	@sleep 10
	$(call INFO,Waiting for resources to stabilize...)
	@sleep 20
	$(call INFO,Running scanner...)
	@$(MAKE) scan

.PHONY: full-test
full-test: cluster-up deploy-examples
	$(call INFO,Running full integration test...)
	@sleep 30
	@$(MAKE) scan || true
	@$(MAKE) undeploy-examples
	$(call OK,Full test completed)

# ====================================================================================
# Info Targets

.PHONY: version
version:
	@echo "Go version: $$($(GO) version)"
	@echo "Project: $(PROJECT_NAME)"
	@echo "Repository: $(PROJECT_REPO)"

.PHONY: go.cachedir
go.cachedir:
	@$(GO) env GOCACHE

.PHONY: go.mod.cachedir
go.mod.cachedir:
	@$(GO) env GOMODCACHE

.DEFAULT_GOAL := help

