## Sigil — Build targets
## SQLite: modernc.org/sqlite (pure Go, no CGO required).
## go-tree-sitter (M4+): CGO required — install MinGW-w64 gcc when reaching M4.

BUILD   := go build
TEST    := go test
BIN_DIR := bin

.PHONY: all build build-cli build-mcp test test-race vet tidy clean

all: build

## Build both binaries into bin/
build: build-cli build-mcp

build-cli: | $(BIN_DIR)
	$(BUILD) -o $(BIN_DIR)/sigil.exe ./cmd/sigil

build-mcp: | $(BIN_DIR)
	$(BUILD) -o $(BIN_DIR)/sigil-mcp.exe ./cmd/mcp

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

## Run all tests
test:
	$(TEST) ./...

## Run tests with race detector
test-race:
	$(TEST) -race ./...

## Run go vet
vet:
	CGO_ENABLED=1 go vet ./...

## Tidy go.mod / go.sum
tidy:
	go mod tidy

## Clean built binaries
clean:
	rm -rf $(BIN_DIR)
