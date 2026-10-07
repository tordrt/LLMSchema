package db

import "testing"

func TestNormalizePostgresNumericType(t *testing.T) {
	precision, scale := 10, 2
	if got := normalizePostgresType("numeric", "numeric", nil, &precision, &scale); got != "numeric(10,2)" {
		t.Errorf("normalizePostgresType(numeric(10,2)) = %q", got)
	}
	if got := normalizePostgresType("numeric", "numeric", nil, nil, nil); got != "numeric" {
		t.Errorf("normalizePostgresType(numeric) = %q", got)
	}
}
