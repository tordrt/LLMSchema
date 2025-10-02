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

func (f *MarkdownFormatter) formatTable(table schema.Table) error {
	// Table header
	_, _ = fmt.Fprintf(f.writer, "## %s\n\n", table.Name)

	// Primary key info
	if len(table.PrimaryKey) > 0 {
		_, _ = fmt.Fprintf(f.writer, "**Primary Key:** `%s`\n\n", strings.Join(table.PrimaryKey, ", "))
	}

	// Columns table
	_, _ = fmt.Fprintln(f.writer, "### Columns")
	_, _ = fmt.Fprintln(f.writer)
	_, _ = fmt.Fprintln(f.writer, "| Column | Type | Constraints |")
	_, _ = fmt.Fprintln(f.writer, "|--------|------|-------------|")

	for _, col := range table.Columns {
		typeStr := col.Type
		if len(col.EnumValues) > 0 {
			typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, "\\|"))
		}
		_, _ = fmt.Fprintf(f.writer, "| %s | `%s` | %s |\n",
			col.Name,
			typeStr,
			f.formatConstraints(col))
	}
	_, _ = fmt.Fprintln(f.writer)

	// Relations
	if len(table.Relations) > 0 {
		_, _ = fmt.Fprintln(f.writer, "### Relations")
		_, _ = fmt.Fprintln(f.writer)
		_, _ = fmt.Fprintln(f.writer, "| Target Table | Target Column | Cardinality |")
		_, _ = fmt.Fprintln(f.writer, "|--------------|---------------|-------------|")
		for _, rel := range table.Relations {
			_, _ = fmt.Fprintf(f.writer, "| `%s` | `%s` | %s |\n",
				rel.TargetTable,
				rel.TargetColumn,
				rel.Cardinality)
		}
		_, _ = fmt.Fprintln(f.writer)
	}

	// Indexes
	if len(table.Indexes) > 0 {
		_, _ = fmt.Fprintln(f.writer, "### Indexes")
		_, _ = fmt.Fprintln(f.writer)
		_, _ = fmt.Fprintln(f.writer, "| Index Name | Columns | Unique |")
		_, _ = fmt.Fprintln(f.writer, "|------------|---------|--------|")
		for _, idx := range table.Indexes {
			unique := "No"
			if idx.IsUnique {
				unique = "Yes"
			}
			_, _ = fmt.Fprintf(f.writer, "| `%s` | `%s` | %s |\n",
				idx.Name,
				strings.Join(idx.Columns, ", "),
				unique)
		}
		_, _ = fmt.Fprintln(f.writer)
	}

	return nil
}

func (f *MarkdownFormatter) formatConstraints(col schema.Column) string {
	var constraints []string

	if col.IsUnique {
		constraints = append(constraints, "UNIQUE")
	}

	if !col.Nullable {
		constraints = append(constraints, "NOT NULL")
	}

	if col.DefaultValue != nil {
		constraints = append(constraints, fmt.Sprintf("DEFAULT `%s`", *col.DefaultValue))
	}

	if len(constraints) == 0 {
		return "-"
	}

	return strings.Join(constraints, ", ")
}
