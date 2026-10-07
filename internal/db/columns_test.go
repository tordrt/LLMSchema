package db

import "testing"

func TestGeneratedAs(t *testing.T) {
	tests := map[string]string{
		"(quantity * 2)":      "GENERATED ALWAYS AS (quantity * 2)",
		"lower(name)":         "GENERATED ALWAYS AS (lower(name))",
		"(a + 1) * (b + 1)":   "GENERATED ALWAYS AS ((a + 1) * (b + 1))",
		"concat(`a`,'!')":     "GENERATED ALWAYS AS (concat(`a`,'!'))",
		"((nested) + (more))": "GENERATED ALWAYS AS ((nested) + (more))",
	}
	for expression, want := range tests {
		if got := generatedAs(expression); got != want {
			t.Errorf("generatedAs(%q) = %q, want %q", expression, got, want)
		}
	}
}
