# LLMSchema

Generate clean, LLM-optimized documentation from your database schema. Extracts tables, columns, relationships, and constraints into a compact format that AI coding assistants can easily understand and reference.

## Why LLMSchema?

When working with AI coding assistants on database-heavy projects, providing schema context is essential. LLMSchema generates token-efficient schema documentation that you can:

- **Version control** alongside your codebase
- **Auto-generate** on database migrations
- **Reference** in AI agent instructions (e.g., `CLAUDE.md`, `.cursorrules`)
- **Keep in sync** with your actual database structure

## Features

- **Multiple database support** - PostgreSQL, MySQL, SQLite
- **Compact format** - Token-efficient text output optimized for LLMs
- **Complete schema** - Tables, columns, data types, relationships, indexes, constraints
- **Flexible filtering** - Extract specific tables or entire schemas
- **Multiple formats** - Text (compact) or Markdown (structured tables)
- **CLI-friendly** - Output to stdout or file

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

## Quick Start

### Extract Schema

```bash
# PostgreSQL
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -o llm-docs/db-schema.txt

# MySQL
llmschema --db-url "mysql://user:password@tcp(localhost:3306)/mydb" -o llm-docs/db-schema.txt

# SQLite
llmschema --db-url "sqlite://database.db" -o llm-docs/db-schema.txt
```

### Extract Specific Tables

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -t "users,posts,comments" -o llm-docs/core-tables.txt
```

### Markdown Format

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -f markdown -o llm-docs/db-schema.md
```

### Specify Schema (PostgreSQL/MySQL)

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -s "my_schema" -o llm-docs/db-schema.txt
```

## Output Format

```
TABLE users (PK: id)
  id: bigserial NOT NULL
  email: varchar UNIQUE NOT NULL
  created_at: timestamp DEFAULT now()

  RELATIONS:
    → posts.user_id (N:1)
    → profiles.user_id (N:1)

  INDEXES:
    idx_users_email (email) UNIQUE

TABLE posts (PK: id)
  id: bigserial NOT NULL
  user_id: bigint NOT NULL
  title: varchar NOT NULL
  content: text
  created_at: timestamp DEFAULT now()

  RELATIONS:
    → users.id (N:1)

  INDEXES:
    idx_posts_user_id (user_id)
```

## Recommended Workflow

### 1. Create a Documentation Directory

```bash
mkdir -p llm-docs/db-schema
```

### 2. Generate Schema Documentation

Manually or as part of your migration process:

```bash
llmschema --db-url "$DATABASE_URL" -o llm-docs/db-schema/schema.txt
```

### 3. Integrate with AI Agents

Add to your `CLAUDE.md` (for Claude Code) or `.cursorrules` (for Cursor):

```markdown
## Database Schema

The current database schema is documented in `llm-docs/db-schema`.
When working with database-related code:
- Reference this file to understand table structures and relationships
- Check column types, constraints, and indexes
- Verify foreign key relationships before writing queries
```

### 4. Automate with Migrations

Add to your migration scripts or CI/CD pipeline:

```bash
#!/bin/bash
# After running migrations
llmschema --db-url "$DATABASE_URL" -o llm-docs/db-schema/schema.txt
git add llm-docs/db-schema/schema.txt
git commit -m "Update database schema documentation"
```

## CLI Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--db-url` | - | Database connection string (required) | - |
| `--output` | `-o` | Output file path | stdout |
| `--output-dir` | `-d` | Output directory for multi-file output | - |
| `--tables` | `-t` | Comma-separated list of tables to extract | All tables |
| `--schema` | `-s` | Database schema name (PostgreSQL/MySQL) | `public` for PostgreSQL, auto-detected for MySQL |
| `--format` | `-f` | Output format: `text` or `markdown` | `text` |
| `--split-threshold` | - | Split into multiple files when table count exceeds this (requires --output-dir) | 0 (disabled) |

## Connection String Formats

**PostgreSQL:**
```
postgres://username:password@host:port/database?sslmode=disable
```

**MySQL:**
```
mysql://username:password@tcp(host:port)/database
```

**SQLite:**
```
sqlite://path/to/database.db
```

## Contributing

Contributions are welcome! Please open an issue or pull request on GitHub.

## License

MIT License - see LICENSE file for details.
