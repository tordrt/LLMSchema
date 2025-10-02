# LLMSchema

Generate clean, LLM-optimized documentation from your database schema. Extracts tables, columns, relationships, and constraints into a compact text format that AI coding assistants can easily understand. Makes working with Claude, ChatGPT, and other LLMs on database projects seamless.

## Features

- **PostgreSQL support** - Extract schema from PostgreSQL databases
- **Compact format** - Token-efficient text output optimized for LLMs
- **Complete schema** - Tables, columns, data types, relationships, indexes
- **Flexible filtering** - Extract specific tables or entire schemas
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

## Usage

### Basic Usage

Extract entire schema from a PostgreSQL database:

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb"
```

### Save to File

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -o schema.txt
```

### Extract Specific Tables

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -t "users,posts,comments"
```

### Specify Schema

```bash
llmschema --db-url "postgres://user:password@localhost:5432/mydb" -s "my_schema"
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

## CLI Flags

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--db-url` | - | PostgreSQL connection string | Yes | - |
| `--output` | `-o` | Output file path | No | stdout |
| `--tables` | `-t` | Comma-separated list of tables | No | All tables |
| `--schema` | `-s` | Database schema name | No | public |

## Connection String Format

```
postgres://username:password@host:port/database?sslmode=disable
```

Examples:
- `postgres://postgres:password@localhost:5432/mydb`
- `postgres://user:pass@db.example.com:5432/production?sslmode=require`

## Use Cases

- **AI-Assisted Development** - Provide schema context to Claude Code, Cursor, or ChatGPT
- **Documentation** - Generate readable schema documentation
- **Schema Review** - Quickly understand database structure
- **Migration Planning** - Analyze existing schemas before migrations

## Project Structure

```
llmschema/
├── cmd/
│   └── llmschema/
│       └── main.go           # CLI entry point
├── internal/
│   ├── db/
│   │   ├── postgres.go       # PostgreSQL connection
│   │   └── extractor.go      # Schema extraction logic
│   ├── schema/
│   │   └── types.go          # Data structures
│   └── formatter/
│       └── text.go           # Text output formatter
├── go.mod
└── README.md
```

## License

See LICENSE file for details.
