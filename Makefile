# Binary name to use when building the binary.
BINARY ?= kubert

.PHONY: all
all: build

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run vet against code
	go vet ./...

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	go test -race -v ./...

.PHONY: test-short
test-short: ## Run tests in short mode
	go test -short -v ./...

.PHONY: lint
lint: ## Run golangci-lint against code
	golangci-lint run

.PHONY: lint-fix
lint-fix: ## Run golangci-lint against code and fix issues
	golangci-lint run --fix

##@ Build

.PHONY: build
build: fmt vet ## Build the binary
	go build -o bin/${BINARY} main.go

.PHONY: run
run: fmt vet ## Run the binary
	go run main.go $(ARGS)

##@ Utils

.PHONY: docs
docs: ## Generate docs
	go run tools/docs.go

.PHONY: generate-default-config
generate-default-config: ## Generate default config
	go run tools/generate_default_cfg.go