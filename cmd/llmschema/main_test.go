package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tordrt/llmschema"
)

const testDatabaseURL = "sqlite://database.db"

func TestParseTableList(t *testing.T) {
	tests := []struct {
		name       string
		tablesStr  string
		wantTables []string
	}{
		{
			name:       "single table",
			tablesStr:  "users",
			wantTables: []string{"users"},
		},
		{
			name:       "multiple tables",
			tablesStr:  "users,posts,comments",
			wantTables: []string{"users", "posts", "comments"},
		},
		{
			name:       "tables with spaces",
			tablesStr:  "users, posts, comments",
			wantTables: []string{"users", "posts", "comments"},
		},
		{
			name:       "empty string",
			tablesStr:  "",
			wantTables: nil,
		},
		{
			name:       "empty entries",
			tablesStr:  "users,, ,posts,",
			wantTables: []string{"users", "posts"},
		},
		{
			name:       "only empty entries",
			tablesStr:  ", ,",
			wantTables: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTables := parseTableList(tt.tablesStr)

			if len(gotTables) != len(tt.wantTables) {
				t.Fatalf("parseTableList() returned %d tables, want %d", len(gotTables), len(tt.wantTables))
			}

			for i, table := range gotTables {
				if table != tt.wantTables[i] {
					t.Errorf("parseTableList() table[%d] = %s, want %s", i, table, tt.wantTables[i])
				}
			}
		})
	}
}

func TestRootCommandDescribesSingleFileAsDefault(t *testing.T) {
	cmd := newRootCmd(llmschema.ExtractAndFormat)

	if got := cmd.Flags().Lookup("output").Usage; got != "Output file for the single-file schema (default: stdout)" {
		t.Errorf("--output usage = %q", got)
	}
	if got := cmd.Flags().Lookup("output-dir").Usage; got != "Output directory for optional multi-file output" {
		t.Errorf("--output-dir usage = %q", got)
	}
	if got := cmd.Flags().Lookup("no-table-index").Usage; got != "Exclude the table index from single-file output" {
		t.Errorf("--no-table-index usage = %q", got)
	}
	if got := cmd.Flags().Lookup("no-database-info").Usage; got != "Exclude database type and version from the output" {
		t.Errorf("--no-database-info usage = %q", got)
	}
}

func TestRootCommandValidation(t *testing.T) {
	t.Setenv(databaseURLEnv, "")

	tests := []struct {
		name        string
		args        []string
		wantErrText string
	}{
		{
			name:        "database URL is required",
			wantErrText: "--db-url is required or DATABASE_URL must be set",
		},
		{
			name:        "positional arguments are rejected",
			args:        []string{"--db-url", "invalid://database", "unexpected"},
			wantErrText: "unknown command",
		},
		{
			name:        "output modes are mutually exclusive",
			args:        []string{"--db-url", "invalid://database", "--output", "schema.md", "--output-dir", "schema"},
			wantErrText: "if any flags in the group [output output-dir] are set none of the others can be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd(func(context.Context, string, *llmschema.Options, *llmschema.OutputOptions) error {
				t.Fatal("ExtractAndFormat called for invalid arguments")
				return nil
			})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("Execute() error = %v, want error containing %q", err, tt.wantErrText)
			}
		})
	}
}

func TestRootCommandDatabaseURLSources(t *testing.T) {
	tests := []struct {
		name      string
		envURL    string
		args      []string
		wantDBURL string
	}{
		{
			name:      "environment variable",
			envURL:    "sqlite://environment.db",
			wantDBURL: "sqlite://environment.db",
		},
		{
			name:      "flag takes precedence",
			envURL:    "sqlite://environment.db",
			args:      []string{"--db-url", testDatabaseURL},
			wantDBURL: testDatabaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(databaseURLEnv, tt.envURL)
			cmd := newRootCmd(func(_ context.Context, databaseURL string, _ *llmschema.Options, _ *llmschema.OutputOptions) error {
				if databaseURL != tt.wantDBURL {
					t.Errorf("database URL = %q, want %q", databaseURL, tt.wantDBURL)
				}
				return nil
			})
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() failed: %v", err)
			}
		})
	}
}

