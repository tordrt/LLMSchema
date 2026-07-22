package formatter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

func TestOverviewFormattersPropagateWriteErrors(t *testing.T) {
	formatter := NewMultiFileFormatter("schema", formatMarkdown)
	s := &schema.Schema{Tables: []schema.Table{{Name: "users"}}}

	if err := formatter.writeMarkdownOverview(failingWriter{}, s); !errors.Is(err, errWriteFailed) {
		t.Fatalf("writeMarkdownOverview() error = %v, want %v", err, errWriteFailed)
	}
	if err := formatter.writeTextOverview(failingWriter{}, s); !errors.Is(err, errWriteFailed) {
		t.Fatalf("writeTextOverview() error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarkdownOverviewIncludesDatabaseInfoByDefault(t *testing.T) {
	outputDir := t.TempDir()
	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	s := &schema.Schema{
		DatabaseType:    "PostgreSQL",
		DatabaseVersion: "17.5",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "_overview.md"))
	if err != nil {
		t.Fatalf("failed to read overview: %v", err)
	}
	if !strings.Contains(string(content), "**Database:** PostgreSQL 17.5") {
		t.Fatalf("overview does not contain database info:\n%s", content)
	}
}

func TestMarkdownOverviewCanOmitDatabaseInfo(t *testing.T) {
	outputDir := t.TempDir()
	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	formatter.OmitDatabaseInfo = true
	s := &schema.Schema{
		DatabaseType:    "PostgreSQL",
		DatabaseVersion: "17.5",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "_overview.md"))
	if err != nil {
		t.Fatalf("failed to read overview: %v", err)
	}
	if strings.Contains(string(content), "Database:") || strings.Contains(string(content), "17.5") {
		t.Fatalf("overview contains omitted database info:\n%s", content)
	}
}

func TestOverviewQualifiesExternalSchemaReferences(t *testing.T) {
	formatter := NewMultiFileFormatter("schema", formatMarkdown)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "users"},
		{
			Name: "external_profiles",
			Relations: []schema.Relation{{
				TargetSchema: "identity",
				TargetTable:  "users",
			}},
		},
	}}

	tests := []struct {
		name  string
		write func(*strings.Builder, *schema.Schema) error
	}{
		{
			name: "markdown",
			write: func(output *strings.Builder, s *schema.Schema) error {
				return formatter.writeMarkdownOverview(output, s)
			},
		},
		{
			name: "text",
			write: func(output *strings.Builder, s *schema.Schema) error {
				return formatter.writeTextOverview(output, s)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			if err := tt.write(&output, s); err != nil {
				t.Fatalf("writing overview failed: %v", err)
			}
			if !strings.Contains(output.String(), "references: identity.users") {
				t.Fatalf("overview does not qualify external reference:\n%s", output.String())
			}
		})
	}
}

func TestTableFileNameIsPortableAndCollisionSafe(t *testing.T) {
	formatter := NewMultiFileFormatter("schema", formatMarkdown)
	tests := []struct {
		name      string
		tableName string
		want      string
	}{
		{name: "common name unchanged", tableName: "users", want: "users.md"},
		{name: "path traversal encoded", tableName: "../outside", want: "~2e~2e~2foutside.md"},
		{name: "path separator encoded", tableName: "billing/invoices", want: "billing~2finvoices.md"},
		{name: "overview collision encoded", tableName: "_overview", want: "~5foverview.md"},
		{name: "uppercase encoded", tableName: "Users", want: "~55sers.md"},
		{name: "windows device name encoded", tableName: "con", want: "~63on.md"},
		{name: "unicode encoded", tableName: "café", want: "caf~c3~a9.md"},
		{name: "empty name encoded", tableName: "", want: "~empty.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.tableFileName(tt.tableName); got != tt.want {
				t.Fatalf("tableFileName(%q) = %q, want %q", tt.tableName, got, tt.want)
			}
		})
	}
}

