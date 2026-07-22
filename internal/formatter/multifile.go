package formatter

import (
	"crypto/sha256"
	"encoding/json"
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
	generatedFilesManifest    = ".llmschema-manifest.json"
	maxGeneratedFileNameBytes = 120
	truncatedNameHashHexChars = 12
)

// MultiFileFormatter writes schema to multiple files in a directory
type MultiFileFormatter struct {
	OutputDir          string
	OutputFormat       string // "text" or "markdown"
	OmitDatabaseInfo   bool
	PreserveStaleFiles bool
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

	previousFiles, err := f.readGeneratedFilesManifest()
	if err != nil {
		return fmt.Errorf("failed to read generated files manifest: %w", err)
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

	currentFiles := f.tableFileNames(s.Tables)
	if f.PreserveStaleFiles {
		currentFiles = mergeFileNames(previousFiles, currentFiles)
	} else if err := f.removeStaleGeneratedFiles(previousFiles, currentFiles); err != nil {
		return err
	}
	if err := f.writeGeneratedFilesManifest(currentFiles); err != nil {
		return fmt.Errorf("failed to write generated files manifest: %w", err)
	}

	return nil
}

func (f *MultiFileFormatter) tableFileNames(tables []schema.Table) []string {
	files := make([]string, 0, len(tables))
	for _, table := range tables {
		files = append(files, f.tableFileName(table.Name))
	}
	sort.Strings(files)
	return files
}

func (f *MultiFileFormatter) readGeneratedFilesManifest() ([]string, error) {
	content, err := os.ReadFile(filepath.Join(f.OutputDir, generatedFilesManifest))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var files []string
	if err := json.Unmarshal(content, &files); err != nil {
		return nil, err
	}
	for _, name := range files {
		if !validGeneratedFileName(name) {
			return nil, fmt.Errorf("invalid generated filename %q", name)
		}
	}
	return files, nil
}

func (f *MultiFileFormatter) writeGeneratedFilesManifest(files []string) error {
	content, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	tempFile, err := os.CreateTemp(f.OutputDir, ".llmschema-manifest-*.tmp")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	defer func() { _ = os.Remove(tempName) }()

	if err := tempFile.Chmod(0644); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempName, filepath.Join(f.OutputDir, generatedFilesManifest))
}

func (f *MultiFileFormatter) removeStaleGeneratedFiles(previousFiles, currentFiles []string) error {
	current := make(map[string]bool, len(currentFiles))
	for _, name := range currentFiles {
		current[name] = true
	}
	for _, name := range previousFiles {
		if current[name] {
			continue
		}
		if err := os.Remove(filepath.Join(f.OutputDir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove stale generated file %q: %w", name, err)
		}
	}
	return nil
}

func validGeneratedFileName(name string) bool {
	validExtension := strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".txt")
	return name != "" && name != "." && name != ".." && name != "_overview.md" && name != "_overview.txt" &&
		filepath.Base(name) == name && len(name) <= maxGeneratedFileNameBytes && validExtension
}

func mergeFileNames(first, second []string) []string {
	seen := make(map[string]bool, len(first)+len(second))
	for _, name := range first {
		seen[name] = true
	}
	for _, name := range second {
		seen[name] = true
	}

	files := make([]string, 0, len(seen))
	for name := range seen {
		files = append(files, name)
	}
	sort.Strings(files)
	return files
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
				targets = append(targets, formatRelationTable(rel))
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
				targets = append(targets, formatRelationTable(rel))
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
				cardinalityDesc := FormatCardinality(rel.Relation.Cardinality, rel.SourceTable, rel.Relation.TargetTable)
				if _, err := fmt.Fprintf(file, "- %s → %s (%s)\n",
					formatIncomingSource(rel.SourceTable, relationSourceColumns(rel.Relation)),
					formatSourceColumns(relationTargetColumns(rel.Relation)),
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
	SourceTable string
	Relation    schema.Relation
}

// findIncomingRelations finds all foreign keys pointing to this table
func (f *MultiFileFormatter) findIncomingRelations(tableName string, s *schema.Schema) []IncomingRelation {
	var incoming []IncomingRelation

	for _, table := range s.Tables {
		for _, rel := range table.Relations {
			// Extractors normalize references to their own schema to an empty
			// TargetSchema. A qualified target belongs to an external schema,
			// even when it happens to share this local table's name.
			if rel.TargetSchema == "" && rel.TargetTable == tableName {
				incoming = append(incoming, IncomingRelation{
					SourceTable: table.Name,
					Relation:    rel,
				})
			}
		}
	}

	return incoming
}

func formatIncomingSource(table string, columns []string) string {
	if len(columns) == 1 {
		return table + "." + columns[0]
	}
	qualified := make([]string, len(columns))
	for i, column := range columns {
		qualified[i] = table + "." + column
	}
	return "(" + strings.Join(qualified, ", ") + ")"
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
