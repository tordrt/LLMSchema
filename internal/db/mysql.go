package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLClient manages the connection to MySQL
type MySQLClient struct {
	db *sql.DB
}

// NewMySQLClient creates a new MySQL client
func NewMySQLClient(ctx context.Context, connString string) (*MySQLClient, error) {
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MySQLClient{db: db}, nil
}

// Close closes the database connection
func (c *MySQLClient) Close() error {
	return c.db.Close()
}

// GetDB returns the underlying database connection
func (c *MySQLClient) GetDB() *sql.DB {
	return c.db
}
