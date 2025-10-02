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
	schema, err := extractor.ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to extract schema: %v", err)
	}

	// Verify tables exist
	expectedTables := []string{"users", "profiles", "posts", "comments"}
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
		Columns    []struct{ Name string }
		PrimaryKey []string
	}

	for i, table := range schema.Tables {
		if table.Name == "users" {
			usersTable = &struct {
				Name       string
				Columns    []struct{ Name string }
				PrimaryKey []string
			}{
				Name:       table.Name,
				PrimaryKey: table.PrimaryKey,
			}
			for _, col := range table.Columns {
				usersTable.Columns = append(usersTable.Columns, struct{ Name string }{Name: col.Name})
			}
			_ = i
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
	expectedColumns := []string{"id", "email", "username", "created_at"}
	columnMap := make(map[string]bool)
	for _, col := range usersTable.Columns {
		columnMap[col.Name] = true
	}

	for _, colName := range expectedColumns {
		if !columnMap[colName] {
			t.Errorf("Expected column %s not found in users table", colName)
		}
	}

	// Verify foreign key relationships
	var postsTable *struct {
		Relations []struct {
			TargetTable  string
			SourceColumn string
		}
	}

	for _, table := range schema.Tables {
		if table.Name == "posts" {
			postsTable = &struct {
				Relations []struct {
					TargetTable  string
					SourceColumn string
				}
			}{}
			for _, rel := range table.Relations {
				postsTable.Relations = append(postsTable.Relations, struct {
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

	if postsTable == nil {
		t.Fatal("Posts table not found")
	}

	// Check that posts has a relation to users
	foundUserRelation := false
	for _, rel := range postsTable.Relations {
		if rel.TargetTable == "users" && rel.SourceColumn == "user_id" {
			foundUserRelation = true
			break
		}
	}

	if !foundUserRelation {
		t.Error("Expected foreign key relationship from posts.user_id to users not found")
	}
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

	// Extract only users and posts tables
	schema, err := extractor.ExtractSchema(ctx, []string{"users", "posts"})
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

	if !tableMap["users"] || !tableMap["posts"] {
		t.Error("Expected users and posts tables")
	}

	if tableMap["comments"] || tableMap["profiles"] {
		t.Error("Should not include comments or profiles tables")
	}
}
