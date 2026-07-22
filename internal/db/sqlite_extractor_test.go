package db

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

func TestSQLiteExtractorIncludesDatabaseMetadata(t *testing.T) {
	ctx := context.Background()
	client, err := NewSQLiteClient(ctx, ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteClient() failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	s, err := NewSQLiteExtractor(client).ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("ExtractSchema() failed: %v", err)
	}
	if s.DatabaseType != "SQLite" {
		t.Errorf("DatabaseType = %q, want SQLite", s.DatabaseType)
	}
	if s.DatabaseVersion == "" {
		t.Error("DatabaseVersion is empty")
	}
}

func TestSQLiteExtractorHandlesQuotedPragmaNames(t *testing.T) {
	const (
		tableName = `order details"'archive`
		indexName = `parent id"'index`
	)

	ctx := context.Background()
	client, err := NewSQLiteClient(ctx, filepath.Join(t.TempDir(), "quoted-names.db"))
	if err != nil {
		t.Fatalf("NewSQLiteClient() failed: %v", err)
	}
	defer func() { _ = client.Close() }()

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
			t.Fatalf("creating test schema failed: %v", err)
		}
	}

	for _, tables := range [][]string{nil, {tableName}} {
		s, err := NewSQLiteExtractor(client).ExtractSchema(ctx, tables)
		if err != nil {
			t.Fatalf("ExtractSchema(%v) failed: %v", tables, err)
		}

		var table *schema.Table
		for i := range s.Tables {
			if s.Tables[i].Name == tableName {
				table = &s.Tables[i]
				break
			}
		}
		if table == nil {
			t.Fatalf("ExtractSchema(%v) did not return quoted table name", tables)
		}
		if !slices.Equal(table.PrimaryKey, []string{"id"}) {
			t.Errorf("PrimaryKey = %v, want [id]", table.PrimaryKey)
		}
		if len(table.Relations) != 1 || table.Relations[0].TargetTable != "parent table" {
			t.Errorf("Relations = %v, want one relation to parent table", table.Relations)
		}
		if len(table.Indexes) != 1 || table.Indexes[0].Name != indexName {
			t.Errorf("Indexes = %v, want %q", table.Indexes, indexName)
		}

		var emailUnique bool
		for _, column := range table.Columns {
			if column.Name == "email address" {
				emailUnique = column.IsUnique
			}
		}
		if !emailUnique {
			t.Error("email address column was not marked unique")
		}
	}
}
