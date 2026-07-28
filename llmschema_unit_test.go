package llmschema

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

func TestMySQLSchemaNameErrorProvidesLibraryAndCLIGuidance(t *testing.T) {
	cause := errors.New("no database name found in connection string")
	err := mySQLSchemaNameError(cause)

	if !errors.Is(err, cause) {
		t.Fatalf("mySQLSchemaNameError() does not wrap its cause: %v", err)
	}
	for _, guidance := range []string{"Options.SchemaName", "--schema"} {
		if !strings.Contains(err.Error(), guidance) {
			t.Errorf("mySQLSchemaNameError() = %q, want guidance containing %q", err, guidance)
		}
	}
}

func TestFormatSchemaCanOmitTableIndex(t *testing.T) {
	s := &schema.Schema{Tables: []schema.Table{{Name: "users"}}}

	var defaultOutput bytes.Buffer
	if err := FormatSchema(s, &OutputOptions{Writer: &defaultOutput}); err != nil {
		t.Fatalf("FormatSchema() with defaults failed: %v", err)
	}
	if !strings.Contains(defaultOutput.String(), "- [users](#users)") {
		t.Fatalf("default output missing table index:\n%s", defaultOutput.String())
	}

	var outputWithoutIndex bytes.Buffer
	if err := FormatSchema(s, &OutputOptions{
		Writer:         &outputWithoutIndex,
		OmitTableIndex: true,
	}); err != nil {
		t.Fatalf("FormatSchema() without table index failed: %v", err)
	}
	if strings.Contains(outputWithoutIndex.String(), "- [users](#users)") {
		t.Fatalf("output contains omitted table index:\n%s", outputWithoutIndex.String())
	}
}

func TestFormatSchemaCanOmitDatabaseInfo(t *testing.T) {
	s := &schema.Schema{
		DatabaseType:    "PostgreSQL",
		DatabaseVersion: "17.5",
		DatabaseName:    "app",
		SchemaName:      "billing",
	}

	var output bytes.Buffer
	if err := FormatSchema(s, &OutputOptions{
		Writer:           &output,
		OmitDatabaseInfo: true,
	}); err != nil {
		t.Fatalf("FormatSchema() without database info failed: %v", err)
	}
	if strings.Contains(output.String(), "Database:") ||
		strings.Contains(output.String(), "Database name:") ||
		strings.Contains(output.String(), "Schema:") ||
		strings.Contains(output.String(), "17.5") {
		t.Fatalf("output contains omitted database info:\n%s", output.String())
	}
}
