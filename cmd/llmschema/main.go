package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tordrt/llmschema/internal/db"
	"github.com/tordrt/llmschema/internal/formatter"
)

var (
	dbURL      string
	outputFile string
	tables     string
	schemaName string
	format     string
)

var rootCmd = &cobra.Command{
	Use:   "llmschema",
	Short: "Extract database schema in LLM-friendly format",
	Long:  `LLMSchema extracts PostgreSQL database schemas and outputs them in a compact, token-efficient format optimized for LLMs.`,
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&dbURL, "db-url", "", "PostgreSQL connection string (required)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().StringVarP(&tables, "tables", "t", "", "Specific tables (comma-separated, optional)")
	rootCmd.Flags().StringVarP(&schemaName, "schema", "s", "public", "Schema name (default: public)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or markdown (default: text)")

	rootCmd.MarkFlagRequired("db-url")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Connect to database
	client, err := db.NewPostgresClient(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close(ctx)

	// Create extractor
	extractor := db.NewExtractor(client, schemaName)

	// Parse table list
	var tableList []string
	if tables != "" {
		tableList = strings.Split(tables, ",")
		for i, t := range tableList {
			tableList[i] = strings.TrimSpace(t)
		}
	}

	// Extract schema
	schema, err := extractor.ExtractSchema(ctx, tableList)
	if err != nil {
		return fmt.Errorf("failed to extract schema: %w", err)
	}

	// Determine output writer
	var writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	// Format and write output
	var err2 error
	switch format {
	case "text":
		textFormatter := formatter.NewTextFormatter(writer)
		err2 = textFormatter.Format(schema)
	case "markdown":
		markdownFormatter := formatter.NewMarkdownFormatter(writer)
		err2 = markdownFormatter.Format(schema)
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
