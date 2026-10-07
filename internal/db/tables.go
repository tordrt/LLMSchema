package db

import (
	"fmt"
	"slices"
	"strings"
)

// selectRequestedTables returns requestedTables in the order given, or all
// tables when none were requested. Unknown table names are reported as an error.
func selectRequestedTables(allTables, requestedTables []string) ([]string, error) {
	if len(requestedTables) == 0 {
		return allTables, nil
	}

	var missing []string
	for _, table := range requestedTables {
		if !slices.Contains(allTables, table) {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("table(s) not found: %s", strings.Join(missing, ", "))
	}

	return requestedTables, nil
}
