package formatter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

func TestFormatIncludesDatabaseInfoByDefault(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{
		DatabaseType:    "PostgreSQL",
		DatabaseVersion: "17.5",
		DatabaseName:    "app",
		SchemaName:      "billing",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n**Database:** PostgreSQL 17.5\n**Name:** `app`\n**Schema:** `billing`\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatCanOmitDatabaseInfo(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	formatter.OmitDatabaseInfo = true
	s := &schema.Schema{
		DatabaseType:    "PostgreSQL",
		DatabaseVersion: "17.5",
		DatabaseName:    "app",
		SchemaName:      "billing",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatDoesNotRepeatMySQLDatabaseNameAsSchema(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{
		DatabaseType: "MySQL",
		DatabaseName: "app",
		SchemaName:   "app",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n**Database:** MySQL\n**Name:** `app`\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatOmitsDefaultPostgresSchema(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{
		DatabaseType: "PostgreSQL",
		DatabaseName: "app",
		SchemaName:   "public",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n**Database:** PostgreSQL\n**Name:** `app`\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatIncludesPostgresSchemaMatchingDatabaseName(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{
		DatabaseType: "PostgreSQL",
		DatabaseName: "app",
		SchemaName:   "app",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n**Database:** PostgreSQL\n**Name:** `app`\n**Schema:** `app`\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatEscapesDatabaseIdentityAsInlineCode(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{
		DatabaseType: "PostgreSQL",
		DatabaseName: "app`name",
		SchemaName:   "billing\nadmin",
	}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "# Database Schema\n\n**Database:** PostgreSQL\n**Name:** `` app`name ``\n**Schema:** `billing admin`\n\n"
	if got := output.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestMarkdownInlineCode(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain", value: "app", want: "`app`"},
		{name: "backtick", value: "app`name", want: "`` app`name ``"},
		{name: "line break", value: "app\r\nname", want: "`app name`"},
		{name: "surrounding spaces", value: " app ", want: "`  app  `"},
		{name: "only space", value: " ", want: "` `"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := markdownInlineCode(tt.value); got != tt.want {
				t.Errorf("markdownInlineCode(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatIncludesLinkedTableIndexByDefault(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "Order Items"},
		{Name: "audit.logs"},
		{Name: "users"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := `# Database Schema

**Conventions:** ` + "`PK`" + ` and ` + "`UNIQUE`" + ` identify unique keys; their backing indexes are omitted from Additional indexes.

**Tables:**

- [Order Items](#order-items)
- [audit.logs](#auditlogs)
- [users](#users)

## Order Items
`
	if got := output.String(); !strings.HasPrefix(got, want) {
		t.Fatalf("output prefix:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatCanOmitTableIndex(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	formatter.OmitTableIndex = true

	if err := formatter.Format(&schema.Schema{Tables: []schema.Table{{Name: "users"}}}); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	if got := output.String(); strings.Contains(got, "**Tables:**") || strings.Contains(got, "- [users](#users)") {
		t.Fatalf("output contains table index:\n%s", got)
	}
}

func TestFormatTableIndexHandlesAnchorCollisions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "Database Schema"},
		{Name: "Database-Schema"},
		{Name: "audit.logs"},
		{Name: "auditlogs"},
		{Name: "!!!"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	for _, want := range []string{
		"- [Database Schema](#database-schema-1)",
		"- [Database-Schema](#database-schema-2)",
		"- [audit.logs](#auditlogs)",
		"- [auditlogs](#auditlogs-1)",
		"- !!!",
	} {
		if got := output.String(); !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatTableIndexAccountsForGeneratedSubheadings(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{
			Name: "orders",
			Indexes: []schema.Index{{
				Name:    "orders_created_at",
				Columns: []string{"created_at"},
			}},
			Relations: []schema.Relation{{
				SourceColumn: "user_id",
				TargetTable:  "users",
				TargetColumn: "id",
			}},
		},
		{Name: "Additional indexes"},
		{Name: "References"},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	for _, want := range []string{
		"- [Additional indexes](#additional-indexes-1)",
		"- [References](#references-1)",
	} {
		if got := output.String(); !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatRelationsSupportsCompositeKeysAndActions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	relations := []schema.Relation{
		{
			SourceColumns: []string{"photo_test_id"},
			TargetTable:   "photo_tests",
			TargetColumns: []string{"id"},
			Cardinality:   "1:1",
			OnDelete:      "CASCADE",
		},
		{
			SourceColumns: []string{"test_id", "photo_id"},
			TargetSchema:  "media",
			TargetTable:   "test_photos",
			TargetColumns: []string{"test_id", "photo_id"},
			Cardinality:   "N:1",
			OnUpdate:      "CASCADE",
		},
	}

	if err := formatter.FormatRelations(&output, "photo_results", relations); err != nil {
		t.Fatalf("FormatRelations() failed: %v", err)
	}

	got := output.String()
	wants := []string{
		"photo_test_id → photo_tests.id (one photo_results to one photo_tests; ON DELETE CASCADE)",
		"(test_id, photo_id) → media.test_photos(test_id, photo_id) (many photo_results to one test_photos; ON UPDATE CASCADE)",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestSingleFileOmitsReferencedBy(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{
		{Name: "users"},
		{
			Name: "orders",
			Relations: []schema.Relation{{
				SourceColumn: "user_id",
				TargetTable:  "users",
				TargetColumn: "id",
				Cardinality:  "N:1",
			}},
		},
	}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "user_id → users.id (many orders to one users)") {
		t.Fatalf("output does not contain outgoing reference:\n%s", got)
	}
	if strings.Contains(got, "### Referenced by") {
		t.Fatalf("single-file output contains redundant incoming references:\n%s", got)
	}
}

func TestFormatColumnsEscapesMarkdownTableCells(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	defaultValue := "'left|right\nnext\\line'"
	checkConstraint := "value <> 'a|b'\r\nAND value IS NOT NULL"

	err := formatter.FormatColumns(&output, []schema.Column{{
		Name:            "config|value",
		Type:            "text",
		Nullable:        true,
		DefaultValue:    &defaultValue,
		CheckConstraint: &checkConstraint,
	}}, nil, nil)
	if err != nil {
		t.Fatalf("FormatColumns() failed: %v", err)
	}

	want := "| config\\|value | text DEFAULT 'left\\|right<br>next\\\\line' | CHECK(value <> 'a\\|b'<br>AND value IS NOT NULL) |"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestFormatIndexesMarksExpressions(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)

	err := formatter.FormatIndexes(&output, []schema.Index{{
		Name:           "expression_children_user_label",
		Columns:        []string{"user_id"},
		IsUnique:       true,
		HasExpressions: true,
	}})
	if err != nil {
		t.Fatalf("FormatIndexes() failed: %v", err)
	}

	want := "- expression_children_user_label on (user_id, `<expression>`), unique, contains expressions"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestFormatOmitsSingleColumnUniqueKeyIndex(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{{
		Name: "users",
		Columns: []schema.Column{{
			Name:     "email",
			Type:     "text",
			Nullable: true,
			IsUnique: true,
		}},
		Indexes: []schema.Index{{
			Name:     "users_email_key",
			Columns:  []string{"email"},
			IsUnique: true,
		}},
	}}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "| email | text UNIQUE |") {
		t.Errorf("output missing UNIQUE marker:\n%s", got)
	}
	if strings.Contains(got, "users_email_key") || strings.Contains(got, "### Additional indexes") {
		t.Errorf("output repeats the unique key as an additional index:\n%s", got)
	}
}

func TestFormatIncludesCompositeKeysAndAdditionalIndexes(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{{
		Name:       "memberships",
		PrimaryKey: []string{"tenant_id", "id"},
		UniqueKeys: [][]string{{"tenant_id", "email"}},
		Columns: []schema.Column{
			{Name: "tenant_id", Type: "integer"},
			{Name: "id", Type: "integer"},
			{Name: "email", Type: "text"},
		},
		Indexes: []schema.Index{
			{Name: "memberships_tenant_email_key", Columns: []string{"tenant_id", "email"}, IsUnique: true},
			{Name: "memberships_email_idx", Columns: []string{"email"}},
			{Name: "memberships_active_email_idx", Columns: []string{"email"}, IsUnique: true, IsPartial: true},
		},
	}}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"**Primary key:** (tenant_id, id)",
		"**Unique keys:**\n\n- (tenant_id, email)",
		"### Additional indexes\n\n- memberships_email_idx on (email)",
		"- memberships_active_email_idx on (email), unique, partial",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "memberships_tenant_email_key") {
		t.Errorf("output repeats the composite unique key as an additional index:\n%s", got)
	}
}

func TestFormatPreservesExpressionIndexOnUniqueColumn(t *testing.T) {
	var output bytes.Buffer
	formatter := NewMarkdownFormatter(&output)
	s := &schema.Schema{Tables: []schema.Table{{
		Name: "expression_children",
		Columns: []schema.Column{{
			Name:     "user_id",
			Type:     "integer",
			IsUnique: true,
		}},
		Indexes: []schema.Index{{
			Name:           "expression_children_user_label",
			Columns:        []string{"user_id"},
			IsUnique:       true,
			HasExpressions: true,
		}},
	}}}

	if err := formatter.Format(s); err != nil {
		t.Fatalf("Format() failed: %v", err)
	}

	want := "- expression_children_user_label on (user_id, `<expression>`), unique, contains expressions"
	if got := output.String(); !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func TestMarkdownFormatterPropagatesWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	s := &schema.Schema{Tables: []schema.Table{{Name: "users"}}}

	if err := f.Format(s); !errors.Is(err, errWriteFailed) {
		t.Fatalf("Format() error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarkdownFormattingMethodsPropagateWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "columns",
			run: func() error {
				return f.FormatColumns(failingWriter{}, []schema.Column{{Name: "id", Type: "integer"}}, nil, nil)
			},
		},
		{
			name: "relations",
			run: func() error {
				return f.FormatRelations(failingWriter{}, "orders", []schema.Relation{{
					SourceColumns: []string{"user_id"},
					TargetTable:   "users",
					TargetColumns: []string{"id"},
				}})
			},
		},
		{
			name: "indexes",
			run: func() error {
				return f.FormatIndexes(failingWriter{}, []schema.Index{{Name: "users_pkey", Columns: []string{"id"}}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, errWriteFailed) {
				t.Fatalf("error = %v, want %v", err, errWriteFailed)
			}
		})
	}
}
