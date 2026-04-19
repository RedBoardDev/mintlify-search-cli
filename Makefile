BINARY  := msc
MODULE  := github.com/redboard/mintlify-search-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/internal/cli.Version=$(VERSION) -X $(MODULE)/internal/mcp.ClientVersion=$(VERSION)"

.PHONY: build test test-integration lint vet install clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/msc

test:
	go test -race ./...

test-integration:
	go test -tags=integration ./integration/...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || cp $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
