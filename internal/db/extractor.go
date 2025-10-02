package db

import (
	"context"
	"fmt"

	"github.com/tordrt/llmschema/internal/schema"
)

// Extractor handles schema extraction from PostgreSQL
type Extractor struct {
	client *PostgresClient
	schema string
}

// NewExtractor creates a new schema extractor
func NewExtractor(client *PostgresClient, schemaName string) *Extractor {
	return &Extractor{
		client: client,
		schema: schemaName,
	}
}

// ExtractSchema extracts the complete schema for specified tables
// If tables is empty, extracts all tables in the schema
func (e *Extractor) ExtractSchema(ctx context.Context, tables []string) (*schema.Schema, error) {
	var extractedTables []schema.Table

	tableNames, err := e.getTableNames(ctx, tables)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}

	for _, tableName := range tableNames {
		table, err := e.extractTable(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract table %s: %w", tableName, err)
		}
		extractedTables = append(extractedTables, *table)
	}

	return &schema.Schema{Tables: extractedTables}, nil
}

// getTableNames returns the list of tables to extract
func (e *Extractor) getTableNames(ctx context.Context, requestedTables []string) ([]string, error) {
	if len(requestedTables) > 0 {
		return requestedTables, nil
	}

	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// extractTable extracts all information for a single table
func (e *Extractor) extractTable(ctx context.Context, tableName string) (*schema.Table, error) {
	table := &schema.Table{Name: tableName}

	// Extract columns
	columns, err := e.extractColumns(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract columns: %w", err)
	}
	table.Columns = columns

	// Extract primary key
	pk, err := e.extractPrimaryKey(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract primary key: %w", err)
	}
	table.PrimaryKey = pk

	// Extract relations
	relations, err := e.extractRelations(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract relations: %w", err)
	}
	table.Relations = relations

	// Extract indexes
	indexes, err := e.extractIndexes(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract indexes: %w", err)
	}
	table.Indexes = indexes

	return table, nil
}

// extractColumns extracts column information for a table
func (e *Extractor) extractColumns(ctx context.Context, tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			CASE WHEN EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.constraint_column_usage ccu
					ON tc.constraint_name = ccu.constraint_name
					AND tc.table_schema = ccu.table_schema
				WHERE tc.table_schema = $1
					AND tc.table_name = $2
					AND tc.constraint_type = 'UNIQUE'
					AND ccu.column_name = c.column_name
			) THEN true ELSE false END as is_unique
		FROM information_schema.columns c
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var col schema.Column
		var nullable string
		var defaultVal *string

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultVal, &col.IsUnique); err != nil {
			return nil, err
		}

		col.Nullable = (nullable == "YES")
		col.DefaultValue = defaultVal

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// extractPrimaryKey extracts primary key columns
func (e *Extractor) extractPrimaryKey(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = $1
			AND table_name = $2
			AND constraint_name IN (
				SELECT constraint_name
				FROM information_schema.table_constraints
				WHERE table_schema = $1
					AND table_name = $2
					AND constraint_type = 'PRIMARY KEY'
			)
		ORDER BY ordinal_position
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pk []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			return nil, err
		}
		pk = append(pk, colName)
	}

	return pk, rows.Err()
}

// extractRelations extracts foreign key relationships
func (e *Extractor) extractRelations(ctx context.Context, tableName string) ([]schema.Relation, error) {
	query := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []schema.Relation
	for rows.Next() {
		var rel schema.Relation
		if err := rows.Scan(&rel.SourceColumn, &rel.TargetTable, &rel.TargetColumn); err != nil {
			return nil, err
		}

		// Determine cardinality (simplified: assume 1:N for now, would need more logic for 1:1)
		rel.Cardinality = "N:1"

		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// extractIndexes extracts index information
func (e *Extractor) extractIndexes(ctx context.Context, tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS column_names
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE t.relkind = 'r'
			AND n.nspname = $1
			AND t.relname = $2
			AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique
		ORDER BY i.relname
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var idx schema.Index
		if err := rows.Scan(&idx.Name, &idx.IsUnique, &idx.Columns); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}
