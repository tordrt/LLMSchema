package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/tordrt/llmschema/internal/schema"
)

// TextFormatter formats schema as compact text
type TextFormatter struct {
	writer io.Writer
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(w io.Writer) *TextFormatter {
	return &TextFormatter{writer: w}
}

// Format writes the schema in compact text format
func (f *TextFormatter) Format(s *schema.Schema) error {
	for i, table := range s.Tables {
		if i > 0 {
			_, _ = fmt.Fprintln(f.writer) // Blank line between tables
		}

		if err := f.formatTable(table); err != nil {
			return err
		}
	}
	return nil
}

func (f *TextFormatter) formatTable(table schema.Table) error {
	// Table header with primary key
	pkStr := ""
	if len(table.PrimaryKey) > 0 {
		pkStr = fmt.Sprintf(" (PK: %s)", strings.Join(table.PrimaryKey, ", "))
	}
	_, _ = fmt.Fprintf(f.writer, "TABLE %s%s\n", table.Name, pkStr)

	// Columns
	for _, col := range table.Columns {
		_, _ = fmt.Fprintf(f.writer, "  %s\n", f.formatColumn(col))
	}

	// Relations
	if len(table.Relations) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = fmt.Fprintln(f.writer, "  RELATIONS:")
		for _, rel := range table.Relations {
			_, _ = fmt.Fprintf(f.writer, "    → %s.%s (%s)\n", rel.TargetTable, rel.TargetColumn, rel.Cardinality)
		}
	}

	// Indexes
	if len(table.Indexes) > 0 {
		_, _ = fmt.Fprintln(f.writer)
		_, _ = fmt.Fprintln(f.writer, "  INDEXES:")
		for _, idx := range table.Indexes {
			unique := ""
			if idx.IsUnique {
				unique = " UNIQUE"
			}
			_, _ = fmt.Fprintf(f.writer, "    %s (%s)%s\n", idx.Name, strings.Join(idx.Columns, ", "), unique)
		}
	}

	return nil
}

func (f *TextFormatter) formatColumn(col schema.Column) string {
	parts := []string{col.Name + ":"}

	// Type with enum values if present
	typeStr := col.Type
	if len(col.EnumValues) > 0 {
		typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, "|"))
	}
	parts = append(parts, typeStr)

	// Unique
	if col.IsUnique {
		parts = append(parts, "UNIQUE")
	}

	// Nullable
	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}

	// Default value
	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
	}

	return strings.Join(parts, " ")
}
