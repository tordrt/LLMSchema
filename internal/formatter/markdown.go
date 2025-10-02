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
	fmt.Fprintln(f.writer, "# Database Schema")
	fmt.Fprintln(f.writer)

	for _, table := range s.Tables {
		if err := f.formatTable(table); err != nil {
			return err
		}
	}
	return nil
}

func (f *MarkdownFormatter) formatTable(table schema.Table) error {
	// Table header
	fmt.Fprintf(f.writer, "## %s\n\n", table.Name)

	// Primary key info
	if len(table.PrimaryKey) > 0 {
		fmt.Fprintf(f.writer, "**Primary Key:** `%s`\n\n", strings.Join(table.PrimaryKey, ", "))
	}

	// Columns table
	fmt.Fprintln(f.writer, "### Columns")
	fmt.Fprintln(f.writer)
	fmt.Fprintln(f.writer, "| Column | Type | Constraints |")
	fmt.Fprintln(f.writer, "|--------|------|-------------|")

	for _, col := range table.Columns {
		typeStr := col.Type
		if len(col.EnumValues) > 0 {
			typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, "\\|"))
		}
		fmt.Fprintf(f.writer, "| %s | `%s` | %s |\n",
			col.Name,
			typeStr,
			f.formatConstraints(col))
	}
	fmt.Fprintln(f.writer)

	// Relations
	if len(table.Relations) > 0 {
		fmt.Fprintln(f.writer, "### Relations")
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, "| Target Table | Target Column | Cardinality |")
		fmt.Fprintln(f.writer, "|--------------|---------------|-------------|")
		for _, rel := range table.Relations {
			fmt.Fprintf(f.writer, "| `%s` | `%s` | %s |\n",
				rel.TargetTable,
				rel.TargetColumn,
				rel.Cardinality)
		}
		fmt.Fprintln(f.writer)
	}

	// Indexes
	if len(table.Indexes) > 0 {
		fmt.Fprintln(f.writer, "### Indexes")
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, "| Index Name | Columns | Unique |")
		fmt.Fprintln(f.writer, "|------------|---------|--------|")
		for _, idx := range table.Indexes {
			unique := "No"
			if idx.IsUnique {
				unique = "Yes"
			}
			fmt.Fprintf(f.writer, "| `%s` | `%s` | %s |\n",
				idx.Name,
				strings.Join(idx.Columns, ", "),
				unique)
		}
		fmt.Fprintln(f.writer)
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
