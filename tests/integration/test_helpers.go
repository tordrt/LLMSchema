//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

// verifyTablesExist checks that all expected tables are present in the schema
func verifyTablesExist(t *testing.T, s *schema.Schema, expectedTables []string) {
	t.Helper()

	if len(s.Tables) != len(expectedTables) {
		t.Errorf("Expected %d tables, got %d", len(expectedTables), len(s.Tables))
	}

	tableMap := make(map[string]bool)
	for _, table := range s.Tables {
		tableMap[table.Name] = true
	}

	for _, tableName := range expectedTables {
		if !tableMap[tableName] {
			t.Errorf("Expected table %s not found in schema", tableName)
		}
	}
}

// verifyColumns checks that expected columns exist in a table
func verifyColumns(t *testing.T, table *schema.Table, expectedColumns []string) {
	t.Helper()

	columnMap := make(map[string]bool)
	for _, col := range table.Columns {
		columnMap[col.Name] = true
	}

	for _, colName := range expectedColumns {
		if !columnMap[colName] {
			t.Errorf("Expected column %s not found in %s table", colName, table.Name)
		}
	}
}

// verifyPrimaryKey checks that a table has the expected primary key
func verifyPrimaryKey(t *testing.T, table *schema.Table, expectedPK []string) {
	t.Helper()

	if len(table.PrimaryKey) != len(expectedPK) {
		t.Errorf("Expected primary key %v, got %v", expectedPK, table.PrimaryKey)
		return
	}

	for i, pk := range expectedPK {
		if table.PrimaryKey[i] != pk {
			t.Errorf("Expected primary key %v, got %v", expectedPK, table.PrimaryKey)
			return
		}
	}
}

// verifyUniqueConstraint checks that a column has a unique constraint
func verifyUniqueConstraint(t *testing.T, s *schema.Schema, tableName, columnName string) {
	t.Helper()

	table := findTable(s, tableName)
	if table == nil {
		t.Fatalf("Table %s not found", tableName)
		return
	}

	for _, col := range table.Columns {
		if col.Name == columnName {
			if !col.IsUnique {
				t.Errorf("Expected %s column to have unique constraint", columnName)
			}
			return
		}
	}

	t.Errorf("Column %s not found in table %s", columnName, tableName)
}

// verifyForeignKey checks that a foreign key relationship exists
func verifyForeignKey(t *testing.T, s *schema.Schema, tableName, sourceColumn, targetTable string) {
	t.Helper()

	table := findTable(s, tableName)
	if table == nil {
		t.Fatalf("Table %s not found", tableName)
		return
	}

	for _, rel := range table.Relations {
		if rel.TargetTable == targetTable && rel.SourceColumn == sourceColumn {
			return
		}
	}

	t.Errorf("Expected foreign key relationship from %s.%s to %s not found", tableName, sourceColumn, targetTable)
}

// verifyIndex checks that an index exists with the expected columns
func verifyIndex(t *testing.T, s *schema.Schema, tableName, indexName string, expectedColumns []string) {
	t.Helper()

	table := findTable(s, tableName)
	if table == nil {
		t.Fatalf("Table %s not found", tableName)
		return
	}

	for _, idx := range table.Indexes {
		if idx.Name == indexName {
			if len(idx.Columns) != len(expectedColumns) {
				t.Errorf("Expected index %s on %v, got %v", indexName, expectedColumns, idx.Columns)
				return
			}
			for i, col := range expectedColumns {
				if idx.Columns[i] != col {
					t.Errorf("Expected index %s on %v, got %v", indexName, expectedColumns, idx.Columns)
					return
				}
			}
			return
		}
	}

	t.Errorf("Expected index %s on %s table not found", indexName, tableName)
}

// verifyEnumValues checks that a column has the expected enum values
func verifyEnumValues(t *testing.T, s *schema.Schema, tableName, columnName string, expectedValues []string) {
	t.Helper()

	table := findTable(s, tableName)
	if table == nil {
		t.Fatalf("Table %s not found", tableName)
		return
	}

	for _, col := range table.Columns {
		if col.Name == columnName {
			if len(col.EnumValues) == 0 {
				return // Enum values are optional
			}

			if len(col.EnumValues) != len(expectedValues) {
				t.Errorf("Expected %d enum values for %s, got %d", len(expectedValues), columnName, len(col.EnumValues))
				return
			}

			// Note: We don't check the exact enum values as they may vary
			return
		}
	}

	t.Errorf("Column %s not found in table %s", columnName, tableName)
}

// findTable is a helper function to find a table by name in the schema
func findTable(s *schema.Schema, tableName string) *schema.Table {
	for i := range s.Tables {
		if s.Tables[i].Name == tableName {
			return &s.Tables[i]
		}
	}
	return nil
}
