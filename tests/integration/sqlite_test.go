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
	schema, err := extractor.ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to extract schema: %v", err)
	}

	// Verify tables exist
	expectedTables := []string{"users", "products", "orders", "order_items"}
	if len(schema.Tables) != len(expectedTables) {
		t.Errorf("Expected %d tables, got %d", len(expectedTables), len(schema.Tables))
	}

	tableMap := make(map[string]bool)
	for _, table := range schema.Tables {
		tableMap[table.Name] = true
	}

	for _, tableName := range expectedTables {
		if !tableMap[tableName] {
			t.Errorf("Expected table %s not found in schema", tableName)
		}
	}

	// Verify users table structure
	var usersTable *struct {
		Name       string
		Columns    []struct{ Name, Type string }
		PrimaryKey []string
	}

	for _, table := range schema.Tables {
		if table.Name == "users" {
			usersTable = &struct {
				Name       string
				Columns    []struct{ Name, Type string }
				PrimaryKey []string
			}{
				Name:       table.Name,
				PrimaryKey: table.PrimaryKey,
			}
			for _, col := range table.Columns {
				usersTable.Columns = append(usersTable.Columns, struct{ Name, Type string }{
					Name: col.Name,
					Type: col.Type,
				})
			}
			break
		}
	}

	if usersTable == nil {
		t.Fatal("Users table not found")
	}

	// Check primary key
	if len(usersTable.PrimaryKey) != 1 || usersTable.PrimaryKey[0] != "id" {
		t.Errorf("Expected primary key [id], got %v", usersTable.PrimaryKey)
	}

	// Check columns
	expectedColumns := []string{"id", "username", "email", "status", "created_at"}
	columnMap := make(map[string]bool)
	for _, col := range usersTable.Columns {
		columnMap[col.Name] = true
	}

	for _, colName := range expectedColumns {
		if !columnMap[colName] {
			t.Errorf("Expected column %s not found in users table", colName)
		}
	}

	// Verify unique constraint on username
	var usernameIsUnique bool
	for _, table := range schema.Tables {
		if table.Name == "users" {
			for _, col := range table.Columns {
				if col.Name == "username" && col.IsUnique {
					usernameIsUnique = true
					break
				}
			}
			break
		}
	}

	if !usernameIsUnique {
		t.Error("Expected username column to have unique constraint")
	}

	// Verify foreign key relationships
	var ordersTable *struct {
		Relations []struct {
			TargetTable  string
			SourceColumn string
		}
	}

	for _, table := range schema.Tables {
		if table.Name == "orders" {
			ordersTable = &struct {
				Relations []struct {
					TargetTable  string
					SourceColumn string
				}
			}{}
			for _, rel := range table.Relations {
				ordersTable.Relations = append(ordersTable.Relations, struct {
					TargetTable  string
					SourceColumn string
				}{
					TargetTable:  rel.TargetTable,
					SourceColumn: rel.SourceColumn,
				})
			}
			break
		}
	}

	if ordersTable == nil {
		t.Fatal("Orders table not found")
	}

	// Check that orders has a relation to users
	foundUserRelation := false
	for _, rel := range ordersTable.Relations {
		if rel.TargetTable == "users" && rel.SourceColumn == "user_id" {
			foundUserRelation = true
			break
		}
	}

	if !foundUserRelation {
		t.Error("Expected foreign key relationship from orders.user_id to users not found")
	}

	// Verify indexes
	var productsTable *struct {
		Indexes []struct {
			Name    string
			Columns []string
		}
	}

	for _, table := range schema.Tables {
		if table.Name == "products" {
			productsTable = &struct {
				Indexes []struct {
					Name    string
					Columns []string
				}
			}{}
			for _, idx := range table.Indexes {
				productsTable.Indexes = append(productsTable.Indexes, struct {
					Name    string
					Columns []string
				}{
					Name:    idx.Name,
					Columns: idx.Columns,
				})
			}
			break
		}
	}

	if productsTable == nil {
		t.Fatal("Products table not found")
	}

	// Check for category index
	foundCategoryIndex := false
	for _, idx := range productsTable.Indexes {
		if idx.Name == "idx_category" {
			foundCategoryIndex = true
			if len(idx.Columns) != 1 || idx.Columns[0] != "category" {
				t.Errorf("Expected idx_category on [category], got %v", idx.Columns)
			}
			break
		}
	}

	if !foundCategoryIndex {
		t.Error("Expected index idx_category on products table not found")
	}
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
