.PHONY: build clean install run test help deps fmt lint

BINARY=sw
MAIN_PATH=./cmd/sw
BUILD_DIR=./build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(MAIN_PATH)

install: build
	@INSTALL_DIR=$${PREFIX:-$$HOME/.local/bin}; \
	mkdir -p $$INSTALL_DIR; \
	cp $(BUILD_DIR)/$(BINARY) $$INSTALL_DIR/$(BINARY); \
	chmod +x $$INSTALL_DIR/$(BINARY); \
	echo "Installed to $$INSTALL_DIR/$(BINARY)"; \
	echo "Make sure $$INSTALL_DIR is in your PATH"

run:
	go run $(MAIN_PATH) $(ARGS)

clean:
	rm -rf $(BUILD_DIR)
	go clean

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

deps:
	go mod download
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary to ./build/"
	@echo "  install  - Install to ~/.local/bin (use PREFIX=/path to override)"
	@echo "  run      - Run directly"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  deps     - Download dependencies"
