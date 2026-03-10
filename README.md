# Sigil

**Token-efficient codebase intelligence for AI agents.**

Sigil indexes your source code into a local SQLite database and serves surgical symbol
retrieval to AI agents — returning only the exact functions, types, and call edges they
need instead of full files. Typical token reduction: **70–97%** per query.

---

## Why Sigil

When an AI agent needs to understand a codebase it usually reads entire files. A 400-line
service file might contain one relevant function. Sigil solves this:

- **Index once, query forever** — incremental re-indexing in milliseconds
- **Symbol-precision retrieval** — get one function body, not the whole file
- **Call graph traversal** — trace callers and callees without reading source
- **File outline** — all symbols in a file as a structured list, no source
- **Token savings ledger** — tracks exactly how many tokens were saved per session

Supports **Go, TypeScript, JavaScript, and Python** out of the box.

---

## Installation

### Download a binary (recommended)

Pre-compiled binaries are available on the [Releases](../../releases) page.

| Platform | Binary |
|---|---|
| Windows x64 | `sigil-windows-amd64.exe` → rename to `sigil.exe` |
| Linux x64 | `sigil-linux-amd64` → rename to `sigil` |
| macOS ARM64 | `sigil-darwin-arm64` → rename to `sigil` |

Add the binary to your `PATH`.

### Build from source

Requires Go 1.21+ and gcc (for the tree-sitter parser).

```bash
git clone https://github.com/teocci/go-sigil
cd go-sigil
CGO_ENABLED=1 go build -o sigil ./cmd/sigil
CGO_ENABLED=1 go build -o sigil-mcp ./cmd/mcp
```

---

## Quick Start

```bash
# 1. Index your repo (run from anywhere inside it)
sigil index .

# 2. Search for a symbol
sigil search "validateToken"

# 3. Get the source of a specific symbol
sigil get --id a3f9c1b2

# 4. See all symbols in a file
sigil outline internal/auth/service.go

# 5. Check index health
sigil status
```

---

## Commands

### `sigil index [path]`

Index a repository. Run once to build the index, then `sigil sync` for incremental updates.

```bash
sigil index .                  # index current directory
sigil index /path/to/repo      # index a specific repo
sigil index . --force          # full rebuild (ignores cache)
sigil index . --enrich         # add LLM summaries (requires API key)
```

### `sigil sync [path]`

Incrementally re-index only changed files. Much faster than a full index.

```bash
sigil sync .
```

### `sigil search <query>`

Full-text search across all indexed symbols. Returns names, signatures, and summaries —
no source code.

```bash
sigil search "parseToken"
sigil search "auth" --kind method
sigil search "handler" --language typescript --limit 20
sigil search "database" --file internal/db/db.go
```

**Flags:** `--kind`, `--language`, `--file`, `--limit`

### `sigil get`

Retrieve symbol source by ID, or raw file content by path.

```bash
sigil get --id a3f9c1b2
sigil get --id a3f9c1b2 --context 5     # 5 lines of context around the symbol
sigil get --file proto/auth.proto        # raw file for unsupported types
sigil get --id a3f9c1b2 --id b2e1a0f3   # batch retrieval
```

### `sigil deps <id>`

Traverse the call graph for a symbol. Returns summaries — no source.

```bash
sigil deps a3f9c1b2                      # callers and callees
sigil deps a3f9c1b2 --direction calls    # only what this symbol calls
sigil deps a3f9c1b2 --direction callers  # only what calls this symbol
sigil deps a3f9c1b2 --depth 2            # two hops
```

### `sigil outline <file>`

List all symbols in a file as a structured hierarchy.

```bash
sigil outline internal/auth/service.go
sigil outline src/api/handlers.ts
```

Example output:
```
AuthService (class) — Handles JWT-based authentication
  constructor(config: AuthConfig)
  validateToken(token, opts?) → Promise<User>
  revokeToken(jti: string) → void
```

### `sigil tree [path]`

Repository file structure with languages and symbol counts.

```bash
sigil tree
sigil tree --scope src/auth --depth 3
sigil tree --counts                    # include symbol counts per file
```

### `sigil overview`

High-level repository summary: languages, top packages, symbol totals, index age.

```bash
sigil overview
sigil overview /path/to/other-repo
```

### `sigil env`

Inspect `.env` file configuration state. Never reveals values — only reports whether
variables are `set`, `empty`, or `placeholder`. Useful for debugging auth/connection issues.

```bash
sigil env
```

### `sigil diff --since <ref>`

Symbol-level diff since a git ref. Shows which symbols were added, modified, or deleted.

```bash
sigil diff --since HEAD~1
sigil diff --since main
sigil diff --since abc1234
```

### `sigil status`

Index health: last indexed commit, symbol counts, unresolved call edges, savings summary.

```bash
sigil status
sigil status --verify    # re-read sources to verify content hashes
```

