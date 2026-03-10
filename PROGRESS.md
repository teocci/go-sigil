# Sigil — Implementation Progress

> Tracks milestone status, files created, and token efficiency notes.
> Update this file as each milestone completes.

---

## Phase 1 Overview

| Milestone | Status | Files | Notes |
|---|---|---|---|
| M1 — Scaffold, Config, Logger | ✅ Done | 18 files | Go 1.23, Cobra+Viper, slog |
| M2 — SQLite Database Layer | ✅ Done | 11 files | 13 tests |
| M3 — Discovery & Security Filter | ✅ Done | 13 files | git ls-files + gitignore |
| M4 — Tree-sitter Parsing | ✅ Done | 8 files | Go parser, 9 tests |
| M4b — TS/JS/Python Parsers | ✅ Done | 12 files | 21 tests |
| M5 — Indexing Pipeline | ✅ Done | 5 files | Smart index, incremental sync |
| M6 — CLI Query Commands | ✅ Done | 26 files | search, get, deps, outline, tree, overview, env, diff, cache, savings, status, hook |
| M7 — Enrichment Pipeline | ✅ Done | 8 files | Anthropic→Google→OpenAI→Ollama→template, BatchEnrich, --enrich/--no-enrich |
| M8 — MCP Server | ✅ Done | 7 files | Official go-sdk v1.2.0, 9 tools, sigil mcp subcommand |

**Critical path:** M1 → M2 → M3 → M4 → M5 → M6 → M8
**Parallel:** M7 ∥ M6

---

## M1 — Scaffold, Config, Logger ✅

**Goal:** Runnable binary with `sigil version`, config loading, storage layout.

**Completed:** 2026-03-06

### Files Created

| File | Purpose |
|---|---|
| `cmd/sigil/main.go` | CLI entry point |
| `cmd/mcp/main.go` | MCP server entry point (stub) |
| `internal/cli/root.go` | Cobra root command + Viper binding |
| `internal/cli/version.go` | `sigil version` subcommand |
| `internal/config/config.go` | Config struct + Load() (env > toml > defaults) |
| `internal/config/defaults.go` | Platform-aware cache root detection |
| `internal/constants/constants.go` | Symbol kinds, confidence, env states, limits |
| `internal/constants/security.go` | Built-in redacted patterns, secret regexes |
| `internal/constants/extensions.go` | Language extension map, package manifests |
| `internal/logger/logger.go` | slog setup (JSON + file output) |
| `internal/models/file.go` | File domain type |
| `internal/models/symbol.go` | Symbol domain type (maps to DB schema) |
| `internal/models/edge.go` | CallEdge domain type |
| `internal/models/savings.go` | SavingsEntry, SavingsSummary, SessionInfo |
| `internal/models/metadata.go` | Metadata envelope (timing, savings) |
| `internal/storage/paths.go` | RepoHash(), EnsureLayout(), all path helpers |
| `internal/storage/meta.go` | meta.json read/write |
| `internal/storage/repos.go` | repos.json manifest read/write/upsert |

### Dependencies Added

```
github.com/spf13/cobra  v1.10.2
github.com/spf13/viper  v1.21.0
```

### Verification

```
✅ go build ./...
✅ go vet ./...
✅ go mod tidy
✅ sigil version → "sigil 0.1.0"
✅ sigil --help  → shows subcommand structure
```

### Notes

- Go toolchain auto-upgraded from 1.21 (installed) to 1.23.0 (required by Viper)
- `go.mod` now at `go 1.23.0`
- Design doc specifies Go 1.25 target — bump when locally available

---

## M2 — SQLite Database Layer ✅

**Goal:** Schema creation, migration runner, SymbolStore interface + SQLite impl.

**Completed:** 2026-03-09

### Files Created

| File | Purpose |
|---|---|
| `internal/db/db.go` | Open() with WAL, foreign_keys, busy_timeout |
| `internal/db/schema.go` | Schema v1 DDL (files, symbols, call_edges, symbols_fts, savings_log, 9 indexes, 3 FTS5 triggers) |
| `internal/db/migrate.go` | Migration runner via PRAGMA user_version + FTS5 availability probe |
| `internal/db/db_test.go` | TestOpen_InMemory, TestMigrate |
| `internal/store/store.go` | SymbolStore interface + SearchOptions |
| `internal/store/sqlite.go` | SQLiteStore + withTx() + nullable helpers |
| `internal/store/files.go` | UpsertFile (ON CONFLICT DO UPDATE), GetFile, DeleteFile, ListFiles |
| `internal/store/symbols.go` | ReplaceFileSymbols, GetSymbolByID, GetSymbolsByFile, GetSymbolsByIDs, MarkPossibleUnresolved |
| `internal/store/edges.go` | ReplaceFileEdges, GetCalls, GetCalledBy |
| `internal/store/search.go` | SearchSymbols (FTS5 JOIN with kind/language/file filters) |
| `internal/store/savings.go` | AppendSavings, GetSessionSavings, GetRepoSavings |
| `internal/store/sqlite_test.go` | Full test suite (11 tests) |