func TestTableFileNameBoundsLongEncodedNames(t *testing.T) {
	formatter := NewMultiFileFormatter("schema", formatMarkdown)
	firstTable := strings.Repeat("A", 100)
	secondTable := strings.Repeat("A", 99) + "B"

	firstFile := formatter.tableFileName(firstTable)
	secondFile := formatter.tableFileName(secondTable)
	if len(firstFile) != maxGeneratedFileNameBytes {
		t.Fatalf("long filename has %d bytes, want %d: %q", len(firstFile), maxGeneratedFileNameBytes, firstFile)
	}
	if firstFile == secondFile {
		t.Fatalf("distinct long table names mapped to the same filename %q", firstFile)
	}
	if !strings.HasSuffix(firstFile, ".md") {
		t.Fatalf("long filename %q does not retain its extension", firstFile)
	}
	stem := strings.TrimSuffix(firstFile, ".md")
	hashSeparator := strings.LastIndex(stem, "~")
	if hashSeparator == -1 || len(stem[hashSeparator+1:]) != truncatedNameHashHexChars {
		t.Fatalf("long filename %q does not have a %d-character hash suffix", firstFile, truncatedNameHashHexChars)
	}

	outputDir := t.TempDir()
	formatter.OutputDir = outputDir
	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: firstTable}, {Name: secondTable}}}); err != nil {
		t.Fatalf("Format() failed for long table names: %v", err)
	}
	for _, name := range []string{firstFile, secondFile} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("expected output file %q: %v", name, err)
		}
	}
}

