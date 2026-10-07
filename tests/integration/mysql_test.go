//go:build integration
// +build integration

package integration

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/tordrt/llmschema"
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
	if s.DatabaseName != "testdb" {
		t.Errorf("Expected database name testdb, got %q", s.DatabaseName)
	}
	if s.SchemaName != "testdb" {
		t.Errorf("Expected schema name testdb, got %q", s.SchemaName)
	}

	// Verify tables exist
	expectedTables := []string{"users", "products", "orders", "order_items", "profiles", "composite_parents", "composite_children", "expression_children", "external_profiles", "generated_values"}
	verifyTablesExist(t, s, expectedTables)
	if findTable(s, "active_users") != nil {
		t.Error("Views should not be extracted unless requested")
	}

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
	verifyColumnType(t, s, "users", "status", "enum")
	verifyColumnType(t, s, "products", "price", "decimal(10,2)")
	verifyColumnGenerated(t, s, "generated_values", map[string]string{
		"id":       "AUTO_INCREMENT",
		"quantity": "",
		"doubled":  "GENERATED ALWAYS AS (`quantity` * 2)",
		"tripled":  "GENERATED ALWAYS AS (`quantity` * 3)",
	})

	// Verify foreign key relationships
	verifyForeignKey(t, s, "orders", "user_id", "users")
	verifyConstraintExtraction(t, s)
	verifyExternalSchemaRelation(t, s, "external_profiles", "identity", "users")
	verifyExpressionIndexMarked(t, s, "expression_children_user_label")
	verifyKeyAndIndexMarkdown(t, s)

	_, err = db.NewMySQLExtractor(client, "no_such_schema").ExtractSchema(ctx, nil)
	verifyUnknownSchemaRejected(t, err)
}

func TestMySQLURLWithoutDriverNetwork(t *testing.T) {
	connString := os.Getenv("MYSQL_TEST_URL")
	if connString == "" {
		connString = "root:testpassword@tcp(localhost:3306)/testdb"
	}
	cfg, err := mysql.ParseDSN(connString)
	if err != nil {
		t.Fatalf("Failed to parse MYSQL_TEST_URL: %v", err)
	}
	databaseURL := (&url.URL{
		Scheme: "mysql",
		User:   url.UserPassword(cfg.User, cfg.Passwd),
		Host:   cfg.Addr,
		Path:   "/" + cfg.DBName,
	}).String()

	s, err := llmschema.ExtractSchema(context.Background(), databaseURL, &llmschema.Options{Tables: []string{"users"}})
	if err != nil {
		t.Fatalf("Failed to extract schema from %s: %v", databaseURL, err)
	}
	if len(s.Tables) != 1 || findTable(s, "users") == nil {
		t.Errorf("Expected only users table, got %d tables", len(s.Tables))
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

	_, err = extractor.ExtractSchema(ctx, []string{"users", "no_such_table"})
	verifyUnknownTableRejected(t, err)

	views, err := extractor.ExtractSchema(ctx, []string{"active_users"})
	if err != nil {
		t.Fatalf("Failed to extract requested view: %v", err)
	}
	verifyColumns(t, findTable(views, "active_users"), []string{"id", "username"})
}
