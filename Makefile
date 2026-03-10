## Sigil — Build targets
## SQLite: modernc.org/sqlite (pure Go, no CGO required).
## go-tree-sitter (M4+): CGO required — install MinGW-w64 gcc when reaching M4.

BUILD := go build
TEST  := go test

.PHONY: all build build-cli build-mcp test test-race vet tidy clean

all: build

## Build both binaries
build: build-cli build-mcp

build-cli:
	$(BUILD) -o sigil.exe ./cmd/sigil

build-mcp:
	$(BUILD) -o sigil-mcp.exe ./cmd/mcp

## Run all tests
test:
	$(TEST) ./...

## Run tests with race detector
test-race:
	$(TEST) -race ./...

## Run go vet
vet:
	go vet ./...

## Tidy go.mod / go.sum
tidy:
	go mod tidy

## Clean built binaries
clean:
	rm -f sigil.exe sigil-mcp.exe
