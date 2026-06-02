BINARY := scrnshot
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build universal test clean install deps

# Populate go.sum for all deps (sftp/x/crypto are added on first run here).
deps:
	go mod tidy

# Build for the host architecture.
build: deps
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Build a macOS universal binary (Intel + Apple Silicon). Requires macOS (lipo).
universal: deps
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-arm64 .
	lipo -create -output $(BINARY) $(BINARY)-amd64 $(BINARY)-arm64
	rm -f $(BINARY)-amd64 $(BINARY)-arm64
	@file $(BINARY)

test:
	go test ./...

# Copy to a directory on your PATH.
install: build
	install -m 0755 $(BINARY) $(HOME)/bin/$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-amd64 $(BINARY)-arm64
