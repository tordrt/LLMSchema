# LLMSchema

Generate database schema documentation optimized for AI agents. Extracts schemas from PostgreSQL, MySQL, and SQLite into markdown files that AI assistants can efficiently browse and which humans can easily reference.

**Primary use:** AI agent consumption (Claude Code, Cursor, etc.)

**Secondary use:** Human-readable documentation

## Features

- Simple and concise markdown output with structured tables
- Multi-file output (one file per table) for efficient context referencing and browsing by agents
- Complete schema extraction: columns, types, relationships, indexes, constraints
- Multiple database support: PostgreSQL, MySQL, SQLite.

## Installation

```bash
go install github.com/tordrt/llmschema/cmd/llmschema@latest
```

Or build from source:

```bash
git clone https://github.com/tordrt/llmschema.git
cd llmschema
go build -o llmschema ./cmd/llmschema
```

## Workflow

**1. Generate schema documentation** (ideally automate this with your DB migrations):
```bash
llmschema --db-url "$DATABASE_URL" -d llm-docs/db-schema
```

**2. Add instructions to your AI context file** (`CLAUDE.md`, `.cursorrules`, etc.):
```markdown
## Database Schema

Schema docs are in `llm-docs/db-schema/`:
- `_overview.md` - lists all tables and their relationships
- `<table_name>.md` - detailed schema for each table

When working with database code, read `_overview.md` for an overview and load specific table files as needed.
```

**3. AI agents can now:** Browse the overview to understand structure, then load specific tables on-demand for efficient context usage.

## Usage

```bash
# Multi-file output (recommended)
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -d llm-docs/db-schema

# Single file output
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -o schema.md

# Specific tables only
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -t "users,posts" -d output

# Exclude specific tables
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -e "migrations,audit_logs" -d output
```

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

| Flag | Short | Description                                          | Default |
|------|-------|------------------------------------------------------|---------|
| `--db-url` | - | Database connection string (required)                | - |
| `--output` | `-o` | Output file path                                     | stdout |
| `--output-dir` | `-d` | Output directory for multi-file output (Recommended) | - |
| `--tables` | `-t` | Comma-separated list of tables to extract            | All tables |
| `--exclude-tables` | `-e` | Comma-separated list of tables to exclude            | - |
| `--schema` | `-s` | Database schema name (PostgreSQL/MySQL)              | `public` for PostgreSQL, auto-detected for MySQL |


## Output Format

Multi-file output creates an overview file plus one file per table:

### `_overview.md`

```markdown
# Schema Overview

Each table has a corresponding file: `<table_name>.md`

## Tables

- **order_items** (references: orders, products)
- **orders** (references: users)
- **products**
- **users**
```

### `orders.md`

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