func TestMultiFileFormatterRejectsFilenameConflictsBeforeWriting(t *testing.T) {
	tests := []struct {
		name      string
		tables    []schema.Table
		wantError string
	}{
		{
			name: "truncated name collides with encoded name",
			tables: []schema.Table{
				{Name: strings.Repeat("a", 118) + "32"},
				{Name: strings.Repeat("a", 104) + "@c1b7b7ae2f"},
			},
			wantError: "table filename collision",
		},
		{
			name:      "duplicate table",
			tables:    []schema.Table{{Name: "users"}, {Name: "users"}},
			wantError: `duplicate table "users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := filepath.Join(t.TempDir(), "schema")
			formatter := NewMultiFileFormatter(outputDir, formatMarkdown)

			err := formatter.Format(&schema.Schema{Tables: tt.tables})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Format() error = %v, want an error containing %q", err, tt.wantError)
			}
			if _, statErr := os.Stat(outputDir); !os.IsNotExist(statErr) {
				t.Fatalf("output directory was created before validation; stat error = %v", statErr)
			}
		})
	}
}

func TestMultiFileFormatterKeepsTableFilesInsideOutputDirectory(t *testing.T) {
	rootDir := t.TempDir()
	outputDir := filepath.Join(rootDir, "schema")
	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "users"},
		{Name: "Users"},
		{Name: "../outside"},
		{Name: "_overview"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	wantFiles := []string{
		"_overview.md",
		"users.md",
		"~55sers.md",
		"~2e~2e~2foutside.md",
		"~5foverview.md",
	}
	for _, name := range wantFiles {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("expected output file %q: %v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(rootDir, "outside.md")); !os.IsNotExist(err) {
		t.Fatalf("table file escaped output directory; stat error = %v", err)
	}

	overview, err := os.ReadFile(filepath.Join(outputDir, "_overview.md"))
	if err != nil {
		t.Fatalf("failed to read overview: %v", err)
	}
	for _, reference := range []string{"`users.md`", "`~55sers.md`", "`~2e~2e~2foutside.md`", "`~5foverview.md`"} {
		if !strings.Contains(string(overview), reference) {
			t.Errorf("overview does not reference %s:\n%s", reference, overview)
		}
	}
}

func TestMultiFileFormatterRemovesOnlyStaleGeneratedFiles(t *testing.T) {
	outputDir := t.TempDir()
	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	initialSchema := &schema.Schema{Tables: []schema.Table{{Name: "users"}, {Name: "posts"}}}

	if err := formatter.Format(initialSchema); err != nil {
		t.Fatalf("initial Format() failed: %v", err)
	}
	supplementalFile := filepath.Join(outputDir, "notes.md")
	if err := os.WriteFile(supplementalFile, []byte("supplemental"), 0644); err != nil {
		t.Fatalf("failed to write supplemental file: %v", err)
	}

	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("second Format() failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "posts.md")); !os.IsNotExist(err) {
		t.Fatalf("stale generated file was not removed; stat error = %v", err)
	}
	if content, err := os.ReadFile(supplementalFile); err != nil || string(content) != "supplemental" {
		t.Fatalf("supplemental file was changed: content = %q, error = %v", content, err)
	}
}

func TestMultiFileFormatterCanPreserveStaleGeneratedFiles(t *testing.T) {
	outputDir := t.TempDir()
	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	initialSchema := &schema.Schema{Tables: []schema.Table{{Name: "users"}, {Name: "posts"}}}

	if err := formatter.Format(initialSchema); err != nil {
		t.Fatalf("initial Format() failed: %v", err)
	}
	formatter.PreserveStaleFiles = true
	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("preserving Format() failed: %v", err)
	}
	staleFile := filepath.Join(outputDir, "posts.md")
	if _, err := os.Stat(staleFile); err != nil {
		t.Fatalf("stale generated file was not preserved: %v", err)
	}

	formatter.PreserveStaleFiles = false
	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("cleanup Format() failed: %v", err)
	}
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Fatalf("preserved file was not removed by later cleanup; stat error = %v", err)
	}
}

func TestMultiFileFormatterDoesNotDeleteFilesWithoutManifest(t *testing.T) {
	outputDir := t.TempDir()
	legacyFile := filepath.Join(outputDir, "old_table.md")
	if err := os.WriteFile(legacyFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}
	if _, err := os.Stat(legacyFile); err != nil {
		t.Fatalf("file not recorded in a manifest was removed: %v", err)
	}
}

func TestGeneratedFilesManifestRejectsUnsafePaths(t *testing.T) {
	for _, unsafeName := range []string{"../outside.md", "..", "notes"} {
		t.Run(unsafeName, func(t *testing.T) {
			outputDir := t.TempDir()
			manifest := filepath.Join(outputDir, generatedFilesManifest)
			content := []byte("[" + fmt.Sprintf("%q", unsafeName) + "]\n")
			if err := os.WriteFile(manifest, content, 0644); err != nil {
				t.Fatalf("failed to write manifest: %v", err)
			}

			formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
			err := formatter.Format(&schema.Schema{})
			if err == nil || !strings.Contains(err.Error(), "invalid generated filename") {
				t.Fatalf("Format() error = %v, want invalid generated filename", err)
			}
		})
	}
}

func TestWriteGeneratedFilesManifestReplacesFileAtomically(t *testing.T) {
	outputDir := t.TempDir()
	manifestPath := filepath.Join(outputDir, generatedFilesManifest)
	if err := os.WriteFile(manifestPath, []byte("[\"old.md\"]\n"), 0644); err != nil {
		t.Fatalf("failed to write initial manifest: %v", err)
	}
	initialInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("failed to stat initial manifest: %v", err)
	}

	formatter := NewMultiFileFormatter(outputDir, formatMarkdown)
	if err := formatter.writeGeneratedFilesManifest([]string{"users.md"}); err != nil {
		t.Fatalf("writeGeneratedFilesManifest() failed: %v", err)
	}

	updatedInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("failed to stat updated manifest: %v", err)
	}
	if os.SameFile(initialInfo, updatedInfo) {
		t.Fatal("manifest was updated in place instead of replaced")
	}
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read updated manifest: %v", err)
	}
	if string(content) != "[\n  \"users.md\"\n]\n" {
		t.Fatalf("updated manifest = %q, want complete JSON", content)
	}
	tempFiles, err := filepath.Glob(filepath.Join(outputDir, ".llmschema-manifest-*.tmp"))
	if err != nil {
		t.Fatalf("failed to check temporary files: %v", err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("temporary manifest files were not cleaned up: %v", tempFiles)
	}
}

func TestFindIncomingRelationsExcludesExternalSchemas(t *testing.T) {
	formatter := NewMultiFileFormatter(t.TempDir(), formatMarkdown)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "users"},
		{Name: "local_profiles", Relations: []schema.Relation{{TargetTable: "users"}}},
		{Name: "external_profiles", Relations: []schema.Relation{{TargetSchema: "identity", TargetTable: "users"}}},
	}}

	incoming := formatter.findIncomingRelations("users", s)
	if len(incoming) != 1 || incoming[0].SourceTable != "local_profiles" {
		t.Fatalf("incoming relations = %#v, want only local_profiles", incoming)
	}
}
