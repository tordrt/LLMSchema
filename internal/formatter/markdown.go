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

	f.FormatColumns(f.writer, table.Columns, table.PrimaryKey)
	f.FormatRelations(f.writer, table.Name, table.Relations)
	f.FormatIndexes(f.writer, table.Indexes)

	return nil
}

// FormatColumns writes column information as a markdown table
func (f *MarkdownFormatter) FormatColumns(w io.Writer, columns []schema.Column, primaryKey []string) {
	_, _ = fmt.Fprintln(w, "| Column | Type | Nullable | Default | Constraints |")
	_, _ = fmt.Fprintln(w, "|--------|------|----------|---------|-------------|")

	for _, col := range columns {
		typeStr := col.Type
		if len(col.EnumValues) > 0 {
			typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, "|"))
		}

		nullable := "YES"
		if !col.Nullable {
			nullable = "NO"
		}

		defaultVal := ""
		if col.DefaultValue != nil {
			defaultVal = *col.DefaultValue
		}

		constraints := FormatTableConstraints(col, primaryKey)

		_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
			col.Name, typeStr, nullable, defaultVal, constraints)
	}
	_, _ = fmt.Fprintln(w)
}

// FormatRelations writes relationship information
func (f *MarkdownFormatter) FormatRelations(w io.Writer, tableName string, relations []schema.Relation) {
	if len(relations) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "### References")
	_, _ = fmt.Fprintln(w)
	for _, rel := range relations {
		cardinalityDesc := FormatCardinality(rel.Cardinality, tableName, rel.TargetTable)
		_, _ = fmt.Fprintf(w, "- %s → %s.%s (%s)\n",
			rel.SourceColumn,
			rel.TargetTable,
			rel.TargetColumn,
			cardinalityDesc)
	}
	_, _ = fmt.Fprintln(w)
}

// FormatIndexes writes index information
func (f *MarkdownFormatter) FormatIndexes(w io.Writer, indexes []schema.Index) {
	if len(indexes) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "### Index")
	_, _ = fmt.Fprintln(w)
	for _, idx := range indexes {
		if idx.IsUnique {
			_, _ = fmt.Fprintf(w, "- %s on (%s), unique\n",
				idx.Name,
				strings.Join(idx.Columns, ", "))
		} else {
			_, _ = fmt.Fprintf(w, "- %s on (%s)\n",
				idx.Name,
				strings.Join(idx.Columns, ", "))
		}
	}
	_, _ = fmt.Fprintln(w)
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

// FormatTableConstraints formats constraints for table output (excludes nullable and default which have their own columns)
func FormatTableConstraints(col schema.Column, primaryKey []string) string {
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

	if col.CheckConstraint != nil {
		constraints = append(constraints, fmt.Sprintf("CHECK(%s)", *col.CheckConstraint))
	}

	return strings.Join(constraints, ", ")
}

// FormatCardinality converts cardinality notation to human-readable format
func FormatCardinality(cardinality, sourceTable, targetTable string) string {
	switch cardinality {
	case "N:1":
		return fmt.Sprintf("many %s to one %s", sourceTable, targetTable)
	case "1:N":
		return fmt.Sprintf("one %s to many %s", sourceTable, targetTable)
	case "1:1":
		return fmt.Sprintf("one %s to one %s", sourceTable, targetTable)
	default:
		return cardinality
	}
}
