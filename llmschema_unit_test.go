package llmschema

import (
	"errors"
	"strings"
	"testing"
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
