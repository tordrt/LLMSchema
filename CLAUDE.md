# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LLMSchema is a CLI tool that extracts PostgreSQL database schemas and outputs them in a compact, LLM-optimized format. The tool supports two output formats: text (compact) and markdown (structured tables).

## Building and Running

Build the binary:
```bash
go build -o llmschema ./cmd/llmschema
```

## Architecture

The codebase follows a clean layered architecture:

### Data Flow
1. **CLI layer** (`cmd/llmschema/main.go`): Parses flags, orchestrates execution
2. **Database layer** (`internal/db/`): Manages PostgreSQL connection and schema extraction
3. **Schema layer** (`internal/schema/types.go`): Core data structures (Table, Column, Relation, Index)
4. **Formatter layer** (`internal/formatter/`): Outputs schema in different formats

### Key Components

- **PostgresClient** (`internal/db/postgres.go`): Wraps pgx connection, handles connection lifecycle
- **Extractor** (`internal/db/extractor.go`): Queries information_schema and pg_catalog to extract:
  - Tables and columns (with types, nullability, defaults, unique constraints)
  - Primary keys
  - Foreign key relationships (with N:1 cardinality detection)
  - Indexes (excluding primary key indexes)
- **TextFormatter** (`internal/formatter/text.go`): Produces compact text output
- **MarkdownFormatter** (`internal/formatter/markdown.go`): Produces markdown tables with section headers

### Database Queries

All schema extraction uses PostgreSQL system catalogs:
- `information_schema.tables/columns` for basic table/column metadata
- `information_schema.table_constraints` for primary keys and unique constraints
- `pg_catalog` tables for indexes

The extractor queries are optimized for a single schema at a time (default: "public").

## Testing

No automated tests exist yet. Manual testing uses `test_schema.sql` for setting up a test database.

## Dependencies

- **github.com/jackc/pgx/v5**: PostgreSQL driver (low-level connection handling)
- **github.com/spf13/cobra**: CLI framework (flag parsing, command structure)

## Adding New Features

### Adding a new output format
1. Create new formatter in `internal/formatter/` implementing `Format(*schema.Schema) error`
2. Add format option in `cmd/llmschema/main.go` switch statement
3. Instantiate formatter and call `Format()` method

### Extending schema extraction
1. Add new fields to relevant structs in `internal/schema/types.go`
2. Implement extraction query in `internal/db/extractor.go`
3. Update both formatters to display new data

### Supporting new database types
1. Create new client in `internal/db/` (e.g., `mysql.go`)
2. Implement equivalent extraction queries for that database's system tables
3. Update CLI to detect database type from connection string