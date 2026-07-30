package formatter

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/tordrt/llmschema/internal/schema"
)

// MarkdownFormatter formats schema as markdown
type MarkdownFormatter struct {
	writer           io.Writer
	OmitDatabaseInfo bool
	OmitTableIndex   bool
}

// NewMarkdownFormatter creates a new markdown formatter
func NewMarkdownFormatter(w io.Writer) *MarkdownFormatter {
	return &MarkdownFormatter{writer: w}
}

// Format writes the schema in markdown format
func (f *MarkdownFormatter) Format(s *schema.Schema) error {
	if _, err := fmt.Fprintln(f.writer, "# Database Schema"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f.writer); err != nil {
		return err
	}

	if !f.OmitDatabaseInfo && s.DatabaseType != "" {
		if _, err := fmt.Fprintf(f.writer, "**Database:** %s", s.DatabaseType); err != nil {
			return err
		}
		if s.DatabaseVersion != "" {
			if _, err := fmt.Fprintf(f.writer, " %s", s.DatabaseVersion); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(f.writer, "\n"); err != nil {
			return err
		}
		if s.DatabaseName != "" {
			if _, err := fmt.Fprintf(f.writer, "**Name:** %s\n", markdownInlineCode(s.DatabaseName)); err != nil {
				return err
			}
		}
		if shouldFormatSchemaName(s) {
			if _, err := fmt.Fprintf(f.writer, "**Schema:** %s\n", markdownInlineCode(s.SchemaName)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(f.writer); err != nil {
			return err
		}
	}

	if !f.OmitTableIndex && len(s.Tables) > 0 {
		if err := f.formatTableIndex(s.Tables); err != nil {
			return err
		}
	}

	for _, table := range s.Tables {
		if err := f.formatTable(table); err != nil {
			return err
		}
	}
	return nil
}

func shouldFormatSchemaName(s *schema.Schema) bool {
	return s.SchemaName != "" &&
		!(s.DatabaseType == "PostgreSQL" && s.SchemaName == "public") &&
		!(s.DatabaseType == "MySQL" && s.SchemaName == s.DatabaseName)
}

func markdownInlineCode(value string) string {
	normalized := strings.NewReplacer(
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
	).Replace(value)

	longestRun := 0
	currentRun := 0
	for _, r := range normalized {
		if r == '`' {
			currentRun++
			longestRun = max(longestRun, currentRun)
		} else {
			currentRun = 0
		}
	}

	delimiter := strings.Repeat("`", longestRun+1)
	if longestRun > 0 ||
		((strings.HasPrefix(normalized, " ") || strings.HasSuffix(normalized, " ")) &&
			strings.Trim(normalized, " ") != "") {
		return delimiter + " " + normalized + " " + delimiter
	}
	return delimiter + normalized + delimiter
}

func (f *MarkdownFormatter) formatTableIndex(tables []schema.Table) error {
	if _, err := fmt.Fprint(f.writer, "**Tables:**\n\n"); err != nil {
		return err
	}

	usedAnchors := make(map[string]bool)
	reserveMarkdownHeadingAnchor("Database Schema", usedAnchors)

	for _, table := range tables {
		anchor := reserveMarkdownHeadingAnchor(table.Name, usedAnchors)
		tableName := escapeMarkdownLinkText(table.Name)
		if anchor == "" {
			if _, err := fmt.Fprintf(f.writer, "- %s\n", tableName); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(f.writer, "- [%s](#%s)\n", tableName, anchor); err != nil {
				return err
			}
		}

		if len(table.Indexes) > 0 {
			reserveMarkdownHeadingAnchor("Index", usedAnchors)
		}
		if len(table.Relations) > 0 {
			reserveMarkdownHeadingAnchor("References", usedAnchors)
		}
	}
	_, err := fmt.Fprintln(f.writer)
	return err
}

func reserveMarkdownHeadingAnchor(heading string, usedAnchors map[string]bool) string {
	baseAnchor := markdownHeadingAnchor(heading)
	anchor := baseAnchor
	for suffix := 1; anchor != "" && usedAnchors[anchor]; suffix++ {
		anchor = fmt.Sprintf("%s-%d", baseAnchor, suffix)
	}
	if anchor != "" {
		usedAnchors[anchor] = true
	}
	return anchor
}

func markdownHeadingAnchor(heading string) string {
	var anchor strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), r == '_', r == '-':
			anchor.WriteRune(r)
		case unicode.IsSpace(r):
			anchor.WriteByte('-')
		}
	}
	return anchor.String()
}

