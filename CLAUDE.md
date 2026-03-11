# Sigil — CLAUDE.md

Token-efficient codebase intelligence framework. Indexes source code into SQLite with byte-offset symbol retrieval for AI agents. Design spec: `SIGIL_FRAMEWORK_DRAFT_v04.md`.

## Quick Commands

```bash
# Build (CGO_ENABLED=1 required from M4 onwards — tree-sitter)
CGO_ENABLED=1 go build ./...
CGO_ENABLED=1 go build -o sigil.exe ./cmd/sigil
CGO_ENABLED=1 go build -o sigil-mcp.exe ./cmd/mcp

# Or via Makefile
make build
make build-cli
make build-mcp

# Test
CGO_ENABLED=1 go test ./...
CGO_ENABLED=1 go test -race ./...
CGO_ENABLED=1 go test -run TestName ./internal/store/...

# Lint / vet
CGO_ENABLED=1 go vet ./...

# Run CLI directly
go run ./cmd/sigil version
go run ./cmd/sigil --help

# Add dependency
go get github.com/some/pkg
go mod tidy
```

## Architecture

Two binaries sharing all `internal/` packages:

```
cmd/sigil/main.go      → CLI (index, sync, search, get, deps, etc.)
cmd/mcp/main.go        → MCP stdio server (AI agent integration)
```

**Strict layer dependency flow — never skip layers:**

```
cli/*        → input validation + output formatting ONLY
   ↓
service/*    → business logic, orchestration
   ↓
store/*      → SQLite data access (via SymbolStore interface)
   ↓
models/*     → domain types, no internal deps
constants/*  → immutable values, no internal deps
```

**Other internal packages (consumed by service layer):**

| Package | Role |
|---|---|
| `discovery/` | File walking: git mode (blob SHA) + filesystem fallback |
| `security/` | 3-tier filter: Normal / Redacted / Ignored |
| `parser/` | tree-sitter extraction per language |
| `enrichment/` | LLM summary generation (Anthropic→Google→OpenAI→Ollama→template) |
| `mcpserver/` | MCP tool registration + handlers |
| `storage/` | `~/.sigil/` layout, `RepoHash()`, meta.json, repos.json |
| `config/` | Config load: env > `~/.sigil/config.toml` > defaults |
| `logger/` | slog setup — JSON handler, file output for MCP mode |

## Key Files

| File | Purpose |
|---|---|
| `SIGIL_FRAMEWORK_DRAFT_v04.md` | Authoritative design spec (schema, tools, security rules) |
| `internal/models/symbol.go` | Core Symbol type — maps 1:1 to SQLite schema |
| `internal/constants/constants.go` | Symbol kinds, confidence levels, env var states |
| `internal/constants/security.go` | Built-in redacted patterns, secret value regexes |
| `internal/constants/extensions.go` | Language extension allow-lists, package manifests |
| `internal/storage/paths.go` | `RepoHash()`, `EnsureLayout()`, all `~/.sigil/` paths |
| `internal/config/config.go` | Full Config struct + Load() |
| `internal/discovery/walker.go` | `FileEntry`, `Walker` interface, `NewWalker(root, extraIgnore)` |
| `internal/security/filter.go` | `Tier` (Normal/Redacted/Ignored), `Filter` interface |
| `internal/security/classify.go` | `NewFilter(extraIgnore, extraSecret, extraValuePatterns)` |
| `internal/mcpserver/server.go` | `NewServer(cfg)`, tool registration, `Run(ctx)` |
| `internal/mcpserver/handlers.go` | All 9 MCP tool handlers |
| `internal/mcpserver/resolve.go` | `resolveRepoRoot()`, `openRepo()` |

## Interfaces (all defined at consumer side)

```go
store.SymbolStore   — consumed by service/*, implemented by store/sqlite.go
parser.Parser       — consumed by service/indexer.go, implemented per language
discovery.Walker    — consumed by service/indexer.go; GitWalker + FilesystemWalker
security.Filter     — consumed by service/indexer.go; SecurityFilter in classify.go
enrichment.Enricher — consumed by service/indexer.go, implemented in enrichment/
```

All interfaces are mockable. Use mock structs in unit tests, never real SQLite unless testing the store itself.

### Implemented constructor signatures (M3–M8)

