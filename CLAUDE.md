# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LLMSchema is a Go tool that extracts database schemas (PostgreSQL, MySQL, SQLite) and generates markdown documentation optimized for AI agent consumption. It produces either single-file or multi-file output with structured tables, relationships, indexes, and constraints.

## Build and Test Commands

```bash
# Build the CLI binary
make build
go build -o llmschema ./cmd/llmschema

# Run all tests
make test

# Run unit tests
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Start test databases (Docker)
make docker-up              # All databases
make docker-up-postgres     # PostgreSQL only
make docker-up-mysql        # MySQL only

# Stop databases
make docker-down
make docker-clean           # Also remove volumes

# Setup SQLite test database
make setup-sqlite

# Quick manual tests
make test-postgres          # PostgreSQL to stdout
make test-mysql             # MySQL to stdout
make test-sqlite            # SQLite to stdout

# Test specific output formats
make test-postgres-file     # Single file output
make test-postgres-dir      # Multi-file output (recommended)
```

## Architecture

### Public API (llmschema.go)

The package exposes a minimal public API surface:

- **`ExtractAndFormat()`** - Recommended one-call function that extracts and formats schema
- **`ExtractSchema()`** - Extracts schema from database URL into `Schema` struct
- **`FormatSchema()`** - Formats a `Schema` to markdown (single or multi-file)
- **`Options`** - Configuration for extraction (Tables, ExcludeTables, SchemaName)
- **`OutputOptions`** - Configuration for output (Writer, OutputDir)

### Internal Structure

- **`internal/schema/types.go`** - Core data structures (`Schema`, `Table`, `Column`, `Relation`, `Index`)
- **`internal/db/`** - Database-specific extractors:
  - `postgres.go`, `postgres_extractor.go` - PostgreSQL client and extraction
  - `mysql.go`, `mysql_extractor.go` - MySQL client and extraction
  - `sqlite.go`, `sqlite_extractor.go` - SQLite client and extraction
- **`internal/formatter/`** - Output formatters:
  - `markdown.go` - Single-file markdown formatter
  - `multifile.go` - Multi-file output (creates `_overview.md` + one file per table)
- **`cmd/llmschema/main.go`** - CLI implementation using Cobra, delegates to public API

### Key Design Patterns

1. **Database abstraction**: Each database has a client (`NewPostgresClient`, `NewMySQLClient`, `NewSQLiteClient`) and an extractor that implements schema extraction for that specific database
2. **URL parsing**: Database type is detected from URL scheme (`postgres://`, `mysql://`, `sqlite://`)
3. **Two-phase processing**: Extract → Format (allows inspection/modification of schema between steps)
4. **Formatter interface**: Both single-file and multi-file formatters implement `Format(*schema.Schema) error`

## Database Connection Strings

```bash
postgres://username:password@host:port/database
mysql://username:password@tcp(host:port)/database
sqlite://path/to/database.db
```

## Testing

Integration tests are in `tests/integration/` and require:
- PostgreSQL container: `postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable`
- MySQL container: `mysql://root:testpassword@tcp(localhost:3306)/testdb`
- SQLite file: `test.db` (created via `make setup-sqlite`)

Test schema files: `test_postgres_schema.sql`, `test_mysql_schema.sql`, `test_sqlite_schema.sql`

## Important Notes

- Multi-file output (`-d` flag) is recommended over single-file (`-o`) for AI agent consumption
- PostgreSQL defaults to `public` schema; MySQL auto-detects from connection string
- Use `--exclude-tables` for migrations, audit logs, etc.
- The public API in `llmschema.go` is intentionally minimal; internal packages provide implementation details