### New Dependency

```
github.com/mattn/go-sqlite3  v1.14.34 (CGO — requires gcc + -tags fts5)
```

### Verification

```
✅ CGO_ENABLED=1 go build -tags fts5 ./...
✅ go vet ./...
✅ go test -tags fts5 ./internal/db/...    (2 tests)
✅ go test -tags fts5 ./internal/store/... (11 tests)
```

### Notes

- call_edges uses `id INTEGER PRIMARY KEY AUTOINCREMENT` + two partial unique indexes (not COALESCE PK — invalid SQL)
- FTS5 must be explicitly enabled: `-tags fts5`; `migrate.Run()` probes availability and returns clear error if missing
- SQL query concatenation requires explicit spaces: `` `SELECT ` + symbolColumns + ` FROM` ``

---

## M3 — Discovery & Security Filter ✅

**Goal:** Walk repo files via git/filesystem, classify via 3-tier security model.

**Completed:** 2026-03-09

### Files Created

| File | Purpose                                                       |
|---|---------------------------------------------------------------|
| `internal/discovery/walker.go` | FileEntry + Walker interface + NewWalker() factory            |
| `internal/discovery/git.go` | git version detect (≥2.25/≥2.36), ls-files parsing, untracked |
| `internal/discovery/filesystem.go` | WalkDir fallback, per-dir ignorer stack                       |
| `internal/discovery/gitignore.go` | .gitignore + .sigilignore parser (full glob/negate/anchor)    |
| `internal/discovery/exclusions.go` | Built-in exclusion matching (dirs, globs, exact)              |
| `internal/discovery/packages.go` | FindPackageRoot() via manifest file detection                 |
| `internal/discovery/walker_test.go` | 15 tests covering both walkers + ignorer + exclusions         |
| `internal/security/filter.go` | Tier type (Normal/Redacted/Ignored) + Filter interface        |
| `internal/security/classify.go` | SecurityFilter + NewFilter() + patternMatches()               |
| `internal/security/binary.go` | IsBinary() — null-byte ratio check (8KB probe)                |
| `internal/security/pathcheck.go` | IsPathSafe() — symlink + path traversal guard                 |
| `internal/security/secrets.go` | IsPlaceholder(), MatchSecretPattern(), helpers                |
| `internal/security/filter_test.go` | 16 tests covering all tiers + binary + pathcheck              |

### Verification

```
✅ go build ./...
✅ go vet ./...
✅ go test ./internal/discovery/...  (15 tests)
✅ go test ./internal/security/...   (16 tests)
✅ go test ./...                     (all packages pass)
```

### Notes

- `FileEntry` gains `MTime` and `Size` fields (zero in git mode) to avoid double-stat in filesystem mode
- `NewWalker(root, extraIgnore)` — API extended with `extraIgnore` vs the stub's single-param signature
- GitWalker uses `--format=%(objectname) %(path)` for git ≥ 2.36; `ls-files -s` for 2.25–2.35
- Gitignore parser translates patterns to regex; anchoring rules match spec (leading/middle `/` anchors)
- `isIgnoredByStack` processes ancestor .gitignore files from root → leaf; last match wins (enables negation override)
- No new external dependencies — pure stdlib

---

## M4 — Tree-sitter Parsing ✅

**Goal:** Parse Go files → symbols + call graph edges. Vertical slice enabler.

**Completed:** 2026-03-09

### New Dependency

```
github.com/smacker/go-tree-sitter  v0.0.0-20240827094217-dd81d9e9be82 (CGO — requires gcc)
```

### Files Created

