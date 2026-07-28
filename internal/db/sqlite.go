package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// SQLiteClient manages the connection to SQLite
type SQLiteClient struct {
	db           *sql.DB
	databaseName string
}

// NewSQLiteClient creates a new SQLite client
func NewSQLiteClient(ctx context.Context, path string) (*SQLiteClient, error) {
	querySeparator := "?"
	if strings.Contains(path, "?") {
		querySeparator = "&"
	}
	dsn := path + querySeparator + "_pragma=busy_timeout%285000%29"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteClient{
		db:           db,
		databaseName: sqliteDatabaseName(path),
	}, nil
}

// Close closes the database connection
func (c *SQLiteClient) Close() error {
	return c.db.Close()
}

// GetDB returns the underlying database connection
func (c *SQLiteClient) GetDB() *sql.DB {
	return c.db
}

// GetDatabaseName returns a display-safe name for the main SQLite database.
func (c *SQLiteClient) GetDatabaseName() string {
	return c.databaseName
}

func sqliteDatabaseName(path string) string {
	path, _, _ = strings.Cut(path, "?")
	path = strings.TrimPrefix(path, "file:")
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}
