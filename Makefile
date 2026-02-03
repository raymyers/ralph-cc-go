.PHONY: all build test test-slow test-all lint check coverage clean

BINARY_NAME := ralph-cc
BUILD_DIR := bin
COVERAGE_DIR := coverage

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ralph-cc

test:
	go test -skip 'TestE2ERuntimeYAML' ./...

test-slow:
	go test -run 'TestE2ERuntimeYAML' ./...

test-all: test test-slow

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed, using go vet" && go vet ./...)
	@which golangci-lint > /dev/null && golangci-lint run ./... || true

coverage:
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "HTML coverage report: $(COVERAGE_DIR)/coverage.html"
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

check: lint test-all

clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	go clean
