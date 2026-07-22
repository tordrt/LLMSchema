# AGENTS.md

This file provides repository-specific guidance for coding agents working on LLMSchema.

## Project Overview

LLMSchema is a Go 1.23 module that extracts PostgreSQL, MySQL, and SQLite schemas and renders concise Markdown for humans and AI agents. It supports a public library API and a Cobra-based CLI.

Key areas:

- `llmschema.go`: intentionally small public library API
- `cmd/llmschema/`: CLI behavior and flag parsing
- `internal/db/`: database clients and schema extractors
- `internal/schema/`: shared schema types
- `internal/formatter/`: single-file and multi-file Markdown output
- `tests/integration/`: Docker-backed database integration tests

## Behavioral Guidelines

### Think Before Coding

- Inspect the relevant implementation, tests, and nearby conventions before editing.
- State assumptions when requirements or behavior are ambiguous. If different interpretations would materially change the result, ask instead of choosing silently.
- Prefer the simplest approach that meets the request, and point out a meaningfully simpler alternative when one exists.
- Do not conceal uncertainty. Name missing information or conflicting behavior clearly.
- For multi-step work, give a brief plan with a verification check for each step.

### Keep It Simple

- Write the minimum code needed to satisfy the request.
- Do not add speculative features, configuration, abstractions, or defensive handling for impossible cases.
- Avoid one-use abstractions when direct code is clearer.
- Preserve the deliberately small public API. Do not export new identifiers unless the task requires them.
- If an implementation becomes substantially larger than the problem warrants, simplify it before finishing.

### Make Surgical Changes

- Touch only files and lines that directly support the requested change.
- Do not refactor, reformat, or clean up unrelated code.
- Match the style and patterns of the surrounding package.
- Mention unrelated problems instead of fixing or deleting them without approval.
- Remove only imports, variables, functions, or files made obsolete by your own change.
- Preserve user changes already present in the worktree.

### Work Toward Verifiable Outcomes

- Define success in observable terms before implementation.
- For a bug fix, add or update a test that fails for the reported behavior, then make it pass.
- For validation changes, cover both accepted and rejected inputs.
- For refactors, establish passing tests before and after the change when practical.
- Continue until the relevant checks pass or report the exact blocker and what remains unverified.

## Implementation Conventions

- Format changed Go files with `gofmt`.
- Follow standard Go error wrapping with `%w` where callers may need the underlying error.
- Keep database-specific logic in the corresponding `internal/db` implementation and shared output logic in `internal/formatter`.
- Keep the library API and CLI behavior consistent when changing capabilities exposed through both.
- Preserve deterministic Markdown output; formatter changes should assert exact or structurally meaningful output.
- Never commit credentials, database dumps, generated binaries, `test.db`, or files under `output/`.

## Verification

Run the narrowest relevant test while iterating, then complete the applicable checks before finishing:

```bash
# Required for ordinary code changes
gofmt -w <changed-go-files>
go test ./...

# Useful additional checks
go vet ./...
go build ./cmd/llmschema
```

Use `make test-unit` as the repository alias for `go test -v ./...`.

The full integration suite requires Docker and starts PostgreSQL and MySQL containers:

```bash
make test
```

Run it when changing database connection, extraction, cross-database behavior, integration fixtures, or when the user explicitly requests the full suite. For a database-specific change, the corresponding `make test-postgres`, `make test-mysql`, or `make test-sqlite` target may provide a faster focused check. Clean up started services with `make docker-down` when appropriate.

If a required command cannot run because Docker, a database, SQLite tooling, or another dependency is unavailable, report that explicitly; do not claim the affected behavior was verified.

## Change Checklist

Before handing off a change:

1. Confirm every changed line traces to the requested outcome.
2. Add or update focused tests for behavior changes.
3. Run `gofmt` on changed Go files.
4. Run `go test ./...` and any relevant focused or integration checks.
5. Summarize what changed, what was verified, and any remaining limitations.
