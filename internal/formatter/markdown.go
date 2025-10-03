package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/tordrt/llmschema/internal/schema"
)

// MarkdownFormatter formats schema as markdown
type MarkdownFormatter struct {
	writer io.Writer
}

// NewMarkdownFormatter creates a new markdown formatter
func NewMarkdownFormatter(w io.Writer) *MarkdownFormatter {
	return &MarkdownFormatter{writer: w}
}

// Format writes the schema in markdown format
func (f *MarkdownFormatter) Format(s *schema.Schema) error {
	_, _ = fmt.Fprintln(f.writer, "# Database Schema")
	_, _ = fmt.Fprintln(f.writer)

	for _, table := range s.Tables {
		if err := f.formatTable(table); err != nil {
			return err
		}
	}
	return nil
}

// FormatTable formats a single table (exported for use by multifile formatter)
func (f *MarkdownFormatter) FormatTable(table schema.Table) error {
	return f.formatTable(table)
}

func (f *MarkdownFormatter) formatTable(table schema.Table) error {
	// Table header
	_, _ = fmt.Fprintf(f.writer, "## %s\n\n", table.Name)

	// Columns
	_, _ = fmt.Fprintln(f.writer, "### Columns")
	_, _ = fmt.Fprintln(f.writer)

	for _, col := range table.Columns {
		typeStr := col.Type
		if len(col.EnumValues) > 0 {
			typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, "|"))
		}

		constraintStr := f.formatConstraints(col, table.PrimaryKey)
		if constraintStr != "" {
			_, _ = fmt.Fprintf(f.writer, "- **%s:** %s, %s\n", col.Name, typeStr, constraintStr)
		} else {
			_, _ = fmt.Fprintf(f.writer, "- **%s:** %s\n", col.Name, typeStr)
		}
	}
	_, _ = fmt.Fprintln(f.writer)

	// Relations
	if len(table.Relations) > 0 {
		_, _ = fmt.Fprintln(f.writer, "### References")
		_, _ = fmt.Fprintln(f.writer)
		for _, rel := range table.Relations {
			_, _ = fmt.Fprintf(f.writer, "- %s → %s.%s (%s)\n",
				rel.SourceColumn,
				rel.TargetTable,
				rel.TargetColumn,
				rel.Cardinality)
		}
		_, _ = fmt.Fprintln(f.writer)
	}

	// Indexes
	if len(table.Indexes) > 0 {
		_, _ = fmt.Fprintln(f.writer, "### Idx")
		_, _ = fmt.Fprintln(f.writer)
		for _, idx := range table.Indexes {
			if idx.IsUnique {
				_, _ = fmt.Fprintf(f.writer, "- %s on (%s), unique\n",
					idx.Name,
					strings.Join(idx.Columns, ", "))
			} else {
				_, _ = fmt.Fprintf(f.writer, "- %s on (%s)\n",
					idx.Name,
					strings.Join(idx.Columns, ", "))
			}
		}
		_, _ = fmt.Fprintln(f.writer)
	}

	return nil
}

func (f *MarkdownFormatter) formatConstraints(col schema.Column, primaryKey []string) string {
	var constraints []string

	// Check if this column is part of the primary key
	isPK := false
	for _, pk := range primaryKey {
		if pk == col.Name {
			isPK = true
			break
		}
	}

	if isPK {
		constraints = append(constraints, "PK")
	}

	if col.IsUnique {
		constraints = append(constraints, "UNIQUE")
	}

	if !col.Nullable {
		constraints = append(constraints, "NOT NULL")
	}

	if col.DefaultValue != nil {
		constraints = append(constraints, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
	}

	if col.CheckConstraint != nil {
		constraints = append(constraints, fmt.Sprintf("CHECK(%s)", *col.CheckConstraint))
	}

	return strings.Join(constraints, ", ")
}
