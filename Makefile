.PHONY: help build build-with-lattice build-with-picogrid build-authorized test lint vet fmt clean install

BIN := bin/goat-translator
PKG := ./cmd/goat-translator
GOFLAGS := -trimpath
LDFLAGS := -s -w

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the default binary (open-standards adapters only).
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

build-with-lattice: ## Build with the Anduril Lattice adapter (requires authorized goat-lattice-shim clone).
	go build $(GOFLAGS) -tags lattice -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

build-with-picogrid: ## Build with the Picogrid ECN adapter (requires authorized goat-picogrid-shim clone).
	go build $(GOFLAGS) -tags picogrid -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

build-authorized: ## Build with all available vendor adapters (requires all private adapter repos).
	go build $(GOFLAGS) -tags "lattice picogrid" -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

test: ## Run unit tests.
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

lint: ## Run golangci-lint (must be installed locally).
	golangci-lint run ./...

vet: ## Run go vet.
	go vet ./...

fmt: ## Format Go source files.
	gofmt -w -s .

clean: ## Remove build artifacts.
	rm -rf bin/ dist/ build/ coverage.txt coverage.html

install: build ## Install the default binary to $$GOBIN.
	go install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(PKG)
