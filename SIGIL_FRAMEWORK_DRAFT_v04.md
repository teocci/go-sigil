# Sigil — Token-Efficient Codebase Intelligence Framework
> Draft v0.4 — Final Design Document

---

## 0. The Problem We Are Solving

Every AI coding agent today reads code the same expensive way:

```
agent thinks: "I need to understand AuthService.validateToken()"
→ open src/auth/service.ts          [~800 tokens]
→ skim 600 irrelevant lines
→ open src/auth/types.ts            [~400 tokens]
→ open src/auth/middleware.ts       [~550 tokens]
→ maybe find the function
→ total: ~1,750 tokens to read ~60 useful tokens
```

This is a **30:1 waste ratio**. Across a 200-file repository, a single "understand this
codebase" session costs 80,000–500,000 tokens — most of it noise.

The same problem compounds for commit descriptions, refactor planning, dependency tracing,
and onboarding. Agents re-run `git log`, `git diff`, `find`, `grep`, and `cat` on every
session because nothing is persisted in a queryable, structured form.

**Sigil's thesis**: index once, retrieve with surgical precision, pay only for what you need.

---

## 1. Design Goals

| Goal | Metric |
|---|---|
| Token reduction | ≥ 70% fewer tokens for symbol-level tasks |
| Retrieval latency | < 5ms per symbol; < 20ms for 50-symbol batch |
| Index build time | < 30s for a 100k-LOC repo |
| Cross-agent compatibility | Claude Code, Gemini CLI, OpenAI Codex, Cursor, Aider |
| Security | No secrets, no binary data, traversal-safe |
| Portability | Single binary, works offline, no daemon required |
| Observability | Cumulative token savings persisted per repo and globally |

---

## 2. Core Concept — The Symbol

A **Symbol** is the atomic unit of Sigil. Every function, class, method, interface, constant,
and type becomes a Symbol at index time.

```jsonc
{
  // Identity
  "id": "a3f9c1b2",                          // stable 8-char hash of qualified_name
  "kind": "method",                           // function | class | method | interface | type | const | var
  "name": "validateToken",
  "qualified_name": "auth/service.AuthService.validateToken",
  "language": "typescript",
  "package_root": "services/auth",            // detected from package manifest, null if unknown

  // Navigation — the core innovation
  "file": "src/auth/service.ts",
  "byte_start": 14823,                        // O(1) seek target
  "byte_end": 15210,
  "line_start": 412,
  "line_end": 428,

  // Lightweight summary
  "signature": "validateToken(token: string, opts?: TokenOptions): Promise<User>",
  "summary": "Verifies JWT signature, expiry, and revocation; returns hydrated User or throws AuthError",
  "tags": ["exported", "async"],

  // Graph edges — populated at query time from call_edges table, never stored on symbol
  "calls": [
    { "id": "b2e1a0f3", "confidence": "static",   "summary": "..." },
    { "id": null,        "confidence": "dynamic",  "raw_expression": "handler.execute" }
  ],
  "called_by": [
    { "id": "f0a2b3c4", "confidence": "static", "summary": "..." }
  ],
  "imports": ["User", "TokenOptions", "AuthError"],

  // Hierarchy
  "parent_id": "d7c3b1a0",
  "children": ["e8f2a3b1", "f0c4d2e3"],
  "depth": 2,

  // Audit
  "content_hash": "sha256:9f3c...",           // parse-time audit and corruption detection ONLY
  "possible_unresolved": false,               // true when a dependency was renamed/deleted
  "untracked": false,
  "embedding": null,                          // float32 BLOB, populated by --embed flag (Phase 2)
  "indexed_at": "2025-09-14T10:22:00Z"
}
```

### Why byte offsets matter

With `byte_start` and `byte_end`, retrieval is:

```
1. SQLite:  WHERE id = 'a3f9c1b2'  →  {file, byte_start, byte_end}   O(log n) ~0.1ms
2. OS:      fd.ReadAt(buf, byte_start)  →  fd.Read(size)              O(1)     ~0.3ms
```

No line counting. No regex scanning. No file-length traversal. Direct byte addressing —
the same technique used by database storage engines and git's packfile format.

`fd.ReadAt` is used throughout (not `fd.Seek` + `fd.Read`) — it is goroutine-safe and
does not move the shared file cursor, making concurrent batch retrieval safe.

---

## 3. Local Git Repository Indexing

### 3.1 Git version detection and command selection

Sigil checks the git version once at startup, caches it for the session:

```
git installed?          NO  → filesystem mode (§3.4)
git version ≥ 2.25?     NO  → filesystem mode + one-time version warning
git repo exists?        NO  → filesystem mode + one-time warning
                        YES → git mode (version-appropriate command)
```

**Minimum supported git version: 2.25** (Ubuntu 20.04 era).

| Git version | Command |
|---|---|
| ≥ 2.36 | `git ls-files --format='%(objectname) %(path)'` |
| 2.25 – 2.35 | `git ls-files -s` (blob SHA = field 2, path = field 4) |
| < 2.25 | filesystem mode |

### 3.2 Discovery walk (git mode)

```bash
sigil index .
```

1. Reads `.gitignore` (all levels) + `.glyphignore` + built-in exclusions
2. Runs version-appropriate `git ls-files` — one invocation, full tracked file list
   with blob SHAs
3. Blob SHA is the **sole file-level sync trigger** (see §3.3)
4. Detects package boundaries via manifest files (`package.json`, `go.mod`,
   `pyproject.toml`, `Cargo.toml`, `pom.xml`) → populates `package_root` on symbols
5. Untracked files: `git ls-files --others` — indexed with `mtime + content_hash`,
   tagged `untracked = 1`, excluded from `sigil_diff`
6. On completion: reads `git rev-parse HEAD` → stores in `meta.json` as
   `last_indexed_commit`

**Built-in exclusion list (always skipped):**
```
node_modules/   vendor/   .git/   __pycache__/   target/   dist/   build/
*.min.js        *.min.css *.lock  *.map          *.wasm
```

### 3.3 Sync strategy — blob SHA comparison

`sigil sync` uses `git ls-files` exclusively. **`git diff` is never used for sync** —
it is reserved solely for the `sigil_diff` MCP tool.