### `sigil savings`

Token savings ledger: how many tokens were saved this session and historically.

```bash
sigil savings
sigil savings --sessions    # per-session breakdown
sigil savings --top 10      # top 10 sessions
```

### `sigil cache`

Manage the index cache.

```bash
sigil cache status
sigil cache invalidate .     # force re-index on next run
sigil cache clear            # remove all cached data
```

---

## MCP Server (AI Agent Integration)

Sigil includes an MCP (Model Context Protocol) server so AI agents — Claude Code, Cursor,
Windsurf, and others — can query the index directly without shell commands.

### Configure in Claude Code

Add to your MCP settings (`~/.claude.json` or workspace settings):

```json
{
  "mcpServers": {
    "sigil": {
      "command": "sigil",
      "args": ["mcp"],
      "env": {
        "SIGIL_LOG_FILE": "/tmp/sigil-mcp.log"
      }
    }
  }
}
```

> **Important:** `SIGIL_LOG_FILE` must be set. Logs written to stdout corrupt the
> MCP JSON-RPC stream.

### MCP Tools

Once connected, the agent has access to 9 tools:

| Tool | What it does |
|---|---|
| `sigil_overview` | Repo summary — call this first at every session |
| `sigil_search` | Find symbols by name or keyword |
| `sigil_get` | Retrieve symbol source or raw file content |
| `sigil_outline` | All symbols in a file, structured |
| `sigil_deps` | Call graph traversal |
| `sigil_tree` | File structure |
| `sigil_env` | Environment variable state |
| `sigil_diff` | Symbol-level git diff |
| `sigil_status` | Index health |

All tools accept an optional `path` parameter for multi-repo setups. Sigil auto-selects
the correct index by walking up to the git root.

### Standalone MCP binary

If you prefer a separate process:

```bash
SIGIL_LOG_FILE=/tmp/sigil-mcp.log sigil-mcp
```

---

## LLM Enrichment

Sigil can add AI-generated summaries to indexed symbols. Summaries are stored in the
index and included in search results and outlines, giving agents more context without
needing to read source.

```bash
sigil index . --enrich
```

Provider priority (first available key wins):

| Priority | Provider | Environment Variable |
|---|---|---|
| 1 | Anthropic (Claude Haiku) | `ANTHROPIC_API_KEY` |
| 2 | Google (Gemini Flash) | `GOOGLE_API_KEY` |
| 3 | OpenAI-compatible | `OPENAI_API_BASE` + `OPENAI_API_KEY` |
| 4 | Ollama (local) | auto-detected at `localhost:11434` |
| 5 | Template | always available, no API key needed |

---

## Configuration

Sigil reads configuration from `~/.sigil/config.toml` (created automatically on first run).
All settings can be overridden with environment variables.

### Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `CODE_INDEX_PATH` | Override cache directory | `~/.sigil/` |
| `SIGIL_LOG_LEVEL` | `DEBUG` / `INFO` / `WARNING` / `ERROR` | `WARNING` |
| `SIGIL_LOG_FILE` | Log file path (required in MCP mode) | stderr |
| `SIGIL_MAX_INDEX_FILES` | Max files indexed per repo | `500` |
| `SIGIL_ENRICH_BATCH_SIZE` | Concurrent LLM requests during enrichment | `4` |

### Config File (`~/.sigil/config.toml`)

```toml
[security]
# Extra filenames to treat as secrets (redacted, byte offsets omitted)
extra_secret_filenames = ["my-secrets.yaml"]

# Extra glob patterns to ignore entirely
extra_ignore_filenames = ["*_generated.go", "vendor/**"]

[indexing]
max_files = 1000

[enrichment]
batch_size = 8
disabled = false
```

---

## Storage

All index data lives in `~/.sigil/` and is never sent anywhere. Each repository gets its
own isolated subdirectory identified by a hash of the repo path.

```
~/.sigil/
  repos.json              ← global repo list
  tokens_saved.json       ← lifetime savings rollup
  config.toml
  {repo-hash}/
    index.db              ← SQLite index (WAL mode)
    meta.json             ← repo metadata
    files/                ← raw source mirror (content-addressed)
```

---

## Ignoring Files

Sigil respects `.gitignore` automatically. To add Sigil-specific exclusions, create
`.sigilignore` in your repo root (same syntax as `.gitignore`):

```
# .sigilignore
*_generated.go
testdata/
fixtures/
```

---

## Supported Languages

| Language | Extensions |
|---|---|
| Go | `.go` |
| TypeScript | `.ts`, `.tsx` |
| JavaScript | `.js`, `.jsx`, `.mjs` |
| Python | `.py` |

Other file types (`.proto`, `.sql`, `.yaml`, etc.) are tracked and retrievable via
`sigil get --file`, but are not parsed for symbols.

---

## License

See [LICENSE](LICENSE).
