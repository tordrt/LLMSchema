# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LLMSchema is a CLI tool that extracts database schemas from PostgreSQL, MySQL, and SQLite databases, outputting them in compact, LLM-optimized formats. The tool supports text (compact) and markdown (structured tables) output formats, with optional multi-file output for large schemas.

## Building and Testing

Build the binary:
```bash
go build -o llmschema ./cmd/llmschema
```

Run all tests (unit + integration):
```bash
make test
```

Run integration tests only:
```bash
make test-integration
```

Test specific databases:
```bash
make test-postgres    # PostgreSQL text output
make test-mysql       # MySQL text output
make test-sqlite      # SQLite text output
make test-all         # All databases
```

Start/stop test databases (requires Docker):
```bash
make docker-up        # Start all test databases
make docker-down      # Stop all test databases
make docker-clean     # Stop and remove volumes
```

Run linter:
```bash
golangci-lint run
```

## Architecture

The codebase follows a clean layered architecture with separation of concerns:

### Data Flow
1. **CLI layer** (`cmd/llmschema/main.go`): Parses flags, detects database type from URL scheme, orchestrates execution
2. **Database layer** (`internal/db/`): Database-specific clients and extractors
3. **Schema layer** (`internal/schema/types.go`): Core data structures (Schema, Table, Column, Relation, Index)
4. **Formatter layer** (`internal/formatter/`): Outputs schema in different formats

### Database Support

All three database types follow the same pattern:

- **Client** (`postgres.go`, `mysql.go`, `sqlite.go`): Wraps database connection, handles connection lifecycle
- **Extractor** (`*_extractor.go`): Queries system catalogs/information_schema to extract:
  - Tables and columns (with types, nullability, defaults, unique constraints, CHECK constraints)
  - Primary keys
  - Foreign key relationships (with N:1 cardinality detection)
  - Indexes (excluding primary key indexes)
  - Enum values (for PostgreSQL USER-DEFINED types)

**PostgreSQL**: Uses `pgx/v5` driver, queries `information_schema` and `pg_catalog`
**MySQL**: Uses `database/sql` with `go-sql-driver/mysql`, queries `information_schema`
**SQLite**: Uses `database/sql` with `go-sqlite3`, queries `sqlite_master` and `pragma` commands

### Database Type Detection

The CLI automatically detects database type from connection string scheme:
- `postgres://` or `postgresql://` ’ PostgreSQL
- `mysql://` ’ MySQL (strips scheme for driver)
- `sqlite://` ’ SQLite (strips scheme to get file path)

### Output Formatters

- **TextFormatter** (`internal/formatter/text.go`): Produces compact text output
- **MarkdownFormatter** (`internal/formatter/markdown.go`): Produces markdown tables with section headers
- **MultiFileFormatter** (`internal/formatter/multifile.go`): Splits schema into multiple files (one per table plus overview)

### Key Data Structures

All defined in `internal/schema/types.go`:

- **Schema**: Contains array of Tables
- **Table**: Name, Columns, Relations, Indexes, PrimaryKey
- **Column**: Name, Type, Nullable, DefaultValue, IsUnique, EnumValues, CheckConstraint
- **Relation**: TargetTable, TargetColumn, SourceColumn, Cardinality (1:1, 1:N, N:1)
- **Index**: Name, Columns, IsUnique

## Testing Infrastructure

Integration tests live in `tests/integration/` and use build tags:
```go
//go:build integration
```

Each database has its own test file and Docker setup:
- PostgreSQL: Uses docker-compose, initializes with `test_postgres_schema.sql`
- MySQL: Uses docker-compose, initializes with `test_mysql_schema.sql`
- SQLite: Uses local file, initializes with `test_sqlite_schema.sql`

Run integration tests with:
```bash
go test -v -tags=integration ./tests/integration/...
```

## Adding New Features

### Adding a new output format
1. Create new formatter in `internal/formatter/` implementing `Format(*schema.Schema) error`
2. Add format option in `cmd/llmschema/main.go` formatOutput() switch statement
3. Instantiate formatter and call `Format()` method

### Extending schema extraction
1. Add new fields to relevant structs in `internal/schema/types.go`
2. Implement extraction queries in relevant `internal/db/*_extractor.go` files
3. Update all three formatters (text, markdown, multifile) to display new data

### Supporting new database types
1. Create new client in `internal/db/` (e.g., `oracle.go`)
2. Create corresponding extractor (e.g., `oracle_extractor.go`)
3. Implement equivalent extraction queries for that database's system tables
4. Update `parseDatabaseURL()` in `cmd/llmschema/main.go` to detect new scheme
5. Add new case in `extractSchema()` function
6. Add integration tests in `tests/integration/`