```
On sigil sync:

1. Run git ls-files (version-appropriate command)
   → current blob SHA per tracked file

2. Compare against blob_sha stored in files table

3. Four cases:
   SHA matches stored   → skip entirely
   SHA differs          → full file reparse (DELETE WHERE file = ? + re-insert)
   File not in DB       → new file, parse and insert
   File in DB but gone  → deleted, DELETE WHERE file = ?

4. Always update last_indexed_commit = git rev-parse HEAD
   (even if zero files changed — handles rebase/force-push correctly)
```

**Pre-delete edge marking:**
Before `DELETE FROM symbols WHERE file = ?`, run:
```sql
SELECT DISTINCT caller_id FROM call_edges
WHERE callee_id IN (SELECT id FROM symbols WHERE file = ?);
```
Those callers are flagged `possible_unresolved = 1`. Flag clears on next reparse of
those files.

**Rule:** a changed blob SHA always triggers full reparse of that file. All symbols
for that file are deleted and re-inserted wholesale — no per-symbol merging, no
partial updates. This guarantees byte offsets are always fresh.

### 3.4 Filesystem fallback mode (no git)

When git is unavailable, too old, or the directory is not a git repo:

```
mtime matches AND size matches  → skip (no file read)
mtime differs  OR size differs  → read file → recompute content_hash
                                   hash matches stored → update mtime/size, skip reparse
                                   hash differs        → full reparse
```

**One-time user message (never repeated):**
```bash
⚠  Git not detected in ./my-project
   Sigil will use filesystem mode (mtime + size tracking).
   For faster, more accurate sync: git init && git add .
   Indexing in filesystem mode...
```

### 3.5 Git rebase / history rewrite

Rebase rewrites commit SHAs but typically does not change file blob SHAs.
Since sync compares blob SHAs only, a rebase that doesn't change file contents
produces zero unnecessary reparsing — correct behavior automatically.

`last_indexed_commit` is always updated at end of sync (§3.3 step 4).

`sigil status` validates the stored commit before display:
```bash
Last indexed commit: a1b2c3d  ← valid
Last indexed commit: a1b2c3d (⚠ no longer in history — run sigil sync)  ← ghost
Index content:       current (all blob SHAs match)   ← always shown separately
```

`sigil_diff` returns a clear error when the stored commit is a ghost:
```json
{
  "error": "stale_commit_reference",
  "message": "Last indexed commit no longer exists (likely rebased). Run sigil sync.",
  "recoverable": true
}
```

### 3.6 Shallow git clones

`git ls-files` works on shallow clones — blob SHAs are present regardless of history
depth. Sync works normally. `sigil_diff` against a ref truncated by the shallow clone
returns a clear error message rather than crashing.

### 3.7 Remote mode (GitHub API)

```bash
sigil index --remote owner/repo --token $GITHUB_TOKEN
```

Single API call: `GET /repos/{owner}/{repo}/git/trees/{sha}?recursive=1` returns the
full file tree with blob SHAs. Only changed/new blobs are fetched and re-parsed.
Same blob SHA comparison logic as §3.3.

---

## 4. Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                          SIGIL FRAMEWORK                             │
│                                                                      │
│  ┌──────────┐   ┌──────────┐   ┌────────────┐   ┌────────────────┐  │
│  │Discovery │──▶│ Security │──▶│   Parser   │──▶│    Indexer     │  │
│  │  Walker  │   │  Filter  │   │(tree-sitter)│  │   (SQLite)     │  │
│  └──────────┘   └──────────┘   └────────────┘   └────────────────┘  │
│        │                             │                  │            │
│  git ls-files / mtime         Enrichment LLM            │            │
│                               (optional, async)         ▼            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                        SYMBOL STORE                             │  │
│  │   ~/.sigil/{repo_hash}/index.db        SQLite WAL mode         │  │
│  │   ~/.sigil/{repo_hash}/files/          raw source mirror       │  │
│  │   ~/.sigil/{repo_hash}/meta.json       repo-level metadata     │  │
│  │   ~/.sigil/repos.json                  global repo manifest    │  │
│  │   ~/.sigil/tokens_saved.json           global savings rollup   │  │
│  │   ~/.sigil/config.toml                 user configuration      │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│          ┌────────────────────┼─────────────────┐                   │
│          ▼                    ▼                  ▼                   │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────┐     │
│  │  CLI (sigil) │   │  MCP Server  │   │  HTTP/JSON API       │     │
│  └──────────────┘   └──────────────┘   │  (Phase 2+, local)   │     │
│                                        └──────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 5. Pipeline — Five Stages

### Stage 1: Discovery (see §3)

### Stage 2: Security Filtering

Every file passes a mandatory security gate before parsing.

**Three-tier security model:**

| Tier | Trigger | Behavior | `sigil_env` access |
|---|---|---|---|
| **Redacted** | Built-in secret file patterns (default) | Keys visible, values nulled, appears in `sigil_tree` | ✓ Live read, state classification only |
| **Fully ignored** | `extra_ignore_filenames` in config | Completely invisible to all Sigil tools | ✗ Returns `exists: false` with explanation |
| **Normal** | Everything else | Full symbol extraction | N/A |

**Built-in redacted files (cannot be removed via config):**
`.env`, `*.env.*`, `*.pem`, `*.key`, `*.p12`, `id_rsa*`, `credentials.*`, `.netrc`

**Additional security checks:**

| Check | Method | On Failure |
|---|---|---|
| Path traversal | Canonicalize path, verify under repo root | Skip + warn |
| Symlink escape | `filepath.EvalSymlinks`, verify target under root | Skip + warn |
| Binary detection | First 8KB: > 0.1% null bytes or non-UTF-8 | Skip silently |
| File size | Default 2MB, configurable | Skip + warn |
| Extension allow-list | Per-language known source extensions | Skip silently |
| Secret value patterns | Regex on constant values: API keys, private keys, DSNs | Index symbol, redact value |
| `.gitignore` | Full spec parsing at every directory level | Skip silently |
| `.glyphignore` | User-defined exclusions (gitignore syntax) | Skip silently |

