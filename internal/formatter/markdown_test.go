package formatter

import (
	"errors"
	"testing"

	"github.com/tordrt/llmschema/internal/schema"
)

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

func TestMarkdownFormatterPropagatesWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	s := &schema.Schema{Tables: []schema.Table{{Name: "users"}}}

	if err := f.Format(s); !errors.Is(err, errWriteFailed) {
		t.Fatalf("Format() error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarkdownFormattingMethodsPropagateWriteErrors(t *testing.T) {
	f := NewMarkdownFormatter(failingWriter{})
	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "columns",
			run: func() error {
				return f.FormatColumns(failingWriter{}, []schema.Column{{Name: "id", Type: "integer"}}, nil, nil)
			},
		},
		{
			name: "relations",
			run: func() error {
				return f.FormatRelations(failingWriter{}, "orders", []schema.Relation{{
					SourceColumn: "user_id",
					TargetTable:  "users",
					TargetColumn: "id",
				}})
			},
		},
		{
			name: "indexes",
			run: func() error {
				return f.FormatIndexes(failingWriter{}, []schema.Index{{Name: "users_pkey", Columns: []string{"id"}}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, errWriteFailed) {
				t.Fatalf("error = %v, want %v", err, errWriteFailed)
			}
		})
	}
}
