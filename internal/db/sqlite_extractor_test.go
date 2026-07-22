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
		if !slices.ContainsFunc(table.Indexes, func(index schema.Index) bool {
			return index.Name == indexName
		}) {
			t.Errorf("Indexes = %v, want one named %q", table.Indexes, indexName)
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

func TestSQLiteExtractorPreservesConstraintsAndCardinality(t *testing.T) {
	ctx := context.Background()
	client, err := NewSQLiteClient(ctx, ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteClient() failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	statements := []string{
		`CREATE TABLE parents (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE composite_parents (a INTEGER, b INTEGER, PRIMARY KEY (a, b))`,
		`CREATE TABLE reverse_key_parents (a INTEGER, b INTEGER, PRIMARY KEY (b, a))`,
		`CREATE TABLE profiles (
			parent_id INTEGER PRIMARY KEY REFERENCES parents(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE implicit_target_children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER REFERENCES parents
		)`,
		`CREATE TABLE unique_children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL UNIQUE REFERENCES parents(id)
		)`,
		`CREATE TABLE ratings (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL REFERENCES parents(id),
			rater_id INTEGER NOT NULL,
			UNIQUE (parent_id, rater_id)
		)`,
		`CREATE TABLE composite_children (
			id INTEGER PRIMARY KEY,
			parent_a INTEGER NOT NULL,
			parent_b INTEGER NOT NULL,
			FOREIGN KEY (parent_a, parent_b)
				REFERENCES composite_parents(a, b) ON UPDATE CASCADE
		)`,
		`CREATE TABLE implicit_composite_children (
			id INTEGER PRIMARY KEY,
			parent_b INTEGER NOT NULL,
			parent_a INTEGER NOT NULL,
			FOREIGN KEY (parent_b, parent_a) REFERENCES reverse_key_parents
		)`,
		`CREATE TABLE expression_children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id),
			label TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX expression_children_parent_label
			ON expression_children(parent_id, lower(label))`,
		`CREATE INDEX expression_children_lower_label
			ON expression_children(lower(label))`,
		`CREATE TABLE partial_children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id),
			active INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX partial_children_parent_active
			ON partial_children(parent_id) WHERE active = 1`,
	}
	for _, statement := range statements {
		if _, err := client.GetDB().ExecContext(ctx, statement); err != nil {
			t.Fatalf("ExecContext(%q) failed: %v", statement, err)
		}
	}

	extracted, err := NewSQLiteExtractor(client).ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("ExtractSchema() failed: %v", err)
	}

	profiles := tableNamed(t, extracted.Tables, "profiles")
	assertRelation(t, profiles.Relations, []string{"parent_id"}, []string{"id"}, "1:1")
	if profiles.Relations[0].OnDelete != "CASCADE" {
		t.Errorf("profiles ON DELETE = %q, want CASCADE", profiles.Relations[0].OnDelete)
	}
	if profiles.Relations[0].SourceColumn != "parent_id" || profiles.Relations[0].TargetColumn != "id" {
		t.Errorf("single-column compatibility aliases were not populated: %#v", profiles.Relations[0])
	}

	uniqueChildren := tableNamed(t, extracted.Tables, "unique_children")
	assertRelation(t, uniqueChildren.Relations, []string{"parent_id"}, []string{"id"}, "1:1")
	if !columnNamed(t, uniqueChildren.Columns, "parent_id").IsUnique {
		t.Error("single-column UNIQUE was not preserved")
	}

	implicitTarget := tableNamed(t, extracted.Tables, "implicit_target_children")
	assertRelation(t, implicitTarget.Relations, []string{"parent_id"}, []string{"id"}, "N:1")

	ratings := tableNamed(t, extracted.Tables, "ratings")
	assertRelation(t, ratings.Relations, []string{"parent_id"}, []string{"id"}, "N:1")
	if columnNamed(t, ratings.Columns, "parent_id").IsUnique || columnNamed(t, ratings.Columns, "rater_id").IsUnique {
		t.Error("members of a composite UNIQUE constraint were marked individually unique")
	}

	composite := tableNamed(t, extracted.Tables, "composite_children")
	assertRelation(t, composite.Relations, []string{"parent_a", "parent_b"}, []string{"a", "b"}, "N:1")
	if len(composite.Relations) != 1 {
		t.Fatalf("composite foreign key produced %d relations, want 1", len(composite.Relations))
	}
	if composite.Relations[0].OnUpdate != "CASCADE" {
		t.Errorf("composite ON UPDATE = %q, want CASCADE", composite.Relations[0].OnUpdate)
	}

	implicitComposite := tableNamed(t, extracted.Tables, "implicit_composite_children")
	assertRelation(t, implicitComposite.Relations,
		[]string{"parent_b", "parent_a"}, []string{"b", "a"}, "N:1")
	if !slices.Equal(tableNamed(t, extracted.Tables, "reverse_key_parents").PrimaryKey, []string{"b", "a"}) {
		t.Error("composite primary key was not returned in constraint order")
	}

	expression := tableNamed(t, extracted.Tables, "expression_children")
	assertRelation(t, expression.Relations, []string{"parent_id"}, []string{"id"}, "N:1")
	if columnNamed(t, expression.Columns, "parent_id").IsUnique {
		t.Error("column from a unique expression index was marked individually unique")
	}
	for _, indexName := range []string{"expression_children_parent_label", "expression_children_lower_label"} {
		if !slices.ContainsFunc(expression.Indexes, func(index schema.Index) bool {
			return index.Name == indexName && index.HasExpressions
		}) {
			t.Errorf("expression index %q not identified as incomplete: %#v", indexName, expression.Indexes)
		}
	}

	partial := tableNamed(t, extracted.Tables, "partial_children")
	assertRelation(t, partial.Relations, []string{"parent_id"}, []string{"id"}, "N:1")
	if columnNamed(t, partial.Columns, "parent_id").IsUnique {
		t.Error("partial unique index was treated as unconditional uniqueness")
	}
}

func TestRelationCardinalityUsesColumnSets(t *testing.T) {
	tests := []struct {
		name       string
		source     []string
		primaryKey []string
		indexes    []schema.Index
		want       string
	}{
		{name: "primary key foreign key", source: []string{"user_id"}, primaryKey: []string{"user_id"}, want: "1:1"},
		{name: "composite primary key foreign key", source: []string{"a", "b"}, primaryKey: []string{"a", "b"}, want: "1:1"},
		{name: "only part of composite primary key", source: []string{"a"}, primaryKey: []string{"a", "b"}, want: "N:1"},
		{name: "unique foreign key", source: []string{"user_id"}, indexes: []schema.Index{{Columns: []string{"user_id"}, IsUnique: true}}, want: "1:1"},
		{name: "composite unique key", source: []string{"a", "b"}, indexes: []schema.Index{{Columns: []string{"a", "b"}, IsUnique: true}}, want: "1:1"},
		{name: "partial unique index", source: []string{"user_id"}, indexes: []schema.Index{{Columns: []string{"user_id"}, IsUnique: true, IsPartial: true}}, want: "N:1"},
		{name: "unique expression index", source: []string{"user_id"}, indexes: []schema.Index{{Columns: []string{"user_id"}, IsUnique: true, HasExpressions: true}}, want: "N:1"},
		{name: "non-unique index", source: []string{"user_id"}, indexes: []schema.Index{{Columns: []string{"user_id"}}}, want: "N:1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relationCardinality(tt.source, tt.primaryKey, tt.indexes); got != tt.want {
				t.Errorf("relationCardinality() = %q, want %q", got, tt.want)
			}
		})
	}
}

func tableNamed(t *testing.T, tables []schema.Table, name string) schema.Table {
	t.Helper()
	for _, table := range tables {
		if table.Name == name {
			return table
		}
	}
	t.Fatalf("table %q not found", name)
	return schema.Table{}
}

func columnNamed(t *testing.T, columns []schema.Column, name string) schema.Column {
	t.Helper()
	for _, column := range columns {
		if column.Name == name {
			return column
		}
	}
	t.Fatalf("column %q not found", name)
	return schema.Column{}
}

func assertRelation(t *testing.T, relations []schema.Relation, source, target []string, cardinality string) {
	t.Helper()
	for _, relation := range relations {
		if slices.Equal(relation.SourceColumns, source) && slices.Equal(relation.TargetColumns, target) {
			if relation.Cardinality != cardinality {
				t.Errorf("relationship cardinality = %q, want %q", relation.Cardinality, cardinality)
			}
			return
		}
	}
	t.Errorf("relationship %v -> %v not found in %#v", source, target, relations)
}