func escapeMarkdownLinkText(value string) string {
	return strings.NewReplacer(
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
		`\`, `\\`,
		"[", `\[`,
		"]", `\]`,
	).Replace(value)
}

// FormatTable formats a single table (exported for use by multifile formatter)
func (f *MarkdownFormatter) FormatTable(table schema.Table) error {
	return f.formatTable(table)
}

func (f *MarkdownFormatter) formatTable(table schema.Table) error {
	// Table header
	if _, err := fmt.Fprintf(f.writer, "## %s\n\n", table.Name); err != nil {
		return err
	}

	if err := f.FormatColumns(f.writer, table.Columns, table.PrimaryKey, table.Relations); err != nil {
		return err
	}
	if err := f.FormatIndexes(f.writer, table.Indexes); err != nil {
		return err
	}
	if err := f.FormatRelations(f.writer, table.Name, table.Relations); err != nil {
		return err
	}

	return nil
}

// FormatColumns writes column information as a markdown table
func (f *MarkdownFormatter) FormatColumns(w io.Writer, columns []schema.Column, primaryKey []string, relations []schema.Relation) error {
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
		if _, err := fmt.Fprintln(w, "| Column | Type | Constraints |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "|--------|------|-------------|"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "| Column | Type |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "|--------|------|"); err != nil {
			return err
		}
	}

	for _, col := range columns {
		// Build type string with PK prefix, nullability, and default
		typeStr := buildTypeString(col, primaryKey)

		if hasConstraints {
			constraints := FormatTableConstraints(col, primaryKey)
			if _, err := fmt.Fprintf(w, "| %s | %s | %s |\n",
				escapeMarkdownTableCell(col.Name),
				escapeMarkdownTableCell(typeStr),
				escapeMarkdownTableCell(constraints)); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "| %s | %s |\n",
				escapeMarkdownTableCell(col.Name),
				escapeMarkdownTableCell(typeStr)); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func escapeMarkdownTableCell(value string) string {
	return strings.NewReplacer(
		"\r\n", "<br>",
		"\r", "<br>",
		"\n", "<br>",
		`\`, `\\`,
		"|", `\|`,
	).Replace(value)
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
func (f *MarkdownFormatter) FormatRelations(w io.Writer, tableName string, relations []schema.Relation) error {
	if len(relations) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w, "### References"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for _, rel := range relations {
		cardinalityDesc := FormatCardinality(rel.Cardinality, tableName, rel.TargetTable)
		details := []string{cardinalityDesc}
		if rel.OnDelete != "" && rel.OnDelete != "NO ACTION" {
			details = append(details, "ON DELETE "+rel.OnDelete)
		}
		if rel.OnUpdate != "" && rel.OnUpdate != "NO ACTION" {
			details = append(details, "ON UPDATE "+rel.OnUpdate)
		}
		if _, err := fmt.Fprintf(w, "- %s → %s (%s)\n",
			formatSourceColumns(relationSourceColumns(rel)),
			formatRelationTarget(rel),
			strings.Join(details, "; ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func formatSourceColumns(columns []string) string {
	if len(columns) == 1 {
		return columns[0]
	}
	return "(" + strings.Join(columns, ", ") + ")"
}

func formatRelationTable(rel schema.Relation) string {
	table := rel.TargetTable
	if rel.TargetSchema != "" {
		table = rel.TargetSchema + "." + table
	}
	return table
}

func formatRelationTarget(rel schema.Relation) string {
	table := formatRelationTable(rel)
	targetColumns := relationTargetColumns(rel)
	if len(targetColumns) == 1 {
		return table + "." + targetColumns[0]
	}
	return table + "(" + strings.Join(targetColumns, ", ") + ")"
}

func relationSourceColumns(rel schema.Relation) []string {
	if len(rel.SourceColumns) == 0 && rel.SourceColumn != "" {
		return []string{rel.SourceColumn}
	}
	return rel.SourceColumns
}

func relationTargetColumns(rel schema.Relation) []string {
	if len(rel.TargetColumns) == 0 && rel.TargetColumn != "" {
		return []string{rel.TargetColumn}
	}
	return rel.TargetColumns
}

// FormatIndexes writes index information
func (f *MarkdownFormatter) FormatIndexes(w io.Writer, indexes []schema.Index) error {
	if len(indexes) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w, "### Index"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for _, idx := range indexes {
		attributes := make([]string, 0, 3)
		if idx.IsUnique {
			attributes = append(attributes, "unique")
		}
		if idx.IsPartial {
			attributes = append(attributes, "partial")
		}
		if idx.HasExpressions {
			attributes = append(attributes, "contains expressions")
		}
		suffix := ""
		if len(attributes) > 0 {
			suffix = ", " + strings.Join(attributes, ", ")
		}
		indexColumns := append([]string{}, idx.Columns...)
		if idx.HasExpressions {
			indexColumns = append(indexColumns, "`<expression>`")
		}
		if _, err := fmt.Fprintf(w, "- %s on (%s)%s\n",
			idx.Name,
			strings.Join(indexColumns, ", "), suffix); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
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
