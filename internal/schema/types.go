package schema

// Schema represents a complete database schema
type Schema struct {
	Tables []Table
}

// Table represents a database table
type Table struct {
	Name       string
	Columns    []Column
	Relations  []Relation
	Indexes    []Index
	PrimaryKey []string
}

// Column represents a table column
type Column struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	IsUnique     bool
}

// Relation represents a foreign key relationship
type Relation struct {
	TargetTable  string
	TargetColumn string
	SourceColumn string
	Cardinality  string // 1:1, 1:N, N:1
}

// Index represents a database index
type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}
