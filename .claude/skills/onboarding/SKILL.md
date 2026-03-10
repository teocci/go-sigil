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

Extract: what languages are used, how many packages or modules exist, whether
enrichment summaries are available. The language mix tells you where to look for
different concerns (e.g., Go backend in `internal/`, TypeScript frontend in `src/`).

### Step 2 — Architectural skeleton

```bash
sigil tree --depth 2
```

Returns: top-level directory structure. Token budget: ~300 tokens.

Follow up with density information to find where the logic lives:

```bash
sigil tree --counts --depth 3
```

Token budget: ~500 tokens.

Identify the major architectural layers (e.g., `cmd/`, `internal/`, `pkg/`, `api/`,
`src/components/`) and which directories hold the most symbols. These become the
"zones" in your architecture summary.

### Step 3 — Entry points and public API

```bash
sigil search "main" --kind function
sigil search "New" --kind function
sigil search "Handler" --kind function
sigil search "Service" --kind type
```

Returns: symbol names, signatures, file locations, summaries. Token budget: ~200
tokens per search.

Locate the primary entry points: main functions, constructors, HTTP handlers,
service factories. These are the "doors" a new developer should understand first.
Stop after 2–3 searches — you want breadth here, not depth.

### Step 4 — File-by-file outlines for core files

```bash
sigil outline cmd/main.go
sigil outline internal/service/service.go
sigil outline internal/api/handlers.go
```

Returns: complete symbol hierarchy for each file — names, kinds, summaries. Token
budget: ~200–400 tokens per file.

Outline the 3–5 files that appeared most structurally central in Steps 2–3. This
reveals internal organization without reading source. Use the symbol list to decide
which specific symbols to retrieve for the guide.

### Step 5 — Retrieve key symbol signatures

```bash
sigil get --id <symbol_id> --context 2
```

Retrieve only the 3–5 most architecturally significant symbols (primary service
interface, main request dispatcher, core data types). Use `--context 2` to see
surrounding declarations without reading the whole file.

Token budget: ~100–300 tokens per symbol.

## Output Format

Organize the findings as a developer guide:

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

1. [A concrete task that teaches the data flow — e.g., "Add a log statement inside X
   and call it from Y to see how requests flow through the system"]
2. [A task that teaches the persistence layer]
3. [A task that teaches the test patterns]
```

## Tips

- Use enrichment summaries from `search` as the primary source of symbol descriptions
  in the guide — they are pre-computed and add no token cost
- If the repo has no enrichment (`sigil overview` shows enrichment: none), run
  `sigil index . --enrich` before starting — the guide quality improves significantly
- Limit `get` calls to 3–5 critical symbols — the goal is architecture understanding,
  not complete source review
- Keep Suggested First Tasks concrete and runnable, not vague ("read the README")
- Hand off to `explore-codebase` when the user is ready to investigate a specific feature
- Hand off to `pre-refactor` when the user is ready to make their first change
