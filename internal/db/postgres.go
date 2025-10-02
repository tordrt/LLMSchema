package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PostgresClient manages the connection to PostgreSQL
type PostgresClient struct {
	conn *pgx.Conn
}

// NewPostgresClient creates a new PostgreSQL client
func NewPostgresClient(ctx context.Context, connString string) (*PostgresClient, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresClient{conn: conn}, nil
}

// Close closes the database connection
func (c *PostgresClient) Close(ctx context.Context) error {
	return c.conn.Close(ctx)
}

// GetConnection returns the underlying connection
func (c *PostgresClient) GetConnection() *pgx.Conn {
	return c.conn
}
