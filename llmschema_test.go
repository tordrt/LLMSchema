//go:build integration
// +build integration

package llmschema

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSchema(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		url        string
		opts       *Options
		wantTables []string
		wantErr    bool
	}{
		{
			name:       "SQLite all tables",
			url:        "sqlite://test.db",
			opts:       nil,
			wantTables: []string{"users", "products", "orders", "order_items"},
			wantErr:    false,
		},
		{
			name:       "SQLite specific tables",
			url:        "sqlite://test.db",
			opts:       &Options{Tables: []string{"users", "products"}},
			wantTables: []string{"users", "products"},
			wantErr:    false,
		},
		{
			name:    "Invalid URL scheme",
			url:     "invalid://test.db",
			opts:    nil,
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			opts:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ExtractSchema(ctx, tt.url, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(schema.Tables) != len(tt.wantTables) {
				t.Errorf("Expected %d tables, got %d", len(tt.wantTables), len(schema.Tables))
			}

			tableMap := make(map[string]bool)
			for _, table := range schema.Tables {
				tableMap[table.Name] = true
			}

			for _, tableName := range tt.wantTables {
				if !tableMap[tableName] {
					t.Errorf("Expected table %s not found", tableName)
				}
			}
		})
	}
}

func TestExtractSchemaWithExclusion(t *testing.T) {
	ctx := context.Background()

	opts := &Options{
		ExcludeTables: []string{"orders", "order_items"},
	}

	schema, err := ExtractSchema(ctx, "sqlite://test.db", opts)
	if err != nil {
		t.Fatalf("ExtractSchema failed: %v", err)
	}

	// Note: ExtractSchema doesn't apply exclusions, only ExtractAndFormat does
	// So we test that exclusions are ignored here
	if len(schema.Tables) != 4 {
		t.Errorf("Expected 4 tables (exclusions not applied in ExtractSchema), got %d", len(schema.Tables))
	}
}

func TestFormatSchemaToWriter(t *testing.T) {
	ctx := context.Background()

	schema, err := ExtractSchema(ctx, "sqlite://test.db", &Options{
		Tables: []string{"users"},
	})
	if err != nil {
		t.Fatalf("ExtractSchema failed: %v", err)
	}

	var buf bytes.Buffer
	opts := &OutputOptions{
		Writer: &buf,
	}

	err = FormatSchema(schema, opts)
	if err != nil {
		t.Fatalf("FormatSchema failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "users") {
		t.Error("Expected output to contain 'users' table")
	}
	if !strings.Contains(output, "username") {
		t.Error("Expected output to contain 'username' column")
	}
}

func TestFormatSchemaToDirectory(t *testing.T) {
	ctx := context.Background()

	schema, err := ExtractSchema(ctx, "sqlite://test.db", &Options{
		Tables: []string{"users", "products"},
	})
	if err != nil {
		t.Fatalf("ExtractSchema failed: %v", err)
	}

	tmpDir := t.TempDir()
	opts := &OutputOptions{
		OutputDir: tmpDir,
	}

	err = FormatSchema(schema, opts)
	if err != nil {
		t.Fatalf("FormatSchema failed: %v", err)
	}

	// Check that files were created
	usersFile := filepath.Join(tmpDir, "users.md")
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		t.Error("Expected users.md to be created")
	}

	productsFile := filepath.Join(tmpDir, "products.md")
	if _, err := os.Stat(productsFile); os.IsNotExist(err) {
		t.Error("Expected products.md to be created")
	}

	// Verify content
	content, err := os.ReadFile(usersFile)
	if err != nil {
		t.Fatalf("Failed to read users.md: %v", err)
	}

	if !strings.Contains(string(content), "username") {
		t.Error("Expected users.md to contain 'username' column")
	}
}

func TestExtractAndFormat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		url     string
		opts    *Options
		outOpts *OutputOptions
		wantErr bool
		verify  func(t *testing.T, outOpts *OutputOptions)
	}{
		{
			name: "Single file output",
			url:  "sqlite://test.db",
			opts: &Options{
				Tables: []string{"users"},
			},
			outOpts: &OutputOptions{
				Writer: &bytes.Buffer{},
			},
			wantErr: false,
			verify: func(t *testing.T, outOpts *OutputOptions) {
				buf := outOpts.Writer.(*bytes.Buffer)
				output := buf.String()
				if !strings.Contains(output, "users") {
					t.Error("Expected output to contain 'users'")
				}
			},
		},
		{
			name: "Multi-file output",
			url:  "sqlite://test.db",
			opts: &Options{
				Tables: []string{"users", "products"},
			},
			outOpts: &OutputOptions{
				OutputDir: t.TempDir(),
			},
			wantErr: false,
			verify: func(t *testing.T, outOpts *OutputOptions) {
				usersFile := filepath.Join(outOpts.OutputDir, "users.md")
				if _, err := os.Stat(usersFile); os.IsNotExist(err) {
					t.Error("Expected users.md to be created")
				}
			},
		},
		{
			name: "With exclusions",
			url:  "sqlite://test.db",
			opts: &Options{
				ExcludeTables: []string{"orders", "order_items"},
			},
			outOpts: &OutputOptions{
				Writer: &bytes.Buffer{},
			},
			wantErr: false,
			verify: func(t *testing.T, outOpts *OutputOptions) {
				buf := outOpts.Writer.(*bytes.Buffer)
				output := buf.String()
				if strings.Contains(output, "orders") {
					t.Error("Expected orders to be excluded from output")
				}
				if !strings.Contains(output, "users") {
					t.Error("Expected users to be in output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExtractAndFormat(ctx, tt.url, tt.opts, tt.outOpts)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, tt.outOpts)
			}
		})
	}
}

func TestDatabaseURLParsing(t *testing.T) {
	tests := []struct {
		url         string
		wantType    string
		wantConnStr string
		wantErr     bool
	}{
		{
			url:         "postgres://user:pass@localhost/db",
			wantType:    "postgres",
			wantConnStr: "postgres://user:pass@localhost/db",
			wantErr:     false,
		},
		{
			url:         "postgresql://user:pass@localhost/db",
			wantType:    "postgres",
			wantConnStr: "postgresql://user:pass@localhost/db",
			wantErr:     false,
		},
		{
			url:         "mysql://user:pass@tcp(localhost:3306)/db",
			wantType:    "mysql",
			wantConnStr: "user:pass@tcp(localhost:3306)/db",
			wantErr:     false,
		},
		{
			url:         "sqlite://test.db",
			wantType:    "sqlite",
			wantConnStr: "test.db",
			wantErr:     false,
		},
		{
			url:     "invalid://test",
			wantErr: true,
		},
		{
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			dbType, connStr, err := parseDatabaseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if dbType != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, dbType)
			}

			if connStr != tt.wantConnStr {
				t.Errorf("Expected connStr %s, got %s", tt.wantConnStr, connStr)
			}
		})
	}
}
