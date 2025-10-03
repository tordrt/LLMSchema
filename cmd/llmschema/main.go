package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tordrt/llmschema/internal/db"
	"github.com/tordrt/llmschema/internal/formatter"
	"github.com/tordrt/llmschema/internal/schema"
)

var (
	dbURL         string
	outputFile    string
	outputDir     string
	tables        string
	excludeTables string
	schemaName    string
)

var rootCmd = &cobra.Command{
	Use:   "llmschema",
	Short: "Extract database schema in LLM-friendly format",
	Long:  `LLMSchema extracts database schemas from PostgreSQL, MySQL, or SQLite and outputs them in a compact, token-efficient format optimized for LLMs.`,
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&dbURL, "db-url", "", "Database connection string (postgres://, mysql://, or sqlite://)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", "Output directory for multi-file output (recommended)")
	rootCmd.Flags().StringVarP(&tables, "tables", "t", "", "Specific tables (comma-separated, optional)")
	rootCmd.Flags().StringVarP(&excludeTables, "exclude-tables", "e", "", "Tables to exclude (comma-separated, optional)")
	rootCmd.Flags().StringVarP(&schemaName, "schema", "s", "", "Database schema name (optional: defaults to 'public' for PostgreSQL, auto-detected from connection string for MySQL)")
}

type dbConfig struct {
	dbType        string // "postgres", "mysql", or "sqlite"
	connectionStr string // processed connection string for the specific driver
}

func parseDatabaseURL(url string) (*dbConfig, error) {
	if url == "" {
		return nil, fmt.Errorf("--db-url is required")
	}

	// Detect database type from scheme
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return &dbConfig{
			dbType:        "postgres",
			connectionStr: url,
		}, nil
	}

	if strings.HasPrefix(url, "mysql://") {
		// Strip mysql:// prefix for the Go MySQL driver
		connectionStr := strings.TrimPrefix(url, "mysql://")
		return &dbConfig{
			dbType:        "mysql",
			connectionStr: connectionStr,
		}, nil
	}

	if strings.HasPrefix(url, "sqlite://") {
		// Strip sqlite:// prefix to get file path
		filePath := strings.TrimPrefix(url, "sqlite://")
		return &dbConfig{
			dbType:        "sqlite",
			connectionStr: filePath,
		}, nil
	}

	return nil, fmt.Errorf("invalid database URL scheme (must start with postgres://, mysql://, or sqlite://)")
}

func parseTableList(tablesStr string) []string {
	if tablesStr == "" {
		return nil
	}
	tableList := strings.Split(tablesStr, ",")
	for i, t := range tableList {
		tableList[i] = strings.TrimSpace(t)
	}
	return tableList
}

func filterExcludedTables(s *schema.Schema, excludeList []string) {
	if len(excludeList) == 0 {
		return
	}

	// Create a set of excluded table names for fast lookup
	excludeSet := make(map[string]bool)
	for _, tableName := range excludeList {
		excludeSet[tableName] = true
	}

	// Filter out excluded tables
	filteredTables := make([]schema.Table, 0, len(s.Tables))
	for _, table := range s.Tables {
		if !excludeSet[table.Name] {
			filteredTables = append(filteredTables, table)
		}
	}
	s.Tables = filteredTables
}

func extractSchema(ctx context.Context, config *dbConfig, tableList []string) (*schema.Schema, error) {
	switch config.dbType {
	case "sqlite":
		return extractSQLiteSchema(ctx, config.connectionStr, tableList)
	case "mysql":
		return extractMySQLSchema(ctx, config.connectionStr, tableList)
	case "postgres":
		return extractPostgresSchema(ctx, config.connectionStr, tableList)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.dbType)
	}
}

func extractSQLiteSchema(ctx context.Context, filePath string, tableList []string) (*schema.Schema, error) {
	client, err := db.NewSQLiteClient(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close SQLite connection: %v\n", err)
		}
	}()

	extractor := db.NewSQLiteExtractor(client)
	extractedSchema, err := extractor.ExtractSchema(ctx, tableList)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w", err)
	}
	return extractedSchema, nil
}

func extractMySQLSchema(ctx context.Context, connectionStr string, tableList []string) (*schema.Schema, error) {
	client, err := db.NewMySQLClient(ctx, connectionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close MySQL connection: %v\n", err)
		}
	}()

	// Auto-detect database name from connection string if schema not specified
	mysqlSchema := schemaName
	if mysqlSchema == "" {
		mysqlSchema, err = db.ParseDatabaseName(connectionStr)
		if err != nil {
			return nil, fmt.Errorf("failed to determine database name: %w (please specify --schema)", err)
		}
	}

	extractor := db.NewMySQLExtractor(client, mysqlSchema)
	extractedSchema, err := extractor.ExtractSchema(ctx, tableList)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w", err)
	}
	return extractedSchema, nil
}

func extractPostgresSchema(ctx context.Context, connectionStr string, tableList []string) (*schema.Schema, error) {
	client, err := db.NewPostgresClient(ctx, connectionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer func() {
		if err := client.Close(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close PostgreSQL connection: %v\n", err)
		}
	}()

	// Default to "public" schema if not specified
	pgSchema := schemaName
	if pgSchema == "" {
		pgSchema = "public"
	}

	extractor := db.NewExtractor(client, pgSchema)
	extractedSchema, err := extractor.ExtractSchema(ctx, tableList)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w", err)
	}
	return extractedSchema, nil
}

func formatOutput(extractedSchema *schema.Schema) error {
	// Validate flag combinations
	if outputDir != "" && outputFile != "" {
		return fmt.Errorf("cannot use both --output-dir and --output flags")
	}

	// Multi-file output
	if outputDir != "" {
		multiFormatter := formatter.NewMultiFileFormatter(outputDir, "markdown")
		if err := multiFormatter.Format(extractedSchema); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
		return nil
	}

	// Single-file output
	var writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close output file: %v\n", err)
			}
		}()
		writer = f
	}

	// Format and write output
	markdownFormatter := formatter.NewMarkdownFormatter(writer)
	return markdownFormatter.Format(extractedSchema)
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse and validate database URL
	config, err := parseDatabaseURL(dbURL)
	if err != nil {
		return err
	}

	// Parse table list
	tableList := parseTableList(tables)

	// Extract schema based on database type
	extractedSchema, err := extractSchema(ctx, config, tableList)
	if err != nil {
		return err
	}

	// Apply table exclusions
	excludeList := parseTableList(excludeTables)
	filterExcludedTables(extractedSchema, excludeList)

	// Format and output the schema
	return formatOutput(extractedSchema)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
