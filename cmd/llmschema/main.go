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
	dbURL          string
	mysqlURL       string
	sqlitePath     string
	outputFile     string
	outputDir      string
	tables         string
	schemaName     string
	format         string
	splitThreshold int
)

var rootCmd = &cobra.Command{
	Use:   "llmschema",
	Short: "Extract database schema in LLM-friendly format",
	Long:  `LLMSchema extracts database schemas from PostgreSQL, MySQL, or SQLite and outputs them in a compact, token-efficient format optimized for LLMs.`,
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&dbURL, "db-url", "", "PostgreSQL connection string")
	rootCmd.Flags().StringVar(&mysqlURL, "mysql-url", "", "MySQL connection string")
	rootCmd.Flags().StringVar(&sqlitePath, "sqlite", "", "SQLite database file path")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", "Output directory for multi-file output")
	rootCmd.Flags().StringVarP(&tables, "tables", "t", "", "Specific tables (comma-separated, optional)")
	rootCmd.Flags().StringVarP(&schemaName, "schema", "s", "public", "Database schema name (default: public for PostgreSQL)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or markdown (default: text)")
	rootCmd.Flags().IntVar(&splitThreshold, "split-threshold", 0, "Split into multiple files when table count exceeds this (requires --output-dir)")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate database flags
	dbCount := 0
	if dbURL != "" {
		dbCount++
	}
	if mysqlURL != "" {
		dbCount++
	}
	if sqlitePath != "" {
		dbCount++
	}
	if dbCount == 0 {
		return fmt.Errorf("one of --db-url, --mysql-url, or --sqlite must be specified")
	}
	if dbCount > 1 {
		return fmt.Errorf("only one of --db-url, --mysql-url, or --sqlite can be specified")
	}

	// Parse table list
	var tableList []string
	if tables != "" {
		tableList = strings.Split(tables, ",")
		for i, t := range tableList {
			tableList[i] = strings.TrimSpace(t)
		}
	}

	// Extract schema based on database type
	var extractedSchema *schema.Schema

	if sqlitePath != "" {
		// SQLite mode
		client, err := db.NewSQLiteClient(ctx, sqlitePath)
		if err != nil {
			return fmt.Errorf("failed to connect to SQLite: %w", err)
		}
		defer func() {
			if err := client.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close SQLite connection: %v\n", err)
			}
		}()

		extractor := db.NewSQLiteExtractor(client)
		extractedSchema, err = extractor.ExtractSchema(ctx, tableList)
		if err != nil {
			return fmt.Errorf("failed to extract schema: %w", err)
		}
	} else if mysqlURL != "" {
		// MySQL mode
		client, err := db.NewMySQLClient(ctx, mysqlURL)
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
		defer func() {
			if err := client.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close MySQL connection: %v\n", err)
			}
		}()

		extractor := db.NewMySQLExtractor(client, schemaName)
		extractedSchema, err = extractor.ExtractSchema(ctx, tableList)
		if err != nil {
			return fmt.Errorf("failed to extract schema: %w", err)
		}
	} else {
		// PostgreSQL mode
		client, err := db.NewPostgresClient(ctx, dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		defer func() {
			if err := client.Close(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close PostgreSQL connection: %v\n", err)
			}
		}()

		extractor := db.NewExtractor(client, schemaName)
		extractedSchema, err = extractor.ExtractSchema(ctx, tableList)
		if err != nil {
			return fmt.Errorf("failed to extract schema: %w", err)
		}
	}

	// Check if we should use multi-file output
	shouldSplit := outputDir != "" && (splitThreshold == 0 || len(extractedSchema.Tables) > splitThreshold)

	// Validate flag combinations
	if outputDir != "" && outputFile != "" {
		return fmt.Errorf("cannot use both --output-dir and --output flags")
	}

	// Multi-file output
	if shouldSplit {
		multiFormatter := formatter.NewMultiFileFormatter(outputDir, format)
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
	var err2 error
	switch format {
	case "text":
		textFormatter := formatter.NewTextFormatter(writer)
		err2 = textFormatter.Format(extractedSchema)
	case "markdown":
		markdownFormatter := formatter.NewMarkdownFormatter(writer)
		err2 = markdownFormatter.Format(extractedSchema)
	default:
		return fmt.Errorf("invalid format: %s (must be 'text' or 'markdown')", format)
	}

	if err2 != nil {
		return fmt.Errorf("failed to format output: %w", err2)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
