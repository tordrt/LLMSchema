package db

import (
	"fmt"
	"slices"
	"strings"
)

// selectRequestedTables returns the requested tables in the order given, or all
// tables when none were requested. Unknown table names are reported as an error.
func selectRequestedTables(allTables, requestedTables []string) ([]string, error) {
	if len(requestedTables) == 0 {
		return allTables, nil
	}

	var selected, missing []string
	for _, requested := range requestedTables {
		table, ok := matchTableName(allTables, requested)
		if !ok {
			missing = append(missing, requested)
			continue
		}
		if !slices.Contains(selected, table) {
			selected = append(selected, table)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("table(s) not found: %s", strings.Join(missing, ", "))
	}

	return selected, nil
}

// matchTableName finds the table named requested, falling back to a
// case-insensitive match when exactly one table matches that way.
func matchTableName(allTables []string, requested string) (string, bool) {
	if slices.Contains(allTables, requested) {
		return requested, true
	}

	match, matches := "", 0
	for _, table := range allTables {
		if strings.EqualFold(table, requested) {
			match = table
			matches++
		}
	}
	return match, matches == 1
}
