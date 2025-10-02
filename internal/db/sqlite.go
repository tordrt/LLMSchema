package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteClient manages the connection to SQLite
type SQLiteClient struct {
	db *sql.DB
}

// NewSQLiteClient creates a new SQLite client
func NewSQLiteClient(ctx context.Context, path string) (*SQLiteClient, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteClient{db: db}, nil
}

// Close closes the database connection
func (c *SQLiteClient) Close() error {
	return c.db.Close()
}

// GetDB returns the underlying database connection
func (c *SQLiteClient) GetDB() *sql.DB {
	return c.db
}
