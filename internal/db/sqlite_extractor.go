package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tordrt/llmschema/internal/schema"
)

// SQLiteExtractor handles schema extraction from SQLite
type SQLiteExtractor struct {
	client *SQLiteClient
}

// NewSQLiteExtractor creates a new SQLite schema extractor
func NewSQLiteExtractor(client *SQLiteClient) *SQLiteExtractor {
	return &SQLiteExtractor{
		client: client,
	}
}

// ExtractSchema extracts the complete schema for specified tables
// If tables is empty, extracts all tables in the database
func (e *SQLiteExtractor) ExtractSchema(ctx context.Context, tables []string) (*schema.Schema, error) {
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
func (e *SQLiteExtractor) getTableNames(ctx context.Context, requestedTables []string) ([]string, error) {
	if len(requestedTables) > 0 {
		return requestedTables, nil
	}

	query := `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableList []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tableList = append(tableList, tableName)
	}

	return tableList, rows.Err()
}

// extractTable extracts all information for a single table
func (e *SQLiteExtractor) extractTable(ctx context.Context, tableName string) (*schema.Table, error) {
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
func (e *SQLiteExtractor) extractColumns(ctx context.Context, tableName string) ([]schema.Column, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []schema.Column
	var pkColumns []string

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		col := schema.Column{
			Name:     name,
			Type:     colType,
			Nullable: notNull == 0,
		}

		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}

		// Track primary key columns
		if pk > 0 {
			pkColumns = append(pkColumns, name)
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Check for unique constraints
	for i := range columns {
		isUnique, err := e.isColumnUnique(ctx, tableName, columns[i].Name, pkColumns)
		if err != nil {
			return nil, err
		}
		columns[i].IsUnique = isUnique
	}

	return columns, nil
}

// isColumnUnique checks if a column has a unique constraint
func (e *SQLiteExtractor) isColumnUnique(ctx context.Context, tableName, columnName string, pkColumns []string) (bool, error) {
	// Check if column is part of primary key
	for _, pk := range pkColumns {
		if pk == columnName {
			return false, nil // Primary keys are handled separately
		}
	}

	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return false, err
		}

		if unique == 1 {
			// Check if this unique index is for our column
			indexQuery := fmt.Sprintf("PRAGMA index_info(%s)", name)
			indexRows, err := e.client.GetDB().QueryContext(ctx, indexQuery)
			if err != nil {
				indexRows.Close()
				continue
			}

			var indexColumns []string
			for indexRows.Next() {
				var seqno, cid int
				var colName sql.NullString

				if err := indexRows.Scan(&seqno, &cid, &colName); err != nil {
					indexRows.Close()
					continue
				}

				if colName.Valid {
					indexColumns = append(indexColumns, colName.String)
				}
			}
			indexRows.Close()

			// If this is a single-column unique index on our column
			if len(indexColumns) == 1 && indexColumns[0] == columnName {
				return true, nil
			}
		}
	}

	return false, rows.Err()
}

// extractPrimaryKey extracts primary key columns
func (e *SQLiteExtractor) extractPrimaryKey(ctx context.Context, tableName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pk []string

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pkOrder int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pkOrder); err != nil {
			return nil, err
		}

		if pkOrder > 0 {
			pk = append(pk, name)
		}
	}

	return pk, rows.Err()
}

// extractRelations extracts foreign key relationships
func (e *SQLiteExtractor) extractRelations(ctx context.Context, tableName string) ([]schema.Relation, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName)

	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []schema.Relation

	for rows.Next() {
		var id, seq int
		var targetTable, fromCol, toCol, onUpdate, onDelete, match string

		if err := rows.Scan(&id, &seq, &targetTable, &fromCol, &toCol, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		rel := schema.Relation{
			SourceColumn: fromCol,
			TargetTable:  targetTable,
			TargetColumn: toCol,
			Cardinality:  "N:1", // Simplified assumption
		}

		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// extractIndexes extracts index information
func (e *SQLiteExtractor) extractIndexes(ctx context.Context, tableName string) ([]schema.Index, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)

	rows, err := e.client.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []schema.Index

	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Skip auto-generated primary key indexes
		if strings.HasPrefix(name, "sqlite_autoindex") {
			continue
		}

		// Get index columns
		indexQuery := fmt.Sprintf("PRAGMA index_info(%s)", name)
		indexRows, err := e.client.GetDB().QueryContext(ctx, indexQuery)
		if err != nil {
			return nil, err
		}

		var columns []string
		for indexRows.Next() {
			var seqno, cid int
			var colName sql.NullString

			if err := indexRows.Scan(&seqno, &cid, &colName); err != nil {
				indexRows.Close()
				return nil, err
			}

			if colName.Valid {
				columns = append(columns, colName.String)
			}
		}
		indexRows.Close()

		if len(columns) > 0 {
			idx := schema.Index{
				Name:     name,
				IsUnique: unique == 1,
				Columns:  columns,
			}
			indexes = append(indexes, idx)
		}
	}

	return indexes, rows.Err()
}
