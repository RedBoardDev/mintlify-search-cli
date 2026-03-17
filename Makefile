BINARY  := msc
MODULE  := github.com/redboard/mintlify-search-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/internal/cli.Version=$(VERSION)"

.PHONY: build test lint vet install clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/msc

test:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || cp $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
