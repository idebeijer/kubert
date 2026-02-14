# Binary name to use when building the binary.
BINARY ?= kubert

.PHONY: all
all: build lint

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

.PHONY: test-cover
test-cover: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## Run golangci-lint against code
	golangci-lint run

.PHONY: lint-fix
lint-fix: ## Run golangci-lint against code and fix issues
	golangci-lint run --fix

.PHONY: vulncheck
vulncheck: ## Run govulncheck against code
	go tool govulncheck ./...

##@ Build

.PHONY: build
build: fmt vet ## Build the binary
	go build -o bin/${BINARY} main.go

.PHONY: run
run: fmt vet ## Run the binary
	go run main.go $(ARGS)

.PHONY: goreleaser-release-snapshot
goreleaser-release-snapshot: ## Build a snapshot release locally with goreleaser at ./dist (does not publish).
	goreleaser release --snapshot --clean --skip=sign

##@ Utils

.PHONY: docs
docs: ## Generate docs
	go run tools/docs.go

.PHONY: generate-default-config
generate-default-config: ## Generate default config
	go run tools/generate_default_cfg.go

##@ Testing in Docker (experimental tests)

.PHONY: test-docker-build
test-docker-build: ## Build the test Docker image (used for testing in Docker)
	docker build -t kubert-test -f testdata/Dockerfile .

.PHONY: test-docker-run
test-docker-run: test-docker-build ## Run tests inside the test Docker image (used for testing in Docker)
	docker run --rm -e RUN_SHELL_TESTS=true kubert-test:latest

.PHONY: test-docker-exec-shell
test-docker-exec-shell: test-docker-build ## Open a shell in the Docker test container
    docker run --rm -it kubert-test:latest /bin/sh

.PHONY: test-docker-shells
test-docker-bash: test-docker-build ## Test with bash shell in Docker
    docker run --rm -e RUN_SHELL_TESTS=true -e SHELL=/bin/bash kubert-test:latest go test -run TestLaunchShells ./cmd -v

.PHONY: test-docker-full
test-docker-full: test-docker-build ## Run all tests (including shell tests) in Docker
    docker run --rm -e RUN_SHELL_TESTS=true kubert-test:latest go test ./... -v
