package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

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
