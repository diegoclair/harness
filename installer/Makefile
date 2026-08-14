# harness installer Makefile
#
#   make build       Build native binary into bin/harness
#   make build-all   Cross-compile for darwin/linux/windows
#   make test        Run unit tests
#   make fmt / lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
PKG     := .
BIN     := harness

.PHONY: all build build-all test install clean clean-dist fmt lint deps

all: build

deps:
	go mod tidy

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

# Asset names must match what install.sh / install.ps1 download.
build-all: clean-dist
	@mkdir -p dist
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-amd64      $(PKG)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-arm64      $(PKG)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-amd64       $(PKG)
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-arm64       $(PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-windows-amd64.exe $(PKG)
	@echo "built dist/ artifacts for version $(VERSION)"

test:
	go test ./...

install: build
	@mkdir -p $(HOME)/.local/bin
	cp bin/$(BIN) $(HOME)/.local/bin/$(BIN)
	@echo "installed to $(HOME)/.local/bin/$(BIN)"

clean: clean-dist
	rm -rf bin

clean-dist:
	rm -rf dist

fmt:
	go fmt ./...

lint:
	go vet ./...
