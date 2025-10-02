package main

import (
	"testing"

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
