.PHONY: build test test-unit test-integration docker-up docker-down docker-clean \
	test-postgres test-postgres-file test-postgres-dir \
	test-mysql test-mysql-file test-mysql-dir \
	test-sqlite test-sqlite-file test-sqlite-dir \
	test-all setup-sqlite clean logs logs-postgres logs-mysql help

# Build the binary
build:
	go build -o llmschema ./cmd/llmschema

# Run all tests
test: test-unit test-integration

# Run unit tests (when they exist)
test-unit:
	go test -v ./...

# Start all test databases
docker-up:
	docker-compose up -d
	@echo "Waiting for databases to be ready..."
	@sleep 5
	@docker-compose ps

# Start only PostgreSQL
docker-up-postgres:
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3

# Start only MySQL
docker-up-mysql:
	docker-compose up -d mysql
	@echo "Waiting for MySQL to be ready..."
	@sleep 3

# Stop all test databases
docker-down:
	docker-compose down

# Stop and remove volumes
docker-clean:
	docker-compose down -v

# Setup SQLite test database
setup-sqlite:
	@echo "Creating SQLite test database..."
	@sqlite3 test.db < test_sqlite_schema.sql
	@echo "SQLite test database created at test.db"

# Test PostgreSQL output (stdout)
test-postgres: build docker-up-postgres
	@echo "\n=== Testing PostgreSQL ==="
	./llmschema --db-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"

# Test PostgreSQL output (to file)
test-postgres-file: build docker-up-postgres
	@mkdir -p output
	@echo "\n=== Testing PostgreSQL (output to file) ==="
	./llmschema --db-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable" -o output/postgres-schema.md
	@echo "Output written to output/postgres-schema.md"

# Test PostgreSQL output (multi-file to directory)
test-postgres-dir: build docker-up-postgres
	@mkdir -p output
	@echo "\n=== Testing PostgreSQL (multi-file output) ==="
	./llmschema --db-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable" -d output/postgres-schema/
	@echo "Multi-file output written to output/postgres-schema/"

# Test MySQL output (stdout)
test-mysql: build docker-up-mysql
	@echo "\n=== Testing MySQL ==="
	./llmschema --db-url "mysql://root:testpassword@tcp(localhost:3306)/testdb"

# Test MySQL output (to file)
test-mysql-file: build docker-up-mysql
	@mkdir -p output
	@echo "\n=== Testing MySQL (output to file) ==="
	./llmschema --db-url "mysql://root:testpassword@tcp(localhost:3306)/testdb" -o output/mysql-schema.md
	@echo "Output written to output/mysql-schema.md"

# Test MySQL output (multi-file to directory)
test-mysql-dir: build docker-up-mysql
	@mkdir -p output
	@echo "\n=== Testing MySQL (multi-file output) ==="
	./llmschema --db-url "mysql://root:testpassword@tcp(localhost:3306)/testdb" -d output/mysql-schema/
	@echo "Multi-file output written to output/mysql-schema/"

# Test SQLite output (stdout)
test-sqlite: build setup-sqlite
	@echo "\n=== Testing SQLite ==="
	./llmschema --db-url "sqlite://test.db"

# Test SQLite output (to file)
test-sqlite-file: build setup-sqlite
	@mkdir -p output
	@echo "\n=== Testing SQLite (output to file) ==="
	./llmschema --db-url "sqlite://test.db" -o output/sqlite-schema.md
	@echo "Output written to output/sqlite-schema.md"

# Test SQLite output (multi-file to directory)
test-sqlite-dir: build setup-sqlite
	@mkdir -p output
	@echo "\n=== Testing SQLite (multi-file output) ==="
	./llmschema --db-url "sqlite://test.db" -d output/sqlite-schema/
	@echo "Multi-file output written to output/sqlite-schema/"

# Run integration tests against all databases
test-integration: build docker-up setup-sqlite
	@echo "\n=== Running integration tests ==="
	go test -v -tags=integration ./tests/integration/...

# Quick test - build and test all databases
test-all: build docker-up setup-sqlite
	@echo "\n=== Testing PostgreSQL ==="
	./llmschema --db-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
	@echo "\n=== Testing MySQL ==="
	./llmschema --db-url "mysql://root:testpassword@tcp(localhost:3306)/testdb"
	@echo "\n=== Testing SQLite ==="
	./llmschema --db-url "sqlite://test.db"

# Show database logs
logs:
	docker-compose logs -f

# Show PostgreSQL logs
logs-postgres:
	docker-compose logs -f postgres

# Show MySQL logs
logs-mysql:
	docker-compose logs -f mysql

# Clean generated files
clean:
	@echo "Cleaning generated schema files..."
	@rm -rf output/ test.db
	@echo "Clean complete"

# Help
help:
	@echo "LLMSchema Makefile Commands:"
	@echo ""
	@echo "  make build              - Build the llmschema binary"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run unit tests"
	@echo "  make test-integration   - Run integration tests against databases"
	@echo ""
	@echo "  make docker-up          - Start all test databases"
	@echo "  make docker-up-postgres - Start only PostgreSQL"
	@echo "  make docker-up-mysql    - Start only MySQL"
	@echo "  make docker-down        - Stop all test databases"
	@echo "  make docker-clean       - Stop databases and remove volumes"
	@echo ""
	@echo "  make test-postgres      - Build and test PostgreSQL (stdout)"
	@echo "  make test-postgres-file - Build and test PostgreSQL (output to file)"
	@echo "  make test-postgres-dir  - Build and test PostgreSQL (multi-file to directory)"
	@echo "  make test-mysql         - Build and test MySQL (stdout)"
	@echo "  make test-mysql-file    - Build and test MySQL (output to file)"
	@echo "  make test-mysql-dir     - Build and test MySQL (multi-file to directory)"
	@echo "  make test-sqlite        - Build and test SQLite (stdout)"
	@echo "  make test-sqlite-file   - Build and test SQLite (output to file)"
	@echo "  make test-sqlite-dir    - Build and test SQLite (multi-file to directory)"
	@echo "  make test-all           - Build and test all databases (stdout)"
	@echo ""
	@echo "  make clean              - Remove generated schema files and test databases"
	@echo ""
	@echo "  make logs               - Show logs from all databases"
	@echo "  make logs-postgres      - Show PostgreSQL logs"
	@echo "  make logs-mysql         - Show MySQL logs"