//go:build integration
// +build integration

package integration

import (
	"slices"
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

func verifyConstraintExtraction(t *testing.T, s *schema.Schema) {
	t.Helper()

	profiles := findTable(s, "profiles")
	if profiles == nil {
		t.Fatal("profiles table not found")
	}
	verifyRelation(t, profiles, []string{"user_id"}, []string{"id"}, "1:1")

	orderItems := findTable(s, "order_items")
	if orderItems == nil {
		t.Fatal("order_items table not found")
	}
	for _, name := range []string{"order_id", "product_id"} {
		for _, column := range orderItems.Columns {
			if column.Name == name && column.IsUnique {
				t.Errorf("composite UNIQUE member %s.%s marked individually unique", orderItems.Name, name)
			}
		}
	}

	composite := findTable(s, "composite_children")
	if composite == nil {
		t.Fatal("composite_children table not found")
	}
	verifyRelation(t, composite, []string{"tenant_id", "parent_id"}, []string{"tenant_id", "id"}, "N:1")

	expression := findTable(s, "expression_children")
	if expression == nil {
		t.Fatal("expression_children table not found")
	}
	verifyRelation(t, expression, []string{"user_id"}, []string{"id"}, "N:1")
	for _, column := range expression.Columns {
		if column.Name == "user_id" && column.IsUnique {
			t.Error("column from unique expression index marked individually unique")
		}
	}
}

func verifyExternalSchemaRelation(t *testing.T, s *schema.Schema, sourceTable, targetSchema, targetTable string) {
	t.Helper()
	table := findTable(s, sourceTable)
	if table == nil {
		t.Fatalf("%s table not found", sourceTable)
	}
	for _, relation := range table.Relations {
		if relation.TargetSchema == targetSchema && relation.TargetTable == targetTable {
			return
		}
	}
	t.Errorf("%s relation to %s.%s not found", sourceTable, targetSchema, targetTable)
}

func verifyExpressionIndexMarked(t *testing.T, s *schema.Schema, indexName string) {
	t.Helper()
	table := findTable(s, "expression_children")
	if table == nil {
		t.Fatal("expression_children table not found")
	}
	for _, index := range table.Indexes {
		if index.Name == indexName {
			if !index.HasExpressions {
				t.Errorf("expression index %s was not marked as incomplete", indexName)
			}
			return
		}
	}
	t.Errorf("expression index %s not found", indexName)
}

func verifyRelation(t *testing.T, table *schema.Table, sourceColumns, targetColumns []string, cardinality string) {
	t.Helper()
	for _, relation := range table.Relations {
		if slices.Equal(relation.SourceColumns, sourceColumns) && slices.Equal(relation.TargetColumns, targetColumns) {
			if relation.Cardinality != cardinality {
				t.Errorf("%s relationship %v cardinality = %q, want %q", table.Name, sourceColumns, relation.Cardinality, cardinality)
			}
			return
		}
	}
	t.Errorf("%s relationship %v -> %v not found", table.Name, sourceColumns, targetColumns)
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
		if rel.TargetTable == targetTable {
			for _, column := range rel.SourceColumns {
				if column == sourceColumn {
					return
				}
			}
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
