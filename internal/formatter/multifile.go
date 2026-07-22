package formatter

import (
	"crypto/sha256"
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
	formatMarkdown            = "markdown"
	formatText                = "text"
	maxGeneratedFileNameBytes = 120
	truncatedNameHashHexChars = 12
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
	if err := f.validateTableFileNames(s.Tables); err != nil {
		return err
	}

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
	if _, err := fmt.Fprint(file, "Each table has its own documentation file listed below.\n\n"); err != nil {
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
		if _, err := fmt.Fprintf(file, "- **%s** (file: `%s`)", table.Name, f.tableFileName(table.Name)); err != nil {
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
	if _, err := fmt.Fprint(file, "Each table has its own documentation file listed below.\n\n"); err != nil {
		return err
	}

	// Sort tables alphabetically
	sortedTables := make([]schema.Table, len(s.Tables))
	copy(sortedTables, s.Tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		if _, err := fmt.Fprintf(file, "%s (file: %s)", table.Name, f.tableFileName(table.Name)); err != nil {
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
	filename := filepath.Join(f.OutputDir, f.tableFileName(table.Name))

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

// tableFileName returns a portable filename that cannot escape OutputDir.
// Common lowercase SQL identifiers remain readable, while all other bytes are
// encoded as ~ followed by two lowercase hexadecimal digits. Encoding uppercase
// bytes also prevents collisions on case-insensitive filesystems.
func (f *MultiFileFormatter) tableFileName(tableName string) string {
	stem := encodeTableFileStem(tableName)
	if isReservedTableFileStem(stem) {
		// The encoding is injective because a literal '~' is itself encoded.
		stem = fmt.Sprintf("~%02x%s", stem[0], stem[1:])
	}

	ext := f.getFileExtension()
	maxStemBytes := maxGeneratedFileNameBytes - len(ext)
	if len(stem) > maxStemBytes {
		digest := sha256.Sum256([]byte(tableName))
		hexDigest := fmt.Sprintf("%x", digest)
		suffix := "~" + hexDigest[:truncatedNameHashHexChars]
		stem = stem[:maxStemBytes-len(suffix)] + suffix
	}
	return stem + ext
}

func (f *MultiFileFormatter) validateTableFileNames(tables []schema.Table) error {
	seen := make(map[string]string, len(tables))
	for _, table := range tables {
		filename := f.tableFileName(table.Name)
		previousTable, exists := seen[filename]
		if !exists {
			seen[filename] = table.Name
			continue
		}
		if previousTable == table.Name {
			return fmt.Errorf("duplicate table %q would write %q more than once", table.Name, filename)
		}
		return fmt.Errorf("table filename collision: %q and %q both map to %q", previousTable, table.Name, filename)
	}
	return nil
}

func encodeTableFileStem(tableName string) string {
	if tableName == "" {
		return "~empty"
	}

	var encoded strings.Builder
	for i := 0; i < len(tableName); i++ {
		b := tableName[i]
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-' {
			encoded.WriteByte(b)
			continue
		}
		_, _ = fmt.Fprintf(&encoded, "~%02x", b)
	}
	return encoded.String()
}

func isReservedTableFileStem(stem string) bool {
	if stem == "_overview" {
		return true
	}

	// Windows reserves these names even when they have a file extension.
	switch stem {
	case "con", "prn", "aux", "nul",
		"com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9",
		"lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9":
		return true
	default:
		return false
	}
}