**Secret value redaction example:**
```json
{
  "kind": "const",
  "name": "DB_PASSWORD",
  "summary": "[REDACTED — matched secret pattern: password assignment]",
  "byte_start": null,
  "byte_end": null,
  "tags": ["redacted"]
}
```

**`~/.sigil/config.toml` security section:**
```toml
[security]
max_file_size_bytes    = 2_097_152
binary_null_threshold  = 0.001

# Built-in secret files are redacted by default (listed for reference, cannot be removed)
# .env, *.env.*, *.pem, *.key, *.p12, id_rsa*, credentials.*, .netrc

# Extra patterns added to redaction tier (keys visible, values nulled, sigil_env has access)
extra_secret_filenames = []

# Extra patterns added to fully-invisible tier (sigil_env returns exists: false)
extra_ignore_filenames = []

# Extra regex patterns applied to constant values
extra_secret_patterns  = []
```

### Stage 3: Parsing — tree-sitter AST Extraction

tree-sitter: fast, incremental, error-tolerant, consistent C API with mature Go bindings.

**Supported languages:**

| Phase | Languages |
|---|---|
| Phase 1 | TypeScript, JavaScript, Python, Go |
| Phase 2 | Rust, Java, C, C++, Ruby, PHP, protobuf, SQL |
| Phase 3 | YAML, TOML, GraphQL, shell |

**Unsupported files:** appear in `sigil_tree` with `0 symbols · unsupported`. Raw content
retrievable via `sigil_get` using file path (not symbol ID). Agents always have a complete
picture of repo structure regardless of language support status.

**Summary generation priority:**
1. Docstring / leading comment immediately preceding symbol → extract first sentence
2. Enrichment LLM (if configured) → semantic summary generated once, cached until
   `content_hash` changes
3. Template fallback → `"[kind] [name] accepting [params], returns [type]"`

**Enrichment provider priority chain:**
```
1. ANTHROPIC_API_KEY set?   → Claude Haiku (via ANTHROPIC_BASE_URL if set)
2. GOOGLE_API_KEY set?      → Gemini Flash (via GOOGLE_MODEL if set)
3. OPENAI_API_BASE set?     → OpenAI-compatible endpoint
4. Ollama at :11434?        → auto-detect, use OPENAI_MODEL default
5. None available           → template summaries, log info message
```

Enrichment is **default-on** when any provider is detected. The `--no-enrich` flag
disables it. Enrichment never blocks indexing — if the provider fails, Sigil falls
back to templates gracefully.

**Call graph extraction — single pass:**
tree-sitter identifies call expressions within each symbol's body. For each detected call:
```sql
INSERT OR IGNORE INTO call_edges (caller_id, callee_id, raw_expression, confidence)
VALUES (current_id, resolved_id, raw_expr, 'static'|'inferred'|'dynamic');
```

**Call graph confidence levels:**

| Pattern | Language | Confidence |
|---|---|---|
| Direct call to known symbol | All | `static` |
| Method call on typed variable | TypeScript, Java, Go, Rust | `static` |
| Method call with type hints | Python (typed) | `inferred` |
| Decorator call | Python | `inferred` |
| Method call on untyped variable | Python, JS, Ruby | `dynamic` |
| Callback / higher-order function | JS, Python | `dynamic` |
| `__getattr__` / `method_missing` | Python, Ruby | `dynamic` |

Dynamic edges: `callee_id = null`, `raw_expression` = call site text.
`calls` and `called_by` are never stored on the symbol row — hydrated at query time
from `call_edges`.

### Stage 4: Storage — SQLite + Raw File Mirror

**Schema version: 1** (`PRAGMA user_version = 1`)

```sql
PRAGMA foreign_keys = ON;

-- File-level change detection
CREATE TABLE files (
  path         TEXT PRIMARY KEY,
  blob_sha     TEXT,              -- NULL in filesystem mode
  mtime        TEXT,              -- filesystem mode
  size         INTEGER,           -- filesystem mode
  last_indexed TEXT NOT NULL
);

-- Symbol store
CREATE TABLE symbols (
  id                  TEXT PRIMARY KEY,      -- SHA256(qualified_name)[:8]
  kind                TEXT NOT NULL,         -- function|class|method|interface|type|const|var
  name                TEXT NOT NULL,
  qualified_name      TEXT NOT NULL,         -- full repo-relative path + symbol
  language            TEXT NOT NULL,
  file                TEXT NOT NULL REFERENCES files(path),
  package_root        TEXT,                  -- "services/auth" | null
  byte_start          INTEGER,               -- NULL if redacted
  byte_end            INTEGER,
  line_start          INTEGER,
  line_end            INTEGER,
  signature           TEXT,
  summary             TEXT,
  tags                TEXT,                  -- JSON array
  parent_id           TEXT REFERENCES symbols(id),
  children            TEXT,                  -- JSON array of symbol IDs
  depth               INTEGER DEFAULT 0,
  imports             TEXT,                  -- JSON array of type names
  content_hash        TEXT NOT NULL,         -- audit/corruption detection ONLY, not invalidation
  possible_unresolved INTEGER DEFAULT 0,     -- set when a dependency is deleted/renamed
  untracked           INTEGER DEFAULT 0,
  embedding           BLOB,                  -- float32 embeddings, NULL until --embed run
  indexed_at          TEXT NOT NULL
);

-- Call graph (normalized — replaces calls/called_by arrays on symbol row)
CREATE TABLE call_edges (
  caller_id       TEXT NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
  callee_id       TEXT,                      -- NULL for dynamic/unresolvable edges
  raw_expression  TEXT,                      -- populated for dynamic edges
  confidence      TEXT NOT NULL DEFAULT 'static',  -- 'static'|'inferred'|'dynamic'
  PRIMARY KEY (caller_id, COALESCE(callee_id, raw_expression))
);

-- Full-text search
CREATE VIRTUAL TABLE symbols_fts USING fts5(
  name, qualified_name, signature, summary,
  content=symbols, content_rowid=rowid
);

-- Savings ledger (append-only)
CREATE TABLE savings_log (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   TEXT NOT NULL,
  tool_name    TEXT NOT NULL,
  call_at      TEXT NOT NULL,
  timing_ms    REAL,
  tokens_saved INTEGER
);

-- Indexes
CREATE INDEX idx_symbols_name       ON symbols(name);
CREATE INDEX idx_symbols_file       ON symbols(file);
CREATE INDEX idx_symbols_kind       ON symbols(kind);
CREATE INDEX idx_symbols_parent     ON symbols(parent_id);
CREATE INDEX idx_symbols_pkg        ON symbols(package_root);
CREATE INDEX idx_edges_caller       ON call_edges(caller_id);
CREATE INDEX idx_edges_callee       ON call_edges(callee_id);
CREATE INDEX idx_edges_confidence   ON call_edges(confidence);
CREATE INDEX idx_savings_session    ON savings_log(session_id);
```

