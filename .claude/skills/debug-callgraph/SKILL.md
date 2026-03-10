---
name: debug-callgraph
description: Tracing execution paths and call relationships through a codebase using sigil's call graph traversal to understand bugs, unexpected behavior, or complex control flow. Use this skill when a user is debugging and needs to understand how execution reaches a certain point, asks "how does the call reach this function", "who calls this method", "trace the execution path to this error", "what is the call chain for this bug", "follow the stack from entry point to X", "map the call graph for this feature", "I don't understand how X gets invoked", or "what triggers this code to run". Also use when a user describes a bug involving unexpected function behavior and wants to understand the invocation context, or says "why is this function being called with these arguments". This skill covers the deps, get, and search commands with a focus on --direction and --depth traversal. Always prefer this skill over reading full files to trace execution paths.
---

# Debug Call Graph Traversal

Trace the execution path to a suspicious symbol using sigil's call graph — without
reading full files.

## Goal

Answer: "How does execution reach this function, and what does it call?"

## Prerequisites

The repo must be indexed: `sigil index .`

For best results, enrichment should be enabled — `sigil deps` with summaries lets
you understand callers without retrieving their source, dramatically reducing token
cost per hop.

## Workflow

### Step 1 — Locate the suspicious symbol

```bash
sigil search <function-name>
sigil search <function-name> --kind function
sigil search <function-name> --file internal/auth/handler.go
```

Returns: symbol list with IDs, signatures, file locations, summaries. Token budget:
~100–300 tokens.

Get the `id` of the symbol you are investigating. If the name is ambiguous (multiple
symbols match), use `--file` or `--kind` to disambiguate. You need the exact `id`
for all subsequent `deps` calls.

### Step 2 — Who calls this symbol?

```bash
sigil deps <id> --direction callers
```

Returns: direct callers with names, file locations, and summaries. No source code.
Token budget: ~100–300 tokens.

This answers: "What code paths lead to this symbol being invoked?" Read the summaries
to understand the context of each caller. If you recognize an entry point (HTTP
handler, goroutine start, test function), you have found the call chain.

### Step 3 — Expand the call tree upward

```bash
sigil deps <id> --direction callers --depth 2
```

Returns: transitive callers up to 2 hops. Token budget: ~200–600 tokens.

Increase depth until you reach a recognizable entry point:

```bash
sigil deps <id> --direction callers --depth 3
```

Stop at depth 3 for most debugging scenarios — deeper graphs become noisy and the
summaries alone are usually enough to trace the chain. Read the output from the
deepest level upward to reconstruct the call sequence.

### Step 4 — What does this symbol call?

```bash
sigil deps <id> --direction calls
```

Returns: everything the target symbol directly invokes, with summaries. Token budget:
~100–300 tokens.

This answers: "What does this symbol depend on that might be misbehaving?" Compare
what it calls against what you expect it to call. Unexpected callees are often
the bug.

### Step 5 — Retrieve source for anomalous symbols

```bash
sigil get --id <id>
sigil get --id <id1> --id <id2>
```

Retrieve source only for the specific symbols whose summaries suggest something
unexpected. Token budget: ~100–300 tokens per symbol.

Use batch `--id` when multiple related symbols need inspection. Avoid retrieving
entire files — target only the anomalous symbols.

### Step 6 — Search for alternate invocation paths

```bash
sigil search <symbol-name>
```

Catches call paths the static call graph misses: goroutine-dispatched calls, dynamic
dispatch via interfaces, or invocations identified by string pattern. Token budget:
~200 tokens.

Run this step when `deps --direction callers` returns fewer callers than expected —
it may reveal interface implementations or goroutine patterns not captured by the
static graph.

## Reading the Call Graph Output

When interpreting `sigil deps` output, read it as a tree:

```
Direct callers (depth 1):
  AuthMiddleware — HTTP middleware that validates JWT tokens
    └─ Depth 2 callers of AuthMiddleware:
         Router.Setup — registers all HTTP routes and middleware
```

Trace upward from your suspicious symbol to an entry point. The path between them
is the call chain you need to understand.

If the target symbol is a callee that behaves unexpectedly, read the **callers**
first to understand the invocation context, then the **callees** to see what the
symbol does with its inputs.

## Tips

- Start with `--direction callers` before `--direction calls` — the invocation context
  is usually the key to understanding the bug
- `--depth 2` covers the majority of debugging scenarios; increase only if depth 1
  reveals no entry points you recognize
- If the caller graph is empty, the symbol may be called via an interface — search
  for all types implementing the interface and run `deps --direction calls` on each
- If enrichment summaries are available, you can often diagnose the bug from the
  summaries alone without calling `get` at all
- Prefer batch `get --id <id1> --id <id2>` over multiple single `get` calls — it
  retrieves multiple symbols in one round trip
- Hand off to `pre-refactor` once you have identified the fix and want to assess the
  change's blast radius before editing
