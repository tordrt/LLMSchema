// Package llmschema extracts database schemas and generates markdown documentation
// optimized for AI agent consumption.
//
// LLMSchema supports PostgreSQL, MySQL, and SQLite databases, producing structured
// markdown documentation that includes tables, columns, relationships, indexes, and
// constraints. The output can be generated as a single file or split across multiple
// files (one per table plus an overview).
//
// # Quick Start
//
// The simplest way to use this package is with ExtractAndFormat:
//
//	err := llmschema.ExtractAndFormat(
//		context.Background(),
//		"postgres://user:pass@localhost/db",
//		&llmschema.Options{ExcludeTables: []string{"migrations"}},
//		&llmschema.OutputOptions{OutputDir: "llm-docs/db-schema"},
//	)
//
// # Database Connection URLs
//
// Supported URL formats:
//   - PostgreSQL: postgres://user:pass@host:port/database or postgresql://...
//   - MySQL: mysql://user:pass@tcp(host:port)/database
//   - SQLite: sqlite://path/to/database.db
//
// # Output Formats
//
// Single-file output writes all tables to one markdown file:
//
//	&OutputOptions{Writer: os.Stdout}  // or any io.Writer
//
// Multi-file output creates a directory with _overview.md and one file per table (Recommended):
//
//	&OutputOptions{OutputDir: "docs/schema"}
//
// Multi-file output is recommended for AI agents as it provides better context management.
package llmschema

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tordrt/llmschema/internal/db"
	"github.com/tordrt/llmschema/internal/formatter"
	"github.com/tordrt/llmschema/internal/schema"
)

// Options configures schema extraction behavior.
//
// All fields are optional. If not specified:
//   - Tables: nil extracts all tables in the schema
//   - ExcludeTables: empty list excludes no tables
//   - SchemaName: defaults to "public" for PostgreSQL, auto-detected from URL for MySQL,
//     not applicable for SQLite
//
// Note: If both Tables and ExcludeTables are specified, Tables takes precedence
// (only specified tables are extracted, then exclusions are applied).
type Options struct {
	// Tables specifies which tables to include in the extraction.
	// If nil or empty, all tables in the schema are extracted.
	// Example: []string{"users", "orders", "products"}
	Tables []string

	// ExcludeTables specifies tables to exclude from extraction.
	// Useful for omitting audit logs, migrations, or temporary tables.
	// Example: []string{"schema_migrations", "audit_log"}
	ExcludeTables []string

	// SchemaName specifies the database schema to extract.
	// PostgreSQL: defaults to "public" if not specified
	// MySQL: auto-detected from connection string if not specified
	// SQLite: not applicable (SQLite has no schema concept)
	SchemaName string
}

// OutputOptions configures schema output formatting.
//
// Choose between single-file and multi-file output:
//
// Single-file (Writer): All tables in one markdown document
//
//	&OutputOptions{Writer: os.Stdout}
//	&OutputOptions{Writer: fileHandle}
//
// Multi-file (OutputDir): Creates _overview.md + one file per table
//
//	&OutputOptions{OutputDir: "docs/schema"}
//
// If both are specified, OutputDir takes precedence and Writer is ignored.
// If neither is specified, defaults to single-file output to os.Stdout.
//
// Multi-file output is recommended for AI agents as it provides better
// context management and allows selective loading of specific tables.
type OutputOptions struct {
	// Writer specifies where to write single-file output.
	// Can be os.Stdout, a file handle, bytes.Buffer, or any io.Writer.
	// Ignored if OutputDir is set.
	// Defaults to os.Stdout if neither Writer nor OutputDir is specified.
	Writer io.Writer

	// OutputDir specifies the directory for multi-file output.
	// If set, creates:
	//   - _overview.md: Summary of all tables and relationships
	//   - <table_name>.md: Detailed schema for each table
	// The directory will be created if it doesn't exist.
	// Takes precedence over Writer if both are set.
	OutputDir string
}