**`content_hash` role — explicitly stated:**
Computed from `file[byte_start:byte_end]` at parse time. Stored as an audit field.
Never used for invalidation (blob SHA at the file level handles that). Used by
`sigil status --verify` to detect byte offset corruption:
```bash
$ sigil status --verify
✓ 308 symbols verified
⚠  4 symbols have mismatched content_hash
   Run: sigil index . --force
```

**Schema migration:**
```go
const (
    SchemaVersion  = 1
    MinSafeVersion = 1
)
```
Migrations run sequentially, `PRAGMA user_version` updated after each step.
All Phase 1 migrations are additive only. Destructive changes deferred to v1.0.

**Raw file mirror:**
`~/.sigil/{repo_hash}/files/{blob_sha}` — content-addressed, deduplicated.
Same file at two paths stored once.

**Storage layout:**
```
~/.sigil/
  repos.json                     ← global repo manifest
  tokens_saved.json              ← global savings rollup
  config.toml                    ← user configuration
  {repo_hash}/                   ← SHA256(canonical_repo_path)[:12]
    index.db                     ← SQLite (WAL mode, foreign_keys ON)
    meta.json                    ← {hash, path, name, mode, initialized_at, last_indexed_commit}
    files/
      {blob_sha}                 ← raw source, content-addressed
```

**`repos.json` — global manifest:**
```json
{
  "repos": [
    {
      "hash": "a3f9c1b2",
      "path": "/home/user/projects/my-api",
      "name": "my-api",
      "initialized_at": "2025-09-14T10:00:00Z",
      "last_sync": "2025-09-14T10:22:00Z"
    }
  ]
}
```

### Stage 5: Retrieval — O(1) Byte-Offset Seeking

```
1. SQLite:  SELECT file, byte_start, byte_end WHERE id = 'a3f9c1b2'   ~0.1ms
2. OS:      fd.ReadAt(buf, 14823) → read 387 bytes                     ~0.3ms O(1)
3. Return:  387 bytes of exact source, zero padding
```

**`include_context_lines` implementation:**
Context after: seek to `byte_end`, scan forward N newlines — trivial.
Context before: bounded backward scan from `byte_start` using `ReadAt` in 512-byte
chunks, counting newlines. Maximum 50 lines. `ReadAt` used throughout (not `Seek`) —
goroutine-safe for concurrent batch retrieval. Context returned inline as one
contiguous string.

---

## 6. MCP Server — Global Always-On

Configured **once**, globally, serves all indexed repos automatically.
No restart needed when switching projects.

```json
{
  "mcpServers": {
    "sigil": {
      "command": "sigil",
      "args": ["mcp"]
    }
  }
}
```

**Repo selection:** inferred from the `path` parameter in each tool call. Sigil walks
up from that path to find the git root, computes `repo_hash`, selects the correct index.
Falls back to server startup CWD if `path` is omitted.

**Session ID:** auto-generated on process start (stdio) or first call per connection
(HTTP). Never passed by the caller. HTTP session boundaries are approximations —
documented as such, directionally correct for optimization analysis.

**Not initialized response:**
```json
{
  "error": "not_initialized",
  "message": "This repo has not been indexed. Run: sigil index .",
  "repo": "/home/user/projects/new-project"
}
```

---

## 7. MCP Server — Complete Tool Catalog

All tools include a `path` parameter (optional — falls back to server CWD).
All tools return a `metadata` envelope.

### `sigil_search`
Find symbols by name, pattern, or FTS. Returns signatures and summaries only — no source.
Optional `semantic: true` for vector search (Phase 2, requires `--embed`).
```json
{
  "query": "validateToken",
  "kind": "method",
  "language": "typescript",
  "semantic": false,
  "limit": 10,
  "path": "/home/user/projects/my-api"
}
```
Token cost: ~15 tokens/result × 10 = **~150 tokens**.

---

### `sigil_get` — Single, Batch, and Raw File Retrieval
```json
{
  "ids":                  ["a3f9c1b2", "b2e1a0f3"],
  "files":                ["proto/auth.proto"],
  "include_context_lines": 3,
  "path":                 "/home/user/projects/my-api"
}
```
`ids`: symbol IDs — returns parsed symbols with source.
`files`: file paths — returns raw content for unsupported file types.
`include_context_lines`: 0–50 (default 0). Context returned inline, one contiguous string.

Response:
```jsonc
{
  "symbols": [
    {
      "id": "a3f9c1b2",
      "qualified_name": "auth/service.AuthService.validateToken",
      "file": "src/auth/service.ts",
      "line_start": 412,
      "line_end": 428,
      "context_before_lines": 3,
      "context_after_lines": 3,
      "source": "...context before...\n...symbol body...\n...context after...",
      "possible_unresolved": false
    }
  ],
  "files": [
    {
      "path": "proto/auth.proto",
      "content": "syntax = \"proto3\";\n...",
      "language": "protobuf",
      "supported": false,
      "size_bytes": 1243
    }
  ],
  "metadata": {
    "timing_ms": 4.3,
    "tokens_saved": 1640,
    "session": {
      "id": "s_a3f9c1b2",
      "tokens_saved": 12480,
      "call_count": 8,
      "started_at": "2025-09-14T10:18:00Z"
    },
    "repo_total": {
      "tokens_saved": 284930,
      "call_count": 847
    }
  }
}
```

---

