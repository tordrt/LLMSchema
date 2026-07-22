//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tordrt/llmschema"
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

func TestSQLiteQuotedIdentifiersThroughPublicAPI(t *testing.T) {
	const (
		tableName = `order details"'archive`
		indexName = `parent id"'index`
	)

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "quoted-identifiers.db")
	client, err := db.NewSQLiteClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}

	statements := []string{
		`CREATE TABLE "parent table" ("id" INTEGER PRIMARY KEY)`,
		`CREATE TABLE "order details""'archive" (
			"id" INTEGER PRIMARY KEY,
			"email address" TEXT UNIQUE,
			"parent id" INTEGER REFERENCES "parent table"("id")
		)`,
		`CREATE INDEX "parent id""'index" ON "order details""'archive"("parent id")`,
	}
	for _, statement := range statements {
		if _, err := client.GetDB().ExecContext(ctx, statement); err != nil {
			_ = client.Close()
			t.Fatalf("Failed to create SQLite test schema: %v", err)
		}
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Failed to close SQLite test database: %v", err)
	}

	for _, opts := range []*llmschema.Options{nil, {Tables: []string{tableName}}} {
		s, err := llmschema.ExtractSchema(ctx, "sqlite://"+dbPath, opts)
		if err != nil {
			t.Fatalf("ExtractSchema(%v) failed: %v", opts, err)
		}

		table := findTable(s, tableName)
		if table == nil {
			t.Fatalf("ExtractSchema(%v) did not return %q", opts, tableName)
		}
		verifyPrimaryKey(t, table, []string{"id"})
		verifyUniqueConstraint(t, s, tableName, "email address")
		verifyForeignKey(t, s, tableName, "parent id", "parent table")
		verifyIndex(t, s, tableName, indexName, []string{"parent id"})
	}
}
