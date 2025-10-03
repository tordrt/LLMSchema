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

	f.FormatColumns(f.writer, table.Columns, table.PrimaryKey, table.Relations)
	f.formatIndexes(f.writer, table.Indexes, table.Columns)
	f.FormatRelations(f.writer, table.Name, table.Relations)

	return nil
}

// FormatColumns writes column information as a markdown table
func (f *MarkdownFormatter) FormatColumns(w io.Writer, columns []schema.Column, primaryKey []string, relations []schema.Relation) {
	// Check if any column has CHECK constraints
	hasConstraints := false
	for _, col := range columns {
		if col.CheckConstraint != nil {
			hasConstraints = true
			break
		}
	}

	// Build header
	if hasConstraints {
		_, _ = fmt.Fprintln(w, "| Column | Type | Constraints |")
		_, _ = fmt.Fprintln(w, "|--------|------|-------------|")
	} else {
		_, _ = fmt.Fprintln(w, "| Column | Type |")
		_, _ = fmt.Fprintln(w, "|--------|------|")
	}

	for _, col := range columns {
		// Build type string with PK prefix, nullability, and default
		typeStr := buildTypeString(col, primaryKey)

		if hasConstraints {
			constraints := FormatTableConstraints(col, primaryKey)
			_, _ = fmt.Fprintf(w, "| %s | %s | %s |\n", col.Name, typeStr, constraints)
		} else {
			_, _ = fmt.Fprintf(w, "| %s | %s |\n", col.Name, typeStr)
		}
	}
	_, _ = fmt.Fprintln(w)
}

// buildTypeString builds SQL-like type string with PK prefix, nullability, and default
func buildTypeString(col schema.Column, primaryKey []string) string {
	var parts []string

	// Check if this column is part of the primary key
	isPK := false
	for _, pk := range primaryKey {
		if pk == col.Name {
			isPK = true
			break
		}
	}

	if isPK {
		parts = append(parts, "PK")
	}

	// Add base type with enum values if present
	typeStr := col.Type
	if len(col.EnumValues) > 0 {
		typeStr = fmt.Sprintf("%s (%s)", col.Type, strings.Join(col.EnumValues, ", "))
	}
	parts = append(parts, typeStr)

	// Add NOT NULL if applicable
	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}

	// Add DEFAULT if present
	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
	}

	// Add UNIQUE if applicable and not PK
	if col.IsUnique && !isPK {
		parts = append(parts, "UNIQUE")
	}

	return strings.Join(parts, " ")
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
	f.formatIndexes(w, indexes, nil)
}

// formatIndexes writes index information, optionally filtering out single-column unique indexes
func (f *MarkdownFormatter) formatIndexes(w io.Writer, indexes []schema.Index, columns []schema.Column) {
	if len(indexes) == 0 {
		return
	}

	// Filter out single-column unique indexes if the column is already marked as UNIQUE
	var filteredIndexes []schema.Index
	for _, idx := range indexes {
		// Skip single-column unique indexes if column already has IsUnique
		if idx.IsUnique && len(idx.Columns) == 1 && columns != nil {
			skip := false
			for _, col := range columns {
				if col.Name == idx.Columns[0] && col.IsUnique {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		filteredIndexes = append(filteredIndexes, idx)
	}

	if len(filteredIndexes) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "### Index")
	_, _ = fmt.Fprintln(w)
	for _, idx := range filteredIndexes {
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

// FormatTableConstraints formats constraints for table output (only CHECK constraints now)
func FormatTableConstraints(col schema.Column, primaryKey []string) string {
	if col.CheckConstraint != nil {
		return fmt.Sprintf("CHECK(%s)", *col.CheckConstraint)
	}
	return ""
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