| File | Purpose |
|---|---|
| `internal/parser/parser.go` | Parser interface + Registry (thread-safe, Register/Get/Languages) |
| `internal/parser/result.go` | ParseResult{Symbols, Edges} |
| `internal/parser/id.go` | SymbolID = SHA256(qualified_name)[:8] hex |
| `internal/parser/signature.go` | BuildFuncSignature, BuildTypeSignature, NormalizeWhitespace |
| `internal/parser/summary.go` | FirstSentence (doc comment extraction) + TemplateSummary fallback |
| `internal/parser/golang/golang.go` | GoParser: functions, methods, types, interfaces, consts, vars, call edges |
| `internal/parser/golang/queries.go` | tree-sitter S-expression queries (6 query patterns) |
| `internal/parser/golang/testdata/sample.go` | Fixture file with all symbol kinds |
| `internal/parser/golang/golang_test.go` | 9 tests covering symbols, IDs, signatures, summaries, edges, qualified names |

### Verification

```
✅ CGO_ENABLED=1 go build ./...
✅ CGO_ENABLED=1 go vet ./...
✅ CGO_ENABLED=1 go test ./internal/parser/...  (9 tests)
✅ CGO_ENABLED=1 go test ./...                  (all packages pass)
```

### Notes

- `CGO_ENABLED=1` required for all builds from M4 onwards (tree-sitter C runtime)
- Qualified name format: `{pkgPath}.{ReceiverType}.{Name}` or `{pkgPath}.{Name}`
- Call edge `callee_id` left empty — resolved by indexer (M5) after all symbols loaded
- Docstring = first sentence of immediately preceding `// comment` block
- Template fallback: `"function Name accepting params, returns type"`
- `symRange` is a package-level type (not local) to avoid Go anonymous struct mismatch
- Additional parsers (TypeScript, JavaScript, Python) deferred to M4b after M5 vertical slice

---

## M4b — TS/JS/Python Parsers ✅

**Goal:** TypeScript, JavaScript, and Python symbol extraction using tree-sitter.

**Completed:** 2026-03-09

### Files Created

| File | Purpose |
|---|---|
| `internal/parser/typescript/typescript.go` | TSParser: functions, classes, methods, interfaces, type aliases, call edges |
| `internal/parser/typescript/queries.go` | tree-sitter S-expression queries (7 patterns) |
| `internal/parser/typescript/testdata/sample.ts` | Fixture with all symbol kinds |
| `internal/parser/typescript/typescript_test.go` | 6 tests: language, symbols, IDs, qualified names, summaries, edges, empty |
| `internal/parser/javascript/javascript.go` | JSParser: functions, classes, methods, call edges |
| `internal/parser/javascript/queries.go` | tree-sitter S-expression queries (5 patterns) |
| `internal/parser/javascript/testdata/sample.js` | Fixture with all symbol kinds |
| `internal/parser/javascript/javascript_test.go` | 6 tests matching Go/TS pattern |
| `internal/parser/python/python.go` | PyParser: functions, classes, methods (via parent-walk), docstring extraction |
| `internal/parser/python/queries.go` | tree-sitter S-expression queries (4 patterns) |
| `internal/parser/python/testdata/sample.py` | Fixture with docstrings and all symbol kinds |
| `internal/parser/python/python_test.go` | 6 tests including docstring-specific assertions |

### Verification

```
✅ CGO_ENABLED=1 go build ./...
✅ CGO_ENABLED=1 go vet ./...
✅ CGO_ENABLED=1 go test ./internal/parser/typescript/...  (6 tests, 7 subtests)
✅ CGO_ENABLED=1 go test ./internal/parser/javascript/...  (6 tests)
✅ CGO_ENABLED=1 go test ./internal/parser/python/...      (6 tests)
✅ CGO_ENABLED=1 go test ./...                             (all packages pass)
```

### Notes

- TypeScript class names use `type_identifier` node; JavaScript uses `identifier` — separate queries required
- Python `function_definition` serves both functions and methods; distinguished by parent-walk: `function_definition → block → class_definition`
- Python docstrings: first `expression_statement(string)` in the body block, `string_content` child strips quotes
- Python calls use `call` node (not `call_expression`); attribute calls use `attribute.attribute` field
- `enclosingClassName` uses `node.Parent()` tree walk — no separate class-range index needed
- `go-tree-sitter` promoted from indirect to direct dependency after new imports

---

## M5 — Indexing Pipeline ✅

**Goal:** `sigil index .` — first end-to-end vertical slice.

**Completed:** 2026-03-10

### Files Created