### `sigil_deps`
Traverse the call graph. Returns summaries with confidence levels — never source.
```json
{
  "id": "a3f9c1b2",
  "direction": "both",
  "depth": 2,
  "path": "/home/user/projects/my-api"
}
```
Response:
```json
{
  "calls": [
    { "id": "b2e1a0f3", "confidence": "static",  "qualified_name": "...", "summary": "..." },
    { "id": null,        "confidence": "dynamic", "raw_expression": "handler.execute" }
  ],
  "called_by": [...],
  "metadata": {
    "has_dynamic_edges": true,
    "possible_unresolved": false,
    "note": "Dynamic dispatch detected — call graph may be incomplete for Python/JS targets."
  }
}
```

---

### `sigil_outline`
All symbols in a file as a structured hierarchy. No source, zero filler tokens.
```json
{ "file": "src/auth/service.ts", "path": "/home/user/projects/my-api" }
```
```
AuthService (class) — Handles JWT-based authentication
  ├── constructor(config: AuthConfig) — Initializes token store
  ├── validateToken(token, opts?) → Promise<User> — Verifies JWT and returns User
  ├── revokeToken(jti: string) → void — Adds JTI to revocation set
  └── _parseHeader(token) → JWTHeader — [private] Decodes JWT header
```

---

### `sigil_tree`
Repository file structure — files, symbol counts, languages, unsupported files clearly marked.
```json
{
  "scope": "src/",
  "depth": 2,
  "include_symbol_counts": true,
  "path": "/home/user/projects/my-api"
}
```
```
src/
  auth/                    3 files · 18 symbols · typescript
    service.ts             8 symbols
    middleware.ts          6 symbols
    types.ts               4 symbols
proto/                     2 files · 0 symbols · unsupported
  auth.proto               unsupported · 1.2KB
  payments.proto           unsupported · 0.8KB
database/migrations/       4 files · 0 symbols · unsupported
scripts/                   2 files · 0 symbols · unsupported
```
Token cost: **~80 tokens** for a 50-file repo.

---

### `sigil_overview`
High-level repository summary. **First call at every session start.**
```json
{ "path": "/home/user/projects/my-api" }
```
```json
{
  "repo": "my-api",
  "languages": [
    { "name": "TypeScript", "files": 48, "symbols": 312 },
    { "name": "Go",         "files": 3,  "symbols": 21  }
  ],
  "packages": [
    { "root": "services/auth",     "symbols": 118 },
    { "root": "services/payments", "symbols": 94  }
  ],
  "top_level_modules": ["src/auth", "src/api", "src/db", "src/utils"],
  "entry_points": ["src/index.ts", "src/server.ts"],
  "total_symbols": 333,
  "index_age_seconds": 412,
  "last_indexed_commit": "a1b2c3d",
  "possible_unresolved_count": 0
}
```
Token cost: **~60 tokens**.

---

### `sigil_env`
Environment variable configuration state. **Never returns values.** First call for
any connection/authentication debugging task.
```json
{ "path": "/home/user/projects/my-api" }
```
```json
{
  "env_files": [
    { "path": ".env",          "exists": true  },
    { "path": ".env.template", "exists": true  }
  ],
  "variables": [
    { "key": "DATABASE_URL",      "state": "set"         },
    { "key": "ANTHROPIC_API_KEY", "state": "placeholder", "note": "value matches placeholder pattern" },
    { "key": "STRIPE_SECRET_KEY", "state": "empty"       },
    { "key": "SENDGRID_API_KEY",  "state": "unset",       "note": "missing from .env" }
  ],
  "warnings": [
    "ANTHROPIC_API_KEY is a placeholder — likely causing authentication failures",
    "STRIPE_SECRET_KEY is empty — dependent features will fail",
    "SENDGRID_API_KEY is missing from .env"
  ]
}
```

**Variable states:** `set` · `empty` · `placeholder` · `unset` · `missing`

**Placeholder detection patterns:** `^x+$`, `YOUR_.*`, `<[^>]+>`, `TODO`, `changeme`,
`^dummy`, `^fake`, `^example`, `^0+$`, `^\*+$`, `REPLACE_.*`

**Explicitly ignored file response:**
```json
{
  "path": ".env",
  "exists": false,
  "reason": "explicitly ignored by config — remove from extra_ignore_filenames to inspect"
}
```

---

### `sigil_diff`
Symbol-level diff since a git ref. Uses `git diff <ref>` internally — the **only**
place `git diff` is used in the entire system.
```json
{
  "since": "HEAD~1",
  "include_source": false,
  "path": "/home/user/projects/my-api"
}
```
Response groups changes: `added`, `modified_signature`, `modified_body`, `deleted`.

---

### `sigil_status`
Index health, last indexed commit (validated), symbol counts, possible_unresolved count,
session savings summary.
```json
{ "path": "/home/user/projects/my-api" }
```

---

## 8. Token Savings Observability

Every MCP response includes a `metadata` envelope. Savings use **exact measurement**
where possible, static coefficients as fallback.

### 8.1 Baseline calculation per tool

| Tool | Formula |
|---|---|
| `sigil_get` (symbol) | `(file_size_bytes - symbol_size_bytes) / 4` |
| `sigil_get` (raw file) | `0` (agent requested the file directly — no savings) |
| `sigil_outline` | `(file_size_bytes - outline_text_bytes) / 4` |
| `sigil_diff` | `(git_diff_byte_size - symbol_diff_bytes) / 4` |
| `sigil_search` | static coefficient, benchmark-calibrated |
| `sigil_overview` | static coefficient, benchmark-calibrated |
| `sigil_tree` | static coefficient, benchmark-calibrated |
| `sigil_env` | static coefficient, benchmark-calibrated |

No pricing calculations. `tokens_saved` is the only metric — objective, model-agnostic,
never stale.

### 8.2 Persistent savings ledger

`savings_log` table (SQLite, append-only) rolls up to `~/.sigil/tokens_saved.json`:

```json
{
  "schema_version": 1,
  "updated_at": "2025-09-14T10:22:00Z",
  "repos": {
    "a3f9c1b2": {
      "path": "/home/user/projects/my-api",
      "tokens_saved": 284930,
      "call_count": 847
    }
  },
  "global": {
    "tokens_saved": 1280837,
    "call_count": 3921
  }
}
```

