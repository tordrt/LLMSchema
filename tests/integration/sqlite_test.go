//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/tordrt/llmschema/internal/db"
)

func TestSQLiteExtraction(t *testing.T) {
	ctx := context.Background()

	// Use environment variable if set, otherwise use default test database
	dbPath := os.Getenv("SQLITE_TEST_PATH")
	if dbPath == "" {
		dbPath = "../../test.db"
	}

	// Create client
	client, err := db.NewSQLiteClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer client.Close()

	// Create extractor
	extractor := db.NewSQLiteExtractor(client)

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

	// Verify unique constraint on username
	verifyUniqueConstraint(t, s, "users", "username")

	// Verify foreign key relationships
	verifyForeignKey(t, s, "orders", "user_id", "users")

	// Verify indexes
	verifyIndex(t, s, "products", "idx_category", []string{"category"})
}

func TestSQLiteSpecificTables(t *testing.T) {
	ctx := context.Background()

	dbPath := os.Getenv("SQLITE_TEST_PATH")
	if dbPath == "" {
		dbPath = "../../test.db"
	}

	client, err := db.NewSQLiteClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer client.Close()

	extractor := db.NewSQLiteExtractor(client)

	// Extract only users and products tables
	schema, err := extractor.ExtractSchema(ctx, []string{"users", "products"})
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

	if !tableMap["users"] || !tableMap["products"] {
		t.Error("Expected users and products tables")
	}

	if tableMap["orders"] || tableMap["order_items"] {
		t.Error("Should not include orders or order_items tables")
	}
}
