.PHONY: all build test lint check clean

BINARY_NAME := ralph-cc
BUILD_DIR := bin

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ralph-cc

test:
	go test -v ./...

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed, using go vet" && go vet ./...)
	@which golangci-lint > /dev/null && golangci-lint run ./... || true

check: lint test

clean:
	rm -rf $(BUILD_DIR)
	go clean
