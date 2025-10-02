# LLMSchema

Extract database schemas into LLM-optimized documentation. Supports PostgreSQL, MySQL, and SQLite with token-efficient text or markdown output.

## Features

- Multiple database support (PostgreSQL, MySQL, SQLite)
- Compact, LLM-friendly output format
- Complete schema extraction (tables, columns, types, relationships, indexes, constraints)
- Multi-file output on a per-table basis for efficiency
- CLI-friendly with stdout or file output

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

## Usage

```bash
# Multi-file output (recommended)
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -d llm-docs/db-schema

# Single file output (recommended for small schemas)
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -o schema.txt

# Specific tables only
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -t "users,posts" -o schema.txt

# Markdown format (More readable for humans but more verbose)
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -f markdown -d llm-docs/db-schema
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

## Workflow

Generate schema docs (ideally automate by coupling command with db migrations):

```bash
llmschema --db-url "$DATABASE_URL" -d llm-docs/db-schema
```

Reference in your `CLAUDE.md` or `.cursorrules`:

```markdown
## Database Schema

The current database schema is documented in `llm-docs/db-schema/`.

- Read `_overview.txt` to see all available tables
- Load specific table files (e.g., `users.txt`) when working with that table
- Each file contains: columns, types, constraints, relationships, and indexes

When working with database-related code:
- Check `_overview.txt` to understand the overall structure
- Reference specific table files to understand details
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

## Connection String Formats

```bash
# PostgreSQL
postgres://username:password@host:port/database

# MySQL
mysql://username:password@tcp(host:port)/database

# SQLite
sqlite://path/to/database.db
```

## Contributing

Contributions are welcome! Please open an issue or pull request on GitHub.

## License

MIT License - see LICENSE file for details.
