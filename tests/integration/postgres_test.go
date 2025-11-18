//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/tordrt/llmschema/internal/db"
)

func TestPostgresExtraction(t *testing.T) {
	ctx := context.Background()

	// Use environment variable if set, otherwise use default test connection string
	connString := os.Getenv("POSTGRES_TEST_URL")
	if connString == "" {
		connString = "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
	}

	// Create client
	client, err := db.NewPostgresClient(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer client.Close(ctx)

	// Create extractor
	extractor := db.NewExtractor(client, "public")

	// Extract schema
	s, err := extractor.ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to extract schema: %v", err)
	}

	// Verify tables exist
	expectedTables := []string{"users", "products", "orders", "order_items"}
	verifyTablesExist(t, s, expectedTables)

	// Verify users table structure
	table := findTable(s, "users")
	if table == nil {
		t.Fatal("Users table not found")
	}
	verifyPrimaryKey(t, table, []string{"id"})
	expectedColumns := []string{"id", "username", "email", "status", "created_at"}
	verifyColumns(t, table, expectedColumns)

	// Verify foreign key relationships
	verifyForeignKey(t, s, "orders", "user_id", "users")
}

func TestPostgresSpecificTables(t *testing.T) {
	ctx := context.Background()

	connString := os.Getenv("POSTGRES_TEST_URL")
	if connString == "" {
		connString = "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
	}

	client, err := db.NewPostgresClient(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer client.Close(ctx)

	extractor := db.NewExtractor(client, "public")

	// Extract only users and orders tables
	schema, err := extractor.ExtractSchema(ctx, []string{"users", "orders"})
	if err != nil {
		t.Fatalf("Failed to extract schema: %v", err)
	}

	if len(schema.Tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(schema.Tables))
	}

	tableMap := make(map[string]bool)
	for _, table := range schema.Tables {
		tableMap[table.Name] = true
	}

	if !tableMap["users"] || !tableMap["orders"] {
		t.Error("Expected users and orders tables")
	}

	if tableMap["products"] || tableMap["order_items"] {
		t.Error("Should not include products or order_items tables")
	}
}