### 8.3 Session granularity

Session IDs enable optimization analysis across workflow types:

```bash
sigil savings --sessions          # list all sessions with totals
sigil savings --session s_a3f9    # drill into one session
sigil savings --top 10            # highest-impact sessions
```

Insight example: *"SIGIL_COMMIT sessions save 3× more tokens per call than
SIGIL_EXPLORE sessions"* → invest in improving SIGIL_COMMIT skill first.

---

## 9. Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `GITHUB_API_TOKEN` | GitHub API auth for remote indexing | — |
| `ANTHROPIC_API_KEY` | Enrichment via Claude Haiku **(priority 1)** | — |
| `ANTHROPIC_BASE_URL` | Override Anthropic endpoint (z.ai, Bedrock, etc.) | `https://api.anthropic.com` |
| `ANTHROPIC_MODEL` | Model for Anthropic enrichment | `claude-haiku-4-5-20251001` |
| `GOOGLE_API_KEY` | Enrichment via Gemini Flash **(priority 2)** | — |
| `GOOGLE_MODEL` | Model for Google enrichment | `gemini-2.0-flash` |
| `OPENAI_API_BASE` | OpenAI-compatible endpoint **(priority 3)** | — |
| `OPENAI_API_KEY` | Key for OpenAI-compatible endpoint | `local-llm` |
| `OPENAI_MODEL` | Model for OpenAI-compatible endpoint | `qwen2.5-coder:7b` |
| `OPENAI_TIMEOUT` | Timeout in seconds | `60` |
| `CODE_INDEX_PATH` | Custom cache root | `~/.sigil/` / `%APPDATA%\sigil\` |
| `SIGIL_MAX_INDEX_FILES` | Max files to index per repo | `500` |
| `SIGIL_ENRICH_BATCH_SIZE` | Concurrent enrichment requests | `4` |
| `SIGIL_LOG_LEVEL` | `DEBUG` `INFO` `WARNING` `ERROR` | `WARNING` |
| `SIGIL_LOG_FILE` | Log file path. **Must be set in MCP stdio mode** to avoid corrupting the MCP JSON stream. | stderr |
| `SIGIL_SERVER_TOKEN` | Shared auth token for HTTP MCP server (Phase 3) | — |

---

## 10. CLI Interface

```bash
# Indexing (smart: full on first run, incremental on subsequent)
sigil index .
sigil index ./my-project
sigil index . --force                              # always full rebuild
sigil index . --enrich                             # force enrichment even if provider not auto-detected
sigil index . --no-enrich                          # skip enrichment
sigil index . --embed                              # generate vector embeddings (Phase 2)
sigil index . --force src/auth/service.ts          # force specific files only
sigil index --remote owner/repo --token $GITHUB_API_TOKEN

# Incremental update (used by git hook)
sigil sync

# Symbol operations
sigil search "validateToken" --kind method
sigil search "JWT verification" --semantic         # Phase 2
sigil get a3f9c1b2
sigil get a3f9c1b2 b2e1a0f3 c9d4e5f1              # batch
sigil deps a3f9c1b2 --direction both --depth 2
sigil diff HEAD~1

# Repository overview
sigil overview
sigil status
sigil status --verify                              # recompute content_hash for all symbols
sigil env                                          # environment variable states

# Cache management
sigil cache invalidate
sigil cache remove
sigil cache clear --all
sigil cache status
sigil cache prune --older-than 30d

# Token savings
sigil savings
sigil savings --repo .
sigil savings --sessions
sigil savings --session s_a3f9c1b2
sigil savings --top 10
sigil savings --format json
sigil savings prune --older-than 90d
sigil savings reset                                # confirmation required

# MCP server
sigil mcp                                          # stdio, global always-on
sigil mcp --http --port 8765                       # HTTP (Phase 2)
sigil mcp --http --port 8765 --read-only           # Phase 3 team sharing

# Git hook
sigil hook install                                 # pre-commit → sigil sync
sigil hook uninstall

# Distribution (Phase 2)
sigil pull-index --ci github --repo owner/repo
sigil pull-index --ci gitlab --repo owner/repo
sigil pull-index --url https://internal/sigil-index.tar.gz

