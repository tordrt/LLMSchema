.PHONY: build test test-postgres test-mysql test-sqlite test-integration docker-up docker-down docker-clean help

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

# Test PostgreSQL output (text format)
test-postgres: build docker-up-postgres
	@echo "\n=== Testing PostgreSQL (text format) ==="
	./llmschema --pg-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"

# Test PostgreSQL output (markdown format)
test-postgres-md: build docker-up-postgres
	@echo "\n=== Testing PostgreSQL (markdown format) ==="
	./llmschema --pg-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable" --format markdown

# Test MySQL output (text format)
test-mysql: build docker-up-mysql
	@echo "\n=== Testing MySQL (text format) ==="
	./llmschema --mysql-url "root:testpassword@tcp(localhost:3306)/testdb"

# Test MySQL output (markdown format)
test-mysql-md: build docker-up-mysql
	@echo "\n=== Testing MySQL (markdown format) ==="
	./llmschema --mysql-url "root:testpassword@tcp(localhost:3306)/testdb" --format markdown

# Setup SQLite test database
setup-sqlite:
	@echo "Creating SQLite test database..."
	@sqlite3 test.db < test_sqlite_schema.sql
	@echo "SQLite test database created at test.db"

# Test SQLite output (text format)
test-sqlite: build setup-sqlite
	@echo "\n=== Testing SQLite (text format) ==="
	./llmschema --sqlite test.db

# Test SQLite output (markdown format)
test-sqlite-md: build setup-sqlite
	@echo "\n=== Testing SQLite (markdown format) ==="
	./llmschema --sqlite test.db --format markdown

# Run integration tests against all databases
test-integration: build docker-up setup-sqlite
	@echo "\n=== Running integration tests ==="
	go test -v -tags=integration ./tests/integration/...

# Quick test - build and test all databases
test-all: build docker-up setup-sqlite
	@echo "\n=== Testing PostgreSQL ==="
	./llmschema --db-url "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
	@echo "\n=== Testing MySQL ==="
	./llmschema --mysql-url "root:testpassword@tcp(localhost:3306)/testdb"
	@echo "\n=== Testing SQLite ==="
	./llmschema --sqlite test.db

# Show database logs
logs:
	docker-compose logs -f

# Show PostgreSQL logs
logs-postgres:
	docker-compose logs -f postgres

# Show MySQL logs
logs-mysql:
	docker-compose logs -f mysql

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
	@echo "  make test-postgres      - Build and test PostgreSQL (text format)"
	@echo "  make test-postgres-md   - Build and test PostgreSQL (markdown)"
	@echo "  make test-mysql         - Build and test MySQL (text format)"
	@echo "  make test-mysql-md      - Build and test MySQL (markdown)"
	@echo "  make test-sqlite        - Build and test SQLite (text format)"
	@echo "  make test-sqlite-md     - Build and test SQLite (markdown)"
	@echo "  make test-all           - Build and test all databases"
	@echo ""
	@echo "  make logs               - Show logs from all databases"
	@echo "  make logs-postgres      - Show PostgreSQL logs"
	@echo "  make logs-mysql         - Show MySQL logs"