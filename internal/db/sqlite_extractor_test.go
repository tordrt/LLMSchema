package db

import (
	"context"
	"testing"
)

func TestSQLiteExtractorIncludesDatabaseMetadata(t *testing.T) {
	ctx := context.Background()
	client, err := NewSQLiteClient(ctx, ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteClient() failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	s, err := NewSQLiteExtractor(client).ExtractSchema(ctx, nil)
	if err != nil {
		t.Fatalf("ExtractSchema() failed: %v", err)
	}
	if s.DatabaseType != "SQLite" {
		t.Errorf("DatabaseType = %q, want SQLite", s.DatabaseType)
	}
	if s.DatabaseVersion == "" {
		t.Error("DatabaseVersion is empty")
	}
}