# Pricing (removed — tokens_saved is the only metric)
sigil update-pricing                               # NOT IMPLEMENTED — not needed
```

---

## 11. Security — Built-in Protections Summary

| Protection | Mechanism |
|---|---|
| Path traversal | All paths canonicalized, verified under repo root before access |
| Symlink escape | `filepath.EvalSymlinks`; target must be under root |
| Secret file redaction (default) | `.env`, `*.pem`, `*.key` etc — keys visible, values nulled |
| Secret file exclusion (opt-in) | `extra_ignore_filenames` — fully invisible to all tools |
| Secret value redaction | Regex on constant values; byte range nulled; summary redacted |
| Placeholder detection | `sigil_env` classifies values as set/empty/placeholder/unset |
| Binary detection | First 8KB: null byte ratio + non-UTF-8 ratio threshold |
| File size limit | Configurable (default 2MB) |
| `.gitignore` respect | Full spec parsing at every directory level |
| `.glyphignore` | User-defined exclusions with gitignore syntax |
| No outbound network (default) | Indexing fully local; GitHub mode is explicit opt-in |
| No credential storage | Tokens via env var only, never written to disk |
| Index file permissions | `~/.sigil/` mode `0700`; `index.db` mode `0600` |
| MCP stdio log isolation | `SIGIL_LOG_FILE` must be set — logs never written to stdout |

---

## 12. Use Cases

**Large multi-module repositories** — `sigil_overview` and `sigil_tree` replace expensive
exploration. `sigil_get` batch retrieval replaces opening entire modules.

**Agent-driven refactors** — `sigil_deps(direction: "called_by")` finds every callsite
in one query. No grep across 300 files.

**Architecture exploration** — `sigil_overview` → `sigil_tree` → `sigil_outline` →
`sigil_get`. Structured onboarding without reading anything.

**Faster developer onboarding** — Symbol hierarchy and summaries provide a navigable
map. `sigil_env` surfaces missing or placeholder configuration on day one.

**Environment and configuration debugging** — `sigil_env` identifies placeholder,
empty, and missing variables before the agent wastes tool calls debugging application
code. Ends the `.env.template` infinite loop.

**Token-efficient multi-agent workflows** — Multiple specialized agents sharing a
Sigil index via the HTTP MCP server. Each retrieves only what it needs.

**Commit messages and changelogs** — `sigil_diff("HEAD")` returns structured
symbol-level changes. Writing a precise commit message becomes formatting, not code
reading.

---

## 13. Non-Goals (Phase 1)

**LSP diagnostics or completions** — Sigil provides navigation and summarization,
not type checking, error detection, or autocomplete.

**Editing workflows** — Sigil is read-only. It does not write, patch, or modify
source files.

**Real-time file watching** — No daemon, no `inotify`. Invalidation is explicit via
`sigil sync` or the pre-commit git hook.

**Cross-repository global indexing** — Each repo has its own index. Cross-repo
symbol resolution is Phase 3+ territory.

**Semantic program analysis** — No type inference, data flow, or points-to analysis.
tree-sitter gives syntactic structure, not semantic meaning.

**IDE plugin** — Sigil integrates via MCP, already supported by Claude Code, Cursor,
Cody, and others.

---

## 14. Language — Go 1.23

**Decision: Go 1.23. Final.**

- Design is still evolving — Go allows fast restructuring
- `go-tree-sitter` bindings are mature and production-proven
- `mattn/go-sqlite3` is battle-tested for this exact use case
- Compile times under 10 seconds — iteration speed matters for Phase 1
- Goroutines make parallel file parsing trivial
- `goreleaser` + GitHub Actions: signed binaries for Linux/macOS/Windows/ARM
- GC latency on `fd.ReadAt(387 bytes)` is unmeasurable in practice

**CGO cross-compilation:** tree-sitter requires CGO. Workaround: pre-built
tree-sitter C objects per platform using `zig cc` as cross-compiler — same approach
as `esbuild` and `bun`.

**Future:** if profiling shows the indexer as a bottleneck at monorepo scale (10M+
LOC), a Rust parser/indexer core called from Go via CGO is a viable path. That is a
v1.0+ decision.

---

## 15. Skills for AI Agents

### `SIGIL_EXPLORE`
```
When exploring an unfamiliar codebase:
1. Call sigil_status — confirm index is current
2. Call sigil_overview — orient to languages, packages, entry points
3. Call sigil_tree — understand file/directory structure
4. Call sigil_env — check for placeholder or missing environment variables
5. Call sigil_search with broad queries to map specific areas (summaries only)
6. Call sigil_outline on key files to understand their symbol hierarchy
7. Call sigil_get ONLY for symbols you have confirmed are relevant
8. Use sigil_deps to trace dependencies without reading files
Never open raw files until sigil_get has been tried and is insufficient.
```

### `SIGIL_COMMIT`
```
To write a commit message:
1. Call sigil_diff("HEAD") — get structured symbol-level changes
2. Group: added symbols, modified signatures, body-only changes, deleted symbols
3. Write subject from the most significant change
4. Write body from all symbol-level changes with qualified names
Do NOT run git log, git diff, or read any source files.
```

### `SIGIL_REFACTOR`
```
To plan a rename or refactor:
1. Call sigil_search to find the target symbol
2. Call sigil_deps with direction "called_by" to find all callers
3. Note any possible_unresolved flags — run sigil index . --force if present
4. Call sigil_get (batch) only for symbols that need changing
5. Output a precise change plan with file:line references
```

### `SIGIL_DEBUG_ENV`
```
When debugging a connection, authentication, or API failure:
1. Call sigil_env FIRST — check for placeholder, empty, or missing variables
2. If any variable state is "placeholder" or "empty" — report to user immediately
3. Do NOT proceed to debug application code until environment state is confirmed
4. If .env is explicitly ignored, inform user that sigil_env cannot inspect it
Never attempt to read .env directly — it is redacted by default.
```

---

## 16. Integration Roadmap

> Document versions (v0.1, v0.2...) track design iteration.
> Release phases below are independent.

### Phase 1 — Core
- [ ] Go 1.23 CLI scaffolding (cobra + viper)
- [ ] Git version detection + command selection (§3.1)
- [ ] Local directory walker + full `.gitignore` parser
- [ ] Package boundary detection (`package.json`, `go.mod`, `pyproject.toml`, etc.)
- [ ] Filesystem fallback mode (§3.4)
- [ ] Three-tier security filter (§5 Stage 2)
- [ ] `sigil_env` tool + placeholder detection
- [ ] tree-sitter parsing: TypeScript, JavaScript, Python, Go
- [ ] Single-pass call graph extraction into `call_edges` table
- [ ] Full SQLite schema (`PRAGMA user_version = 1`)
- [ ] `files` table for blob SHA storage
- [ ] `possible_unresolved` flag + pre-delete marking
- [ ] Schema migration runner
- [ ] Enrichment pipeline (provider priority chain, default-on, graceful fallback)
- [ ] `sigil_search`, `sigil_get` (single + batch + raw files), `sigil_outline`,
      `sigil_tree`, `sigil_overview`, `sigil_env`, `sigil_diff`, `sigil_status` tools
- [ ] `metadata` envelope (per-call + session + repo_total) on every response
- [ ] Auto session ID generation (stdio: startup, HTTP: first call)
- [ ] `~/.sigil/tokens_saved.json` rollup
- [ ] `repos.json` global manifest
- [ ] stdio MCP server (global, always-on)
- [ ] `sigil cache` subcommands
- [ ] `sigil hook install` (pre-commit → sigil sync)
- [ ] `sigil status --verify` (content_hash audit)
- [ ] `last_indexed_commit` always updated on sync, validated before display
- [ ] Claude Desktop + Claude Code integration test

### Phase 2 — Full Toolset
- [ ] HTTP MCP server mode
- [ ] `sigil pull-index` (CI artifact download)
- [ ] CI workflow templates (GitHub Actions, GitLab CI)
- [ ] `--embed` flag + `sigil_search --semantic` (vector search)
- [ ] Ollama / LM Studio enrichment (auto-detect)
- [ ] GitHub remote indexing mode
- [ ] Rust, Java, C, C++, Ruby, PHP language support
- [ ] Protobuf + SQL lightweight extraction
- [ ] Strategy B monorepo namespacing (package-boundary qualified names)
- [ ] Gemini CLI + OpenAI Codex integration test
- [ ] Benchmark suite (3 repos, 5 task categories)
- [ ] `sigil savings --sessions`, `--session`, `--top` commands

### Phase 3 — Ecosystem
- [ ] `sigil mcp --http --read-only` + `SIGIL_SERVER_TOKEN`
- [ ] YAML, TOML, GraphQL, shell lightweight extraction
- [ ] `sqlite-vec` extension upgrade path (if brute-force cosine too slow)
- [ ] Published benchmark results + leaderboard
- [ ] Homebrew / winget / apt distribution
- [ ] Config TUI for browsing index
- [ ] Cross-repo symbol resolution (research)

---

## 17. Token Savings Validation — Benchmark Suite

### 17.1 Benchmark tasks

Canonical task set across 3 open-source repos (small: ~5k LOC, medium: ~50k LOC,
large: ~200k LOC):

| Category | Task |
|---|---|
| Symbol lookup | "What does `validateToken` do?" |
| Dependency trace | "What does `AuthService` depend on?" |
| Commit description | "Describe the changes in the last commit" |
| Refactor planning | "Where must I change to rename `UserID` → `AccountID`?" |
| Cold start | "Give me an overview of this codebase" |
| Env debugging | "Why is the API authentication failing?" |

### 17.2 Measurement protocol

```
1. Fresh session — zero conversation context
2. Baseline: no Sigil tools available
   Record: input_tokens, output_tokens, tool_call_count, quality (human 1–5)
