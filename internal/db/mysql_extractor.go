package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tordrt/llmschema/internal/schema"
)

// MySQLExtractor handles schema extraction from MySQL
type MySQLExtractor struct {
	client     *MySQLClient
	schemaName string
}

// NewMySQLExtractor creates a new MySQL schema extractor
func NewMySQLExtractor(client *MySQLClient, schemaName string) *MySQLExtractor {
	return &MySQLExtractor{
		client:     client,
		schemaName: schemaName,
	}
}

// ExtractSchema extracts the complete schema for specified tables
// If tables is empty, extracts all tables in the schema
func (e *MySQLExtractor) ExtractSchema(ctx context.Context, tables []string) (*schema.Schema, error) {
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
func (e *MySQLExtractor) getTableNames(ctx context.Context, requestedTables []string) ([]string, error) {
	if len(requestedTables) > 0 {
		return requestedTables, nil
	}

	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query, e.schemaName)
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
func (e *MySQLExtractor) extractTable(ctx context.Context, tableName string) (*schema.Table, error) {
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
func (e *MySQLExtractor) extractColumns(ctx context.Context, tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			c.column_name,
			c.column_type,
			c.is_nullable,
			c.column_default,
			CASE WHEN EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
					ON tc.constraint_name = kcu.constraint_name
					AND tc.table_schema = kcu.table_schema
					AND tc.table_name = kcu.table_name
				WHERE tc.table_schema = ?
					AND tc.table_name = ?
					AND tc.constraint_type = 'UNIQUE'
					AND kcu.column_name = c.column_name
			) THEN true ELSE false END as is_unique,
			c.data_type
		FROM information_schema.columns c
		WHERE c.table_schema = ? AND c.table_name = ?
		ORDER BY c.ordinal_position
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query, e.schemaName, tableName, e.schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []schema.Column
	var enumColumns []int

	for rows.Next() {
		var col schema.Column
		var columnType string
		var nullable string
		var defaultVal sql.NullString
		var isUnique bool
		var dataType string

		if err := rows.Scan(&col.Name, &columnType, &nullable, &defaultVal, &isUnique, &dataType); err != nil {
			return nil, err
		}

		col.Type = columnType
		col.Nullable = (nullable == "YES")
		col.IsUnique = isUnique
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		// Check if this is an ENUM column
		if dataType == "enum" {
			enumColumns = append(enumColumns, len(columns))
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Extract enum values for enum columns
	for _, idx := range enumColumns {
		enumValues, err := e.extractEnumValues(columns[idx].Type)
		if err != nil {
			return nil, err
		}
		columns[idx].EnumValues = enumValues
	}

	return columns, nil
}

// extractEnumValues parses enum values from the column type string
// MySQL stores enum types as "enum('value1','value2','value3')"
func (e *MySQLExtractor) extractEnumValues(columnType string) ([]string, error) {
	if !strings.HasPrefix(columnType, "enum(") {
		return nil, nil
	}

	// Extract the part between enum( and )
	start := strings.Index(columnType, "(")
	end := strings.LastIndex(columnType, ")")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("invalid enum type format: %s", columnType)
	}

	enumList := columnType[start+1 : end]

	// Split by comma and clean up quotes
	var values []string
	parts := strings.Split(enumList, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove surrounding quotes
		if len(part) >= 2 && part[0] == '\'' && part[len(part)-1] == '\'' {
			part = part[1 : len(part)-1]
		}
		values = append(values, part)
	}

	return values, nil
}

// extractPrimaryKey extracts primary key columns
func (e *MySQLExtractor) extractPrimaryKey(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
			AND table_name = ?
			AND constraint_name = 'PRIMARY'
		ORDER BY ordinal_position
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query, e.schemaName, tableName)
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
func (e *MySQLExtractor) extractRelations(ctx context.Context, tableName string) ([]schema.Relation, error) {
	query := `
		SELECT
			kcu.column_name,
			kcu.referenced_table_name,
			kcu.referenced_column_name
		FROM information_schema.key_column_usage kcu
		WHERE kcu.table_schema = ?
			AND kcu.table_name = ?
			AND kcu.referenced_table_name IS NOT NULL
		ORDER BY kcu.ordinal_position
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query, e.schemaName, tableName)
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

		// Determine cardinality (simplified: assume N:1 for now)
		rel.Cardinality = "N:1"

		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// extractIndexes extracts index information
func (e *MySQLExtractor) extractIndexes(ctx context.Context, tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			s.index_name,
			s.non_unique = 0 AS is_unique,
			GROUP_CONCAT(s.column_name ORDER BY s.seq_in_index) AS column_names
		FROM information_schema.statistics s
		WHERE s.table_schema = ?
			AND s.table_name = ?
			AND s.index_name != 'PRIMARY'
		GROUP BY s.index_name, s.non_unique
		ORDER BY s.index_name
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query, e.schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var idx schema.Index
		var isUnique int
		var columnNames string

		if err := rows.Scan(&idx.Name, &isUnique, &columnNames); err != nil {
			return nil, err
		}

		idx.IsUnique = (isUnique == 1)
		idx.Columns = strings.Split(columnNames, ",")

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}