```go
// Auto-selects GitWalker (git ≥ 2.25) or FilesystemWalker.
// extraIgnore: additional .glyphignore-style patterns from config.
discovery.NewWalker(root string, extraIgnore []string) Walker

// extraIgnore → TierIgnored, extraSecret → TierRedacted,
// extraValuePatterns → additional secret value regexes.
security.NewFilter(extraIgnore, extraSecret, extraValuePatterns []string) (*SecurityFilter, error)

// Returns the nearest ancestor dir containing a package manifest.
discovery.FindPackageRoot(repoRoot, filePath string) string

// One instance per language; goroutine-safe.
golang.New()     *GoParser
typescript.New() *TSParser  // "github.com/smacker/go-tree-sitter/typescript/typescript"
javascript.New() *JSParser  // "github.com/smacker/go-tree-sitter/javascript"
python.New()     *PyParser  // "github.com/smacker/go-tree-sitter/python"
// All four are sub-packages of the same go-tree-sitter module — no extra go get needed.

// Stable 8-char hex symbol ID: lower(hex(SHA256(qualifiedName)))[:8].
parser.SymbolID(qualifiedName string) string

// Qualified name format: "{pkgPath}.{ReceiverType}.{Name}" or "{pkgPath}.{Name}".
// pkgPath = repo-relative directory of the file (e.g. "internal/parser/golang").

// Full indexing pipeline: Walker → SecurityFilter → Parser → Store.
// filesDir = storage.FilesDir(cacheRoot, repoHash) — raw file mirror.
// pkgPath passed to Parse() = filepath.Dir(entry.Path), NOT FindPackageRoot().
// FindPackageRoot() result → symbol.PackageRoot only.
service.NewIndexer(walker, filter, registry, st, filesDir, repoRoot string, maxFiles int) *Indexer
service.NewSyncer(idx *Indexer) *Syncer

// IndexOptions.Force = true → full rebuild; false → smart incremental (blob SHA / mtime+size).
// IndexResult.SymbolsTotal counts only symbols from the current run's indexed files.

// M6 — Query services (all consume store.SymbolStore).
service.NewSearcher(st) *Searcher        // Search(ctx, query, SearchOptions) (*SearchResult, error)
service.NewRetriever(st, filesDir, repoRoot) *Retriever  // Get(ctx, ids, files, contextLines) (*GetResult, error)
service.NewDeps(st) *Deps                // Trace(ctx, symbolID, direction, depth) (*DepsResult, error)
service.NewOutline(st) *Outline          // ForFile(ctx, file) (*OutlineResult, error)
service.NewTree(st, repoRoot) *Tree      // Build(ctx, scope, maxDepth, includeSymbolCounts) (*TreeResult, error)
service.NewOverview(st, meta) *OverviewService  // Summary(ctx) (*OverviewResult, error)
service.NewEnvService(repoRoot) *EnvService     // Inspect(ctx) (*EnvResult, error)
service.NewDiffer(st, repoRoot) *Differ         // Diff(ctx, since) (*DiffResult, error)
service.NewStatus(st, meta, repoRoot) *StatusService  // Check(ctx, verify) (*StatusResult, error)
service.NewCacheManager(cacheRoot) *CacheManager
service.NewSavings(st) *SavingsService

// M7 — Enrichment.
enrichment.Detect(timeout) Enricher     // auto-selects: Anthropic→Google→OpenAI→Ollama→template
enrichment.IsAvailable() bool
enrichment.BatchEnrich(ctx, enricher, symbols, srcMap, batchSize)
indexer.SetEnricher(e enrichment.Enricher)  // wire before calling Index()

// M8 — MCP server.
mcpserver.NewServer(cfg *config.Config) (*Server, error)
server.Run(ctx context.Context) error   // stdio transport; blocks until EOF/cancel
// Handler signature: func(ctx, *mcp.CallToolRequest, InputStruct) (*mcp.CallToolResult, struct{}, error)
// resolveRepoRoot(path, serverCWD) — git rev-parse --show-toplevel walk-up
// openRepo(repoRoot, cacheRoot) → (store.SymbolStore, *storage.RepoMeta, error)
```

## Tree-sitter Grammar Gotchas

- TS class names → `type_identifier` node; JS class names → `identifier` — separate query files required
- Python calls → `call` node (not `call_expression`); attribute calls → `(call function: (attribute attribute: (identifier)))`
- Python method vs function: walk `node.Parent()` — `function_definition → block → class_definition` = method
- Python docstring: first `expression_statement(string)` in body block; use `string_content` child to strip quotes
- `enclosingClassName` for TS/JS: `method_definition → class_body → class_declaration.ChildByFieldName("name")`
- Go `const`/`var` queries match ALL declarations including function-local ones — always guard with `defNode.Parent().Type() == "source_file"` to extract only package-level symbols
- Verify node types before writing queries: `sitter.ParseCtx(ctx, src, lang)` then print `node.String()`

## CGO Requirements

**SQLite (M1–M3, done):** `modernc.org/sqlite` is pure Go — no CGO, no gcc needed.

**Tree-sitter (M4+, done):** `go-tree-sitter` requires CGO. gcc 10.2.0 confirmed on this machine.
All builds from M4 onwards require `CGO_ENABLED=1`:
```bash
CGO_ENABLED=1 go build ./...
CGO_ENABLED=1 go test ./...

# Cross-compilation (see design doc §14)
CC="zig cc -target x86_64-linux-musl" CGO_ENABLED=1 GOOS=linux go build ./cmd/sigil
```

## Environment Variables

