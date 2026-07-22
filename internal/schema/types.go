package schema

// Schema represents a complete database schema
type Schema struct {
	DatabaseType    string
	DatabaseVersion string
	Tables          []Table
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
	Name            string
	Type            string
	Nullable        bool
	DefaultValue    *string
	IsUnique        bool
	EnumValues      []string // For USER-DEFINED enum types
	CheckConstraint *string  // For CHECK constraints
}

// Relation represents a foreign key relationship
type Relation struct {
	Name          string
	TargetSchema  string
	TargetTable   string
	TargetColumns []string
	SourceColumns []string
	Cardinality   string // 1:1 or N:1, expressed from source to target
	OnUpdate      string
	OnDelete      string

	// Deprecated: use TargetColumns and SourceColumns. These aliases remain
	// populated for single-column relationships for API compatibility.
	TargetColumn string
	SourceColumn string
}

// Index represents a database index
type Index struct {
	Name           string
	Columns        []string
	IsUnique       bool
	IsPartial      bool // Conditional PostgreSQL or SQLite index
	HasExpressions bool // Columns is incomplete and cannot prove key uniqueness
}
