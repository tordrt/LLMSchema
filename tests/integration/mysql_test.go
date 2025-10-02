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

	// Verify ENUM type extraction for status column
	var statusColumn *struct {
		Name       string
		Type       string
		EnumValues []string
	}

	for _, col := range usersTable.Columns {
		if col.Name == "status" {
			// Find the actual column in the schema
			for _, table := range schema.Tables {
				if table.Name == "users" {
					for _, c := range table.Columns {
						if c.Name == "status" {
							statusColumn = &struct {
								Name       string
								Type       string
								EnumValues []string
							}{
								Name:       c.Name,
								Type:       c.Type,
								EnumValues: c.EnumValues,
							}
							break
						}
					}
					break
				}
			}
			break
		}
	}

	if statusColumn != nil && len(statusColumn.EnumValues) > 0 {
		expectedEnumValues := []string{"active", "inactive", "banned"}
		if len(statusColumn.EnumValues) != len(expectedEnumValues) {
			t.Errorf("Expected %d enum values for status, got %d", len(expectedEnumValues), len(statusColumn.EnumValues))
		}
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
