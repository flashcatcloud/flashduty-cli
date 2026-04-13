BINARY=flashduty
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build check lint test clean

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/flashduty

check: lint test build

lint:
	golangci-lint run ./...

test:
	go test -race -cover ./...

clean:
	rm -rf bin/
