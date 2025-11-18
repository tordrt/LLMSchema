//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/tordrt/llmschema/internal/db"
)

func TestMySQLExtraction(t *testing.T) {
	ctx := context.Background()

	// Use environment variable if set, otherwise use default test connection string
	connString := os.Getenv("MYSQL_TEST_URL")
	if connString == "" {
		connString = "root:testpassword@tcp(localhost:3306)/testdb"
	}

	// Create client
	client, err := db.NewMySQLClient(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer client.Close()

	// Create extractor
	extractor := db.NewMySQLExtractor(client, "testdb")

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

	// Verify ENUM type extraction for status column
	expectedEnumValues := []string{"active", "inactive", "banned"}
	verifyEnumValues(t, s, "users", "status", expectedEnumValues)

	// Verify foreign key relationships
	verifyForeignKey(t, s, "orders", "user_id", "users")
}

func TestMySQLSpecificTables(t *testing.T) {
	ctx := context.Background()

	connString := os.Getenv("MYSQL_TEST_URL")
	if connString == "" {
		connString = "root:testpassword@tcp(localhost:3306)/testdb"
	}

	client, err := db.NewMySQLClient(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer client.Close()

	extractor := db.NewMySQLExtractor(client, "testdb")

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
