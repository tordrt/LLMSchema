package formatter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tordrt/llmschema/internal/schema"
)

const (
	formatMarkdown = "markdown"
	formatText     = "text"
)

// MultiFileFormatter writes schema to multiple files in a directory
type MultiFileFormatter struct {
	OutputDir        string
	OutputFormat     string // "text" or "markdown"
	OmitDatabaseInfo bool
}

// NewMultiFileFormatter creates a new multi-file formatter
func NewMultiFileFormatter(outputDir, format string) *MultiFileFormatter {
	return &MultiFileFormatter{
		OutputDir:    outputDir,
		OutputFormat: format,
	}
}

// Format writes the schema to multiple files
func (f *MultiFileFormatter) Format(s *schema.Schema) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(f.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write overview file
	if err := f.writeOverview(s); err != nil {
		return fmt.Errorf("failed to write overview: %w", err)
	}

	// Write per-table files
	for _, table := range s.Tables {
		if err := f.writeTableFile(&table, s); err != nil {
			return fmt.Errorf("failed to write table file for %s: %w", table.Name, err)
		}
	}

	return nil
}

// writeOverview writes the overview file
func (f *MultiFileFormatter) writeOverview(s *schema.Schema) (err error) {
	ext := f.getFileExtension()
	filename := filepath.Join(f.OutputDir, "_overview"+ext)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	if f.OutputFormat == formatMarkdown {
		return f.writeMarkdownOverview(file, s)
	}
	return f.writeTextOverview(file, s)
}

func (f *MultiFileFormatter) writeMarkdownOverview(file io.Writer, s *schema.Schema) error {
	if _, err := fmt.Fprintf(file, "# Schema Overview\n\n"); err != nil {
		return err
	}
	if !f.OmitDatabaseInfo && s.DatabaseType != "" {
		if _, err := fmt.Fprintf(file, "**Database:** %s", s.DatabaseType); err != nil {
			return err
		}
		if s.DatabaseVersion != "" {
			if _, err := fmt.Fprintf(file, " %s", s.DatabaseVersion); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(file, "\n\n"); err != nil {
			return err
		}
	}
	examplePath := filepath.Join(f.OutputDir, "<table_name>"+f.getFileExtension())
	if _, err := fmt.Fprintf(file, "Each table has its own documentation file `%s`\n\n", examplePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "## Tables\n\n"); err != nil {
		return err
	}

	// Sort tables alphabetically
	sortedTables := make([]schema.Table, len(s.Tables))
	copy(sortedTables, s.Tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		if _, err := fmt.Fprintf(file, "- **%s**", table.Name); err != nil {
			return err
		}

		// Show outgoing relationships
		if len(table.Relations) > 0 {
			targets := []string{}
			for _, rel := range table.Relations {
				targets = append(targets, rel.TargetTable)
			}
			if _, err := fmt.Fprintf(file, " (references: %s)", strings.Join(targets, ", ")); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(file, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func (f *MultiFileFormatter) writeTextOverview(file io.Writer, s *schema.Schema) error {
	if _, err := fmt.Fprintf(file, "SCHEMA OVERVIEW\n"); err != nil {
		return err
	}
	examplePath := filepath.Join(f.OutputDir, "<table_name>"+f.getFileExtension())
	if _, err := fmt.Fprintf(file, "Each table has a file: %s\n\n", examplePath); err != nil {
		return err
	}

	// Sort tables alphabetically
	sortedTables := make([]schema.Table, len(s.Tables))
	copy(sortedTables, s.Tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		if _, err := fmt.Fprintf(file, "%s", table.Name); err != nil {
			return err
		}
		if len(table.Relations) > 0 {
			targets := []string{}
			for _, rel := range table.Relations {
				targets = append(targets, rel.TargetTable)
			}
			if _, err := fmt.Fprintf(file, " (references: %s)", strings.Join(targets, ",")); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(file, "\n"); err != nil {
			return err
		}
	}

	return nil
}

// writeTableFile writes a single table to its own file
func (f *MultiFileFormatter) writeTableFile(table *schema.Table, s *schema.Schema) (err error) {
	ext := f.getFileExtension()
	filename := filepath.Join(f.OutputDir, table.Name+ext)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	if f.OutputFormat == formatMarkdown {
		// Create a markdown formatter to reuse formatting logic
		mdFormatter := NewMarkdownFormatter(file)

		// Format table header
		if _, err := fmt.Fprintf(file, "## %s\n\n", table.Name); err != nil {
			return err
		}

		// Use shared formatting methods
		if err := mdFormatter.FormatColumns(file, table.Columns, table.PrimaryKey, table.Relations); err != nil {
			return err
		}
		if err := mdFormatter.formatIndexes(file, table.Indexes, table.Columns); err != nil {
			return err
		}
		if err := mdFormatter.FormatRelations(file, table.Name, table.Relations); err != nil {
			return err
		}

		// Add incoming relationships
		incomingRels := f.findIncomingRelations(table.Name, s)
		if len(incomingRels) > 0 {
			if _, err := fmt.Fprintf(file, "### Referenced by\n\n"); err != nil {
				return err
			}
			for _, rel := range incomingRels {
				cardinalityDesc := FormatCardinality(rel.Cardinality, rel.SourceTable, rel.TargetTable)
				if _, err := fmt.Fprintf(file, "- %s.%s → %s (%s)\n",
					rel.SourceTable, rel.SourceColumn,
					rel.TargetColumn,
					cardinalityDesc); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(file); err != nil {
				return err
			}
		}
	}

	return nil
}

// IncomingRelation represents a relationship pointing to this table
type IncomingRelation struct {
	SourceTable  string
	SourceColumn string
	TargetTable  string
	TargetColumn string
	Cardinality  string
}

// findIncomingRelations finds all foreign keys pointing to this table
func (f *MultiFileFormatter) findIncomingRelations(tableName string, s *schema.Schema) []IncomingRelation {
	var incoming []IncomingRelation

	for _, table := range s.Tables {
		for _, rel := range table.Relations {
			if rel.TargetTable == tableName {
				incoming = append(incoming, IncomingRelation{
					SourceTable:  table.Name,
					SourceColumn: rel.SourceColumn,
					TargetTable:  rel.TargetTable,
					TargetColumn: rel.TargetColumn,
					Cardinality:  rel.Cardinality,
				})
			}
		}
	}

	return incoming
}

func (f *MultiFileFormatter) getFileExtension() string {
	if f.OutputFormat == formatMarkdown {
		return ".md"
	}
	return ".txt"
}
