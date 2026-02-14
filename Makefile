.PHONY: build test clean install format lint

BINARY_NAME=inodes
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY_NAME) .

test:
	go test ./... -v

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

install:
	go install -ldflags "-X main.version=$(VERSION)" .

format:
	gofmt -w .

lint:
	gofmt -l . | grep -q . && echo "Files need formatting" && exit 1 || true
	go vet ./...