// ExtractAndFormat extracts a database schema and formats it as markdown in one call.
// This is the recommended function for most use cases.
//
// The function connects to the database, extracts the schema, applies any
// table filters (Tables/ExcludeTables), and writes the formatted markdown
// to the specified output (single file or directory).
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - databaseURL: Database connection URL (postgres://, mysql://, or sqlite://)
//   - opts: Extraction options (can be nil for defaults)
//   - outOpts: Output options (can be nil for stdout)
//
// Returns an error if:
//   - Database connection fails
//   - Schema extraction fails
//   - Output writing fails (e.g., directory creation, file write)
//
// Example (multi-file output):
//
//	err := llmschema.ExtractAndFormat(
//		ctx,
//		"postgres://user:pass@localhost/db",
//		&llmschema.Options{
//			ExcludeTables: []string{"migrations"},
//		},
//		&llmschema.OutputOptions{
//			OutputDir: "llm-docs/db-schema",
//		},
//	)
//
// Example (single-file to stdout):
//
//	err := llmschema.ExtractAndFormat(
//		ctx,
//		"sqlite://data.db",
//		nil,  // Extract all tables
//		nil,  // Write to stdout
//	)
func ExtractAndFormat(ctx context.Context, databaseURL string, opts *Options, outOpts *OutputOptions) error {
	s, err := ExtractSchema(ctx, databaseURL, opts)
	if err != nil {
		return err
	}

	// Apply exclusions
	if opts != nil && len(opts.ExcludeTables) > 0 {
		filterExcludedTables(s, opts.ExcludeTables)
	}

	return FormatSchema(s, outOpts)
}

// ExtractSchema extracts database schema metadata from the given connection URL.
//
// Use this function when you need to inspect or modify the schema before formatting.
// For most use cases, use ExtractAndFormat instead which combines extraction and formatting.
//
// The returned schema.Schema contains all tables, columns, relationships, indexes,
// and constraints. You can inspect or modify this structure before passing it to
// FormatSchema.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - databaseURL: Database connection URL
//   - opts: Extraction options (can be nil for defaults)
//
// Supported URL schemes:
//   - postgres:// or postgresql://
//   - mysql://
//   - sqlite://
//
// Returns an error if:
//   - URL format is invalid
//   - Database connection fails
//   - Schema extraction fails (e.g., permission issues)
//
// Example (extract specific tables):
//
//	schema, err := llmschema.ExtractSchema(
//		ctx,
//		"postgres://user:pass@localhost/db",
//		&llmschema.Options{
//			Tables: []string{"users", "orders"},
//		},
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Inspect or modify schema before formatting
//	fmt.Printf("Found %d tables\n", len(schema.Tables))
func ExtractSchema(ctx context.Context, databaseURL string, opts *Options) (*schema.Schema, error) {
	if opts == nil {
		opts = &Options{}
	}

	dbType, connStr, err := parseDatabaseURL(databaseURL)
	if err != nil {
		return nil, err
	}

	switch dbType {
	case "postgres":
		return extractPostgresSchema(ctx, connStr, opts)
	case "mysql":
		return extractMySQLSchema(ctx, connStr, opts)
	case "sqlite":
		return extractSQLiteSchema(ctx, connStr, opts)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// FormatSchema formats a schema structure as markdown and writes it to the specified output.
//
// Use this function when you've already extracted a schema with ExtractSchema and
// potentially modified it. For most use cases, use ExtractAndFormat instead which
// combines extraction and formatting.
//
// The function supports two output modes:
//   - Single-file: Writes all tables to one markdown document (Writer)
//   - Multi-file: Creates a directory with _overview.md + one file per table (OutputDir)
//
// Parameters:
//   - s: The schema to format (obtained from ExtractSchema)
//   - opts: Output options (can be nil for stdout)
//
// Returns an error if:
//   - Directory creation fails (multi-file mode)
//   - File writing fails
//   - Writer errors occur
//
// Example (two-phase workflow):
//
//	// Phase 1: Extract
//	schema, err := llmschema.ExtractSchema(ctx, dbURL, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Phase 2: Modify schema if needed (e.g., remove or add info)
//	// ... custom logic ...
//
//	// Phase 3: Format
//	err = llmschema.FormatSchema(schema, &llmschema.OutputOptions{
//		OutputDir: "docs/schema",
//	})
func FormatSchema(s *schema.Schema, opts *OutputOptions) error {
	if opts == nil {
		opts = &OutputOptions{Writer: os.Stdout}
	}

	// Multi-file output
	if opts.OutputDir != "" {
		f := formatter.NewMultiFileFormatter(opts.OutputDir, "markdown")
		return f.Format(s)
	}

	// Single-file output
	writer := opts.Writer
	if writer == nil {
		writer = os.Stdout
	}
	f := formatter.NewMarkdownFormatter(writer)
	return f.Format(s)
}

// parseDatabaseURL detects database type and returns connection string
func parseDatabaseURL(url string) (dbType, connectionStr string, err error) {
	if url == "" {
		return "", "", fmt.Errorf("database URL is required")
	}

	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return "postgres", url, nil
	}

	if strings.HasPrefix(url, "mysql://") {
		// Strip mysql:// prefix for the Go MySQL driver
		connectionStr := strings.TrimPrefix(url, "mysql://")
		return "mysql", connectionStr, nil
	}

	if strings.HasPrefix(url, "sqlite://") {
		// Strip sqlite:// prefix to get file path
		filePath := strings.TrimPrefix(url, "sqlite://")
		return "sqlite", filePath, nil
	}

	return "", "", fmt.Errorf("invalid database URL scheme (must start with postgres://, mysql://, or sqlite://)")
}

func extractPostgresSchema(ctx context.Context, connectionStr string, opts *Options) (*schema.Schema, error) {
	client, err := db.NewPostgresClient(ctx, connectionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	schemaName := opts.SchemaName
	if schemaName == "" {
		schemaName = "public"
	}

	extractor := db.NewExtractor(client, schemaName)
	return extractor.ExtractSchema(ctx, opts.Tables)
}

func extractMySQLSchema(ctx context.Context, connectionStr string, opts *Options) (*schema.Schema, error) {
	client, err := db.NewMySQLClient(ctx, connectionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() { _ = client.Close() }()

	schemaName := opts.SchemaName
	if schemaName == "" {
		schemaName, err = db.ParseDatabaseName(connectionStr)
		if err != nil {
			return nil, fmt.Errorf("failed to determine database name: %w (please specify SchemaName in Options)", err)
		}
	}

	extractor := db.NewMySQLExtractor(client, schemaName)
	return extractor.ExtractSchema(ctx, opts.Tables)
}

func extractSQLiteSchema(ctx context.Context, filePath string, opts *Options) (*schema.Schema, error) {
	client, err := db.NewSQLiteClient(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	defer func() { _ = client.Close() }()

	extractor := db.NewSQLiteExtractor(client)
	return extractor.ExtractSchema(ctx, opts.Tables)
}

func filterExcludedTables(s *schema.Schema, excludeList []string) {
	if len(excludeList) == 0 {
		return
	}

	excludeSet := make(map[string]bool)
	for _, tableName := range excludeList {
		excludeSet[tableName] = true
	}

	filteredTables := make([]schema.Table, 0, len(s.Tables))
	for _, table := range s.Tables {
		if !excludeSet[table.Name] {
			filteredTables = append(filteredTables, table)
		}
	}
	s.Tables = filteredTables
}