| Variable | Purpose |
|---|---|
| `CODE_INDEX_PATH` | Override `~/.sigil/` cache root |
| `SIGIL_LOG_LEVEL` | DEBUG / INFO / WARNING / ERROR (default: WARNING) |
| `SIGIL_LOG_FILE` | **Must be set in MCP stdio mode** — logs corrupt stdout otherwise |
| `SIGIL_MAX_INDEX_FILES` | Max files per repo (default: 500) |
| `SIGIL_ENRICH_BATCH_SIZE` | Concurrent LLM enrichment requests (default: 4) |
| `ANTHROPIC_API_KEY` | Enrichment via Claude Haiku (priority 1) |
| `GOOGLE_API_KEY` | Enrichment via Gemini Flash (priority 2) |
| `OPENAI_API_BASE` | OpenAI-compatible endpoint (priority 3) |

## Storage Layout

```
~/.sigil/
  repos.json                   ← global repo manifest
  tokens_saved.json            ← global savings rollup
  config.toml                  ← user config ([security], [indexing], [enrichment])
  {SHA256(path)[:12]}/
    index.db                   ← SQLite WAL mode, PRAGMA foreign_keys=ON
    meta.json                  ← {hash, path, name, mode, last_indexed_commit}
    files/
      {blob_sha}               ← raw source, content-addressed
```

## SQLite Schema Constraints

- Schema v1 — `PRAGMA user_version = 1`
- WAL mode + `busy_timeout = 5000ms` (concurrent read-heavy access)
- `content_hash` = audit only (never used for invalidation — blob SHA handles that)
- `byte_start`/`byte_end` = NULL when symbol is redacted
- All symbol inserts: `DELETE WHERE file = ?` + bulk re-insert in one transaction

## MCP Server Gotchas

- **Never write to stdout** in MCP mode — it corrupts the JSON-RPC stream
- Always set `SIGIL_LOG_FILE` when running `sigil-mcp` or `sigil mcp`
- Session ID is auto-generated at server start (process lifetime)
- All 9 MCP tools accept a `path` parameter (optional, falls back to server CWD)
- Repo selected by walking up from `path` to git root via `git rev-parse --show-toplevel`
- `sigil_env` does not open a store — no DB required for `.env` inspection
- Token savings persisted to `savings_log` via `st.AppendSavings()` on each tool call
- MCP client config: `{"command": "sigil", "args": ["mcp"], "env": {"SIGIL_LOG_FILE": "..."}}`

## Testing Patterns

```bash
# Unit tests use in-memory SQLite — no filesystem deps:
db, _ := sql.Open("sqlite3", ":memory:")

# Parser tests use fixture .go/.ts files in testdata/ directories
# Discovery tests use a temp git repo created via exec.Command("git", "init", ...)

# Race detector on all concurrency-sensitive packages:
go test -race ./internal/service/... ./internal/store/...
```

## Code Style Reminders

- `slog` for all logging — never `log.Printf` or `fmt.Println` in library code
- CLI-only printing via `fmt.Fprintf(cmd.OutOrStdout(), ...)` — never `fmt.Println`
- Guard clauses over nested ifs — early returns preferred
- Interfaces at consumer side — never in the implementing package
- `fd.ReadAt` not `fd.Seek+fd.Read` — goroutine-safe for concurrent retrieval
- Error wrap with `%w`, lowercase messages, no trailing punctuation

## Distribution & Release Builds

End users run a pre-compiled binary — no Go, gcc, or tree-sitter required. The C runtime
(tree-sitter) is statically linked at build time. Each OS/arch needs its own native build.

### Platform matrix

| Platform | Binary | Notes |
|---|---|---|
| Windows x64 | `sigil.exe` | Built on `windows-latest` runner |
| Linux x64 | `sigil` | Built on `ubuntu-latest` runner; musl for full static |
| macOS ARM64 | `sigil` | Built on `macos-latest` runner |

### GitHub Actions release workflow (skeleton)

```yaml
jobs:
  release:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            out: sigil-linux-amd64
          - os: windows-latest
            goos: windows
            goarch: amd64
            out: sigil-windows-amd64.exe
          - os: macos-latest
            goos: darwin
            goarch: arm64
            out: sigil-darwin-arm64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: CGO_ENABLED=1 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }}
               go build -ldflags="-s -w" -o ${{ matrix.out }} ./cmd/sigil
      - uses: actions/upload-artifact@v4
        with: { name: ${{ matrix.out }}, path: ${{ matrix.out }} }
```

Linux musl (fully static, no libc dependency):
```bash
CC="zig cc -target x86_64-linux-musl" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w -extldflags=-static" -o sigil-linux-amd64 ./cmd/sigil
```

WSL users use the Linux binary — it runs natively inside WSL2 and can access
Windows filesystems via `/mnt/c/...`.

## Implementation Status

All milestones M1–M8 complete. See `PROGRESS.md` for full details.

## Commit Style

- Never add `Co-Authored-By` trailers to commit messages.
