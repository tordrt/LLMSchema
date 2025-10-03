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

// Options configures schema extraction
type Options struct {
	// Tables to include (nil = all tables)
	Tables []string
	// Tables to exclude
	ExcludeTables []string
	// Schema name (optional: defaults to "public" for PostgreSQL, auto-detected for MySQL)
	SchemaName string
}

// OutputOptions configures schema output formatting
type OutputOptions struct {
	// Writer for single-file output (ignored if OutputDir is set)
	Writer io.Writer
	// OutputDir for multi-file output (if set, creates one file per table)
	OutputDir string
}

// ExtractAndFormat extracts and formats a database schema in one call.
// This is the recommended function for most use cases.
//
// Example:
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

// ExtractSchema extracts database schema from the given connection URL.
// For most use cases, use ExtractAndFormat instead.
// Supported URL schemes: postgres://, postgresql://, mysql://, sqlite://
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

// FormatSchema formats a schema as markdown and writes it to the specified output.
// For most use cases, use ExtractAndFormat instead.
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
