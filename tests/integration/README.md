# Integration Tests

This directory contains integration tests for LLMSchema that test against real database instances.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21 or later

## Running Tests

### Quick Start

The easiest way to run integration tests is using the Makefile:

```bash
# Start databases and run all integration tests
make test-integration

# Or run the full test suite including unit tests
make test
```

### Manual Testing

1. Start the test databases:
```bash
make docker-up
# or
docker-compose up -d
```

2. Run integration tests:
```bash
go test -v -tags=integration ./tests/integration/...
```

3. Stop the databases when done:
```bash
make docker-down
```

### Testing Individual Databases

Test only PostgreSQL:
```bash
make docker-up-postgres
go test -v -tags=integration ./tests/integration/ -run TestPostgres
```

Test only MySQL:
```bash
make docker-up-mysql
go test -v -tags=integration ./tests/integration/ -run TestMySQL
```

## Environment Variables

You can override the default test database connection strings:

```bash
export POSTGRES_TEST_URL="postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
export MYSQL_TEST_URL="root:testpassword@tcp(localhost:3306)/testdb"

go test -v -tags=integration ./tests/integration/...
```

## Test Structure

- `postgres_test.go` - Integration tests for PostgreSQL schema extraction
- `mysql_test.go` - Integration tests for MySQL schema extraction

Each test file contains:
- Full schema extraction tests (all tables)
- Specific table extraction tests (subset of tables)
- Verification of table structure, columns, primary keys, and foreign key relationships

## Build Tags

Integration tests use the `integration` build tag to prevent them from running during normal `go test ./...` commands. This is because they require external database dependencies.

To run integration tests, you must explicitly include the tag:
```bash
go test -tags=integration ./tests/integration/...
```

## Troubleshooting

### Connection Refused
If tests fail with connection errors, ensure the databases are running:
```bash
docker-compose ps
```

If containers aren't healthy, check logs:
```bash
make logs
```

### Port Conflicts
If ports 5432 or 3306 are already in use, you'll need to either:
1. Stop the conflicting service
2. Modify the ports in `docker-compose.yml` and update connection strings accordingly

### Clean Start
To completely reset the test databases:
```bash
make docker-clean  # Stops containers and removes volumes
make docker-up     # Starts fresh containers
```