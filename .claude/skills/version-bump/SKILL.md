---
name: version-bump
description: Identify the correct semver version for pending changes and execute a full release for go-sigil. Use this skill when the user asks "what version should this be", "bump the version", "release vX.Y.Z", "tag a release", "create a new release", "what semver bump is this", or "prepare a release". Also use proactively after a bug fix or feature is committed and the user wants to publish it. This skill uses `sigil diff --since <last-tag>` to enumerate changed symbols, infers the semver bump, updates version constants, updates PROGRESS.md and RELEASES.md, commits, tags, pushes, and creates the GitHub release via `gh`.
---

# Version Bump & Release

Determine the correct semver version for pending changes and publish a GitHub release.

## Goal

By the end of this workflow:

1. Know which symbols changed since the last tag
2. Have the correct semver bump applied (patch / minor / major)
3. Version constants updated in source
4. PROGRESS.md and RELEASES.md updated
5. Commit tagged and pushed
6. GitHub release created

## Prerequisites

- `sigil` indexed and synced: `sigil sync .`
- `gh` authenticated: `gh auth status`
- Clean working tree (all changes staged or committed)

## Step 1 — Find the last release tag

```bash
git describe --tags --abbrev=0
```

Store the result as `<last-tag>` (e.g. `v0.1.1`).

## Step 2 — List changed symbols since last tag

```bash
sigil diff --since <last-tag>
```

Read the output:

| Symbol diff output | Semver signal |
|---|---|
| Only `~` (modified) lines | patch |
| Any `+` (added) functions / methods / types | minor |
| Any `-` (deleted) exported symbols or changed interfaces | major |
| Only test files or internal unexported symbols changed | patch |

## Step 3 — Determine next version

Given current version `vMAJOR.MINOR.PATCH`:

- **patch** → `vMAJOR.MINOR.(PATCH+1)`
- **minor** → `vMAJOR.(MINOR+1).0`
- **major** → `v(MAJOR+1).0.0`

State the version and the reason before proceeding. Ask the user to confirm if the
bump type is ambiguous.

## Step 4 — Update version constants

Two files always require updating:

```
internal/constants/constants.go   → AppVersion = "MAJOR.MINOR.PATCH"
internal/mcpserver/server.go      → serverVersion = "vMAJOR.MINOR.PATCH"
```

Use the Edit tool to change both.

## Step 5 — Update PROGRESS.md

Insert a new row at the top of the Release History table (just after the header row):

```markdown
| vX.Y.Z | YYYY-MM-DD | <one-line summary of changes> |
```

The summary should be derived from the `sigil diff` output — name the key symbols or
areas that changed, not raw symbol IDs.

## Step 6 — Update RELEASES.md

Insert the same row at the top of the Release History table in `RELEASES.md`.

## Step 7 — Commit

Stage only the relevant files — never stage `.claude/settings.local.json`:

```bash
git add internal/constants/constants.go \
        internal/mcpserver/server.go \
        PROGRESS.md \
        RELEASES.md \
        <any other changed source files>
git commit -m "Release vX.Y.Z — <short description>

- <bullet 1>
- <bullet 2>"
```

No `Co-Authored-By` trailer.

## Step 8 — Tag and push

```bash
git tag vX.Y.Z
git push origin main
git push origin vX.Y.Z
```

## Step 9 — Create GitHub release

```bash
gh release create vX.Y.Z \
  --title "vX.Y.Z" \
  --notes "## What's Changed
- <change 1>
- <change 2>

**Full Changelog**: https://github.com/teocci/go-sigil/compare/<last-tag>...vX.Y.Z"
```

## Step 10 — Verify

```bash
sigil version          # → sigil X.Y.Z
gh release view vX.Y.Z # → release exists with correct notes
```

## Quick Reference — Semver Rules for go-sigil

| Change type | Bump |
|---|---|
| Bug fix, test addition, doc update | patch |
| New CLI flag or command (backward-compatible) | minor |
| New MCP tool or sigil query service | minor |
| Removed/renamed exported symbol or interface change | major |
| SQLite schema version bump | major |
