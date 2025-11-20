# LLMSchema

[![Go Version](https://img.shields.io/github/go-mod/go-version/tordrt/llmschema)](go.mod)
[![Release](https://img.shields.io/github/v/release/tordrt/llmschema)](https://github.com/tordrt/llmschema/releases)
[![License](https://img.shields.io/github/license/tordrt/llmschema)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/tordrt/llmschema.svg)](https://pkg.go.dev/github.com/tordrt/llmschema)
[![Go Report Card](https://goreportcard.com/badge/github.com/tordrt/llmschema)](https://goreportcard.com/report/github.com/tordrt/llmschema)

**Dead simple database schema docs for LLMs and AI agents.**

LLMSchema extracts database schemas from PostgreSQL, MySQL, and SQLite into concise markdown files. These files are optimized for AI agents (Claude Code, Cursor, etc.) to browse efficiently and for humans to reference easily.

## Why?

Most database documentation tools generate output that is too dense or complex for efficient consumption by Large Language Models. LLMSchema focuses on the bare essentials: table structures, column types, indexes, constraints, and relationships.

By providing a lightweight, structured overview, AI agents can understand your data model without being overwhelmed by irrelevant details.

> **Note:** This tool is intended for development databases to aid AI-assisted coding. Do not rely on it for production-critical documentation.

## Features

- **Concise Markdown Output:** Structured tables and relationships optimized for token efficiency.
- **Multi-file Support:** Generates one file per table for targeted context loading.
- **Broad Compatibility:** Supports PostgreSQL, MySQL, and SQLite.
- **Deep Extraction:** Captures columns, types, foreign keys, indexes, and constraints.

## Installation

### CLI Tool

**With Go installed:**
```bash
go install github.com/tordrt/llmschema/cmd/llmschema@latest
```

**Quick install (macOS/Linux):**
```bash
curl -fsSL https://raw.githubusercontent.com/tordrt/llmschema/main/install.sh | sh
```

### Go Library

```bash
go get github.com/tordrt/llmschema
```

## CLI Usage

### Quick Start

Generate schema documentation for your database:

```bash
llmschema --db-url "postgres://user:pass@localhost:5432/mydb" -d docs/db-schema
```

### Connection Strings

| Database | Format |
|----------|--------|
| **PostgreSQL** | `postgres://username:password@host:port/database` |
| **MySQL** | `mysql://username:password@tcp(host:port)/database` |
| **SQLite** | `sqlite://path/to/database.db` |

### Common Examples

**Filter Specific Tables**
```bash
llmschema --db-url "$DB_URL" -d docs/db-schema -t "users,posts,comments"
```

**Exclude Tables**
```bash
llmschema --db-url "$DB_URL" -d docs/db-schema -e "migrations,audit_logs"
```

**Single File Output**
```bash
llmschema --db-url "$DB_URL" -o schema.md
```

**Automated CI/Migration Integration**
Add to your `Makefile` or migration script to keep docs up-to-date:
```makefile
migrate:
    goose postgres "$(DB_URL)" up
    llmschema --db-url "$(DB_URL)" -d docs/db-schema
```

### Command Line Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--db-url` | | Database connection string (required) | - |
| `--output-dir` | `-d` | Output directory for multi-file output (Recommended) | - |
| `--output` | `-o` | Output file path (for single-file output) | stdout |
| `--tables` | `-t` | Comma-separated list of tables to extract | All tables |
| `--exclude-tables` | `-e` | Comma-separated list of tables to exclude | - |
| `--schema` | `-s` | Database schema name (PostgreSQL/MySQL) | `public` (PG) / Auto (MySQL) |

## AI Context Integration

For tools like **Claude Code** or **Cursor**, simply reference the generated overview file in your project instructions (e.g., `CLAUDE.md` or `.cursorrules`).

```markdown
--- CLAUDE.md ---

@docs/db-schema/_overview.md

<!-- The agent can now read the overview and pull in specific table docs as needed. -->
```

## Library Usage

Use `LLMSchema` programmatically in your Go applications.

```go
package main

import (
    "context"
    "log"
    "github.com/tordrt/llmschema"
)

func main() {
    err := llmschema.ExtractAndFormat(
        context.Background(),
        "postgres://user:pass@localhost:5432/mydb",
        &llmschema.Options{
            ExcludeTables: []string{"migrations"},
        },
        &llmschema.OutputOptions{
            OutputDir: "llm-docs/db-schema",
        },
    )
    if err != nil {
        log.Fatal(err)
    }
}

## Output Format

Multi-file output creates an overview file plus one file per table:

### `docs/db-schema/_overview.md`

```markdown
# Schema Overview

Each table has a corresponding file: `docs/db-schema/<table_name>.md`

## Tables

- **order_items** (references: orders, products)
- **orders** (references: users)
- **products**
- **users**
```

### `docs/db-schema/orders.md`

```markdown
## orders

| Column       | Type                                                                                                                 |
|--------------|----------------------------------------------------------------------------------------------------------------------|
| id           | PK integer NOT NULL DEFAULT nextval('orders_id_seq'::regclass)                                                      |
| user_id      | integer NOT NULL                                                                                                     |
| total_amount | numeric NOT NULL                                                                                                     |
| order_date   | timestamp DEFAULT CURRENT_TIMESTAMP                                                                                  |
| status       | order_status (pending, processing, shipped, delivered, cancelled) DEFAULT 'pending'::order_status                    |

### Index

- idx_status on (status)
- idx_user_date on (user_id, order_date)

### References

- user_id → users.id (many orders to one users)

### Referenced by

- order_items.order_id → id (many order_items to one orders)
```

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests for new features, database support, or bug fixes.

## License

MIT License - see [LICENSE](LICENSE) file for details.
