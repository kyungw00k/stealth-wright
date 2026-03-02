.PHONY: build clean install run test help

BINARY=sw
MAIN_PATH=./cmd/sw

build:
	go build -o bin/$(BINARY) $(MAIN_PATH)

install:
	go install $(MAIN_PATH)

run:
	go run $(MAIN_PATH) $(ARGS)

clean:
	rm -rf bin/
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
	@echo "  build    - Build the binary"
	@echo "  install  - Install to GOPATH/bin"
	@echo "  run      - Run directly"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  deps     - Download dependencies"
