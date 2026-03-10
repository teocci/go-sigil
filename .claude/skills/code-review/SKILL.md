---
name: code-review
description: Performing symbol-aware code review of git diffs using sigil to see exactly which functions, types, and methods changed and to fetch their context efficiently. Use this skill when a user wants to review a pull request, asks "what changed in this branch", "review the diff against main", "what functions were added or modified since HEAD~1", "help me review this PR", "show me the symbol-level changes", "what was affected by this commit", "do a code review of these changes", or "what symbols did this PR touch". Also use when a user wants to understand the scope of a commit or branch before merging, needs to verify which call edges were affected by a change, or wants to check whether all callers were updated after a signature change. This skill covers the diff, outline, deps, and get commands. Always prefer sigil diff over reading raw git diff output — it shows changed symbols, not changed lines.
---

# Symbol-Aware Code Review

Review a PR or commit at the symbol level using sigil — see exactly what changed,
fetch only the symbols you need, and verify call graph impact.

## Goal

By the end of this workflow:

1. Know which symbols were added, modified, or deleted
2. Have source for any symbol that warrants deeper review
3. Verified that callers are updated for any modified signatures
4. Identified any new symbols with unexpected dependencies

## Prerequisites

The repo must be indexed and synced to the branch being reviewed:

```bash
sigil sync .
```

Run `sync` after checking out the feature branch or after fetching the latest commits.

## Workflow

### Step 1 — Get the symbol-level diff

```bash
sigil diff --since main
sigil diff --since HEAD~1
sigil diff --since <branch-name>
sigil diff --since <commit-hash>
```

Returns: list of symbols that were added, modified, or deleted since the given ref,
with their status and file locations. Token budget: ~200–800 tokens depending on PR
size.

Use `--since main` for PR review. Use `--since HEAD~1` for single-commit review.

This is the primary review entry point. Unlike `git diff`, it operates at the symbol
level — a 200-line file change might show only 3 modified functions, immediately
focusing the review.

### Step 2 — Outline changed files for structural context

```bash
sigil outline path/to/changed/file.go
sigil outline src/api/changed-handler.ts
```

Returns: complete symbol hierarchy for the file. Token budget: ~200–400 tokens per
file.

Outline the 2–3 most significant changed files to understand the full structure of
what was touched, not just the diff subset. This reveals if a method was added to a
type, or if a file's overall responsibility shifted.

### Step 3 — Retrieve modified symbol source

```bash
sigil get --id <id>
sigil get --id <id1> --id <id2> --id <id3>
```

Retrieve source for symbols marked `modified` or `added` in the diff output that
need deeper scrutiny. Token budget: ~100–300 tokens per symbol.

Use batch `--id` to fetch multiple symbols in one call. Skip symbols marked
`deleted` — they no longer exist and cannot be retrieved. Skip `added` symbols that
look trivially correct from the diff summary.

### Step 4 — Check callers of modified symbols

```bash
sigil deps <id> --direction callers
```

For any symbol whose **signature changed**, check callers to verify they were all
updated. Token budget: ~100–300 tokens per symbol.

This is the most common review miss: a function signature changes, the function
itself looks correct, but some callers still pass the old argument pattern.

### Step 5 — Check callees of new symbols

```bash
sigil deps <id> --direction calls
```

For **newly added** symbols, verify that what they call is appropriate: no
unexpected cross-layer dependencies, no calls to deprecated functions, no circular
imports. Token budget: ~100–300 tokens per symbol.

## Review Output Format

Structure the review findings as:

```markdown
## Symbol-Level Diff Summary

- Added:    N symbols
- Modified: M symbols
- Deleted:  K symbols

## Modified Symbols

### `SymbolName` (modified) — path/to/file.go

[One paragraph: what changed, whether the change is correct, any concerns]

Callers checked: yes — [N callers all updated / concern: CallerX still uses old signature]

### `NewSymbol` (added) — path/to/file.go

[One paragraph: what it does, whether the implementation is sound]

Callees reviewed: yes — [looks appropriate / concern: calls deprecated DepX]

## Deleted Symbols

- `DeletedFn` — path/to/file.go — [safe: had 0 callers before deletion]
- `OldType` — path/to/file.go — [concern: CallerY may still reference this]

## Verdict

**Approve** / **Request changes** / **Needs discussion**

Key concerns:
- [Concern 1]
- [Concern 2]
```

## Tips

- `sigil diff --since main` vs `--since HEAD~1`: use `main` for PR review,
  `HEAD~1` for single-commit review
- If `diff` output is very large (>50 symbols changed), focus first on `modified`
  symbols — additions and deletions are usually lower risk than modifications
- Check callers for **every** `modified` function — this step catches the most common
  review miss
- For deleted symbols, verify they had zero callers before deletion: run
  `sigil deps <id> --direction callers` on the pre-deletion ID (if available) or
  look for them in the caller graph of remaining symbols
- Avoid `sigil get --file` on changed files — use `sigil get --id` per symbol;
  whole-file retrieval is almost never necessary in code review
- Hand off to `debug-callgraph` if a symbol's behavior looks correct but you suspect
  it is called incorrectly at runtime