| File | Purpose |
|---|---|
| `internal/service/session.go` | `NewSessionID()` — crypto/rand, "s_" + 8 hex chars |
| `internal/service/indexer.go` | `Indexer`: Walker→Filter→Parser→Store pipeline |
| `internal/service/syncer.go` | `Syncer`: incremental wrapper + `StagedFiles()` helper |
| `internal/cli/index.go` | `sigil index [path] [--force]` command |
| `internal/cli/sync.go` | `sigil sync [path]` command |

### Bug Fixed (M4 carry-over)

- `GoParser.extractConsts` and `extractVars` were matching function-local const/var
  declarations, producing duplicate qualified names across files in the same package.
  Fixed by checking `defNode.Parent().Type() == "source_file"` before extracting.

### Verification

```
✅ CGO_ENABLED=1 go build ./...
✅ CGO_ENABLED=1 go vet ./...
✅ CGO_ENABLED=1 go test ./...   (all packages pass)
✅ sigil index .                  → 97 files, 409 symbols, 0 warnings
✅ sigil index .  (second run)   → 96 skipped, 1 re-indexed, 37ms (incremental)
✅ sigil index . --force         → full rebuild
```

### Notes

- `pkgPath` passed to parsers = repo-relative directory of the file (e.g. `internal/parser/golang`)
  NOT `FindPackageRoot()`. `FindPackageRoot()` → `symbol.PackageRoot` (manifest ancestor).
- `symbol.ContentHash` = `hex(SHA256(src[byte_start:byte_end]))[:16]` (16 hex chars)
- File mirror: `~/.sigil/{hash}/files/{blobSHA}` — SHA256(content) used as blobSHA in filesystem mode
- Callee resolution: best-effort single-match by stripping package qualifier from RawExpression

---

## M6 — CLI Query Commands ⬜

**Goal:** All query subcommands operational.

### Files Created

`internal/service/{searcher,retriever,deps,outline,tree,overview,env,differ,cache,savings,context}.go`
`internal/cli/{search,get,deps,outline,tree,overview,status,env,diff,cache,savings,hook}.go`

---

## M7 — Enrichment Pipeline ⬜

**Goal:** LLM-based summaries with graceful fallback. Runs parallel with M6.

### Files Created

`internal/enrichment/{enricher,provider,anthropic,google,openai,template,detect,batch}.go`

---

## M8 — MCP Server ✅

**Goal:** `sigil-mcp` stdio server with 9 tools registered via official go-sdk.

### New Dependency

```
github.com/modelcontextprotocol/go-sdk  v1.2.0
```

### Files Created

| File | Purpose |
|---|---|
| `internal/mcpserver/server.go` | NewServer(), tool registration, Run() |
| `internal/mcpserver/tools.go` | 9 typed input structs with jsonschema tags |
| `internal/mcpserver/handlers.go` | Handlers: input → service → result |
| `internal/mcpserver/resolve.go` | Repo resolution from path parameter |
| `internal/mcpserver/session.go` | MCP session + Sigil session ID |

### Tools

| Tool | Handler | Service method |
|---|---|---|
| `sigil_search` | `handleSearch` | `service.Searcher.Search()` |
| `sigil_get` | `handleGet` | `service.Retriever.Get()` |
| `sigil_deps` | `handleDeps` | `service.Deps.Trace()` |
| `sigil_outline` | `handleOutline` | `service.Outline.ForFile()` |
| `sigil_tree` | `handleTree` | `service.Tree.Build()` |
| `sigil_overview` | `handleOverview` | `service.Overview.Summary()` |
| `sigil_env` | `handleEnv` | `service.Env.Inspect()` |
| `sigil_diff` | `handleDiff` | `service.Differ.Diff()` |
| `sigil_status` | `handleStatus` | `service.Status.Check()` |

### Verification Plan

```bash
# Start server, pipe JSON-RPC
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | ./sigil-mcp

# Configure in Claude Code:
# {"mcpServers": {"sigil": {"command": "path/to/sigil-mcp"}}}
```

---

## Token Efficiency Notes

> Track observations that help optimize future sessions.

- **Session cost:** Reading this PROGRESS.md + CLAUDE.md ≈ 500 tokens vs reading all source ≈ 8,000+ tokens
- **Design doc:** `SIGIL_FRAMEWORK_DRAFT_v04.md` is 1,261 lines — read only when schema/tool spec details are needed; CLAUDE.md covers the essentials
- **Model:** claude-sonnet-4-6 (switched from opus for implementation — more token-efficient for code generation)
- **Verification pattern:** `go build ./... && go vet ./...` after each milestone before continuing
