PKGNAME=tumble
VERSION=$(shell git describe --tags --always | sed -e 's/-/\./g')
BINARY_NAME=tumble
BUILD_DIR=bin

.PHONY: all build clean test deps docs kill restart reset-db load-fixtures

all: build

deps:
	go mod download

GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X tumble/internal/version.CommitHash=$(GIT_COMMIT)"

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tumble

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

test-api: build
	./tests/api_test.sh

docs:
	@echo "Generating API docs..."
	# Placeholder for Swagger/OpenAPI generation
	# e.g. swag init -g cmd/tumble/main.go --output docs/api

kill:
	-pkill -f $(BUILD_DIR)/$(BINARY_NAME)

restart: kill build
	$(BUILD_DIR)/$(BINARY_NAME) conf/config.yaml &

reset-db:
	rm -f tumble.sqlite

load-fixtures:
	./tests/load_fixtures.sh

