package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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
	var databaseVersion string
	// Version metadata is optional: compatible drivers may not support this
	// query even when schema extraction itself works.
	_ = e.client.GetDB().QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&databaseVersion)

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

	return &schema.Schema{
		DatabaseType:    "SQLite",
		DatabaseVersion: databaseVersion,
		Tables:          extractedTables,
	}, nil
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
	defer func() { _ = rows.Close() }()

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

	// Extract indexes
	indexes, err := e.extractIndexes(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract indexes: %w", err)
	}
	table.Indexes = indexes
	markSingleColumnUnique(table.Columns, indexes)

	// Extract relations after keys and indexes so cardinality can be inferred.
	relations, err := e.extractRelations(ctx, tableName, pk, indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract relations: %w", err)
	}
	table.Relations = relations

	return table, nil
}

// extractColumns extracts column information for a table
func (e *SQLiteExtractor) extractColumns(ctx context.Context, tableName string) ([]schema.Column, error) {
	rows, err := e.client.GetDB().QueryContext(ctx, "SELECT * FROM pragma_table_info(?)", tableName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []schema.Column
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

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Extract CHECK constraints
	checkConstraints, err := e.extractCheckConstraints(ctx, tableName)
	if err != nil {
		return nil, err
	}

	// Apply CHECK constraints to columns
	for i := range columns {
		if check, ok := checkConstraints[columns[i].Name]; ok {
			columns[i].CheckConstraint = &check
		}
	}

	return columns, nil
}

// extractPrimaryKey extracts primary key columns
func (e *SQLiteExtractor) extractPrimaryKey(ctx context.Context, tableName string) ([]string, error) {
	rows, err := e.client.GetDB().QueryContext(ctx, "SELECT * FROM pragma_table_info(?)", tableName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	type primaryKeyColumn struct {
		name  string
		order int
	}
	var keyColumns []primaryKeyColumn

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pkOrder int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pkOrder); err != nil {
			return nil, err
		}

		if pkOrder > 0 {
			keyColumns = append(keyColumns, primaryKeyColumn{name: name, order: pkOrder})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(keyColumns, func(i, j int) bool { return keyColumns[i].order < keyColumns[j].order })
	pk := make([]string, len(keyColumns))
	for i, column := range keyColumns {
		pk[i] = column.name
	}
	return pk, nil
}

// extractRelations extracts foreign key relationships
func (e *SQLiteExtractor) extractRelations(ctx context.Context, tableName string, primaryKey []string, indexes []schema.Index) ([]schema.Relation, error) {
	rows, err := e.client.GetDB().QueryContext(ctx, "SELECT * FROM pragma_foreign_key_list(?)", tableName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var relations []schema.Relation
	relationByID := make(map[int]int)

	for rows.Next() {
		var id, seq int
		var targetTable, fromCol, onUpdate, onDelete, match string
		var toCol sql.NullString

		if err := rows.Scan(&id, &seq, &targetTable, &fromCol, &toCol, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		index, ok := relationByID[id]
		if !ok {
			index = len(relations)
			relationByID[id] = index
			relations = append(relations, schema.Relation{
				TargetTable: targetTable,
				OnUpdate:    onUpdate,
				OnDelete:    onDelete,
			})
		}
		relations[index].SourceColumns = append(relations[index].SourceColumns, fromCol)
		relations[index].TargetColumns = append(relations[index].TargetColumns, toCol.String)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range relations {
		if hasEmptyColumn(relations[i].TargetColumns) {
			targetPrimaryKey, err := e.extractPrimaryKey(ctx, relations[i].TargetTable)
			if err != nil {
				return nil, err
			}
			if len(targetPrimaryKey) == len(relations[i].TargetColumns) {
				relations[i].TargetColumns = targetPrimaryKey
			}
		}
		finalizeRelation(&relations[i], primaryKey, indexes)
	}

	return relations, nil
}

func hasEmptyColumn(columns []string) bool {
	for _, column := range columns {
		if column == "" {
			return true
		}
	}
	return false
}

// extractIndexes extracts index information
func (e *SQLiteExtractor) extractIndexes(ctx context.Context, tableName string) ([]schema.Index, error) {
	rows, err := e.client.GetDB().QueryContext(ctx, "SELECT * FROM pragma_index_list(?)", tableName)
	if err != nil {
		return nil, err
	}
	type indexMetadata struct {
		name      string
		isUnique  bool
		isPartial bool
	}
	var metadata []indexMetadata

	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Primary keys are extracted separately. Keep auto-generated unique
		// indexes because they carry UNIQUE constraint semantics.
		if origin == "pk" {
			continue
		}
		metadata = append(metadata, indexMetadata{
			name:      name,
			isUnique:  unique == 1,
			isPartial: partial == 1,
		})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	var indexes []schema.Index
	for _, item := range metadata {
		// Get index columns
		indexRows, err := e.client.GetDB().QueryContext(ctx, "SELECT * FROM pragma_index_info(?)", item.name)
		if err != nil {
			return nil, err
		}

		var columns []string
		hasExpressions := false
		for indexRows.Next() {
			var seqno, cid int
			var colName sql.NullString

			if err := indexRows.Scan(&seqno, &cid, &colName); err != nil {
				_ = indexRows.Close()
				return nil, err
			}

			if colName.Valid {
				columns = append(columns, colName.String)
			} else {
				hasExpressions = true
			}
		}
		_ = indexRows.Close()

		if len(columns) > 0 || hasExpressions {
			idx := schema.Index{
				Name:           item.name,
				IsUnique:       item.isUnique,
				IsPartial:      item.isPartial,
				HasExpressions: hasExpressions,
				Columns:        columns,
			}
			indexes = append(indexes, idx)
		}
	}

	return indexes, nil
}

// extractCheckConstraints extracts CHECK constraints from the table definition
func (e *SQLiteExtractor) extractCheckConstraints(ctx context.Context, tableName string) (map[string]string, error) {
	sqlText, err := e.getTableSQL(ctx, tableName)
	if err != nil {
		return nil, err
	}

	checkConstraints := make(map[string]string)
	lines := strings.Split(sqlText, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if e.shouldSkipLine(line) {
			continue
		}

		if constraint := e.parseCheckConstraintFromLine(line); constraint != nil {
			checkConstraints[constraint.columnName] = constraint.expression
		}
	}

	return checkConstraints, nil
}

// getTableSQL retrieves the CREATE TABLE SQL for a given table
func (e *SQLiteExtractor) getTableSQL(ctx context.Context, tableName string) (string, error) {
	query := `
		SELECT sql
		FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`

	var sqlText string
	err := e.client.GetDB().QueryRowContext(ctx, query, tableName).Scan(&sqlText)
	return sqlText, err
}

// shouldSkipLine determines if a line should be skipped during CHECK constraint parsing
func (e *SQLiteExtractor) shouldSkipLine(line string) bool {
	if line == "" {
		return true
	}

	skipPrefixes := []string{"CREATE TABLE", "FOREIGN KEY", "PRIMARY KEY", "UNIQUE"}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// checkConstraint represents a parsed CHECK constraint
type checkConstraint struct {
	columnName string
	expression string
}

// parseCheckConstraintFromLine parses a CHECK constraint from a single line
func (e *SQLiteExtractor) parseCheckConstraintFromLine(line string) *checkConstraint {
	checkIdx := strings.Index(line, "CHECK(")
	if checkIdx == -1 {
		return nil
	}

	columnName := e.extractColumnName(line)
	if columnName == "" {
		return nil
	}

	expression := e.extractCheckExpression(line, checkIdx)
	if expression == "" {
		return nil
	}

	return &checkConstraint{
		columnName: columnName,
		expression: expression,
	}
}

// extractColumnName extracts the column name from a line (first word)
func (e *SQLiteExtractor) extractColumnName(line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// extractCheckExpression extracts the CHECK expression from a line
func (e *SQLiteExtractor) extractCheckExpression(line string, checkIdx int) string {
	checkStart := checkIdx + 6 // Length of "CHECK("
	depth := 1
	checkEnd := checkStart

	for i := checkStart; i < len(line) && depth > 0; i++ {
		if line[i] == '(' {
			depth++
		} else if line[i] == ')' {
			depth--
			if depth == 0 {
				checkEnd = i
				break
			}
		}
	}

	if depth != 0 {
		return ""
	}

	return line[checkStart:checkEnd]
}
