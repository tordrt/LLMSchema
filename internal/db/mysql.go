package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-sql-driver/mysql"
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

// ParseDatabaseName extracts the database name from a MySQL connection string
func ParseDatabaseName(connString string) (string, error) {
	cfg, err := mysql.ParseDSN(connString)
	if err != nil {
		return "", fmt.Errorf("failed to parse MySQL connection string: %w", err)
	}

	if cfg.DBName == "" {
		return "", fmt.Errorf("no database name found in connection string")
	}

	return cfg.DBName, nil
}

// MySQLDSN converts a mysql:// URL into a Go MySQL driver DSN. It accepts both
// mysql://user:pass@host:3306/db and the driver form mysql://user:pass@tcp(host:3306)/db.
func MySQLDSN(databaseURL string) string {
	dsn := strings.TrimPrefix(databaseURL, "mysql://")
	if _, err := mysql.ParseDSN(dsn); err == nil {
		return dsn
	}

	u, err := url.Parse(databaseURL)
	if err != nil || u.Host == "" {
		// Let the driver report its own error for the original string.
		return dsn
	}

	cfg := mysql.NewConfig()
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	cfg.DBName = strings.TrimPrefix(u.Path, "/")
	dsn = cfg.FormatDSN()
	if u.RawQuery != "" {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn += separator + u.RawQuery
	}
	return dsn
}
