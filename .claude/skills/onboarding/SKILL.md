---
name: onboarding
description: Structured orientation of a new developer to an unfamiliar codebase using sigil to deliver a guided architectural tour without reading full source files. Use this skill when a user is new to a project and wants to get oriented, asks "I'm new to this codebase, where do I start", "give me a developer onboarding tour", "explain the architecture to a new team member", "orient me to this project", "I just joined this team, help me ramp up", "what are the most important files and functions to know", "create a mental model of this codebase for me", or "walk me through this project like I'm a new developer". Also use when the user wants to produce a structured onboarding document for a colleague, or says "what should I read first" and hasn't started a specific task yet. Unlike the explore-codebase skill (which is task-focused), this skill is persona-focused: it produces a narrative developer guide. Covers the overview, tree, outline, search, and get commands.
---

# New Developer Onboarding

Produce a structured developer orientation guide using sigil — no full file reads
required. The output is a narrative document a new team member can follow, not just
an exploration log.

## Goal

By the end of this workflow, you will have produced:

1. An architecture summary (what the repo is, what it does, how it is organized)
2. A key-files map (the 5–10 most important files, in reading order)
3. A critical-symbols table (the 10–20 most important functions and types)
4. Suggested first tasks for hands-on learning

## Prerequisites

The repo must be indexed: `sigil index .`

For the best onboarding output, enrichment should be enabled — enriched indexes
include one-line summaries in `search` output that become the symbol descriptions
in the guide. If not enriched, run `sigil index . --enrich` first.

## Workflow

### Step 1 — Repository identity

```bash
sigil overview
```

Returns: language mix, package structure, symbol totals, enrichment status, index
age. Token budget: ~200 tokens.

### Step 2 — Architectural skeleton

```bash
sigil tree --source-only
```

Returns: source directory structure with symbol counts. Token budget: ~200–400 tokens.

`--source-only` prunes non-code files (docs, configs, IDE dirs). Default depth is 2.
To explore a subdirectory: `sigil tree internal/ --source-only`

**Never use `--depth 3` on the repo root** — output can exceed 30,000 chars.

### Step 3 — Entry points and public API

```bash
sigil search "main" --kind function
sigil search "New" --kind function
sigil search "Handler" --kind function
sigil search "Service" --kind type
```

Returns: symbol names, signatures, file locations, summaries. Token budget: ~200
tokens per search.

**Search tip:** Bare words auto-expand with `*` (prefix match). If a search returns
0 results, try a shorter prefix or a different word from the symbol name.

Stop after 2–3 searches — breadth over depth at this stage.

### Step 4 — File-by-file outlines for core files

```bash
sigil outline cmd/main.go
sigil outline internal/service/service.go
sigil outline internal/api/handlers.go
```

Returns: complete symbol hierarchy — names, kinds, summaries. Token budget: ~200–400
tokens per file.

**Anti-pattern:** Do NOT `outline` then `get` every symbol in the file — that defeats
the purpose. Only retrieve source for the 2–4 symbols critical for the guide.

### Step 5 — Retrieve key symbol signatures

```bash
# Always batch multiple IDs into one call
sigil get <id1> <id2> <id3> .
```

Retrieve only the 3–5 most architecturally significant symbols. Token budget: ~100–300
tokens per symbol.

## Output Format

```markdown
## Architecture Summary

[2–3 paragraphs: what the repo is, what it does, how it is organized by layer]

## Key Files (read in this order)

1. `path/to/file.go` — [why this file matters to a new developer]
2. `path/to/file.go` — [why]
3. ...

## Critical Symbols

| Symbol | File | What It Does |
|---|---|---|
| `FunctionName` | `path/to/file.go` | [one-line description] |
| `TypeName` | `path/to/file.go` | [one-line description] |

## Suggested First Tasks

1. [A concrete task that teaches the data flow]
2. [A task that teaches the persistence layer]
3. [A task that teaches the test patterns]
```

## Tips

- Use enrichment summaries from `search` as the primary source of symbol descriptions
- Batch `get` calls: `sigil get id1 id2 id3 .` — never loop single-ID calls
- Limit `get` calls to 3–5 critical symbols — architecture understanding, not full review
- Keep Suggested First Tasks concrete and runnable
- Hand off to `explore-codebase` when the user is ready to investigate a specific feature
- Hand off to `pre-refactor` when the user is ready to make their first change
