---
name: index-setup
description: First-time setup and ongoing maintenance of a sigil codebase index, including enrichment with LLM summaries and index health verification. Use this skill when a user wants to index a repo for the first time, asks "how do I set up sigil", "sigil isn't indexed", "run sigil index", "enrich the index with summaries", "my sigil index is stale", "check if sigil is up to date", "sync the index after pulling", "what are my token savings", or encounters errors about missing or outdated index data. Also use this when another skill fails because sigil cannot find symbols, when the user switches branches and the index may be outdated, or when they want to verify index health before a long session. This skill covers the index, sync, status, env, cache, and savings commands. Run this skill before any other sigil skill if the repo has never been indexed.
---

# Index Setup and Maintenance

Build and maintain sigil's local SQLite index so all other skills work correctly.
The index lives entirely on your machine — nothing is sent externally.

## First-Time Setup

### Step 1 — Build the index

```bash
sigil index .
```

Sigil parses all Go, TypeScript, JavaScript, and Python files and writes a local
database at `~/.sigil/{repo-hash}/index.db`. Run from anywhere inside the repo.

For a specific path:

```bash
sigil index /path/to/repo
```

For a full rebuild (if the index seems corrupted or symbols are missing):

```bash
sigil index . --force
```

Typical duration: 5–30 seconds for repos up to 10,000 files. You will see a
progress summary when indexing completes.

### Step 2 — Verify index health

```bash
sigil status
```

Returns: last-indexed commit, file count, symbol count, unresolved call edges,
and a savings summary. Token budget: ~100 tokens.

For a deeper check that re-reads source files and validates content hashes:

```bash
sigil status --verify
```

Use `--verify` after large git operations (rebase, merge with many conflicts,
branch switch) to confirm the index reflects the current file state.

### Step 3 — Enrich with LLM summaries (recommended)

```bash
sigil index . --enrich
```

Adds AI-generated one-line summaries to every symbol. Enrichment makes `search`
and `deps` output dramatically more useful — summaries appear inline so you can
often understand a symbol without calling `get` at all.

First, check which API keys are configured (enrichment requires at least one):

```bash
sigil env
```

Returns: whether `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, `OPENAI_API_KEY`, and
related variables are `set`, `empty`, or `placeholder`. Never reveals values.

Provider priority (first available key wins):

| Priority | Provider | Variable |
|---|---|---|
| 1 | Anthropic (Claude Haiku) | `ANTHROPIC_API_KEY` |
| 2 | Google (Gemini Flash) | `GOOGLE_API_KEY` |
| 3 | OpenAI-compatible | `OPENAI_API_BASE` + `OPENAI_API_KEY` |
| 4 | Ollama (local) | auto-detected at `localhost:11434` |
| 5 | Template | always available, no key needed |

If no key is configured, sigil falls back to template summaries — functional but
less informative.

## Incremental Updates

### Step 4 — Sync after code changes

```bash
sigil sync .
```

Re-indexes only the files changed since the last run. Much faster than a full
`sigil index`. Token budget for the status report: ~50 tokens.

Run `sigil sync` at the start of each work session after pulling new commits, or
after making local edits before running searches.

## Cache Management

### Step 5 — Inspect cache

```bash
sigil cache status
```

Returns: cache size on disk, number of repos indexed.

### Step 6 — Invalidate stale entries

```bash
sigil cache invalidate
```

Forces the next `sync` to re-parse all files (retains the database, discards
parse cache). Use when symbols that should have changed still appear with old
signatures.

For a complete wipe (re-index from scratch after this):

```bash
sigil cache clear
```

## Token Savings

```bash
sigil savings
sigil savings --sessions 10
sigil savings --top 5
```

Reports cumulative tokens saved by using sigil instead of reading raw files.
`--sessions N` shows the last N sessions. `--top N` shows the highest-savings
queries.

## Tips

- Run `sigil status` at the start of any session — 100 tokens to confirm the
  index is current is always worth it
- `sigil sync` is the everyday command; `sigil index` is only needed once per repo
  or after `sigil cache clear`
- `--enrich` is a one-time cost that pays off on every subsequent `search` call
  by reducing the need for `get` calls
- The `~/.sigil/` directory can be deleted safely at any time; `sigil index .`
  rebuilds everything from source
- If `sigil env` shows a key as `placeholder`, the variable is set to a dummy
  value (e.g. `sk-xxx`) — replace it with a real key for enrichment to work
