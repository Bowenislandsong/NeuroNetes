# Makefile for NeuroNetes

# Variables
REGISTRY ?= ghcr.io/bowenislandsong
IMAGE_NAME ?= neuronetes
VERSION ?= v0.1.0
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(VERSION)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOPATH=$(shell $(GOCMD) env GOPATH)
CONTROLLER_GEN=$(GOPATH)/bin/controller-gen

# Directories
CONTROLLER_DIR=controllers
PKG_DIR=pkg
API_DIR=api
TEST_DIR=test

.PHONY: all build test clean install deploy

## help: Display this help message
help:
	@echo "NeuroNetes Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/^## /  /'

## all: Build all components
all: test build

## build: Build all binaries
build:
	@echo "Building controllers..."
	$(GOBUILD) -v -o bin/manager ./cmd/manager/main.go
	$(GOBUILD) -v -o bin/scheduler ./cmd/scheduler/main.go
	$(GOBUILD) -v -o bin/autoscaler ./cmd/autoscaler/main.go

## test: Run unit tests
test:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./test/unit/...
	@echo "Coverage report:"
	$(GOCMD) tool cover -func=coverage.out

## test-metrics: Run metrics tests
test-metrics:
	@echo "Running metrics tests..."
	$(GOTEST) -v -race -coverprofile=metrics-coverage.out ./pkg/metrics/...

## test-plugins: Run plugin tests
test-plugins:
	@echo "Running plugin tests..."
	$(GOTEST) -v -race -coverprofile=plugins-coverage.out ./pkg/plugins/...

## test-scheduler: Run scheduler tests
test-scheduler:
	@echo "Running scheduler tests..."
	$(GOTEST) -v -race -coverprofile=scheduler-coverage.out ./pkg/scheduler/...

## test-autoscaler: Run autoscaler tests
test-autoscaler:
	@echo "Running autoscaler tests..."
	$(GOTEST) -v -race -coverprofile=autoscaler-coverage.out ./pkg/autoscaler/...

## test-all: Run all tests
test-all: test test-metrics test-plugins test-scheduler test-autoscaler test-integration test-e2e

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -timeout 30m ./test/integration/...

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	$(GOTEST) -v -timeout 60m ./test/e2e/...

## lint: Run linters
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## generate: Generate code (CRDs, clients, etc.)
generate:
	@echo "Generating code..."
	@test -f $(CONTROLLER_GEN) || (echo "Installing controller-gen..." && $(GOCMD) install sigs.k8s.io/controller-tools/cmd/controller-gen@latest)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true,crdVersions=v1 rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd

## manifests: Generate Kubernetes manifests
manifests: generate
	@echo "Generating manifests..."
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true,crdVersions=v1 paths="./api/..." output:crd:artifacts:config=config/crd
	@mkdir -p config/deploy
	kustomize build config/default > config/deploy/neuronetes.yaml

## install: Install CRDs into the cluster
install: manifests
	@echo "Installing CRDs..."
	kubectl apply -f config/crd/

## uninstall: Uninstall CRDs from the cluster
uninstall:
	@echo "Uninstalling CRDs..."
	kubectl delete -f config/crd/ --ignore-not-found=true

## deploy: Deploy controllers to the cluster
deploy: manifests
	@echo "Deploying controllers..."
	kubectl apply -f config/deploy/

## undeploy: Remove controllers from the cluster
undeploy:
	@echo "Removing controllers..."
	kubectl delete -f config/deploy/ --ignore-not-found=true

## docker-build: Build docker image
docker-build:
	@echo "Building docker image..."
	docker build -t $(IMG) .

## docker-push: Push docker image
docker-push:
	@echo "Pushing docker image..."
	docker push $(IMG)

## docker-build-push: Build and push docker image
docker-build-push: docker-build docker-push

## examples: Deploy example configurations
examples:
	@echo "Deploying examples..."
	kubectl apply -f config/samples/

## verify: Run all verification steps
verify: fmt vet lint test

## ci: Run CI checks
ci: deps verify test-integration

## dev: Start development environment
dev:
	@echo "Starting development environment..."
	kind create cluster --name neuronetes-dev --config hack/kind-config.yaml || true
	make install
	make deploy

## dev-clean: Clean development environment
dev-clean:
	@echo "Cleaning development environment..."
	kind delete cluster --name neuronetes-dev

## run-local: Run controller locally
run-local:
	@echo "Running controller locally..."
	$(GOCMD) run ./cmd/manager/main.go

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## coverage: Generate coverage report
coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## docs: Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Run 'godoc -http=:6060' to view documentation at http://localhost:6060"

## docker-compose-up: Start local services with docker-compose
docker-compose-up:
	@echo "Starting local services..."
	docker compose up -d
	@echo "Services started. Access:"
	@echo "  - Redis: localhost:6379"
	@echo "  - NATS: localhost:4222"
	@echo "  - Weaviate: localhost:8080"
	@echo "  - Prometheus: localhost:9090"
	@echo "  - Grafana: localhost:3000 (admin/admin)"
	@echo "  - NeuroNetes: localhost:8081"

## docker-compose-down: Stop local services
docker-compose-down:
	@echo "Stopping local services..."
	docker compose down

## docker-compose-logs: View logs from docker-compose services
docker-compose-logs:
	docker compose logs -f

## docker-compose-ps: Check status of docker-compose services
docker-compose-ps:
	docker compose ps

## local-test: Quick local test with docker-compose
local-test: docker-compose-up
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Running tests against local services..."
	$(GOTEST) -v -tags=integration ./test/integration/...

.DEFAULT_GOAL := help
