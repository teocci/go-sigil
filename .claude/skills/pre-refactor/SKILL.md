---
name: pre-refactor
description: Gathering comprehensive symbol-level context before making code changes, using sigil to map all callers, callees, and usages before any edits begin. Use this skill whenever a user is about to refactor, rename, move, or delete a symbol and wants to understand the blast radius, asks "what calls this function", "what would break if I change this", "gather context before I refactor", "show me all usages of X", "impact analysis for this change", "what depends on this symbol", "is it safe to modify this", or "I need to understand everything that touches this code". Also use proactively when a user says "I want to refactor the auth system", "I need to change this interface", or "I'm going to rename this function" — trigger before they start editing. This skill covers the search, outline, deps, and get commands. This is the correct skill any time code modification is imminent.
---

# Pre-Refactor Context Gathering

Map all callers, callees, and usages of a target symbol before touching it — so
refactoring is precise and nothing breaks unexpectedly.

## Goal

Before any edit, produce:

1. The exact symbol to change (ID, file, line, signature)
2. Every caller (direct and transitive) — the blast radius
3. Every symbol it calls — its dependencies
4. A risk assessment based on caller count and distribution

## Prerequisites

The repo must be indexed and up to date:

```bash
sigil sync .
```

Run `sync` before this workflow if you have pulled commits or made local edits
since the last index. Stale data will show outdated callers.

## Workflow

### Step 1 — Find the target symbol

```bash
sigil search <symbol-name>
sigil search <symbol-name> --kind function
sigil search <symbol-name> --file internal/auth/service.go
```

Returns: symbol list with IDs, kinds, signatures, file locations, and summaries.
Token budget: ~200 tokens.

Note the `id` field from the matching result — you need it for all subsequent
commands. If multiple symbols match by name, use `--file` to disambiguate or use
`--kind` to filter by type.

### Step 2 — Outline the containing file

```bash
sigil outline internal/auth/service.go
```

Returns: all symbols in the file as a structured hierarchy. Token budget: ~200–400
tokens.

This reveals the target symbol's context: what interface it implements, which struct
it belongs to, what methods surround it. You need this to understand what a signature
change will ripple through.

### Step 3 — Map callers (blast radius)

```bash
sigil deps <id> --direction callers
```

Returns: all symbols that directly call the target, with summaries. No source code.
Token budget: ~200–400 tokens.

For a wider blast radius, expand to second-order callers:

```bash
sigil deps <id> --direction callers --depth 2
```

Use `--depth 2` when the symbol is a utility called by many things — it shows
callers of callers, revealing the full impact. Avoid `--depth 3` or higher for
widely-used symbols; the graph becomes too large to act on.

### Step 4 — Map callees (dependencies)

```bash
sigil deps <id> --direction calls
```

Returns: everything the target symbol calls, with summaries. Token budget: ~100–300
tokens.

This reveals what the symbol depends on. If you are changing behavior rather than
just a signature, these are the symbols that may need coordinated updates.

### Step 5 — Retrieve the symbol source

```bash
sigil get --id <id>
sigil get --id <id> --context 5
```

Returns: the exact source of the symbol. Token budget: ~100–400 tokens.

Use `--context 5` when the symbol has closely related declarations adjacent to it
(e.g., a method where you need to see the struct definition above it).

### Step 6 — Search for string-based usages (optional)

```bash
sigil search <symbol-name>
```

Catches usages the static call graph may miss: dynamic dispatch via interfaces,
reflection-based calls, or cross-package references not yet resolved as call edges.
Token budget: ~200 tokens.

Run this step when `deps --direction callers` returns unexpectedly few results or
the symbol is used via an interface.

## Output Summary

Before editing, report findings in this format:

```
TARGET: <SymbolName> (id: <id>)
  File:      path/to/file.go:42
  Signature: func SymbolName(args) returnType

CALLERS (direct): N symbols
  - CallerA (id: ...) — path/to/caller.go — [summary]
  - CallerB (id: ...) — path/to/caller.go — [summary]

CALLEES: M symbols
  - DepA (id: ...) — path/to/dep.go — [summary]

RISK: low | medium | high
  Reason: [e.g., "3 callers all in one package — low coordination cost"
           or "12 callers across 5 packages — high blast radius, change carefully"]
```

Risk guide:
- **Low** — 0-3 callers, all in the same package
- **Medium** — 4-10 callers, or callers in multiple packages
- **High** — 10+ callers, or callers in public API / external packages

## Tips

- Run this skill *before* opening an editor — discovering blast radius after editing
  means re-checking everything manually
- If `deps --direction callers` returns 0 results, the symbol may be unexported,
  unreachable, or not yet resolved in the index — run `sigil status --verify` to
  check index completeness
- For interface methods, the call graph tracks each implementing type separately —
  search by method name in addition to running `deps` on the interface itself
- A high caller count is not always high risk — 10 callers all following the same
  pattern in one package is lower risk than 3 callers in different packages with
  different call patterns
- Hand off to `debug-callgraph` if the goal is understanding unexpected runtime
  behavior rather than planning a change
