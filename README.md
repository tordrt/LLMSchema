# LLMSchema

[![Go Version](https://img.shields.io/github/go-mod/go-version/tordrt/llmschema)](go.mod)
[![Release](https://img.shields.io/github/v/release/tordrt/llmschema)](https://github.com/tordrt/llmschema/releases)
[![License](https://img.shields.io/github/license/tordrt/llmschema)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/tordrt/llmschema.svg)](https://pkg.go.dev/github.com/tordrt/llmschema)
[![Go Report Card](https://goreportcard.com/badge/github.com/tordrt/llmschema)](https://goreportcard.com/report/github.com/tordrt/llmschema)

**Dead simple database schema docs for LLM's and AI agents**

Generate simple database schema documentation for LLM's and AI agents. Extracts schemas from PostgreSQL, MySQL, and SQLite into markdown files that agents can efficiently browse and which humans can easily reference.

**Primary use:** AI agent consumption (Claude Code, Cursor, etc.)

## Table of Contents

- [Why?](#why)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Examples](#examples)
- [Connection String Formats](#connection-string-formats)
- [CLI Options](#cli-options)
- [Library Usage](#library-usage)
- [Output Format](#output-format)
- [Contributing](#contributing)
- [License](#license)

## Why?

I wanted an easy way to provide db schema context to Claude Code.

Existing database documentation tools felt like overkill for solving this. Yes, there are many schema documentation tools out there—but I wanted something **simple and concise** that AI agents could easily browse and consume.

This tool isn't about generating beautiful, incredibly detailed, or dense documentation. It's about providing **just the barebones**: table structures, types, indexes, constraints and relationships in a format that's trivial for agents to understand. I found that the database MCPs were overly complex for what I needed.

I vibecoded this over a couple days and have been using successfully for more than a month now. My workflow:

- **Big tasks:** I write detailed prompts and directly reference relevant table docs.
- **Simple tasks:** Importing the overview in `CLAUDE.md` gives Claude Code enough context to pull in relevant table docs on demand

It's been working well for me, and I hope it helps others who want a straightforward way to give their AI agents DB schema context.

> **Note:** This tool was built to solve a specific need in AI-assisted development. It works well for this purpose, but is best used with development databases. Don't rely on it for production-critical documentation or operations requiring high reliability.

## Features

- Simple and concise markdown output with structured tables
- Multi-file output (one file per table) for efficient context referencing and browsing by agents
- Extracts columns, types, relationships, indexes, constraints
- Multiple database support: PostgreSQL, MySQL, SQLite.

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

$Or$ build from source.

### Go Library

```bash
go get github.com/tordrt/llmschema
```

## Quick Start

**1. Generate schema documentation** (ideally automate this to run after DB migrations):

```bash
llmschema --db-url "$DATABASE_URL" -d docs/db-schema
```

**2. Add instructions to your AI context file** (`CLAUDE.md`, `.cursorrules`, etc.)

For claude code I like to just import the overview file directly in the CLAUDE.md:

```markdown
--- CLAUDE.md ---

@docs/db-schema/_overview.md

<!-- Add whatever other database releated context you want below -->

--- CLAUDE.md ---
```

**3. AI agents now:** Has context above your DB structure and can load specific tables on-demand for efficient context usage. You can also easily directly reference table doc files in your prompts

## Examples

```bash
# Multi-file output (recommended)
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -d docs/db-schema

# Single file output
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -o schema.md

# Specific tables only
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -t "users,posts" -d docs/db-schema

# Exclude specific tables
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -e "migrations,audit_logs" -d docs/db-schema
```

**Integrate with migration tools to always have up to date docs:**

Add a target to your Makefile that runs migrations and generates schema docs:

```makefile
# Example Makefile integration
migrate:
    # Replace with your migration tool (goose, migrate, etc.)
    goose -dir migrations postgres "$(DATABASE_URL)" up
    llmschema --db-url "$(DATABASE_URL)" -d docs/db-schema
```

Alternatively, integrate llmschema into git hooks (e.g., post-migration), CI/CD pipelines, or other automation workflows.

## Connection String Formats

```bash
# PostgreSQL
postgres://username:password@host:port/database

# MySQL
mysql://username:password@tcp(host:port)/database

# SQLite
sqlite://path/to/database.db
```

## CLI Options

| Flag               | Short | Description                                          | Default                                          |
| ------------------ | ----- | ---------------------------------------------------- | ------------------------------------------------ |
| `--db-url`         | -     | Database connection string (required)                | -                                                |
| `--output`         | `-o`  | Output file path                                     | stdout                                           |
| `--output-dir`     | `-d`  | Output directory for multi-file output (Recommended) | -                                                |
| `--tables`         | `-t`  | Comma-separated list of tables to extract            | All tables                                       |
| `--exclude-tables` | `-e`  | Comma-separated list of tables to exclude            | -                                                |
| `--schema`         | `-s`  | Database schema name (PostgreSQL/MySQL)              | `public` for PostgreSQL, auto-detected for MySQL |

## Library Usage

You can also use LLMSchema as a Go library in your projects.

### Quick Start (Recommended)

For most use cases, use `ExtractAndFormat` to extract and save schema in one call:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/tordrt/llmschema"
)

func main() {
    ctx := context.Background()

    err := llmschema.ExtractAndFormat(
        ctx,
        "postgres://user:pass@localhost:5432/mydb",
        &llmschema.Options{
            ExcludeTables: []string{"migrations"},
        },
        &llmschema.OutputOptions{
            OutputDir: "llm-docs/db-schema",
        },
    )
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Multi-Step Workflow

For more control, extract and format separately:

```go
ctx := context.Background()

// Step 1: Extract schema
schema, err := llmschema.ExtractSchema(ctx, "postgres://user:pass@localhost:5432/mydb", &llmschema.Options{
    Tables:        []string{"users", "posts"}, // optional: specific tables
    ExcludeTables: []string{"migrations"},      // optional: exclude tables
    SchemaName:    "public",                    // optional: schema name
})
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    return
}

// Step 2: Format to stdout
err = llmschema.FormatSchema(schema, &llmschema.OutputOptions{
    Writer: os.Stdout, // optional: defaults to stdout
})

// Or save to multi-file output
err = llmschema.FormatSchema(schema, &llmschema.OutputOptions{
    OutputDir: "llm-docs/db-schema",
})
```

## Output Format

Multi-file output creates an overview file plus one file per table:

> **See examples:** The output examples below show the structure and format of generated documentation.

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

This is a new project I made for my own usage and likely has rough edges. Issue reports and contributions are very welcome!

**Areas for improvement:**

- Output format refinements
- Support for additional databases
- Alternative output formats
- New features and ideas

Please report bugs or suggest improvements at https://github.com/tordrt/llmschema/issues.

## License

MIT License - see LICENSE file for details.
