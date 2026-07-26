package formatter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

func TestFormatIncludesLinkedTableIndexByDefault(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "Order Items"},
		{Name: "audit.logs"},
		{Name: "users"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := `# Database Schema

- [Order Items](#order-items)
- [audit.logs](#auditlogs)
- [users](#users)

## Order Items
`
	if got := output.String(); !strings.HasPrefix(got, want) {
		t.Fatalf("output prefix:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatCanOmitTableIndex(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	formatter.OmitTableIndex = true

	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	if got := output.String(); strings.Contains(got, "- [users](#users)") {
		t.Fatalf("output contains table index:\n%s", got)
	}
}

func TestFormatTableIndexHandlesAnchorCollisions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "Database Schema"},
		{Name: "Database-Schema"},
		{Name: "audit.logs"},
		{Name: "auditlogs"},
		{Name: "!!!"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	for _, want := range []string{
		"- [Database Schema](#database-schema-1)",
		"- [Database-Schema](#database-schema-2)",
		"- [audit.logs](#auditlogs)",
		"- [auditlogs](#auditlogs-1)",
		"- !!!",
	} {
		if got := output.String(); !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatTableIndexAccountsForGeneratedSubheadings(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{
			Name: "orders",
			Indexes: []schema.Index{{
				Name:    "orders_created_at",
				Columns: []string{"created_at"},
			}},
			Relations: []schema.Relation{{
				SourceColumn: "user_id",
				TargetTable:  "users",
				TargetColumn: "id",
			}},
		},
		{Name: "Index"},
		{Name: "References"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	for _, want := range []string{
		"- [Index](#index-1)",
		"- [References](#references-1)",
	} {
		if got := output.String(); !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatRelationsSupportsCompositeKeysAndActions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	relations := []schema.Relation{
		{
			SourceColumns: []string{"photo_test_id"},
			TargetTable:   "photo_tests",
			TargetColumns: []string{"id"},
			Cardinality:   "1:1",
			OnDelete:      "CASCADE",
		},
		{
			SourceColumns: []string{"test_id", "photo_id"},
			TargetSchema:  "media",
			TargetTable:   "test_photos",
			TargetColumns: []string{"test_id", "photo_id"},
			Cardinality:   "N:1",
			OnUpdate:      "CASCADE",
		},
	}

	if err := formatter.FormatRelations(&output, "photo_results", relations); err != nil {
		t.Fatalf("FormatRelations() failed: %v", err)
	}

	got := output.String()
	wants := []string{
		"photo_test_id → photo_tests.id (one photo_results to one photo_tests; ON DELETE CASCADE)",
		"(test_id, photo_id) → media.test_photos(test_id, photo_id) (many photo_results to one test_photos; ON UPDATE CASCADE)",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatColumnsEscapesMarkdownTableCells(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	defaultValue := "'left|right\nnext\\line'"
	checkConstraint := "value <> 'a|b'\r\nAND value IS NOT NULL"

	err := formatter.FormatColumns(&output, []schema.Column{{
		Name:            "config|value",
		Type:            "text",
		Nullable:        true,
		DefaultValue:    &defaultValue,
		CheckConstraint: &checkConstraint,
	}}, nil, nil)
	if err != nil {
		t.Fatalf("FormatColumns() failed: %v", err)
	}

	want := "| config\\|value | text DEFAULT 'left\\|right<br>next\\\\line' | CHECK(value <> 'a\\|b'<br>AND value IS NOT NULL) |"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestFormatIndexesMarksExpressions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)

	err := formatter.FormatIndexes(&output, []schema.Index{{
		Name:           "expression_children_user_label",
		Columns:        []string{"user_id"},
		IsUnique:       true,
		HasExpressions: true,
	}})
	if err != nil {
		t.Fatalf("FormatIndexes() failed: %v", err)
	}

	want := "- expression_children_user_label on (user_id, `<expression>`), unique, contains expressions"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestFormatPreservesExpressionIndexOnUniqueColumn(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{{
		Name: "expression_children",
		Columns: []schema.Column{{
			Name:     "user_id",
			Type:     "integer",
			IsUnique: true,
		}},
		Indexes: []schema.Index{{
			Name:           "expression_children_user_label",
			Columns:        []string{"user_id"},
			IsUnique:       true,
			HasExpressions: true,
		}},
	}}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "- expression_children_user_label on (user_id, `<expression>`), unique, contains expressions"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestMarkdownFormatterPropagatesWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	s := &schema.Schema{Tables: []schema.Table{{Name: "users"}}}

	if err := f.Format(s); !errors.Is(err, errWriteFailed) {
		t.Fatalf("Format() error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarkdownFormattingMethodsPropagateWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "columns",
			run: func() error {
				return f.FormatColumns(failingWriter{}, []schema.Column{{Name: "id", Type: "integer"}}, nil, nil)
			},
		},
		{
			name: "relations",
			run: func() error {
				return f.FormatRelations(failingWriter{}, "orders", []schema.Relation{{
					SourceColumns: []string{"user_id"},
					TargetTable:   "users",
					TargetColumns: []string{"id"},
				}})
			},
		},
		{
			name: "indexes",
			run: func() error {
				return f.FormatIndexes(failingWriter{}, []schema.Index{{Name: "users_pkey", Columns: []string{"id"}}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, errWriteFailed) {
				t.Fatalf("error = %v, want %v", err, errWriteFailed)
			}
		})
	}
}
