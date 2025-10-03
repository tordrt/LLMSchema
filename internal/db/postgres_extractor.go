package db

import (
	"context"
	"fmt"

	"github.com/tordrt/llmschema/internal/schema"
)

const varcharType = "varchar"

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

// normalizePostgresType maps verbose SQL type names to commonly-used PostgreSQL equivalents
func normalizePostgresType(dataType, udtName string, charMaxLength *int) string {
	switch dataType {
	case "timestamp with time zone":
		return "timestamptz"
	case "timestamp without time zone":
		return "timestamp"
	case "time with time zone":
		return "timetz"
	case "time without time zone":
		return "time"
	case "character varying":
		if charMaxLength != nil {
			return fmt.Sprintf("varchar(%d)", *charMaxLength)
		}
		return varcharType
	case "character":
		if charMaxLength != nil {
			return fmt.Sprintf("char(%d)", *charMaxLength)
		}
		return "char"
	case "ARRAY":
		// udt_name has underscore prefix for arrays (e.g., "_text" for text[], "_int4" for integer[])
		if len(udtName) > 0 && udtName[0] == '_' {
			elementType := normalizeUdtName(udtName[1:])
			return fmt.Sprintf("%s[]", elementType)
		}
		return "array"
	case "USER-DEFINED":
		return udtName
	default:
		return dataType
	}
}

// normalizeUdtName converts PostgreSQL internal type names to more readable forms
func normalizeUdtName(udtName string) string {
	switch udtName {
	case "int4":
		return "integer"
	case "int8":
		return "bigint"
	case "int2":
		return "smallint"
	case "float4":
		return "real"
	case "float8":
		return "double precision"
	case "bool":
		return "boolean"
	case varcharType:
		return varcharType
	default:
		return udtName
	}
}

// extractColumns extracts column information for a table
func (e *Extractor) extractColumns(ctx context.Context, tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			CASE WHEN EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.constraint_column_usage ccu
					ON tc.constraint_name = ccu.constraint_name
					AND tc.table_schema = ccu.table_schema
				WHERE tc.table_schema = $1
					AND tc.table_name = $2
					AND tc.constraint_type = 'UNIQUE'
					AND ccu.column_name = c.column_name
			) THEN true ELSE false END as is_unique,
			c.udt_name,
			c.character_maximum_length
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
	var enumTypes []string

	// First pass: collect all columns and enum type names
	for rows.Next() {
		var col schema.Column
		var nullable string
		var defaultVal *string
		var dataType string
		var udtName string
		var charMaxLength *int

		if err := rows.Scan(&col.Name, &dataType, &nullable, &defaultVal, &col.IsUnique, &udtName, &charMaxLength); err != nil {
			return nil, err
		}

		col.Nullable = (nullable == "YES")
		col.DefaultValue = defaultVal

		// Use SQL standard type names, but apply PostgreSQL-specific shortcuts for verbose types
		col.Type = normalizePostgresType(dataType, udtName, charMaxLength)

		// If it's a USER-DEFINED type, remember it for later lookup of enum values
		if dataType == "USER-DEFINED" {
			enumTypes = append(enumTypes, udtName)
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Second pass: fetch enum values for all USER-DEFINED types
	if len(enumTypes) > 0 {
		enumValuesMap, err := e.extractEnumValuesMap(ctx, enumTypes)
		if err != nil {
			return nil, err
		}

		// Update columns with enum values
		for i := range columns {
			if values, ok := enumValuesMap[columns[i].Type]; ok {
				columns[i].EnumValues = values
			}
		}
	}

	return columns, nil
}

// extractEnumValuesMap extracts enum values for multiple enum types at once
func (e *Extractor) extractEnumValuesMap(ctx context.Context, enumTypeNames []string) (map[string][]string, error) {
	if len(enumTypeNames) == 0 {
		return make(map[string][]string), nil
	}

	query := `
		SELECT t.typname, e.enumlabel
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = $1 AND t.typname = ANY($2)
		ORDER BY t.typname, e.enumsortorder
	`

	rows, err := e.client.GetConnection().Query(ctx, query, e.schema, enumTypeNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var typName, enumLabel string
		if err := rows.Scan(&typName, &enumLabel); err != nil {
			return nil, err
		}
		result[typName] = append(result[typName], enumLabel)
	}

	return result, rows.Err()
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