3. Sigil: same task with Sigil tools
   Record: same metrics
4. token_reduction = (baseline - sigil) / baseline
```

Quality gate: Sigil results must score ≥ baseline quality.

### 17.3 Expected hypothesis

| Task | Baseline | Sigil | Reduction |
|---|---|---|---|
| Symbol lookup | ~1,500 | ~120 | 92% |
| Dependency trace | ~4,000 | ~450 | 89% |
| Commit description | ~6,000 | ~500 | 92% |
| Refactor planning | ~12,000 | ~800 | 93% |
| Cold start | ~25,000 | ~2,000 | 92% |
| Env debugging | ~8,000 | ~200 | 97% |

These are **hypotheses**. The benchmark suite is the proof.

---

## 18. Decision Log — Issues

| # | Issue | Decision |
|---|---|---|
| 🔴 1 | Byte offset invalidation | Full file reparse on blob SHA change. `DELETE WHERE file = ?` + re-insert all symbols. |
| 🔴 2 | `called_by` reverse-index | Separate `call_edges` table. Single pass. `calls`/`called_by` hydrated at query time. |
| 🔴 3 | `session_total` statefulness | Auto session ID + `repo_total`. Both in every envelope. HTTP sessions are approximations. |
| 🔴 4 | `git diff` wrong for sync | Blob SHA comparison only. `git diff` reserved for `sigil_diff` MCP tool exclusively. |
| 🟡 5 | Minimum git version | Version detection + command fallback. Min: 2.25. Below that: filesystem mode. |
| 🟡 6 | MCP repo-selection | Global always-on server. Path-based selection per call. `repos.json` manifest. |
| 🟡 7 | Symbol ID stability | `ON DELETE CASCADE` + `possible_unresolved` flag. Pre-delete marking. Self-healing. |
| 🟡 8 | Schema migration | `PRAGMA user_version` + sequential additive migrations. Four user-facing outcomes. |
| 🟡 9 | `content_hash` vs blob SHA | Three roles. `files` table in `index.db`. `content_hash` = audit only, not invalidation. |
| 🔵 10 | Name still "Glyph" | Renamed to **Sigil** throughout. Repo: `go-sigil-cli` → `sigil`. |
| 🔵 11 | Language choice stale | **Go 1.23. Final.** Section rewritten as statement of record. |
| 🔵 12 | Roadmap version confusion | Document versions and release phases are separate tracks. |

## 19. Decision Log — Open Questions

| # | Question | Decision |
|---|---|---|
| Q1 | Summary quality floor | Enrichment default-on when provider detected. Priority: Anthropic → Google → OpenAI-compatible → Ollama → templates. Never blocking. |
| Q2 | Monorepo namespacing | Full repo-relative path (Phase 1). `package_root` field pre-added. Package-boundary qualified names (Phase 2). |
| Q3 | Dynamic call graphs | `confidence` field on `call_edges`: `static` / `inferred` / `dynamic`. Dynamic edges stored with `raw_expression`, `callee_id` null. |
| Q4 | Counterfactual calibration | Per-repo exact measurement (`file_size - symbol_size`). `tokens_saved` only — no pricing, no cost calculations. |
| Q5 | Vector search | Phase 2 `--embed` flag. Brute-force cosine similarity in Go. `embedding BLOB` pre-added. `sqlite-vec` upgrade path Phase 3. |
| Q6 | Index sharing for teams | CI artifact Phase 2 (`sigil pull-index`). Read-only HTTP + `SIGIL_SERVER_TOKEN` Phase 3. No user management. |
| Q7 | Shallow git clones | Resolved — blob SHA strategy works on shallow clones. `sigil_diff` returns clear error for truncated refs. |
| Q8 | Unsupported language files | Show all files in `sigil_tree` (0 symbols, clearly marked). Raw retrieval via `sigil_get files:[]`. `sigil_env` for env config. Three-tier security model. |
| Q9 | `sigil init` vs `sigil index` | Resolved — `sigil index .` is smart (first-run registers + indexes, subsequent syncs incrementally). `--force` for rebuild. |
| Q10 | Git rebase / history rewrite | Blob SHA handles content correctly. Always update `last_indexed_commit` at end of sync. Validate before display. Clear `stale_commit_reference` error. |
| Q11 | `include_context_lines` backward scan | Strategy A — bounded backward scan with `ReadAt` (goroutine-safe). Max 50 lines. Context inline as one contiguous string. |

---

*Draft v0.4 — all 12 issues and 11 open questions resolved. Ready for implementation.*
