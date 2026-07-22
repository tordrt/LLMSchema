package main

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/tordrt/llmschema/internal/schema"
)

func TestFilterExcludedTables(t *testing.T) {
	tests := []struct {
		name        string
		schema      *schema.Schema
		excludeList []string
		wantTables  []string
	}{
		{
			name: "exclude single table",
			schema: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
					{Name: "posts"},
					{Name: "comments"},
				},
			},
			excludeList: []string{"posts"},
			wantTables:  []string{"users", "comments"},
		},
		{
			name: "exclude multiple tables",
			schema: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
					{Name: "posts"},
					{Name: "comments"},
					{Name: "likes"},
				},
			},
			excludeList: []string{"posts", "likes"},
			wantTables:  []string{"users", "comments"},
		},
		{
			name: "exclude no tables",
			schema: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
					{Name: "posts"},
				},
			},
			excludeList: []string{},
			wantTables:  []string{"users", "posts"},
		},
		{
			name: "exclude non-existent table",
			schema: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
					{Name: "posts"},
				},
			},
			excludeList: []string{"products"},
			wantTables:  []string{"users", "posts"},
		},
		{
			name: "exclude all tables",
			schema: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
					{Name: "posts"},
				},
			},
			excludeList: []string{"users", "posts"},
			wantTables:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExcludedTables(tt.schema, tt.excludeList)

			if len(tt.schema.Tables) != len(tt.wantTables) {
				t.Errorf("filterExcludedTables() resulted in %d tables, want %d", len(tt.schema.Tables), len(tt.wantTables))
				return
			}

			for i, table := range tt.schema.Tables {
				if table.Name != tt.wantTables[i] {
					t.Errorf("filterExcludedTables() table[%d] = %s, want %s", i, table.Name, tt.wantTables[i])
				}
			}
		})
	}
}

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
				t.Errorf("parseTableList() returned %d tables, want %d", len(gotTables), len(tt.wantTables))
				return
			}

			for i, table := range gotTables {
				if table != tt.wantTables[i] {
					t.Errorf("parseTableList() table[%d] = %s, want %s", i, table, tt.wantTables[i])
				}
			}
		})
	}
}

func TestRootCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErrText string
	}{
		{
			name:        "database URL is required",
			wantErrText: `required flag(s) "db-url" not set`,
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
			resetRootCommandFlags(t)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("Execute() error = %v, want error containing %q", err, tt.wantErrText)
			}
		})
	}
}

func TestRootCommandAcceptsValidArguments(t *testing.T) {
	resetRootCommandFlags(t)
	if err := rootCmd.Flags().Set("db-url", "sqlite://database.db"); err != nil {
		t.Fatalf("failed to set --db-url: %v", err)
	}
	if err := rootCmd.Flags().Set("output", "schema.md"); err != nil {
		t.Fatalf("failed to set --output: %v", err)
	}

	if err := rootCmd.Args(rootCmd, nil); err != nil {
		t.Fatalf("argument validation failed: %v", err)
	}
	if err := rootCmd.ValidateRequiredFlags(); err != nil {
		t.Fatalf("required flag validation failed: %v", err)
	}
	if err := rootCmd.ValidateFlagGroups(); err != nil {
		t.Fatalf("flag group validation failed: %v", err)
	}
}

func resetRootCommandFlags(t *testing.T) {
	t.Helper()
	rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("failed to reset --%s: %v", flag.Name, err)
		}
		flag.Changed = false
	})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		})
	})
}

func TestPreserveStaleFilesFlagIsAvailable(t *testing.T) {
	flag := rootCmd.Flags().Lookup("preserve-stale-files")
	if flag == nil {
		t.Fatal("--preserve-stale-files flag is not registered")
	}
	if flag.DefValue != "false" {
		t.Fatalf("--preserve-stale-files default = %q, want false", flag.DefValue)
	}
}
