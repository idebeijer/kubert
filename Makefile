# Container tool to use to build images.
CONTAINER_TOOL ?= docker

# Binary name to use when building the binary.
BINARY ?= kubert

# Image name to use when building images.
IMG ?= kubert:latest

.PHONY: all
all: build

##@ Development

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./...

##@ Build

.PHONY: build
build: fmt vet
	go build -o bin/${BINARY} main.go

.PHONY: run
run: fmt vet
	go run main.go $(filter-out $@,$(MAKECMDGOALS))

%:
	@:

.PHONY: docker-build
docker-build:
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push:
	$(CONTAINER_TOOL) push ${IMG}

##@ Utils

.PHONY: docs
docs:
	go run tools/docs.go

.PHONY: generate-default-config
generate-default-config:
	go run tools/generate_default_cfg.go