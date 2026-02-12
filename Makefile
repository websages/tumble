PKGNAME=tumble
VERSION=$(shell git describe --tags --always | sed -e 's/-/\./g')
BINARY_NAME=tumble
BUILD_DIR=bin

.PHONY: all build clean test test-mysql deps docs kill restart reset-db load-fixtures build-linux help fmt

all: build ## Build the binary (default)

deps: ## Download dependencies
	go mod download

fmt: ## Run go fmt and cleanup whitespace
	go fmt ./...
	find internal/templates -type f \( -name "*.html" -o -name "*.xml" \) -exec sed -i '' 's/[[:blank:]]*$$//' {} +
	find tests -type f -name "*.sh" -exec sed -i '' 's/[[:blank:]]*$$//' {} +
	git diff --check

GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X tumble/internal/version.CommitHash=$(GIT_COMMIT)"

build: ## Build the binary
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tumble

build-linux: ## Build the binary for Linux amd64
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/tumble

clean: ## Clean build directory
	rm -rf $(BUILD_DIR) tumble-test.log tumble-test.sqlite*


test: ## Run unit tests
	go test -v ./...

test-mysql: ## Run MySQL-specific tests (requires MySQL)
	go test -v -tags mysql ./internal/data/

test-api: build ## Run API tests
	./tests/run_integration_tests.sh

docs: ## Generate API docs
	@echo "Generating API docs..."
	# Placeholder for Swagger/OpenAPI generation
	# e.g. swag init -g cmd/tumble/main.go --output docs/api

kill: ## Kill the running process
	-pkill -f $(BUILD_DIR)/$(BINARY_NAME)

restart: kill build ## Restart the application locally
	$(BUILD_DIR)/$(BINARY_NAME) conf/config.yaml &

reset-db: ## Remove sqlite database
	rm -f tumble.sqlite

backup: ## Backup the current database
	cp tumble.sqlite tumble.sqlite.bak
	@echo "Database backed up to tumble.sqlite.bak"

restore: ## Restore the database from backup
	cp tumble.sqlite.bak tumble.sqlite
	@echo "Database restored from tumble.sqlite.bak"

load-fixtures: ## Load test fixtures
	./tests/load_fixtures.sh

run-test: kill build ## Run the application with the test database
	$(BUILD_DIR)/$(BINARY_NAME) conf/config-test.yaml



test-db: kill build ## Create a fresh test database with fixtures
	./tests/setup_test_db.sh

help: ## Show this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
