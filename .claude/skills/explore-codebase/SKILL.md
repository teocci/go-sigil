---
name: explore-codebase
description: Systematically exploring and mapping an unfamiliar codebase using the sigil CLI for token-efficient symbol retrieval. Use this skill whenever a user wants to understand a new repository, asks "what does this codebase do", "how is this project structured", "walk me through this repo", "map out the code", "what are the entry points", "give me an overview of this project", or starts work on a repo they haven't seen before. Also use this when the user wants a high-level summary before reading any source files, or says things like "orient me", "help me understand this project", "explore this repo with me", or "what should I know before touching this code". This skill covers the overview, tree, search, outline, and get commands. Always prefer this skill over reading raw files when sigil is available — it delivers the same understanding at a fraction of the token cost.
---

# Explore an Unfamiliar Codebase

Build a complete mental model of a repository in under 2,000 tokens — without
reading a single full file.

## Prerequisites

The repo must be indexed. If sigil commands return "index not found" errors, run
the `index-setup` skill first: `sigil index .`

## Workflow

### Step 1 — Repository overview

```bash
sigil overview
```

Returns: language breakdown, package count, total symbol count, enrichment status,
and index age. Token budget: ~200 tokens.

Always start here. It tells you the language mix, rough scale, and whether enrichment
summaries are available (which makes every later step more informative).

### Step 2 — File tree with symbol density

```bash
sigil tree --source-only
```

Returns: directory tree showing only source code files, annotated with per-file
symbol counts. Token budget: ~200–400 tokens.

`--source-only` prunes non-code files (docs, configs, IDE dirs). Default depth is 2,
which is sufficient for most repos.

For large monorepos, scope to a subdirectory:

```bash
sigil tree internal/ --source-only
```

**Never use `--depth 3` on the repo root** — it can produce 30,000+ chars of output.
Scope to a subdirectory (`sigil tree internal/`) instead of increasing depth.

### Step 3 — Locate entry points

```bash
sigil search main --kind function
```

Returns: symbol names, signatures, file locations, and enrichment summaries where
available. No source code. Token budget: ~200 tokens per search.

Follow up with targeted searches based on what Step 1 revealed:

```bash
sigil search handler --kind function --limit 20
sigil search "New" --kind function --language go
sigil search "Controller" --kind class --language typescript
```

Use `--kind` to filter by symbol type: `function`, `method`, `class`, `type`,
`interface`, `const`, `var`. Use `--language` in multilingual repos.

**Search tip:** Bare words auto-expand with `*` (prefix match). `sigil search "sav"`
matches `save`, `savings`, `savings_service`. If a search returns 0 results, try a
shorter prefix or a different word in the symbol name.

### Step 4 — Inspect a file's symbol structure

```bash
sigil outline internal/service/indexer.go
sigil outline src/api/handlers.ts
```

Returns: structured hierarchy of all symbols in the file — names, kinds, and
one-line summaries. Token budget: ~200–500 tokens per file.

This is the primary command for understanding a file without reading it. Run it
on the 3–5 files that appear most important from Steps 2–3. Use the output to
decide which specific symbols need deeper inspection.

**Anti-pattern:** Do NOT `outline` a file and then `get` every symbol in it —
that is equivalent to reading the whole file but with more calls and more overhead.
Only `get` the 1–3 symbols you actually need source for.

### Step 5 — Retrieve targeted source

```bash
sigil get <symbol_id>
sigil get <symbol_id> --context 3
```

Returns: the symbol's source code only, with optional N lines of surrounding
context. Token budget: ~100–300 tokens per symbol.

**Always batch multiple IDs into a single call:**

```bash
# Correct — one call for multiple symbols
sigil get <id1> <id2> <id3> .

# Wrong — three separate calls (3x overhead)
sigil get <id1> .
sigil get <id2> .
sigil get <id3> .
```

Only retrieve symbols you cannot understand from the `outline` or enrichment
summaries alone.

## Tips

- Run `overview` + `tree --source-only` before any `search` — together they cost
  under 600 tokens and prevent blind searching
- Use `outline` on any file before `get` — `outline` is typically 10x more token-efficient
- If enrichment summaries appear in `search` output, trust them and skip `get`
  for symbols you already understand
- Batch `get` calls: `sigil get id1 id2 id3 .` — never call `get` in a loop
- Use `--compact` for even lower token cost: `sigil get id1 id2 --compact .`
- Hand off to the `pre-refactor` skill once you have enough context to start making changes
- Hand off to the `onboarding` skill if the goal is to produce a structured developer guide
