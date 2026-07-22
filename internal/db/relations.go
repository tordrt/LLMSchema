package db

import "github.com/tordrt/llmschema/internal/schema"

func relationCardinality(sourceColumns, primaryKey []string, indexes []schema.Index) string {
	if columnsContainKey(sourceColumns, primaryKey) {
		return "1:1"
	}

	for _, index := range indexes {
		if index.IsUnique && !index.IsPartial && !index.HasExpressions && columnsContainKey(sourceColumns, index.Columns) {
			return "1:1"
		}
	}

	return "N:1"
}

func finalizeRelation(relation *schema.Relation, primaryKey []string, indexes []schema.Index) {
	relation.Cardinality = relationCardinality(relation.SourceColumns, primaryKey, indexes)
	if len(relation.SourceColumns) == 1 {
		relation.SourceColumn = relation.SourceColumns[0]
	}
	if len(relation.TargetColumns) == 1 {
		relation.TargetColumn = relation.TargetColumns[0]
	}
}

// columnsContainKey reports whether values in columns are guaranteed unique by
// key. A foreign key can contain extra columns and still be unique when a subset
// of its columns forms a primary or unique key.
func columnsContainKey(columns, key []string) bool {
	if len(key) == 0 || len(key) > len(columns) {
		return false
	}

	present := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		present[column] = struct{}{}
	}
	for _, column := range key {
		if _, ok := present[column]; !ok {
			return false
		}
	}
	return true
}

func markSingleColumnUnique(columns []schema.Column, indexes []schema.Index) {
	uniqueColumns := make(map[string]struct{})
	for _, index := range indexes {
		if index.IsUnique && !index.IsPartial && !index.HasExpressions && len(index.Columns) == 1 {
			uniqueColumns[index.Columns[0]] = struct{}{}
		}
	}
	for i := range columns {
		_, columns[i].IsUnique = uniqueColumns[columns[i].Name]
	}
}