func TestRootCommandVersion(t *testing.T) {
	originalVersion := version
	version = "1.2.3"
	t.Cleanup(func() {
		version = originalVersion
	})

	var output strings.Builder
	cmd := newRootCmd(func(context.Context, string, *llmschema.Options, *llmschema.OutputOptions) error {
		t.Fatal("ExtractAndFormat called for --version")
		return nil
	})
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if got, want := output.String(), "llmschema version 1.2.3\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRootCommandPassesContextAndOptionsToLibrary(t *testing.T) {
	type contextKey string
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey("test"), "command context"))
	cancel()

	var called bool
	cmd := newRootCmd(func(gotCtx context.Context, databaseURL string, opts *llmschema.Options, outOpts *llmschema.OutputOptions) error {
		called = true
		if got := gotCtx.Value(contextKey("test")); got != "command context" {
			t.Errorf("context value = %v, want command context", got)
		}
		if !errors.Is(gotCtx.Err(), context.Canceled) {
			t.Errorf("context error = %v, want context canceled", gotCtx.Err())
		}
		if databaseURL != testDatabaseURL {
			t.Errorf("database URL = %q, want %s", databaseURL, testDatabaseURL)
		}
		assertStringsEqual(t, "tables", opts.Tables, []string{"users", "posts"})
		assertStringsEqual(t, "excluded tables", opts.ExcludeTables, []string{"migrations"})
		if opts.SchemaName != "main" {
			t.Errorf("schema name = %q, want main", opts.SchemaName)
		}
		if outOpts.OutputDir != "docs/schema" {
			t.Errorf("output directory = %q, want docs/schema", outOpts.OutputDir)
		}
		if outOpts.Writer != nil {
			t.Errorf("writer = %v, want nil for multi-file output", outOpts.Writer)
		}
		if !outOpts.OmitDatabaseInfo {
			t.Error("OmitDatabaseInfo = false, want true")
		}
		if !outOpts.PreserveStaleFiles {
			t.Error("PreserveStaleFiles = false, want true")
		}
		if !outOpts.OmitTableIndex {
			t.Error("OmitTableIndex = false, want true")
		}
		return nil
	})
	cmd.SetArgs([]string{
		"--db-url", testDatabaseURL,
		"--tables", "users, posts",
		"--exclude-tables", "migrations",
		"--schema", "main",
		"--output-dir", "docs/schema",
		"--no-database-info",
		"--no-table-index",
		"--preserve-stale-files",
	})

	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() failed: %v", err)
	}
	if !called {
		t.Fatal("ExtractAndFormat was not called")
	}
}

func TestRootCommandWritesToStdoutByDefault(t *testing.T) {
	var gotWriter io.Writer
	cmd := newRootCmd(func(_ context.Context, _ string, _ *llmschema.Options, outOpts *llmschema.OutputOptions) error {
		gotWriter = outOpts.Writer
		return nil
	})
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"--db-url", testDatabaseURL})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if gotWriter != cmd.OutOrStdout() {
		t.Errorf("writer = %v, want command stdout", gotWriter)
	}
}

func TestRootCommandSuggestsSchemaFlagWhenMySQLDatabaseNameIsMissing(t *testing.T) {
	cmd := newRootCmd(func(_ context.Context, _ string, _ *llmschema.Options, _ *llmschema.OutputOptions) error {
		return errors.New("failed to determine database name: no database name found in connection string (please specify --schema in the CLI or Options.SchemaName in the library API)")
	})
	cmd.SetArgs([]string{"--db-url", "mysql://user:pass@tcp(localhost:3306)/"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "please specify --schema") {
		t.Fatalf("Execute() error = %v, want guidance to specify --schema", err)
	}
}

func TestRootCommandPreservesOutputWhenExtractionFails(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "schema.md")
	const original = "existing schema\n"
	if err := os.WriteFile(outputPath, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to create existing output: %v", err)
	}

	extractionErr := errors.New("extraction failed")
	cmd := newRootCmd(func(_ context.Context, _ string, _ *llmschema.Options, _ *llmschema.OutputOptions) error {
		return extractionErr
	})
	cmd.SetArgs([]string{"--db-url", testDatabaseURL, "--output", outputPath})

	if err := cmd.Execute(); !errors.Is(err, extractionErr) {
		t.Fatalf("Execute() error = %v, want %v", err, extractionErr)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read existing output: %v", err)
	}
	if string(content) != original {
		t.Fatalf("output content = %q, want %q", content, original)
	}
}

func TestRootCommandCreatesOutputWhenFormattingStarts(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "schema.md")
	cmd := newRootCmd(func(_ context.Context, _ string, _ *llmschema.Options, outOpts *llmschema.OutputOptions) error {
		_, err := io.WriteString(outOpts.Writer, "new schema\n")
		return err
	})
	cmd.SetArgs([]string{"--db-url", testDatabaseURL, "--output", outputPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(content) != "new schema\n" {
		t.Fatalf("output content = %q, want %q", content, "new schema\n")
	}
}

func assertStringsEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
